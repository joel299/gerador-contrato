package service

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"ia-infinito-contracts/internal/config"
	"ia-infinito-contracts/internal/domain"
	"ia-infinito-contracts/web"
)

type contractTemplateData struct {
	ClientCompanyName       string
	ClientCNPJFormatted     string
	ClientAddress           string
	ClientRepName           string
	ClientRepQualif         string
	ClientRepDocument       string
	SetupFeeFormatted       string
	SetupFeeExtenso         string
	SetupInstallments       int
	SetupInstallmentFormatted string
	SetupInstallmentExtenso   string
	SetupSummaryText        string
	SetupClauseValueText    string
	SetupPaymentForm        string
	HasMonthlyFee           bool
	MonthlyFeeFormatted     string
	MonthlyFeeExtenso       string
	MonthlySummaryText      string
	MonthlyPaymentForm      string
	PayOnSignature          bool
	IncludeWitnesses        bool
	DueDay                  int
	BillingStartDateDisplay string
	CityForum               string
	SignatureDateFull       string

	ContratadaCompanyName    string
	ContratadaCNPJDisplay    string
	ContratadaAddressDisplay string
	ContratadaRepDisplay     string
	ContratadaRepCPFDisplay  string
}

// RenderService compiles and renders the official IA Infinito contract HTML template.
type RenderService struct {
	tmpl *template.Template
	cfg  *config.Config
}

func NewRenderService(cfg *config.Config) (*RenderService, error) {
	tmpl, err := template.ParseFS(web.Assets, "templates/contract_v1.html")
	if err != nil {
		return nil, fmt.Errorf("parse contract_v1.html: %w", err)
	}
	return &RenderService{tmpl: tmpl, cfg: cfg}, nil
}

// RenderHTML renders the contract HTML using html/template (auto-escaping all user inputs).
func (r *RenderService) RenderHTML(in domain.ContractInput) (string, error) {
	in.Normalize(r.cfg.DefaultCityForum)

	qualifParts := nonEmpty(in.ClientRepNationality, in.ClientRepMaritalStatus, in.ClientRepProfession)

	contratadaCompany := firstNonEmpty(in.ContratadaCompanyName, r.cfg.ContratadaRazaoSocial, "JOEL ACOSTA QUINTANA JUNIOR DESENVOLVIMENTO DE PROGRAMAS LTDA - EPP")
	contratadaCNPJ := firstNonEmpty(domain.FormatCNPJ(in.ContratadaCNPJ), domain.FormatCNPJ(r.cfg.ContratadaCNPJ), "63.741.075/0001-43")
	contratadaAddr := firstNonEmpty(in.ContratadaAddress, r.cfg.ContratadaEndereco, "Campo Grande/MS")
	contratadaRep := firstNonEmpty(in.ContratadaRepName, r.cfg.ContratadaRepresentante, "Joel Acosta Quintana Junior")
	contratadaCPF := firstNonEmpty(domain.FormatCPF(in.ContratadaRepCPF), domain.FormatCPF(r.cfg.ContratadaCPF))

	billingStart := formatDateBR(in.BillingStartDate)
	if billingStart == "" {
		billingStart = formatDateBR(in.StartDate)
	}

	installments := in.SetupInstallments
	if installments < 1 {
		installments = 1
	}
	payMethod := in.EffectiveSetupPaymentMethod()
	totalFinalCents, installmentCents := domain.CalculateSetupInstallmentsWithMethod(in.SetupFeeCents, installments, payMethod)
	setupTotalFormatted := domain.FormatBRL(totalFinalCents)
	setupTotalExtenso := domain.CentsToExtenso(totalFinalCents)
	installmentFormatted := domain.FormatBRL(installmentCents)
	installmentExtenso := domain.CentsToExtenso(installmentCents)

	hasMonthly := in.EffectiveHasMonthlyFee()
	payOnSig := in.EffectivePayOnSignature()
	monthlyFormatted := domain.FormatBRL(in.MonthlyFeeCents)
	monthlyExtenso := domain.CentsToExtenso(in.MonthlyFeeCents)

	installmentsExtenso := map[int]string{
		2: "duas", 3: "três", 4: "quatro", 5: "cinco", 6: "seis",
		7: "sete", 8: "oito", 9: "nove", 10: "dez", 11: "onze", 12: "doze",
	}

	var setupSummaryText, setupClauseValueText string
	if installments == 1 {
		setupSummaryText = fmt.Sprintf("Parcela única à vista de R$ %s (%s)", setupTotalFormatted, setupTotalExtenso)
		setupClauseValueText = setupSummaryText
	} else {
		nExtenso := installmentsExtenso[installments]
		setupClauseValueText = fmt.Sprintf(
			"Valor total de R$ %s (%s), dividido em %d (%s) parcelas iguais e sucessivas de R$ %s (%s) cada",
			setupTotalFormatted, setupTotalExtenso, installments, nExtenso, installmentFormatted, installmentExtenso,
		)
		setupSummaryText = setupClauseValueText
	}

	var setupPaymentForm string
	if payOnSig {
		if installments == 1 {
			if payMethod == "card" {
				setupPaymentForm = "Pagamento integral no ato da assinatura deste instrumento, via cartão de crédito."
			} else {
				setupPaymentForm = "Pagamento integral no ato da assinatura deste instrumento, via PIX ou transferência bancária."
			}
		} else {
			setupPaymentForm = fmt.Sprintf(
				"Sendo a 1ª (primeira) parcela no valor de R$ %s paga no ato da assinatura deste instrumento, e as demais %d parcelas com vencimento todo dia %d dos meses subsequentes.",
				installmentFormatted, installments-1, in.DueDay,
			)
		}
	} else {
		setupPaymentForm = fmt.Sprintf("Vencimento todo dia %d de cada mês, via Boleto / PIX / Transferência.", in.DueDay)
	}

	var monthlySummaryText string
	if hasMonthly {
		monthlySummaryText = fmt.Sprintf("Mensalidade: R$ %s (%s) | Vencimento: dia %d de cada mês", monthlyFormatted, monthlyExtenso, in.DueDay)
	} else {
		monthlySummaryText = "Sem mensalidade recorrente"
	}

	data := contractTemplateData{
		ClientCompanyName:         firstNonEmpty(in.ClientCompanyName, "[RAZÃO SOCIAL DA EMPRESA CONTRATANTE]"),
		ClientCNPJFormatted:       firstNonEmpty(domain.FormatCNPJ(in.ClientCNPJ), "[PREENCHER CNPJ]"),
		ClientAddress:             firstNonEmpty(in.ClientAddress, "[ENDEREÇO COMPLETO, BAIRRO, CIDADE/UF, CEP]"),
		ClientRepName:             firstNonEmpty(in.ClientRepName, "[NOME DO REPRESENTANTE LEGAL]"),
		ClientRepQualif:           strings.Join(qualifParts, ", "),
		ClientRepDocument:         formatDoc(in.ClientRepDocument),
		SetupFeeFormatted:         setupTotalFormatted,
		SetupFeeExtenso:           setupTotalExtenso,
		SetupInstallments:         installments,
		SetupInstallmentFormatted: installmentFormatted,
		SetupInstallmentExtenso:   installmentExtenso,
		SetupSummaryText:          setupSummaryText,
		SetupClauseValueText:      setupClauseValueText,
		SetupPaymentForm:          setupPaymentForm,
		HasMonthlyFee:             hasMonthly,
		MonthlyFeeFormatted:       monthlyFormatted,
		MonthlyFeeExtenso:         monthlyExtenso,
		MonthlySummaryText:        monthlySummaryText,
		MonthlyPaymentForm:        in.MonthlyPaymentForm,
		PayOnSignature:            payOnSig,
		IncludeWitnesses:          in.EffectiveIncludeWitnesses(),
		DueDay:                    in.DueDay,
		BillingStartDateDisplay:   billingStart,
		CityForum:                 in.CityForum,
		SignatureDateFull:         formatDatePortugueseFull(in.StartDate),

		ContratadaCompanyName:    contratadaCompany,
		ContratadaCNPJDisplay:    contratadaCNPJ,
		ContratadaAddressDisplay: contratadaAddr,
		ContratadaRepDisplay:     contratadaRep,
		ContratadaRepCPFDisplay:  contratadaCPF,
	}

	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute contract template: %w", err)
	}
	return buf.String(), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

func formatDoc(raw string) string {
	d := domain.OnlyDigits(raw)
	if len(d) == 11 {
		return domain.FormatCPF(d)
	}
	return strings.TrimSpace(raw)
}

func formatDateBR(iso string) string {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(iso))
	if err != nil {
		return strings.TrimSpace(iso)
	}
	return t.Format("02/01/2006")
}

func formatDatePortugueseFull(iso string) string {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(iso))
	if err != nil {
		t = time.Now()
	}
	months := []string{
		"", "janeiro", "fevereiro", "março", "abril", "maio", "junho",
		"julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
	}
	return fmt.Sprintf("%02d de %s de %d", t.Day(), months[int(t.Month())], t.Year())
}
