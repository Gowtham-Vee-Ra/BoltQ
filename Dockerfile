FROM golang:1.20
WORKDIR /app
COPY . .
RUN go mod tidy
CMD ["go", "run", "cmd/api/main.go"]
