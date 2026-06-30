package imageproc

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

type Input struct {
	URL       string
	Width     int
	Height    int
	Format    string // "jpeg" or "png"
	OutputDir string
}

type Result struct {
	SourceURL    string
	Width        int
	Height       int
	Format       string
	OutputPath   string
	SizeBytes    int64
	ProcessedAt  string
	OriginalSize image.Point
}

func Process(in Input) (*Result, error) {
	if in.Width <= 0 {
		in.Width = 1280
	}
	if in.Height <= 0 {
		in.Height = 720
	}
	in.Format = normaliseFormat(in.Format)
	if in.OutputDir == "" {
		in.OutputDir = "./output/images"
	}
	if err := os.MkdirAll(in.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create output dir: %w", err)
	}

	// Only fetch over http(s); reject file://, gopher://, data://, etc.
	parsed, err := url.Parse(in.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", parsed.Scheme)
	}

	// CDNs like Wikimedia block Go's default User-Agent
	req, err := http.NewRequest(http.MethodGet, in.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; BoltQ/1.0; +https://github.com/Gowtham-Vee-Ra/BoltQ)")

	// safeHTTPClient blocks requests that resolve to private/loopback/link-local
	// addresses (SSRF guard) and bounds the fetch with a timeout.
	resp, err := safeHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch returned HTTP %d", resp.StatusCode)
	}

	// Bound how many bytes we read so a huge/slow response can't exhaust memory.
	body := io.LimitReader(resp.Body, maxFetchBytes+1)

	// imaging decodes JPEG, PNG, GIF, BMP, TIFF, WebP; only JPEG/PNG for output
	src, err := imaging.Decode(body, imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	origBounds := src.Bounds()

	resized := imaging.Resize(src, in.Width, in.Height, imaging.Lanczos)

	ext := "." + in.Format
	filename := fmt.Sprintf("img_%d%s", time.Now().UnixNano(), ext)
	outPath := filepath.Join(in.OutputDir, filename)

	f, err := os.Create(outPath)
	if err != nil {
		return nil, fmt.Errorf("cannot create output file: %w", err)
	}
	defer f.Close()

	switch in.Format {
	case "jpeg":
		if err := jpeg.Encode(f, resized, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("JPEG encode failed: %w", err)
		}
	case "png":
		if err := png.Encode(f, resized); err != nil {
			return nil, fmt.Errorf("PNG encode failed: %w", err)
		}
	}

	info, err := os.Stat(outPath)
	if err != nil {
		return nil, err
	}

	return &Result{
		SourceURL:    in.URL,
		Width:        in.Width,
		Height:       in.Height,
		Format:       in.Format,
		OutputPath:   outPath,
		SizeBytes:    info.Size(),
		ProcessedAt:  time.Now().Format(time.RFC3339),
		OriginalSize: image.Point{X: origBounds.Max.X, Y: origBounds.Max.Y},
	}, nil
}

func normaliseFormat(f string) string {
	switch strings.ToLower(strings.TrimSpace(f)) {
	case "jpg", "jpeg":
		return "jpeg"
	case "png":
		return "png"
	default:
		return "jpeg"
	}
}
