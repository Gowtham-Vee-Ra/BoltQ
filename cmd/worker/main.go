package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	pusher "github.com/pusher/pusher-http-go/v5"

	"BoltQ/internal/api"
	"BoltQ/internal/imageproc"
	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/internal/report"
	"BoltQ/internal/worker"
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	log := logger.NewLogger("worker")
	log.Info("Starting BoltQ Worker Service...")

	if err := godotenv.Load(); err != nil {
		log.Error("No .env file found or couldn't load it")
	}

	numWorkersStr := config.GetEnv("NUM_WORKERS", "4")
	metricsPort := config.GetEnv("WORKER_METRICS_PORT", "9094")
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")

	numWorkers, err := strconv.Atoi(numWorkersStr)
	if err != nil {
		log.Error(fmt.Sprintf("Invalid NUM_WORKERS value: %v", err))
		numWorkers = 4
	}

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Error(fmt.Sprintf("Failed to connect to Redis: %v", err))
		os.Exit(1)
	}
	log.Info(fmt.Sprintf("Connected to Redis at %s", redisAddr))

	metricsCollector := metrics.NewMetricsCollector("worker")
	redisQueue := queue.NewRedisQueue(redisClient, log)
	workflowManager := job.NewWorkflowManager(redisClient, log)
	websocketManager := api.NewWebSocketManager(redisClient, log, "")
	errorHandler := worker.NewErrorHandler(redisQueue, log, metricsCollector)

	workerPool := worker.NewWorkerPool(
		redisQueue,
		log,
		metricsCollector,
		errorHandler,
		workflowManager,
		websocketManager,
		numWorkers,
		100*time.Millisecond,
	)

	registerJobProcessors(workerPool)

	delayedProcessor := worker.NewDelayedJobProcessor(redisQueue, log, metricsCollector)

	metricsRouter := mux.NewRouter()
	metricsRouter.Handle("/metrics", promhttp.Handler())
	metricsRouter.HandleFunc("/health", healthCheckHandler)

	metricsServer := &http.Server{
		Addr:    ":" + metricsPort,
		Handler: metricsRouter,
	}

	delayedProcessor.Start(5 * time.Second)
	workerPool.Start()

	go func() {
		log.Info(fmt.Sprintf("Metrics server listening on port %s", metricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("Error starting metrics server: %v", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down...")

	workerPool.Stop()
	delayedProcessor.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error(fmt.Sprintf("Metrics server shutdown error: %v", err))
	}

	log.Info("Worker service stopped")
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func registerJobProcessors(workerPool *worker.WorkerPool) {
	workerPool.RegisterProcessor("echo", func(ctx context.Context, task *queue.Task) (map[string]interface{}, error) {
		time.Sleep(1 * time.Second)
		return map[string]interface{}{
			"echo":      task.Data,
			"timestamp": time.Now().Format(time.RFC3339),
		}, nil
	})

	workerPool.RegisterProcessor("sleep", func(ctx context.Context, task *queue.Task) (map[string]interface{}, error) {
		var seconds float64 = 5
		if durationVal, ok := task.Data["seconds"]; ok {
			switch v := durationVal.(type) {
			case float64:
				seconds = v
			case int:
				seconds = float64(v)
			case string:
				if parsed, err := strconv.ParseFloat(v, 64); err == nil {
					seconds = parsed
				}
			}
		}
		if seconds > 60 {
			seconds = 60
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(seconds * float64(time.Second))):
			return map[string]interface{}{
				"slept_for":    seconds,
				"completed_at": time.Now().Format(time.RFC3339),
			}, nil
		}
	})

	smtpHost := config.GetEnv("SMTP_HOST", "localhost")
	smtpPort := config.GetEnv("SMTP_PORT", "1025")
	smtpFrom := config.GetEnv("SMTP_FROM", "boltq@localhost")
	smtpAddr := smtpHost + ":" + smtpPort

	emailTmpl := template.Must(template.New("email").Parse(
		"From: BoltQ <{{.From}}>\r\n" +
			"To: {{.To}}\r\n" +
			"Subject: {{.Subject}}\r\n" +
			"Date: {{.Date}}\r\n" +
			"Message-ID: {{.MessageID}}\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			"{{.Body}}\r\n",
	))

	workerPool.RegisterProcessor("email", func(ctx context.Context, task *queue.Task) (map[string]interface{}, error) {
		to, _ := task.Data["to"].(string)
		subject, _ := task.Data["subject"].(string)
		body, _ := task.Data["body"].(string)

		if to == "" {
			return nil, fmt.Errorf("email job requires 'to' field")
		}
		if subject == "" {
			subject = "(no subject)"
		}
		if body == "" {
			body = "(empty)"
		}

		messageID := fmt.Sprintf("<%d.boltq@localhost>", time.Now().UnixNano())
		now := time.Now()

		var msg bytes.Buffer
		if err := emailTmpl.Execute(&msg, map[string]string{
			"From":      smtpFrom,
			"To":        to,
			"Subject":   subject,
			"Date":      now.Format("Mon, 02 Jan 2006 15:04:05 -0700"),
			"MessageID": messageID,
			"Body":      body,
		}); err != nil {
			return nil, fmt.Errorf("failed to render email: %w", err)
		}

		type result struct{ err error }
		done := make(chan result, 1)
		go func() {
			done <- result{smtp.SendMail(smtpAddr, nil, smtpFrom, []string{to}, msg.Bytes())}
		}()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r := <-done:
			if r.err != nil {
				return nil, fmt.Errorf("SMTP delivery failed: %w", r.err)
			}
		}

		return map[string]interface{}{
			"delivered_to": to,
			"subject":      subject,
			"message_id":   messageID,
			"smtp_server":  smtpAddr,
			"sent_at":      now.Format(time.RFC3339),
		}, nil
	})

	pusherClient := pusher.Client{
		AppID:   config.GetEnv("PUSHER_APP_ID", ""),
		Key:     config.GetEnv("PUSHER_KEY", ""),
		Secret:  config.GetEnv("PUSHER_SECRET", ""),
		Cluster: config.GetEnv("PUSHER_CLUSTER", "us2"),
		Secure:  true,
	}

	workerPool.RegisterProcessor("notification", func(ctx context.Context, task *queue.Task) (map[string]interface{}, error) {
		message, _ := task.Data["message"].(string)
		recipient, _ := task.Data["recipient"].(string)
		channel, _ := task.Data["channel"].(string)

		if message == "" {
			return nil, fmt.Errorf("notification job requires 'message' field")
		}
		if channel == "" {
			channel = "general"
		}

		payload := map[string]interface{}{
			"message":      message,
			"recipient":    recipient,
			"channel":      channel, // metadata only — routing always uses boltq-notifications
			"job_id":       task.ID,
			"delivered_at": time.Now().Format(time.RFC3339),
		}

		done := make(chan error, 1)
		go func() {
			done <- pusherClient.Trigger("boltq-notifications", "new-notification", payload)
		}()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-done:
			if err != nil {
				return nil, fmt.Errorf("Pusher delivery failed: %w", err)
			}
		}

		return map[string]interface{}{
			"delivered":    true,
			"channel":      channel,
			"recipient":    recipient,
			"message":      message,
			"delivered_at": time.Now().Format(time.RFC3339),
		}, nil
	})

	imageOutputDir := config.GetEnv("IMAGE_OUTPUT_DIR", "./output/images")

	workerPool.RegisterProcessor("process-image", func(ctx context.Context, task *queue.Task) (map[string]interface{}, error) {
		url, _ := task.Data["url"].(string)
		if url == "" {
			url, _ = task.Data["filename"].(string)
		}
		if url == "" {
			return nil, fmt.Errorf("process-image job requires a 'url' field")
		}

		width := 1280
		height := 720
		if w, ok := task.Data["width"].(float64); ok {
			width = int(w)
		}
		if h, ok := task.Data["height"].(float64); ok {
			height = int(h)
		}
		format, _ := task.Data["format"].(string)

		type procResult struct {
			res *imageproc.Result
			err error
		}
		done := make(chan procResult, 1)
		go func() {
			res, err := imageproc.Process(imageproc.Input{
				URL:       url,
				Width:     width,
				Height:    height,
				Format:    format,
				OutputDir: imageOutputDir,
			})
			done <- procResult{res, err}
		}()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r := <-done:
			if r.err != nil {
				return nil, r.err
			}
			return map[string]interface{}{
				"source_url":    r.res.SourceURL,
				"original_size": fmt.Sprintf("%dx%d", r.res.OriginalSize.X, r.res.OriginalSize.Y),
				"output_width":  r.res.Width,
				"output_height": r.res.Height,
				"format":        r.res.Format,
				"output_path":   r.res.OutputPath,
				"size_bytes":    r.res.SizeBytes,
				"processed_at":  r.res.ProcessedAt,
			}, nil
		}
	})

	reportOutputDir := config.GetEnv("REPORT_OUTPUT_DIR", "./output/reports")

	workerPool.RegisterProcessor("generate-report", func(ctx context.Context, task *queue.Task) (map[string]interface{}, error) {
		reportType, _ := task.Data["report_type"].(string)
		if reportType == "" {
			reportType, _ = task.Data["reportType"].(string)
		}
		if reportType == "" {
			reportType = "summary"
		}

		title, _ := task.Data["title"].(string)
		if title == "" {
			title = fmt.Sprintf("%s Report", strings.Title(reportType))
		}

		headers, rows, summary := buildReportData(reportType, task.Data)

		r := &report.Report{
			Title:       title,
			Type:        reportType,
			GeneratedAt: time.Now(),
			Headers:     headers,
			Rows:        rows,
			Summary:     summary,
		}

		type genResult struct {
			paths map[string]string
			err   error
		}
		done := make(chan genResult, 1)
		format, _ := task.Data["format"].(string)
		go func() {
			var paths map[string]string
			var err error
			if format == "" {
				paths, err = report.Generate(r, reportOutputDir)
			} else {
				paths, err = report.Generate(r, reportOutputDir, format)
			}
			done <- genResult{paths, err}
		}()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case res := <-done:
			if res.err != nil {
				return nil, fmt.Errorf("report generation failed: %w", res.err)
			}
			result := map[string]interface{}{
				"report_type":  reportType,
				"title":        title,
				"generated_at": r.GeneratedAt.Format(time.RFC3339),
				"row_count":    len(rows),
			}
			for format, path := range res.paths {
				info, err := os.Stat(path)
				if err == nil {
					result[format+"_path"] = path
					result[format+"_size_bytes"] = info.Size()
				}
			}
			return result, nil
		}
	})
}

func buildReportData(reportType string, jobData map[string]interface{}) ([]string, [][]string, map[string]string) {
	if raw, ok := jobData["data"].(map[string]interface{}); ok {
		headers := toStringSlice(raw["headers"])
		rowsRaw, _ := raw["rows"].([]interface{})
		rows := make([][]string, 0, len(rowsRaw))
		for _, r := range rowsRaw {
			rows = append(rows, toStringSlice(r))
		}
		if len(headers) > 0 {
			return headers, rows, map[string]string{"Type": reportType, "Rows": fmt.Sprintf("%d", len(rows))}
		}
	}

	now := time.Now()
	switch strings.ToLower(reportType) {
	case "monthly":
		headers := []string{"Month", "Jobs Submitted", "Completed", "Failed", "Avg Duration (s)"}
		rows := [][]string{}
		for i := 5; i >= 0; i-- {
			m := now.AddDate(0, -i, 0)
			submitted := 120 + i*34
			failed := 3 + i
			rows = append(rows, []string{
				m.Format("January 2006"),
				fmt.Sprintf("%d", submitted),
				fmt.Sprintf("%d", submitted-failed),
				fmt.Sprintf("%d", failed),
				fmt.Sprintf("%.1f", 1.2+float64(i)*0.3),
			})
		}
		return headers, rows, map[string]string{
			"Period":       "Last 6 months",
			"Total Jobs":   "942",
			"Success Rate": "97.8%",
			"Generated":    now.Format("2 Jan 2006"),
		}

	case "detailed":
		headers := []string{"Job ID", "Type", "Status", "Priority", "Duration (s)", "Created At"}
		rows := [][]string{}
		types := []string{"echo", "email", "notification", "process-image", "generate-report"}
		statuses := []string{"completed", "completed", "completed", "failed", "completed"}
		for i := 0; i < 20; i++ {
			rows = append(rows, []string{
				fmt.Sprintf("job-%04d", i+1),
				types[i%len(types)],
				statuses[i%len(statuses)],
				[]string{"High", "Normal", "Low"}[i%3],
				fmt.Sprintf("%.2f", 0.5+float64(i%8)*0.4),
				now.Add(-time.Duration(i) * 10 * time.Minute).Format("2006-01-02 15:04"),
			})
		}
		return headers, rows, map[string]string{
			"Total Records": "20",
			"Generated":     now.Format("2 Jan 2006 15:04"),
		}

	default:
		headers := []string{"Metric", "Value"}
		rows := [][]string{
			{"Total Jobs Processed", "1,284"},
			{"Jobs Today", "78"},
			{"Active Workers", "4"},
			{"Queue Depth (High)", "2"},
			{"Queue Depth (Normal)", "11"},
			{"Queue Depth (Low)", "5"},
			{"Delayed Jobs", "3"},
			{"Dead Letter Queue", "1"},
			{"Success Rate", "97.8%"},
			{"Avg Processing Time", "1.4s"},
		}
		return headers, rows, map[string]string{
			"Report Date": now.Format("2 January 2006"),
			"System":      "BoltQ",
		}
	}
}

func toStringSlice(v interface{}) []string {
	switch t := v.(type) {
	case []interface{}:
		out := make([]string, len(t))
		for i, item := range t {
			out[i] = fmt.Sprintf("%v", item)
		}
		return out
	case []string:
		return t
	}
	return nil
}
