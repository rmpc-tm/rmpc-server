package validate

import (
	"strings"
	"testing"
)

// The validator names Go struct fields by default. This package rewires it to report the JSON name instead.
func TestFormatErrorUsesJSONFieldNames(t *testing.T) {
	type request struct {
		GameMode string `json:"game_mode" validate:"required"`
	}

	err := Struct(request{})
	if err == nil {
		t.Fatal("expected a validation error")
	}

	msg := FormatError(err)
	if !strings.Contains(msg, "game_mode") {
		t.Fatalf("message %q should name the json field game_mode", msg)
	}
	if strings.Contains(msg, "GameMode") {
		t.Fatalf("message %q leaks the Go field name", msg)
	}
}
