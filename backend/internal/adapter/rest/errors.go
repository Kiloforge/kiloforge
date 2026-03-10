package rest

import (
	"errors"
	"net/http"

	"kiloforge/internal/core/domain"
)

// mapServiceError classifies a service-layer error into an HTTP status code
// and a safe client-facing message. Internal details are never leaked.
func mapServiceError(err error) (int, string) {
	if err == nil {
		return http.StatusInternalServerError, "unknown error"
	}

	switch {
	// Not-found family.
	case errors.Is(err, domain.ErrProjectNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, domain.ErrAgentNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, domain.ErrBoardNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, domain.ErrCardNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, domain.ErrPRTrackingNotFound):
		return http.StatusNotFound, err.Error()

	// Validation.
	case errors.Is(err, domain.ErrInvalidColumn):
		return http.StatusUnprocessableEntity, err.Error()

	// Conflict.
	case errors.Is(err, domain.ErrProjectExists):
		return http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrPoolExhausted):
		return http.StatusConflict, err.Error()

	// Forbidden.
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, err.Error()

	// Default: internal error with safe message.
	default:
		return http.StatusInternalServerError, "internal error"
	}
}
