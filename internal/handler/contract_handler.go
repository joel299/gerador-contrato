package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ia-infinito-contracts/internal/config"
	"ia-infinito-contracts/internal/domain"
	"ia-infinito-contracts/internal/repository"
	"ia-infinito-contracts/internal/service"
)

type ContractHandler struct {
	cfg     *config.Config
	repo    *repository.ContractRepository
	render  *service.RenderService
	pdf     *service.PDFService
	storage service.StorageProvider
}

func NewContractHandler(
	cfg *config.Config,
	repo *repository.ContractRepository,
	render *service.RenderService,
	pdf *service.PDFService,
	storage service.StorageProvider,
) *ContractHandler {
	return &ContractHandler{
		cfg:     cfg,
		repo:    repo,
		render:  render,
		pdf:     pdf,
		storage: storage,
	}
}

func (h *ContractHandler) Preview(w http.ResponseWriter, r *http.Request) {
	var in domain.ContractInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	html, err := h.render.RenderHTML(in)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao renderizar preview"})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in domain.ContractInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	in.Normalize(h.cfg.DefaultCityForum)
	if err := in.Validate(false); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	rawJSON, _ := json.Marshal(in)
	escopoJSON := "{}"
	if in.EscopoMarketing != nil {
		if b, err := json.Marshal(in.EscopoMarketing); err == nil {
			escopoJSON = string(b)
		}
	}
	c := &domain.Contract{
		ClientCompanyName:          in.ClientCompanyName,
		ClientTradeName:            in.ClientTradeName,
		ClientCNPJ:                 in.ClientCNPJ,
		ClientAddress:              in.ClientAddress,
		ClientRepName:              in.ClientRepName,
		ClientRepDocument:          in.ClientRepDocument,
		ClientEmail:                in.ClientEmail,
		ClientPhone:                in.ClientPhone,
		SetupFeeCents:              in.SetupFeeCents,
		SetupInstallments:          in.SetupInstallments,
		SetupPaymentMethod:         in.EffectiveSetupPaymentMethod(),
		SetupTotalFinalCents:       in.SetupTotalFinalCents(),
		SetupInstallmentValueCents: in.SetupInstallmentValueCents(),
		HasMonthlyFee:              in.EffectiveHasMonthlyFee(),
		MonthlyFeeCents:            in.MonthlyFeeCents,
		PayOnSignature:             in.EffectivePayOnSignature(),
		IncludeWitnesses:           in.EffectiveIncludeWitnesses(),
		DueDay:                     in.DueDay,
		StartDate:                  in.StartDate,
		CityForum:                  in.CityForum,
		ModeloContrato:             in.ModeloContrato,
		EscopoJSON:                 escopoJSON,
		PayloadJSON:                string(rawJSON),
		Status:                     domain.StatusDraft,
	}
	if err := h.repo.Create(r.Context(), c); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao salvar contrato"})
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *ContractHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	list, err := h.repo.List(r.Context(), q, status, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao listar contratos"})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *ContractHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao buscar contrato"})
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *ContractHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao buscar contrato"})
		return
	}

	var in domain.ContractInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	in.Normalize(h.cfg.DefaultCityForum)
	if err := in.Validate(false); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	rawJSON, _ := json.Marshal(in)
	escopoJSON := "{}"
	if in.EscopoMarketing != nil {
		if b, err := json.Marshal(in.EscopoMarketing); err == nil {
			escopoJSON = string(b)
		}
	}
	existing.ClientCompanyName = in.ClientCompanyName
	existing.ClientTradeName = in.ClientTradeName
	existing.ClientCNPJ = in.ClientCNPJ
	existing.ClientAddress = in.ClientAddress
	existing.ClientRepName = in.ClientRepName
	existing.ClientRepDocument = in.ClientRepDocument
	existing.ClientEmail = in.ClientEmail
	existing.ClientPhone = in.ClientPhone
	existing.SetupFeeCents = in.SetupFeeCents
	existing.SetupInstallments = in.SetupInstallments
	existing.SetupPaymentMethod = in.EffectiveSetupPaymentMethod()
	existing.SetupTotalFinalCents = in.SetupTotalFinalCents()
	existing.SetupInstallmentValueCents = in.SetupInstallmentValueCents()
	existing.HasMonthlyFee = in.EffectiveHasMonthlyFee()
	existing.MonthlyFeeCents = in.MonthlyFeeCents
	existing.PayOnSignature = in.EffectivePayOnSignature()
	existing.IncludeWitnesses = in.EffectiveIncludeWitnesses()
	existing.DueDay = in.DueDay
	existing.StartDate = in.StartDate
	existing.CityForum = in.CityForum
	existing.ModeloContrato = in.ModeloContrato
	existing.EscopoJSON = escopoJSON
	existing.PayloadJSON = string(rawJSON)

	if err := h.repo.Update(r.Context(), existing); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao atualizar contrato"})
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *ContractHandler) generateAndStorePDF(r *http.Request, c *domain.Contract) ([]byte, *service.StoredObject, error) {
	var in domain.ContractInput
	if err := json.Unmarshal([]byte(c.PayloadJSON), &in); err != nil {
		in = domain.ContractInput{
			ClientCompanyName:  c.ClientCompanyName,
			ClientTradeName:    c.ClientTradeName,
			ClientCNPJ:         c.ClientCNPJ,
			ClientAddress:      c.ClientAddress,
			ClientRepName:      c.ClientRepName,
			ClientRepDocument:  c.ClientRepDocument,
			ClientEmail:        c.ClientEmail,
			ClientPhone:        c.ClientPhone,
			SetupFeeCents:      c.SetupFeeCents,
			SetupInstallments:  c.SetupInstallments,
			SetupPaymentMethod: c.SetupPaymentMethod,
			HasMonthlyFee:      &c.HasMonthlyFee,
			MonthlyFeeCents:    c.MonthlyFeeCents,
			PayOnSignature:     &c.PayOnSignature,
			IncludeWitnesses:   &c.IncludeWitnesses,
			DueDay:             c.DueDay,
			StartDate:          c.StartDate,
			CityForum:          c.CityForum,
			ModeloContrato:     c.ModeloContrato,
		}
	}
	if in.ModeloContrato == "" && c.ModeloContrato != "" {
		in.ModeloContrato = c.ModeloContrato
	}
	if in.EscopoMarketing == nil && c.EscopoJSON != "" && c.EscopoJSON != "{}" {
		var escopo domain.EscopoMarketing
		if err := json.Unmarshal([]byte(c.EscopoJSON), &escopo); err == nil {
			in.EscopoMarketing = &escopo
		}
	}

	html, err := h.render.RenderHTML(in)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao compilar template do contrato: %w", err)
	}

	pdfBytes, err := h.pdf.GeneratePDF(r.Context(), html)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao gerar PDF: %w", err)
	}

	now := time.Now()
	key := fmt.Sprintf("contracts/%04d/%02d/%s_%s.pdf", now.Year(), now.Month(), c.Code, c.ClientCNPJ)
	obj, err := h.storage.SavePDF(r.Context(), key, pdfBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao armazenar PDF: %w", err)
	}

	if err := h.repo.UpdatePDFMetadata(r.Context(), c.ID, obj.Bucket, obj.Key, obj.SHA256Hex, domain.StatusGenerated); err != nil {
		return nil, nil, fmt.Errorf("erro ao atualizar metadados do PDF: %w", err)
	}

	return pdfBytes, obj, nil
}

func (h *ContractHandler) GeneratePDF(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao buscar contrato"})
		return
	}

	_, obj, err := h.generateAndStorePDF(r, c)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"contract_id":      c.ID,
		"code":             c.Code,
		"storage_provider": h.storage.ProviderName(),
		"bucket":           obj.Bucket,
		"key":              obj.Key,
		"sha256":           obj.SHA256Hex,
		"size_bytes":       obj.SizeBytes,
		"download_url":     fmt.Sprintf("/api/contracts/%s/pdf", c.ID),
	})
}

func (h *ContractHandler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao buscar contrato"})
		return
	}

	var data []byte
	if c.S3Key != "" {
		data, err = h.storage.ReadPDF(r.Context(), c.S3Key)
	}
	if c.S3Key == "" || err != nil {
		data, _, err = h.generateAndStorePDF(r, c)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao gerar arquivo PDF"})
			return
		}
	}

	disposition := "attachment"
	if r.URL.Query().Get("inline") == "1" {
		disposition = "inline"
	}

	safeFilename := sanitizePDFFilename(c.ClientCompanyName, c.Code)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, safeFilename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

var accentReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"Á", "A", "À", "A", "Â", "A", "Ã", "A", "Ä", "A",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"É", "E", "È", "E", "Ê", "E", "Ë", "E",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"Í", "I", "Ì", "I", "Î", "I", "Ï", "I",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"Ó", "O", "Ò", "O", "Ô", "O", "Õ", "O", "Ö", "O",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"Ú", "U", "Ù", "U", "Û", "U", "Ü", "U",
	"ç", "c", "Ç", "C", "ñ", "n", "Ñ", "N",
)

func sanitizePDFFilename(companyName, code string) string {
	ascii := accentReplacer.Replace(companyName)
	var b strings.Builder
	prevUnderscore := false
	for _, r := range ascii {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevUnderscore = false
		} else if !prevUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	clean := strings.Trim(b.String(), "_")
	if clean == "" {
		clean = code
	}
	if len(clean) > 48 {
		clean = strings.Trim(clean[:48], "_")
	}
	return fmt.Sprintf("Contrato_IA_Infinito_%s.pdf", clean)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
