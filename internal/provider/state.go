package provider

import (
	"context"
	"errors"
	"net/http"
)

// Classify maps provider errors into stable UI states without exposing bodies.
func Classify(ctx context.Context, err error) State {
	if err == nil {
		return StateAvailable
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return StateCanceled
	}
	if errors.Is(err, ErrNoToken) {
		return StateNotConfigured
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.Status {
		case http.StatusUnauthorized:
			return StateUnauthorized
		case http.StatusForbidden, http.StatusTooManyRequests:
			return StateRateLimited
		}
	}
	if errors.Is(err, ErrRateLimited) {
		return StateRateLimited
	}
	if errors.Is(err, ErrProviderUnavailable) {
		return StateUnavailable
	}
	return StateMalformed
}
