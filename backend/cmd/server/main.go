package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"persboard/backend/internal/platform"
	"persboard/backend/internal/repository/postgres"
	"persboard/backend/internal/service"
	"persboard/backend/internal/transport/httpapi"
)

func main() {
	port := getEnv("PORT", "8080")

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

	repository := postgres.NewRepository(db)
	orgService := service.NewOrgService(repository)
	handler := httpapi.NewHandler(orgService)

	metricsDefs, err := service.LoadCalendarMetricsFromEnv()
	if err != nil {
		log.Fatalf("failed to load CALENDAR_METRICS_JSON: %v", err)
	}
	eazyClient, err := service.BuildEazyBIClientFromEnv()
	if err != nil {
		log.Fatalf("failed to build EazyBI client: %v", err)
	}
	calendarSvc := service.NewCalendarService(repository, eazyClient, metricsDefs)
	calendarHandler := httpapi.NewCalendarHandler(calendarSvc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	calendarHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           platform.WithCORS(mux, getEnv("CORS_ORIGIN", "http://localhost:5173")),
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	log.Printf("backend started on :%s", port)
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
