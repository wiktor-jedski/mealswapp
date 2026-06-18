package compliance

// Implements DESIGN-015 DisclaimerRenderer verification.

import (
	"context"
	"errors"
	"testing"
)

type fakeDisclaimerProvider struct {
	content DisclaimerContent
	err     error
}

func (p fakeDisclaimerProvider) GetDisclaimer(context.Context, string) (DisclaimerContent, error) {
	return p.content, p.err
}

// TestDisclaimerService verifies DESIGN-015 DisclaimerRenderer behavior.
func TestDisclaimerService(t *testing.T) {
	ctx := context.Background()
	service := NewDisclaimerService(fakeDisclaimerProvider{content: DisclaimerContent{Version: "configured-v1", Markdown: "Configured content"}})
	content, err := service.GetDisclaimer(ctx, " Account ")
	if err != nil {
		t.Fatalf("GetDisclaimer() configured error = %v", err)
	}
	if content.Location != "account" || content.Version != "configured-v1" || content.Fallback {
		t.Fatalf("configured content = %#v", content)
	}
	fallback, err := NewDisclaimerService(fakeDisclaimerProvider{err: errors.New("down")}).GetDisclaimer(ctx, "login")
	if err != nil {
		t.Fatalf("GetDisclaimer() fallback error = %v", err)
	}
	if !fallback.Fallback || fallback.Alert != "configured_disclaimer_unavailable" || fallback.Markdown == "" {
		t.Fatalf("fallback content = %#v", fallback)
	}
	if _, err := service.GetDisclaimer(ctx, "checkout"); err == nil {
		t.Fatal("GetDisclaimer() accepted invalid location")
	}
	defaulted, err := NewDisclaimerService(fakeDisclaimerProvider{}).GetDisclaimer(ctx, "")
	if err != nil || defaulted.Location != "login" || !defaulted.Fallback || defaulted.Markdown != fallbackLoginDisclaimer {
		t.Fatalf("default disclaimer=%+v err=%v", defaulted, err)
	}
	account, err := NewDisclaimerService(nil).GetDisclaimer(ctx, "account")
	if err != nil || account.Markdown != fallbackAccountDisclaimer {
		t.Fatalf("account fallback=%+v err=%v", account, err)
	}
}
