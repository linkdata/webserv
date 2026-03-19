package webserv

import (
	"errors"
	"testing"
)

func TestErrServePanic_UnwrapsRecoveredError(t *testing.T) {
	panicErr := errors.New("panic boom")
	err := newErrServePanic(panicErr)
	if got, want := err.Error(), "ServeWith(): panic in http.Server.Serve: panic boom"; got != want {
		t.Fatalf("error text = %q, want %q", got, want)
	}
	if !errors.Is(err, ErrServePanic) {
		t.Fatalf("expected errors.Is(err, ErrServePanic), got: %v", err)
	}
	if !errors.Is(err, panicErr) {
		t.Fatalf("expected errors.Is(err, panicErr), got: %v", err)
	}
	var got errServePanic
	if !errors.As(err, &got) {
		t.Fatalf("expected errors.As(err, errServePanic), got: %v", err)
	}
	if got.recovered != panicErr {
		t.Fatalf("stored recovered value = %#v, want %#v", got.recovered, panicErr)
	}
}

func TestErrServePanic_UnwrapsNilForNonErrorRecoveredValue(t *testing.T) {
	err := newErrServePanic("panic boom")
	if got, want := err.Error(), "ServeWith(): panic in http.Server.Serve: panic boom"; got != want {
		t.Fatalf("error text = %q, want %q", got, want)
	}
	if !errors.Is(err, ErrServePanic) {
		t.Fatalf("expected errors.Is(err, ErrServePanic), got: %v", err)
	}
	if errors.Unwrap(err) != nil {
		t.Fatalf("expected nil unwrap for non-error recovered value, got: %v", errors.Unwrap(err))
	}
	var got errServePanic
	if !errors.As(err, &got) {
		t.Fatalf("expected errors.As(err, errServePanic), got: %v", err)
	}
	if recovered, ok := got.recovered.(string); !ok || recovered != "panic boom" {
		t.Fatalf("stored recovered value = %#v, want %q", got.recovered, "panic boom")
	}
}
