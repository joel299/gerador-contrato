package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"ia-infinito-contracts/internal/domain"
)

var (
	ErrInvalidCNPJ      = errors.New("CNPJ inválido")
	ErrCNPJNotFound     = errors.New("CNPJ não encontrado na BrasilAPI")
	ErrCNPJAPIUnhealthy = errors.New("serviço BrasilAPI indisponível no momento; preencha os dados manualmente")
)

// CNPJData is the normalized company payload returned to the frontend and API clients.
type CNPJData struct {
	CNPJ          string `json:"cnpj"`
	FormattedCNPJ string `json:"formatted_cnpj"`
	CompanyName   string `json:"company_name"`
	TradeName     string `json:"trade_name"`
	Address       string `json:"address"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	LegalRepName  string `json:"legal_rep_name"`
	Status        string `json:"status"`
	Source        string `json:"source"`
}

type brasilAPIResponse struct {
	CNPJ          string `json:"cnpj"`
	RazaoSocial   string `json:"razao_social"`
	NomeFantasia  string `json:"nome_fantasia"`
	DescricaoSitu string `json:"descricao_situacao_cadastral"`
	Logradouro    string `json:"logradouro"`
	DescTipoLogr  string `json:"descricao_tipo_de_logradouro"`
	Numero        string `json:"numero"`
	Complemento   string `json:"complemento"`
	Bairro        string `json:"bairro"`
	Municipio     string `json:"municipio"`
	UF            string `json:"uf"`
	CEP           string `json:"cep"`
	DDDTelefone1  string `json:"ddd_telefone_1"`
	Email         string `json:"email"`
	QSA           []struct {
		NomeSocio string `json:"nome_socio"`
	} `json:"qsa"`
}

type cnpjCacheEntry struct {
	data      CNPJData
	expiresAt time.Time
}

// CNPJService queries BrasilAPI with a 5s timeout, in-memory TTL cache, and optional SQLite fallback cache.
type CNPJService struct {
	baseURL    string
	httpClient *http.Client
	db         *sql.DB
	ttl        time.Duration

	mu    sync.RWMutex
	cache map[string]cnpjCacheEntry
}

func NewCNPJService(db *sql.DB) *CNPJService {
	return &CNPJService{
		baseURL: "https://brasilapi.com.br/api/cnpj/v1",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		db:    db,
		ttl:   24 * time.Hour,
		cache: make(map[string]cnpjCacheEntry),
	}
}

// SetBaseURL overrides the BrasilAPI base URL (useful in unit tests).
func (s *CNPJService) SetBaseURL(u string) {
	s.baseURL = strings.TrimRight(u, "/")
}

// Lookup fetches normalized company data for a 14-digit CNPJ.
func (s *CNPJService) Lookup(ctx context.Context, rawCNPJ string) (*CNPJData, error) {
	digits := domain.OnlyDigits(rawCNPJ)
	if !domain.IsValidCNPJ(digits) {
		return nil, ErrInvalidCNPJ
	}

	// 1. Check in-memory cache
	s.mu.RLock()
	entry, ok := s.cache[digits]
	s.mu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		cp := entry.data
		cp.Source = "memory_cache"
		return &cp, nil
	}

	// 2. Query BrasilAPI with 5s timeout
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/%s", s.baseURL, digits)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return s.fallbackFromDB(ctx, digits, ErrCNPJAPIUnhealthy)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "IA-Infinito-Contracts/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return s.fallbackFromDB(ctx, digits, ErrCNPJAPIUnhealthy)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrCNPJNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return s.fallbackFromDB(ctx, digits, ErrCNPJAPIUnhealthy)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return s.fallbackFromDB(ctx, digits, ErrCNPJAPIUnhealthy)
	}

	var apiResp brasilAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return s.fallbackFromDB(ctx, digits, ErrCNPJAPIUnhealthy)
	}

	data := normalizeBrasilAPI(digits, &apiResp)
	data.Source = "brasilapi"

	s.mu.Lock()
	s.cache[digits] = cnpjCacheEntry{data: data, expiresAt: time.Now().Add(s.ttl)}
	s.mu.Unlock()

	s.saveToDB(ctx, digits, &data)
	return &data, nil
}

func (s *CNPJService) saveToDB(ctx context.Context, digits string, data *CNPJData) {
	if s.db == nil {
		return
	}
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO cnpj_cache (cnpj, response_json, cached_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(cnpj) DO UPDATE SET response_json = excluded.response_json, cached_at = CURRENT_TIMESTAMP`,
		digits, string(b),
	)
}

func (s *CNPJService) fallbackFromDB(ctx context.Context, digits string, origErr error) (*CNPJData, error) {
	if s.db == nil {
		return nil, origErr
	}
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT response_json FROM cnpj_cache WHERE cnpj = ?`, digits).Scan(&raw)
	if err != nil {
		return nil, origErr
	}
	var data CNPJData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, origErr
	}
	data.Source = "sqlite_cache_fallback"
	return &data, nil
}

func normalizeBrasilAPI(digits string, r *brasilAPIResponse) CNPJData {
	street := strings.TrimSpace(strings.Join(nonEmpty(r.DescTipoLogr, r.Logradouro), " "))
	if r.Numero != "" {
		street += ", " + strings.TrimSpace(r.Numero)
	}
	if r.Complemento != "" {
		street += " (" + strings.TrimSpace(r.Complemento) + ")"
	}
	cityUF := ""
	if r.Municipio != "" && r.UF != "" {
		cityUF = fmt.Sprintf("%s/%s", strings.TrimSpace(r.Municipio), strings.TrimSpace(r.UF))
	} else {
		cityUF = strings.TrimSpace(r.Municipio + r.UF)
	}
	cep := formatCEP(r.CEP)
	if cep != "" {
		cep = "CEP " + cep
	}

	addrParts := nonEmpty(street, strings.TrimSpace(r.Bairro), cityUF, cep)
	legalRep := ""
	if len(r.QSA) > 0 {
		legalRep = strings.TrimSpace(r.QSA[0].NomeSocio)
	}

	return CNPJData{
		CNPJ:          digits,
		FormattedCNPJ: domain.FormatCNPJ(digits),
		CompanyName:   strings.TrimSpace(r.RazaoSocial),
		TradeName:     strings.TrimSpace(r.NomeFantasia),
		Address:       strings.Join(addrParts, ", "),
		Phone:         formatPhone(r.DDDTelefone1),
		Email:         strings.ToLower(strings.TrimSpace(r.Email)),
		LegalRepName:  legalRep,
		Status:        strings.TrimSpace(r.DescricaoSitu),
	}
}

func nonEmpty(items ...string) []string {
	var out []string
	for _, item := range items {
		if t := strings.TrimSpace(item); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func formatCEP(cep string) string {
	d := domain.OnlyDigits(cep)
	if len(d) != 8 {
		return strings.TrimSpace(cep)
	}
	return d[:5] + "-" + d[5:]
}

func formatPhone(raw string) string {
	d := domain.OnlyDigits(raw)
	switch len(d) {
	case 10:
		return fmt.Sprintf("(%s) %s-%s", d[:2], d[2:6], d[6:])
	case 11:
		return fmt.Sprintf("(%s) %s-%s", d[:2], d[2:7], d[7:])
	default:
		return strings.TrimSpace(raw)
	}
}
