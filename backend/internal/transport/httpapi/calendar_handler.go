package httpapi

import (
	"context"
	"net/http"
	"time"

	"persboard/backend/internal/domain"
	"persboard/backend/internal/service"
)

type CalendarHandler struct {
	svc *service.CalendarService
}

func NewCalendarHandler(svc *service.CalendarService) *CalendarHandler {
	return &CalendarHandler{svc: svc}
}

func (h *CalendarHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/calendar/metrics", h.getCalendarMetrics)
	mux.HandleFunc("/api/v1/calendar/metric-weights", h.updateMetricWeight)
}

func (h *CalendarHandler) getCalendarMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, to, err := parseDateRange(fromStr, toStr)
	if err != nil {
		badRequest(w, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	resp, err := h.svc.CalendarMetrics(ctx, from, to)
	if err != nil {
		if _, ok := err.(service.ValidationError); ok {
			badRequest(w, err.Error())
			return
		}
		internalError(w)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *CalendarHandler) updateMetricWeight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		methodNotAllowed(w, http.MethodPut)
		return
	}

	var input domain.UpdateMetricWeightInput
	if err := decodeJSONBody(w, r, &input, 64<<10); err != nil {
		badRequest(w, "invalid JSON body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	weight, err := h.svc.UpdateMetricWeight(ctx, input)
	if err != nil {
		if _, ok := err.(service.ValidationError); ok {
			badRequest(w, err.Error())
			return
		}
		internalError(w)
		return
	}

	writeJSON(w, http.StatusOK, weight)
}

func parseDateRange(fromStr, toStr string) (time.Time, time.Time, error) {
	parse := func(s string) (time.Time, error) {
		if s == "" {
			return time.Time{}, nil
		}
		return time.Parse("2006-01-02", s)
	}

	now := time.Now().UTC()
	from := now.AddDate(0, 0, -6)
	to := now

	if d, err := parse(fromStr); err != nil {
		return time.Time{}, time.Time{}, err
	} else if !d.IsZero() {
		from = d
	}
	if d, err := parse(toStr); err != nil {
		return time.Time{}, time.Time{}, err
	} else if !d.IsZero() {
		to = d
	}

	if from.After(to) {
		return time.Time{}, time.Time{}, service.ValidationError{Message: "from must be <= to"}
	}

	days := int(to.Sub(from).Hours()/24) + 1
	if days < 1 || days > 31 {
		return time.Time{}, time.Time{}, service.ValidationError{Message: "date range must be between 1 and 31 days"}
	}

	return from, to, nil
}
