package task

import (
	"errors"
	"fmt"
	"testing"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func TestShouldRetryPullRequestCreate(t *testing.T) {
	if shouldRetryPullRequestCreate(nil) {
		t.Fatal("nil should not retry")
	}
	if shouldRetryPullRequestCreate(fmt.Errorf("%w: bad input", domain.ErrInvalidInput)) {
		t.Fatal("invalid input should not retry")
	}
	transient := fmt.Errorf("%w: timeout", domain.ErrShipping)
	if !shouldRetryPullRequestCreate(transient) {
		t.Fatal("plain ErrShipping should retry")
	}
	authLike := errors.Join(
		domain.ErrShippingNonRetryable,
		fmt.Errorf("%w: unauthorized (check token scopes: repo)", domain.ErrShipping),
	)
	if shouldRetryPullRequestCreate(authLike) {
		t.Fatal("ErrShippingNonRetryable should not retry")
	}
}
