package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/closeloopautomous/arms/internal/domain"
)

type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, errorBody{Error: msg, Code: code})
}

func mapDomainErr(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return true
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return true
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", err.Error())
		return true
	case errors.Is(err, domain.ErrNotMergeQueueHead):
		writeError(w, http.StatusConflict, "merge_queue_head", err.Error())
		return true
	case errors.Is(err, domain.ErrInvalidTransition):
		writeError(w, http.StatusBadRequest, "invalid_transition", err.Error())
		return true
	case errors.Is(err, domain.ErrBudgetExceeded):
		writeError(w, http.StatusPaymentRequired, "budget_exceeded", err.Error())
		return true
	case errors.Is(err, domain.ErrGateway):
		writeError(w, http.StatusBadGateway, "gateway", err.Error())
		return true
	case errors.Is(err, domain.ErrShipping):
		writeError(w, http.StatusBadGateway, "shipping", err.Error())
		return true
	case errors.Is(err, domain.ErrMergeConflict):
		writeError(w, http.StatusConflict, "merge_conflict", err.Error())
		return true
	case errors.Is(err, domain.ErrMergeShipBusy):
		writeError(w, http.StatusServiceUnavailable, "merge_lease_busy", err.Error())
		return true
	default:
		return false
	}
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
