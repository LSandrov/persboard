package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"persboard/backend/internal/domain"
	"persboard/backend/internal/service"
)

type Handler struct {
	service *service.OrgService
}

func NewHandler(service *service.OrgService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", h.health)
	mux.HandleFunc("/api/v1/dashboard/metrics", h.dashboardMetrics)
	mux.HandleFunc("/api/v1/people/stats", h.peopleStats)
	mux.HandleFunc("/api/v1/org-structure", h.orgStructure)
	mux.HandleFunc("/api/v1/teams", h.createTeam)
	mux.HandleFunc("/api/v1/people", h.createPerson)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	status, err := h.service.Health(ctx)
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

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := h.service.DashboardMetrics(ctx)
	if err != nil {
		internalError(w)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) peopleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := h.service.PeopleStats(ctx)
	if err != nil {
		internalError(w)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) orgStructure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	resp, err := h.service.OrgStructure(ctx)
	if err != nil {
		internalError(w)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) createTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var input domain.CreateTeamInput
	if err := decodeJSONBody(w, r, &input, 1<<20); err != nil {
		badRequest(w, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	id, err := h.service.CreateTeam(ctx, input)
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

	var input domain.CreatePersonInput
	if err := decodeJSONBody(w, r, &input, 1<<20); err != nil {
		badRequest(w, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	id, err := h.service.CreatePerson(ctx, input)
	if err != nil {
		writeKnownError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       id,
		"fullName": input.FullName,
	})
}

func writeKnownError(w http.ResponseWriter, err error) {
	var validationErr service.ValidationError
	if errors.As(err, &validationErr) {
		badRequest(w, validationErr.Message)
		return
	}
	internalError(w)
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

func internalError(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}
