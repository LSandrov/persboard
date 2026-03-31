package main

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	dbmigrations "persboard/backend/migrations"
	"persboard/backend/internal/platform"
	"persboard/backend/internal/repository/postgres"
	"persboard/backend/internal/service"
	"persboard/backend/internal/transport/grpcapi"
	orgusecase "persboard/backend/internal/usecase/org"
)

func main() {
	port := getEnv("PORT", "8080")
	logDir := getEnv("LOG_DIR", ".docker/logs")
	debugLog := strings.EqualFold(getEnv("LOG_DEBUG", "false"), "true") ||
		strings.EqualFold(getEnv("LOG_LEVEL", ""), "debug")

	var accessLog *os.File
	var appLog *os.File
	if strings.TrimSpace(logDir) != "" {
		var errLog error
		accessLog, appLog, errLog = platform.InitLogging(logDir, debugLog)
		if errLog != nil {
			fallbackDir := "/tmp/persboard-logs"
			accessLog, appLog, errLog = platform.InitLogging(fallbackDir, debugLog)
			if errLog != nil {
				log.Printf("warning: file logging disabled (%v); using stderr only for slog", errLog)
				level := slog.LevelInfo
				if debugLog {
					level = slog.LevelDebug
				}
				slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
			} else {
				log.Printf("warning: log dir %q unavailable, using fallback %q", logDir, fallbackDir)
				logDir = fallbackDir
				defer func() { _ = accessLog.Close() }()
				defer func() { _ = appLog.Close() }()
			}
		} else {
			defer func() { _ = accessLog.Close() }()
			defer func() { _ = appLog.Close() }()
		}
	} else {
		level := slog.LevelInfo
		if debugLog {
			level = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
	}

	db, err := platform.ConnectDB(platform.DBConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		Name:     getEnv("DB_NAME", "persboard"),
		User:     getEnv("DB_USER", "persboard"),
		Password: getEnv("DB_PASSWORD", "persboard"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	migrationCtx, cancelMigrations := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelMigrations()
	appliedMigrations, err := dbmigrations.Apply(migrationCtx, db)
	if err != nil {
		log.Fatalf("failed to apply database migrations: %v", err)
	}
	if len(appliedMigrations) > 0 {
		log.Printf("applied migrations: %v", appliedMigrations)
	}

	repository := postgres.NewRepository(db)
	orgUC := orgusecase.NewUseCase(repository)

	metricsDefs, err := service.LoadCalendarMetricsFromEnv()
	if err != nil {
		log.Fatalf("failed to load CALENDAR_METRICS_JSON: %v", err)
	}
	eazyClient, err := service.BuildEazyBIClientFromEnv()
	if err != nil {
		log.Fatalf("failed to build EazyBI client: %v", err)
	}
	calendarSvc := service.NewCalendarService(repository, eazyClient, metricsDefs)

	grpcPort := getEnv("GRPC_PORT", "9090")
	grpcAddr := ":" + grpcPort
	grpcEndpoint := getEnv("GRPC_ENDPOINT", "127.0.0.1:"+grpcPort)
	go func() {
		if err := grpcapi.StartGRPCServer(grpcAddr, orgUC, calendarSvc); err != nil {
			log.Fatalf("grpc server error: %v", err)
		}
	}()

	mux := http.NewServeMux()

	gatewayCtx, cancelGateway := context.WithCancel(context.Background())
	defer cancelGateway()
	gatewayMux, err := grpcapi.NewGatewayMux(gatewayCtx, grpcEndpoint)
	if err != nil {
		log.Fatalf("failed to init grpc gateway: %v", err)
	}
	mux.Handle("/", gatewayMux)

	core := platform.WithRequestID(mux)
	core = platform.WithCORS(core, getEnv("CORS_ORIGIN", "http://localhost:5173"))

	accessWriter := io.Writer(io.Discard)
	if accessLog != nil {
		accessWriter = accessLog
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           platform.WithAccessLog(accessWriter, core),
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	slog.Info("backend starting", "http_addr", ":"+port, "grpc_addr", grpcAddr, "log_dir", logDir, "debug", debugLog)
	log.Printf("backend started on :%s", port)
	log.Printf("grpc started on %s (gateway -> %s)", grpcAddr, grpcEndpoint)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
