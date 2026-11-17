package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
	existing.PayloadJSON = string(rawJSON)

	if err := h.repo.Update(r.Context(), existing); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao atualizar contrato"})
		return
	}
	writeJSON(w, http.StatusOK, existing)
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

	var in domain.ContractInput
	if err := json.Unmarshal([]byte(c.PayloadJSON), &in); err != nil {
		in = domain.ContractInput{
			ClientCompanyName: c.ClientCompanyName,
			ClientTradeName:   c.ClientTradeName,
			ClientCNPJ:        c.ClientCNPJ,
			ClientAddress:     c.ClientAddress,
			ClientRepName:     c.ClientRepName,
			ClientRepDocument: c.ClientRepDocument,
			ClientEmail:       c.ClientEmail,
			ClientPhone:       c.ClientPhone,
			SetupFeeCents:     c.SetupFeeCents,
			MonthlyFeeCents:   c.MonthlyFeeCents,
			DueDay:            c.DueDay,
			StartDate:         c.StartDate,
			CityForum:         c.CityForum,
		}
	}

	html, err := h.render.RenderHTML(in)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao compilar template do contrato"})
		return
	}

	pdfBytes, err := h.pdf.GeneratePDF(r.Context(), html)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("erro ao gerar PDF: %v", err)})
		return
	}

	now := time.Now()
	key := fmt.Sprintf("contracts/%04d/%02d/%s_%s.pdf", now.Year(), now.Month(), c.Code, c.ClientCNPJ)
	obj, err := h.storage.SavePDF(r.Context(), key, pdfBytes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao armazenar PDF"})
		return
	}

	if err := h.repo.UpdatePDFMetadata(r.Context(), c.ID, obj.Bucket, obj.Key, obj.SHA256Hex, domain.StatusGenerated); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao atualizar metadados do PDF"})
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
	if c.S3Key == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "PDF ainda não gerado para este contrato"})
		return
	}

	// O PDF é servido pelo próprio app: em MinIO interno, a URL pré-assinada usa
	// o host `http://minio:9000`, que o navegador do cliente não resolve.
	data, err := h.storage.ReadPDF(r.Context(), c.S3Key)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "arquivo PDF não encontrado no armazenamento"})
		return
	}

	filename := fmt.Sprintf("%s_IA_Infinito.pdf", c.Code)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
