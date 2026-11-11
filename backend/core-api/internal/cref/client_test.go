package cref

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const portalPage = `<html><body><form>
<input type="hidden" name="__VIEWSTATE" value="/wEPDwUK/vs" />
<input type="hidden" name="__VIEWSTATEGENERATOR" value="D305D42C" />
<input type="hidden" name="__EVENTVALIDATION" value="/wEdAAq/ev" />
<input type="submit" name="ctl00$ContentPlaceHolder1$Callbackconsulta$btnConsultaTotal" value="Pesquisar" />
</form></body></html>`

func row(number, name, category, status string) string {
	return `<tr id="grid_DXDataRow0" class="dxgvDataRow"><td><span>` + number +
		`</span><script>ASPx.AddDisabledItems('x');</script></td><td>` + name +
		`</td><td>` + category + `</td><td>` + status + `</td></tr>`
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		rows    string
		want    *Profile
		wantErr error
	}{
		{
			name:  "encontra o numero exato entre resultados de prefixo",
			query: "012345",
			rows:  row("MG-000123", "OUTRO", "BACHAREL", "ATIVO") + row("MG-012345", "JOS&#201; MARIA", "LICENCIADO/BACHAREL", "ATIVO"),
			want:  &Profile{Number: "MG-012345", Name: "JOSÉ MARIA", Category: "LICENCIADO/BACHAREL", Status: "ATIVO"},
		},
		{
			name:  "aceita numero sem zero a esquerda",
			query: "12345",
			rows:  row("MG-012345", "ALAN FELIX", "BACHAREL", "ATIVO"),
			want:  &Profile{Number: "MG-012345", Name: "ALAN FELIX", Category: "BACHAREL", Status: "ATIVO"},
		},
		{
			name:  "aceita o formato com UF",
			query: "012345-G/MG",
			rows:  row("MG-012345", "ALAN FELIX", "BACHAREL", "ATIVO"),
			want:  &Profile{Number: "MG-012345", Name: "ALAN FELIX", Category: "BACHAREL", Status: "ATIVO"},
		},
		{
			name:    "numero inexistente",
			query:   "000000",
			rows:    "",
			wantErr: ErrNotFound,
		},
		{
			name:    "numero de outro profissional nao serve",
			query:   "012345",
			rows:    row("MG-999999", "OUTRO", "BACHAREL", "ATIVO"),
			wantErr: ErrNotFound,
		},
		{
			name:    "registro inativo",
			query:   "012345",
			rows:    row("MG-012345", "ALAN FELIX", "BACHAREL", "INATIVO"),
			wantErr: ErrInactive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotQuery string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					_, _ = w.Write([]byte(portalPage))
					return
				}
				if err := r.ParseForm(); err != nil {
					t.Errorf("ParseForm: %v", err)
				}
				if r.PostForm.Get("__VIEWSTATE") != "/wEPDwUK/vs" {
					t.Errorf("__VIEWSTATE nao foi reenviado: %q", r.PostForm.Get("__VIEWSTATE"))
				}
				gotQuery = r.PostForm.Get("ctl00$ContentPlaceHolder1$Callbackconsulta$txtConsultaTotal")
				_, _ = w.Write([]byte(`<div>` + tt.rows + `</div>`))
			}))
			defer server.Close()

			client := NewClient()
			client.portalURL = server.URL
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
			if gotQuery != onlyDigits(tt.query) {
				t.Errorf("portal recebeu %q, want %q", gotQuery, onlyDigits(tt.query))
			}
		})
	}
}

func TestValidatePortalIndisponivel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	client.portalURL = server.URL
	if _, err := client.Validate(context.Background(), "012345"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Validate() erro = %v, want ErrUnavailable", err)
	}
}

func TestValidateBuscaForaDoAr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(portalPage))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient()
	client.portalURL = server.URL
	if _, err := client.Validate(context.Background(), "012345"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Validate() erro = %v, want ErrUnavailable", err)
	}
}

func TestValidateNumeroVazioNaoConsulta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("portal nao deveria ser consultado")
	}))
	defer server.Close()

	client := NewClient()
	client.portalURL = server.URL
	if _, err := client.Validate(context.Background(), "  "); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Validate() erro = %v, want ErrNotFound", err)
	}
}

func TestParseInputsIgnoraBotao(t *testing.T) {
	form := parseInputs(portalPage)
	if form.Get("__VIEWSTATEGENERATOR") != "D305D42C" {
		t.Errorf("campo oculto nao foi lido: %v", form)
	}
	if form.Has("ctl00$ContentPlaceHolder1$Callbackconsulta$btnConsultaTotal") {
		t.Errorf("botao de submit nao deveria ser copiado")
	}
	if !strings.Contains(form.Get("__VIEWSTATE"), "/wEPDwUK") {
		t.Errorf("viewstate nao foi lido: %q", form.Get("__VIEWSTATE"))
	}
}
