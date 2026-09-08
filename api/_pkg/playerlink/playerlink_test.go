package playerlink

import (
	"testing"

	"rmpc-server/api/_pkg/config"
)

// An unset secret must produce no token and verify nothing.
func TestNoSecretSignsNothingAndVerifiesNothing(t *testing.T) {
	prev := config.Env.PlayerLinkSecret
	config.Env.PlayerLinkSecret = ""
	t.Cleanup(func() { config.Env.PlayerLinkSecret = prev })

	if got := Sign("abc-123"); got != "" {
		t.Fatalf("Sign without a secret returned %q, want empty", got)
	}
	if Verify("abc-123", "") {
		t.Fatal("Verify must reject an empty signature when no secret is set")
	}
	if Verify("abc-123", "AAAAAAAA") {
		t.Fatal("Verify must reject any signature when no secret is set")
	}
}
