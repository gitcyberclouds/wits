package target

import (
	"errors"
	"fmt"
	"testing"
)

// Resolvers return these sentinels (sometimes wrapped with %w) so callers branch
// on errors.Is rather than matching message text. This pins that the wrapping is
// transparent and the sentinels stay distinct.
func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	if errors.Is(ErrUnsupported, ErrSocketOwnerUnknown) {
		t.Error("ErrUnsupported and ErrSocketOwnerUnknown must be distinct")
	}

	// Mirrors how ResolveFile wraps ErrUnsupported with context.
	wrapped := fmt.Errorf("finding process by file is %w", ErrUnsupported)
	if !errors.Is(wrapped, ErrUnsupported) {
		t.Errorf("errors.Is should see ErrUnsupported through wrapping; got %q", wrapped)
	}
	if errors.Is(wrapped, ErrSocketOwnerUnknown) {
		t.Error("wrapped ErrUnsupported must not match ErrSocketOwnerUnknown")
	}
}
