package admin

import (
	"errors"
	"testing"
	"time"

	"fitconecta/core-api/internal/cref"
)

func TestCREFCheckError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		reason string
	}{
		{name: "nao encontrado", err: cref.ErrNotFound, reason: "not_found"},
		{name: "inativo", err: cref.ErrInactive, reason: "inactive"},
		{name: "portal indisponivel", err: cref.ErrUnavailable, reason: "unavailable"},
		{name: "erro inesperado", err: errors.New("boom"), reason: "unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := crefCheckError(tt.err)
			if got.Reason != tt.reason {
				t.Errorf("reason = %q, want %q", got.Reason, tt.reason)
			}
			if got.Verified {
				t.Error("verified deveria ser false")
			}
			if got.Message == "" {
				t.Error("mensagem nao deveria ser vazia")
			}
		})
	}
}

func TestDecideCREFUpdate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		reason     string
		wantWrite  bool
		wantVerify bool
	}{
		{name: "verificado grava a data", reason: "ok", wantWrite: true, wantVerify: true},
		{name: "nao encontrado limpa o selo", reason: "not_found", wantWrite: true, wantVerify: false},
		{name: "inativo limpa o selo", reason: "inactive", wantWrite: true, wantVerify: false},
		{name: "portal fora nao altera", reason: "unavailable", wantWrite: false, wantVerify: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			write, verifiedAt := decideCREFUpdate(CREFCheck{Reason: tt.reason}, now)
			if write != tt.wantWrite {
				t.Errorf("write = %v, want %v", write, tt.wantWrite)
			}
			if tt.wantVerify {
				if verifiedAt == nil || !verifiedAt.Equal(now) {
					t.Errorf("verifiedAt = %v, want %v", verifiedAt, now)
				}
			} else if verifiedAt != nil {
				t.Errorf("verifiedAt = %v, want nil", verifiedAt)
			}
		})
	}
}
