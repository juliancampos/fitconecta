// Package cref valida o registro CREF do profissional no portal público do
// CREF6. O portal é um formulário ASP.NET (WebForms) sem API: a consulta é
// feita em duas requisições, GET da página e POST do formulário com o viewstate.
package cref

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const portalURL = "https://portal.cref6.org.br/spw/ConsultaCadastral/TelaConsultaPublicaCompleta.aspx"

// Situação aceita para exercer a profissão.
const statusAtivo = "ATIVO"

// validationTimeout limita a consulta completa (GET + POST), não cada requisição.
const validationTimeout = 15 * time.Second

var (
	ErrNotFound    = errors.New("CREF nao encontrado no portal do CREF6 (MG). Confira o numero informado.")
	ErrInactive    = errors.New("Este CREF nao esta ativo no portal do CREF6.")
	ErrUnavailable = errors.New("Nao foi possivel validar o CREF agora. Tente novamente em instantes.")
)

// Profile é o registro encontrado no portal.
type Profile struct {
	Number   string
	Name     string
	Category string
	Status   string
}

// Validator valida um número de CREF.
type Validator interface {
	Validate(ctx context.Context, number string) (*Profile, error)
}

// Client consulta o portal do CREF6.
type Client struct {
	httpClient *http.Client
	portalURL  string
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		portalURL:  portalURL,
	}
}

// Validate busca o número no portal e devolve o registro quando ele existe e
// está ativo. Números de outros regionais não são encontrados: o portal cobre
// apenas o CREF6 (MG).
func (c *Client) Validate(ctx context.Context, number string) (*Profile, error) {
	ctx, cancel := context.WithTimeout(ctx, validationTimeout)
	defer cancel()

	digits := onlyDigits(number)
	if digits == "" {
		return nil, ErrNotFound
	}

	form, cookies, err := c.openPortal(ctx)
	if err != nil {
		return nil, err
	}

	const panel = "ctl00$ContentPlaceHolder1$Callbackconsulta$"
	const button = panel + "btnConsultaTotal"
	form.Set(panel+"txtConsultaTotal", digits)
	form.Set(button, "Pesquisar")
	form.Set("ctl00$ScriptManager1", "ctl00$ContentPlaceHolder1$UpdatePanel3|"+button)
	form.Set("__EVENTTARGET", "")
	form.Set("__EVENTARGUMENT", "")
	form.Set("__ASYNCPOST", "true")

	body, err := c.search(ctx, form, cookies)
	if err != nil {
		return nil, err
	}
	return findProfile(body, digits)
}

// openPortal carrega a página e devolve os campos ocultos e o cookie de sessão.
func (c *Client) openPortal(ctx context.Context) (url.Values, []*http.Cookie, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.portalURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("%w: GET %d", ErrUnavailable, resp.StatusCode)
	}

	page, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return parseInputs(string(page)), resp.Cookies(), nil
}

// search envia o formulário como um postback assíncrono do ASP.NET e devolve o
// HTML parcial com o resultado.
func (c *Client) search(ctx context.Context, form url.Values, cookies []*http.Cookie) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.portalURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-MicrosoftAjax", "Delta=true")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", c.portalURL)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: POST %d", ErrUnavailable, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return string(body), nil
}

// UserError traduz um erro do portal em status HTTP e mensagem para a API.
func UserError(err error) (int, string, bool) {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusBadRequest, ErrNotFound.Error(), true
	case errors.Is(err, ErrInactive):
		return http.StatusBadRequest, ErrInactive.Error(), true
	case errors.Is(err, ErrUnavailable):
		return http.StatusServiceUnavailable, ErrUnavailable.Error(), true
	}
	return 0, "", false
}

// User-Agent de navegador evita bloqueio do IIS.
const userAgent = "Mozilla/5.0 (compatible; FitConecta/1.0)"

var (
	inputTagRe = regexp.MustCompile(`(?is)<input\b[^>]*>`)
	attrRe     = regexp.MustCompile(`(?is)\b(name|value|type)="([^"]*)"`)
	rowRe      = regexp.MustCompile(`(?is)<tr[^>]*DXDataRow\d+"[^>]*>(.*?)</tr>`)
	cellRe     = regexp.MustCompile(`(?is)<td[^>]*>(.*?)</td>`)
	scriptRe   = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>|<style\b[^>]*>.*?</style>`)
	tagRe      = regexp.MustCompile(`(?s)<[^>]+>`)
)

// parseInputs copia os campos do formulário, exceto o botão de submit.
func parseInputs(page string) url.Values {
	form := url.Values{}
	for _, tag := range inputTagRe.FindAllString(page, -1) {
		attrs := map[string]string{}
		for _, match := range attrRe.FindAllStringSubmatch(tag, -1) {
			attrs[strings.ToLower(match[1])] = html.UnescapeString(match[2])
		}
		name := attrs["name"]
		if name == "" || strings.EqualFold(attrs["type"], "submit") {
			continue
		}
		form.Set(name, attrs["value"])
	}
	return form
}

// findProfile exige número exato: a busca do portal é por prefixo e devolve
// vários registros para um número incompleto.
func findProfile(body, digits string) (*Profile, error) {
	found := false
	for _, row := range rowRe.FindAllStringSubmatch(body, -1) {
		cells := cellsOf(row[1])
		if len(cells) < 4 || normalizeNumber(cells[0]) != normalizeNumber(digits) {
			continue
		}
		found = true
		profile := &Profile{
			Number:   cells[0],
			Name:     cells[1],
			Category: cells[2],
			Status:   cells[3],
		}
		if strings.EqualFold(profile.Status, statusAtivo) {
			return profile, nil
		}
	}
	if found {
		return nil, ErrInactive
	}
	return nil, ErrNotFound
}

func cellsOf(row string) []string {
	row = scriptRe.ReplaceAllString(row, " ")
	cells := make([]string, 0, 4)
	for _, cell := range cellRe.FindAllStringSubmatch(row, -1) {
		text := tagRe.ReplaceAllString(cell[1], " ")
		text = html.UnescapeString(text)
		cells = append(cells, strings.Join(strings.Fields(text), " "))
	}
	return cells
}

func normalizeNumber(value string) string {
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
