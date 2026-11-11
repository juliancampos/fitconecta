package crn

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// cnnRow monta uma linha da resposta falsa no formato da CNN.
type cnnRow struct {
	Nome         string `json:"nome"`
	Registro     string `json:"registro"`
	CRN          int    `json:"crn"`
	DataCadastro string `json:"data_cadastro"`
	Situacao     string `json:"situacao"`
	TipoRegistro string `json:"tipo_registro"`
}

func TestValidate(t *testing.T) {
	nine := "9"
	tests := []struct {
		name         string
		query        string
		rows         []cnnRow
		want         *Profile
		wantErr      error
		wantRegistro string
		wantRegional *string
	}{
		{
			name:  "encontra o numero exato entre resultados de prefixo",
			query: "39104",
			rows: []cnnRow{
				{Nome: "OUTRA PESSOA", Registro: "391045", CRN: 9, Situacao: "ATIVO", TipoRegistro: "NUTRICIONISTA DEFINITIVO"},
				{Nome: "JULIA VIANA RESENDE", Registro: "39104", CRN: 9, DataCadastro: "10-11-2026", Situacao: "ATIVO", TipoRegistro: "NUTRICIONISTA DEFINITIVO"},
			},
			want:         &Profile{Number: "CRN9 39104", Name: "JULIA VIANA RESENDE"},
			wantRegistro: "39104",
		},
		{
			name:  "prefixo regional CRN9 filtra o regional certo",
			query: "CRN9 39104",
			rows: []cnnRow{
				{Nome: "PESSOA DO CRN3", Registro: "39104", CRN: 3, Situacao: "ATIVO", TipoRegistro: "NUTRICIONISTA DEFINITIVO"},
				{Nome: "JULIA VIANA RESENDE", Registro: "39104", CRN: 9, Situacao: "ATIVO", TipoRegistro: "NUTRICIONISTA DEFINITIVO"},
			},
			want:         &Profile{Number: "CRN9 39104", Name: "JULIA VIANA RESENDE"},
			wantRegistro: "39104",
			wantRegional: &nine,
		},
		{
			name:  "prefixo regional 9-39104 filtra o regional certo",
			query: "9-39104",
			rows: []cnnRow{
				{Nome: "PESSOA DO CRN3", Registro: "39104", CRN: 3, Situacao: "ATIVO", TipoRegistro: "NUTRICIONISTA DEFINITIVO"},
				{Nome: "JULIA VIANA RESENDE", Registro: "39104", CRN: 9, Situacao: "ATIVO", TipoRegistro: "NUTRICIONISTA DEFINITIVO"},
			},
			want:         &Profile{Number: "CRN9 39104", Name: "JULIA VIANA RESENDE"},
			wantRegistro: "39104",
			wantRegional: &nine,
		},
		{
			name:         "aceita registro com zero a esquerda na CNN",
			query:        "CRN9 506",
			rows:         []cnnRow{{Nome: "NUTRI SEED", Registro: "0506", CRN: 9, Situacao: "ATIVO", TipoRegistro: "NUTRICIONISTA DEFINITIVO"}},
			want:         &Profile{Number: "CRN9 0506", Name: "NUTRI SEED"},
			wantRegistro: "506",
			wantRegional: &nine,
		},
		{
			name:    "numero inexistente",
			query:   "39104",
			rows:    nil,
			wantErr: ErrNotFound,
		},
		{
			name:    "situacao cancelado",
			query:   "39104",
			rows:    []cnnRow{{Nome: "JULIA VIANA RESENDE", Registro: "39104", CRN: 9, Situacao: "CANCELADO", TipoRegistro: "NUTRICIONISTA DEFINITIVO"}},
			wantErr: ErrInactive,
		},
		{
			name:    "linha AVULSO-PF com o mesmo registro e ignorada",
			query:   "39104",
			rows:    []cnnRow{{Nome: "PRODUTOR RURAL", Registro: "39104", CRN: 9, Situacao: "ATIVO", TipoRegistro: "AVULSO-PF"}},
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody consultaRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Errorf("decode do corpo: %v", err)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
				}
				_ = json.NewEncoder(w).Encode(struct {
					Success bool     `json:"success"`
					Data    []cnnRow `json:"data"`
				}{Success: true, Data: tt.rows})
			}))
			defer server.Close()

			client := NewClient()
			client.endpoint = server.URL
			got, err := client.Validate(context.Background(), tt.query)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Validate() erro = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() erro inesperado: %v", err)
			}
			if *got != *tt.want {
				t.Errorf("Validate() = %+v, want %+v", got, tt.want)
			}
			if gotBody.Comando != "get-nutricionista" {
				t.Errorf("comando = %q, want get-nutricionista", gotBody.Comando)
			}
			if gotBody.Options.Registro != tt.wantRegistro {
				t.Errorf("CNN recebeu registro %q, want %q", gotBody.Options.Registro, tt.wantRegistro)
			}
			if regional := gotBody.Options.CRN; (regional == nil) != (tt.wantRegional == nil) || (regional != nil && *regional != *tt.wantRegional) {
				t.Errorf("CNN recebeu crn %v, want %v", regional, tt.wantRegional)
			}
		})
	}
}

func TestValidateCNNIndisponivel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	client.endpoint = server.URL
	if _, err := client.Validate(context.Background(), "39104"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Validate() erro = %v, want ErrUnavailable", err)
	}
}

func TestValidateCNNSuccessFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(consultaResponse{Success: false})
	}))
	defer server.Close()

	client := NewClient()
	client.endpoint = server.URL
	if _, err := client.Validate(context.Background(), "39104"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Validate() erro = %v, want ErrUnavailable", err)
	}
}

func TestValidateNumeroVazioNaoConsulta(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))
	defer server.Close()

	client := NewClient()
	client.endpoint = server.URL
	if _, err := client.Validate(context.Background(), "  "); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Validate() erro = %v, want ErrNotFound", err)
	}
	if calls != 0 {
		t.Errorf("CNN foi consultada %d vezes, queria nenhuma", calls)
	}
}
