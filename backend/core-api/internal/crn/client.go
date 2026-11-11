// Package crn valida o registro CRN do nutricionista na Consulta Nacional de
// Nutricionistas (CNN) do CFN. A consulta e um POST JSON puro, sem viewstate
// nem CAPTCHA, com cobertura nacional (os 11 regionais).
package crn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const consultaURL = "https://cnn.cfn.org.br/application/front-resource/get"

// Situacao aceita para exercer a profissao.
const statusAtivo = "ATIVO"

// validationTimeout limita a consulta completa, nao cada requisicao.
const validationTimeout = 15 * time.Second

var (
	ErrNotFound    = errors.New("CRN nao encontrado na Consulta Nacional de Nutricionistas (CFN). Confira o numero informado.")
	ErrInactive    = errors.New("Este CRN nao esta ativo na Consulta Nacional de Nutricionistas (CFN).")
	ErrUnavailable = errors.New("Nao foi possivel validar o CRN agora. Tente novamente em instantes.")
)

// Profile e o registro encontrado na CNN; Number vem na forma canonica
// "CRN9 39104".
type Profile struct {
	Number string
	Name   string
}

// Validator valida um numero de CRN.
type Validator interface {
	Validate(ctx context.Context, number string) (*Profile, error)
}

// Client consulta a CNN do CFN.
type Client struct {
	httpClient *http.Client
	endpoint   string
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		endpoint:   consultaURL,
	}
}

// Validate busca o numero na CNN e devolve o registro quando ele existe e esta
// ativo. O regional pode vir como prefixo do numero digitado ("CRN9 39104",
// "9-39104") ou nao vir: sem regional, qualquer linha ATIVO de nutricionista
// com o mesmo registro e aceita.
func (c *Client) Validate(ctx context.Context, number string) (*Profile, error) {
	regional, registro := parseNumber(number)
	if registro == "" {
		return nil, ErrNotFound
	}

	ctx, cancel := context.WithTimeout(ctx, validationTimeout)
	defer cancel()

	rows, err := c.consult(ctx, regional, registro)
	if err != nil {
		return nil, err
	}
	return findProfile(rows, regional, registro)
}

// consultaRequest e o corpo do POST esperado pelo app AngularJS da CNN.
type consultaRequest struct {
	Comando string          `json:"comando"`
	Options consultaOptions `json:"options"`
}

type consultaOptions struct {
	Nome     string  `json:"nome"`
	CRN      *string `json:"crn"` // regional 1-11, null quando nao informado
	Registro string  `json:"registro"`
	Estado   string  `json:"estado"`
	Geral    bool    `json:"geral"`
	Tecnico  *string `json:"tecnico"`
}

type consultaRow struct {
	Nome         string `json:"nome"`
	Registro     string `json:"registro"`
	CRN          int    `json:"crn"`
	Situacao     string `json:"situacao"`
	TipoRegistro string `json:"tipo_registro"`
}

type consultaResponse struct {
	Success bool          `json:"success"`
	Data    []consultaRow `json:"data"`
}

func (c *Client) consult(ctx context.Context, regional, registro string) ([]consultaRow, error) {
	var crn *string
	if regional != "" {
		crn = &regional
	}
	body, err := json.Marshal(consultaRequest{
		Comando: "get-nutricionista",
		Options: consultaOptions{CRN: crn, Registro: registro},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: POST %d", ErrUnavailable, resp.StatusCode)
	}

	var result consultaResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if !result.Success {
		return nil, ErrUnavailable
	}
	return result.Data, nil
}

var (
	// "CRN9 39104" e variantes com hifen, barra ou sem espaco.
	crnPrefixRe = regexp.MustCompile(`(?i)^\s*crn\s*-?\s*(\d{1,2})\s*[-/ ]\s*0*(\d+)\s*$`)
	// Regional digitado direto: "9-39104" / "9/39104".
	regionalRe = regexp.MustCompile(`^\s*(\d{1,2})\s*[-/]\s*0*(\d+)\s*$`)
)

// parseNumber extrai regional e registro; sem prefixo reconhecido, todos os
// digitos sao o registro (regional vazio).
func parseNumber(input string) (regional, registro string) {
	if m := crnPrefixRe.FindStringSubmatch(input); m != nil {
		return m[1], m[2]
	}
	if m := regionalRe.FindStringSubmatch(input); m != nil {
		return m[1], m[2]
	}
	return "", onlyDigits(input)
}

// findProfile exige registro exato (sem zeros a esquerda) e regional coerente;
// linhas de outro tipo (ex.: produtores rurais "AVULSO-PF") sao ignoradas.
func findProfile(rows []consultaRow, regional, registro string) (*Profile, error) {
	want := normalizeRegistro(registro)
	found := false
	for _, row := range rows {
		tipo := strings.ToUpper(strings.TrimSpace(row.TipoRegistro))
		if !strings.HasPrefix(tipo, "NUTRICIONISTA") {
			continue
		}
		if normalizeRegistro(row.Registro) != want {
			continue
		}
		if regional != "" && strconv.Itoa(row.CRN) != regional {
			continue
		}
		found = true
		if strings.ToUpper(strings.TrimSpace(row.Situacao)) == statusAtivo {
			return &Profile{
				Number: fmt.Sprintf("CRN%d %s", row.CRN, row.Registro),
				Name:   row.Nome,
			}, nil
		}
	}
	if found {
		return nil, ErrInactive
	}
	return nil, ErrNotFound
}

// normalizeRegistro compara sem zeros a esquerda: "506" e "0506" sao o
// mesmo registro.
func normalizeRegistro(value string) string {
	return strings.TrimLeft(onlyDigits(value), "0")
}

func onlyDigits(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
