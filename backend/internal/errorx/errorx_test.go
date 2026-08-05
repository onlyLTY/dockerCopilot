package errorx

import (
	"errors"
	"testing"
)

func TestAsCodeError(t *testing.T) {
	err := NewCodeError(400, "bad")
	ce, ok := AsCodeError(err)
	if !ok || ce.Code != 400 || ce.Msg != "bad" {
		t.Fatalf("AsCodeError: %+v ok=%v", ce, ok)
	}
	if _, ok := AsCodeError(errors.New("x")); ok {
		t.Fatal("expected false for plain error")
	}
}

func TestErrDockerUnavailable(t *testing.T) {
	if !IsDockerUnavailable(ErrDockerUnavailable) {
		t.Fatal("expected docker unavailable")
	}
	if IsDockerUnavailable(NewCodeError(500, "x")) {
		t.Fatal("500 should not be docker unavailable")
	}
}
