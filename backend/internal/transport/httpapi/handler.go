package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"persboard/backend/internal/domain"
	"persboard/backend/internal/platform"
	orgusecase "persboard/backend/internal/usecase/org"
)

type Handler struct {
	usecase orgUseCase
}

type orgUseCase interface {
	Health(ctx context.Context) (map[string]string, error)
	DashboardMetrics(ctx context.Context) (domain.DashboardResponse, error)
	PeopleStats(ctx context.Context) (domain.PersonStats, error)
	OrgStructure(ctx context.Context) (domain.OrgStructureResponse, error)
	CreateTeam(ctx context.Context, input domain.CreateTeamInput) (int, error)
	UpdateTeam(ctx context.Context, id int, input domain.UpdateTeamInput) error
	DeleteTeam(ctx context.Context, id int) error
	CreatePerson(ctx context.Context, input domain.CreatePersonInput) (int, error)
	UpdatePerson(ctx context.Context, id int, input domain.UpdatePersonInput) error
	DeletePerson(ctx context.Context, id int) error
}

func NewHandler(usecase orgUseCase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", h.health)
	mux.HandleFunc("/api/v1/dashboard/metrics", h.dashboardMetrics)
	mux.HandleFunc("/api/v1/people/stats", h.peopleStats)
	mux.HandleFunc("/api/v1/org-structure", h.orgStructure)
	mux.HandleFunc("/api/v1/teams", h.createTeam)
	mux.HandleFunc("/api/v1/teams/", h.teamByID)
	mux.HandleFunc("/api/v1/people", h.createPerson)
	mux.HandleFunc("/api/v1/people/", h.personByID)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	slog.DebugContext(r.Context(), "api handler", "route", "/api/health", "method", r.Method, "req_id", platform.RequestIDFromContext(r.Context()))
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	status, err := h.usecase.Health(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, status)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) dashboardMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	slog.DebugContext(r.Context(), "api handler", "route", "/api/v1/dashboard/metrics", "method", r.Method, "req_id", platform.RequestIDFromContext(r.Context()))

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := h.usecase.DashboardMetrics(ctx)
	if err != nil {
		internalErrorLogged(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) peopleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	slog.DebugContext(r.Context(), "api handler", "route", "/api/v1/people/stats", "method", r.Method, "req_id", platform.RequestIDFromContext(r.Context()))

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := h.usecase.PeopleStats(ctx)
	if err != nil {
		internalErrorLogged(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) orgStructure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	slog.DebugContext(r.Context(), "api handler", "route", "/api/v1/org-structure", "method", r.Method, "req_id", platform.RequestIDFromContext(r.Context()))

	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	resp, err := h.usecase.OrgStructure(ctx)
	if err != nil {
		internalErrorLogged(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) createTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	slog.DebugContext(r.Context(), "api handler", "route", "/api/v1/teams", "method", r.Method, "req_id", platform.RequestIDFromContext(r.Context()))

	var input domain.CreateTeamInput
	if err := decodeJSONBody(w, r, &input, 1<<20); err != nil {
		badRequest(w, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	id, err := h.usecase.CreateTeam(ctx, input)
	if err != nil {
		writeKnownError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":   id,
		"name": input.Name,
	})
}

func (h *Handler) createPerson(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	slog.DebugContext(r.Context(), "api handler", "route", "/api/v1/people", "method", r.Method, "req_id", platform.RequestIDFromContext(r.Context()))

	var input domain.CreatePersonInput
	if err := decodeJSONBody(w, r, &input, 1<<20); err != nil {
		badRequest(w, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	id, err := h.usecase.CreatePerson(ctx, input)
	if err != nil {
		writeKnownError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       id,
		"fullName": input.FullName,
	})
}

func (h *Handler) teamByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r.URL.Path, "/api/v1/teams/")
	if err != nil {
		badRequest(w, err.Error())
		return
	}

	switch r.Method {
	case http.MethodPut:
		slog.DebugContext(r.Context(), "api handler", "route", "/api/v1/teams/:id", "method", r.Method, "req_id", platform.RequestIDFromContext(r.Context()), "id", id)
		var input domain.UpdateTeamInput
		if err := decodeJSONBody(w, r, &input, 1<<20); err != nil {
			badRequest(w, err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := h.usecase.UpdateTeam(ctx, id, input); err != nil {
			writeKnownError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"id": id, "name": input.Name})
	case http.MethodDelete:
		slog.DebugContext(r.Context(), "api handler", "route", "/api/v1/teams/:id", "method", r.Method, "req_id", platform.RequestIDFromContext(r.Context()), "id", id)
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := h.usecase.DeleteTeam(ctx, id); err != nil {
			writeKnownError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, http.MethodPut+", "+http.MethodDelete)
	}
}

func (h *Handler) personByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r.URL.Path, "/api/v1/people/")
	if err != nil {
		badRequest(w, err.Error())
		return
	}

	switch r.Method {
	case http.MethodPut:
		slog.DebugContext(r.Context(), "api handler", "route", "/api/v1/people/:id", "method", r.Method, "req_id", platform.RequestIDFromContext(r.Context()), "id", id)
		var input domain.UpdatePersonInput
		if err := decodeJSONBody(w, r, &input, 1<<20); err != nil {
			badRequest(w, err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := h.usecase.UpdatePerson(ctx, id, input); err != nil {
			writeKnownError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"id": id, "fullName": input.FullName})
	case http.MethodDelete:
		slog.DebugContext(r.Context(), "api handler", "route", "/api/v1/people/:id", "method", r.Method, "req_id", platform.RequestIDFromContext(r.Context()), "id", id)
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := h.usecase.DeletePerson(ctx, id); err != nil {
			writeKnownError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, http.MethodPut+", "+http.MethodDelete)
	}
}

func writeKnownError(w http.ResponseWriter, err error) {
	var validationErr orgusecase.ValidationError
	if errors.As(err, &validationErr) {
		badRequest(w, validationErr.Message)
		return
	}
	var notFoundErr orgusecase.NotFoundError
	if errors.As(err, &notFoundErr) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": notFoundErr.Message})
		return
	}
	internalErrorLogged(w, err)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body")
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("JSON body must contain a single object")
	}
	return nil
}

func badRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": message})
}

func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed+", OPTIONS")
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func parseIDFromPath(path, prefix string) (int, error) {
	if !strings.HasPrefix(path, prefix) {
		return 0, fmt.Errorf("invalid path")
	}
	rawID := strings.TrimPrefix(path, prefix)
	if rawID == "" || strings.Contains(rawID, "/") {
		return 0, fmt.Errorf("id must be a positive integer")
	}
	id, err := strconv.Atoi(rawID)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("id must be a positive integer")
	}
	return id, nil
}

func internalErrorLogged(w http.ResponseWriter, err error) {
	if err != nil {
		slog.Error("internal server error", "err", err)
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}
