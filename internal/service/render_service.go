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

type marketingTemplateData struct {
	ClientCompanyName        string
	ClientTradeName          string
	ClientCNPJFormatted      string
	ClientAddress            string
	ClientRepName            string
	ClientRepDocument        string
	ContratadaCompanyName    string
	ContratadaCNPJDisplay    string
	ContratadaAddressDisplay string
	ContratadaRepDisplay     string
	CityForum                string
	SignatureDateFull        string
	IncludeWitnesses         bool

	Escopo domain.EscopoMarketing

	LeadSubtitle            string
	OperationModelText      string
	ObjectServicesText      string
	HasCanais               bool
	CanaisFormatted         string
	CicloEstrategicoDias    int
	CicloEstrategicoExtenso string
	HorasGravacaoExtenso    string
	ClientParticipationText string

	HasSetupFee             bool
	SetupFeeFormatted       string
	SetupFeeExtenso         string
	SetupMarketingCondition string
	MonthlyFeeFormatted     string
	MonthlyFeeExtenso       string
	DueDay                  int

	RemuneracaoClauseNum int
	ConteudoOrganicoDesc string

	Fase1Periodo string
	Fase2Periodo string
	Fase3Periodo string
	Fase1Desc    string
	Fase2Desc    string
	Fase3Desc    string
}

// RenderService compiles and renders the official IA Infinito contract HTML templates.
type RenderService struct {
	tmpl          *template.Template
	marketingTmpl *template.Template
	cfg           *config.Config
}

func NewRenderService(cfg *config.Config) (*RenderService, error) {
	tmpl, err := template.ParseFS(web.Assets, "templates/contract_v1.html")
	if err != nil {
		return nil, fmt.Errorf("parse contract_v1.html: %w", err)
	}
	marketingTmpl, err := template.ParseFS(web.Assets, "templates/contract_marketing_ia_infinito.html")
	if err != nil {
		return nil, fmt.Errorf("parse contract_marketing_ia_infinito.html: %w", err)
	}
	return &RenderService{tmpl: tmpl, marketingTmpl: marketingTmpl, cfg: cfg}, nil
}

// RenderHTML renders the contract HTML using html/template (auto-escaping all user inputs).
func (r *RenderService) RenderHTML(in domain.ContractInput) (string, error) {
	in.Normalize(r.cfg.DefaultCityForum)

	if in.ModeloContrato == domain.ModeloMarketingIAInfinito {
		return r.renderMarketingHTML(in)
	}

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

func (r *RenderService) renderMarketingHTML(in domain.ContractInput) (string, error) {
	escopo := domain.DefaultEscopoMarketing()
	if in.EscopoMarketing != nil {
		escopo = *in.EscopoMarketing
		escopo.Normalize()
	}

	contratadaCompany := firstNonEmpty(in.ContratadaCompanyName, r.cfg.ContratadaRazaoSocial, "JOEL ACOSTA QUINTANA JUNIOR DESENVOLVIMENTO DE PROGRAMAS LTDA - EPP")
	contratadaCNPJ := firstNonEmpty(domain.FormatCNPJ(in.ContratadaCNPJ), domain.FormatCNPJ(r.cfg.ContratadaCNPJ), "63.741.075/0001-43")
	contratadaAddr := firstNonEmpty(in.ContratadaAddress, r.cfg.ContratadaEndereco, "Campo Grande/MS")
	contratadaRep := firstNonEmpty(in.ContratadaRepName, r.cfg.ContratadaRepresentante, "Joel Acosta Quintana Junior")

	hasCanais := escopo.LinkMulticanal && len(escopo.Canais) > 0
	canaisFormatted := joinPortugueseList(escopo.Canais)

	// Lead subtitle & Object clause services
	var leadParts []string
	var objParts []string
	var opParts []string

	if escopo.PosicionamentoEditorial {
		leadParts = append(leadParts, "estratégia de marca")
		objParts = append(objParts, "estratégia de marca")
		opParts = append(opParts, "marketing estratégico")
	}
	if escopo.ConteudoOrganico {
		leadParts = append(leadParts, "produção de conteúdo")
		objParts = append(objParts, "planejamento e produção de conteúdo")
		opParts = append(opParts, "conteúdo")
	}
	if hasCanais {
		leadParts = append(leadParts, "link personalizado multicanal")
		objParts = append(objParts, "implantação de link personalizado multicanal")
		opParts = append(opParts, "presença digital")
	}
	if escopo.TrafegoPago {
		leadParts = append(leadParts, "gestão e investimento em mídia paga")
		objParts = append(objParts, "gestão e custeio de mídia paga")
		opParts = append(opParts, "tráfego pago")
	}
	if escopo.CaptacaoLeads {
		leadParts = append(leadParts, "geração de oportunidades comerciais")
		objParts = append(objParts, "estruturação da jornada de captação de oportunidades comerciais")
	}
	if len(leadParts) == 0 {
		leadParts = []string{"estratégia digital e serviços de marketing"}
	}
	if len(objParts) == 0 {
		objParts = []string{"estratégia digital e marketing"}
	}
	if len(opParts) == 0 {
		opParts = []string{"marketing e presença digital"}
	}

	leadSubtitle := capitalizeFirst(joinPortugueseList(leadParts)) + "."
	objectServicesText := "serviços de " + joinPortugueseList(objParts)
	operationModelText := "Prestação recorrente de serviços estratégicos de " + joinPortugueseList(opParts) + "."

	// Client participation text
	var clientParticipation string
	if escopo.ConteudoOrganico && escopo.HorasGravacaoSemanal > 0 {
		horaWord := "hora"
		if escopo.HorasGravacaoSemanal > 1 {
			horaWord = "horas"
		}
		clientParticipation = fmt.Sprintf(
			"Disponibilidade de aproximadamente %d (%s) %s por semana para gravação em bloco, validações e atendimento comercial dos leads recebidos.",
			escopo.HorasGravacaoSemanal,
			horasExtensoFeminino(escopo.HorasGravacaoSemanal),
			horaWord,
		)
	} else {
		clientParticipation = "Disponibilidade para alinhamentos estratégicos, validações de materiais e atendimento comercial dos leads recebidos."
	}

	// Financial values
	installments := in.SetupInstallments
	if installments < 1 {
		installments = 1
	}
	payMethod := in.EffectiveSetupPaymentMethod()
	totalFinalCents, installmentCents := domain.CalculateSetupInstallmentsWithMethod(in.SetupFeeCents, installments, payMethod)
	hasSetupFee := totalFinalCents > 0
	setupFeeFormatted := domain.FormatBRL(totalFinalCents)
	setupFeeExtenso := domain.CentsToExtenso(totalFinalCents)

	var setupMarketingCondition string
	if installments == 1 {
		if in.EffectivePayOnSignature() {
			setupMarketingCondition = "parcela única na assinatura deste instrumento"
		} else {
			setupMarketingCondition = fmt.Sprintf("parcela única com vencimento no dia %d", in.DueDay)
		}
	} else {
		setupMarketingCondition = fmt.Sprintf(
			"dividido em %d parcelas iguais de R$ %s (%s)",
			installments,
			domain.FormatBRL(installmentCents),
			domain.CentsToExtenso(installmentCents),
		)
	}

	monthlyFeeFormatted := domain.FormatBRL(in.MonthlyFeeCents)
	monthlyFeeExtenso := domain.CentsToExtenso(in.MonthlyFeeCents)

	// Remuneração clause number matching CSS counter(clausula)
	remuneracaoNum := 12
	if escopo.ConteudoOrganico {
		remuneracaoNum++
	}
	if escopo.TrafegoPago {
		remuneracaoNum++
	}

	// Anexo I — Conteúdo Orgânico dynamic text
	conteudoDesc := buildConteudoOrganicoDesc(escopo)

	// Anexo I — Jornada Estratégica Inicial
	dias := escopo.CicloEstrategicoDias
	if dias <= 0 {
		dias = 90
	}
	p1End := dias / 3
	if p1End < 1 {
		p1End = 1
	}
	p2End := (dias * 2) / 3
	if p2End <= p1End {
		p2End = p1End + 1
	}
	fase1Periodo := fmt.Sprintf("0 A %d DIAS", p1End)
	fase2Periodo := fmt.Sprintf("%d A %d DIAS", p1End+1, p2End)
	fase3Periodo := fmt.Sprintf("%d A %d DIAS", p2End+1, dias)

	var f1Parts []string
	f1Parts = append(f1Parts, "Organização da comunicação")
	if hasCanais {
		f1Parts = append(f1Parts, fmt.Sprintf("implantação do link personalizado (%s)", canaisFormatted))
	}
	if escopo.ConteudoOrganico {
		f1Parts = append(f1Parts, "roteiros iniciais", "rotina de gravação")
	}
	if escopo.TrafegoPago {
		f1Parts = append(f1Parts, "ativação das primeiras campanhas custeadas pela IA Infinito")
	}
	fase1Desc := joinPortugueseList(f1Parts) + "."

	var fase2Desc, fase3Desc string
	if escopo.TrafegoPago {
		fase2Desc = "Análise de métricas reais de conversão, otimização de criativos e públicos no tráfego pago, ajuste fino nas abordagens e identificação dos formatos que geram maior volume de oportunidades qualificadas."
		fase3Desc = "Concentração de esforços e verba de mídia nos canais e anúncios validados, maximizando a previsibilidade de geração de leads e oportunidades comerciais para a CONTRATANTE."
	} else {
		fase2Desc = "Análise de métricas reais de engajamento e conversão, otimização de pautas e criativos, ajuste fino nas abordagens e identificação dos formatos que geram maior volume de oportunidades qualificadas."
		fase3Desc = "Concentração de esforços nos canais e formatos validados, maximizando a previsibilidade de geração de leads e oportunidades comerciais para a CONTRATANTE."
	}

	data := marketingTemplateData{
		ClientCompanyName:        firstNonEmpty(in.ClientCompanyName, "[RAZÃO SOCIAL DA CONTRATANTE]"),
		ClientTradeName:          strings.TrimSpace(in.ClientTradeName),
		ClientCNPJFormatted:      firstNonEmpty(domain.FormatCNPJ(in.ClientCNPJ), "[CNPJ DA CONTRATANTE]"),
		ClientAddress:            firstNonEmpty(in.ClientAddress, "[ENDEREÇO COMPLETO DA CONTRATANTE]"),
		ClientRepName:            firstNonEmpty(in.ClientRepName, "[NOME DO REPRESENTANTE LEGAL]"),
		ClientRepDocument:        formatDoc(in.ClientRepDocument),
		ContratadaCompanyName:    contratadaCompany,
		ContratadaCNPJDisplay:    contratadaCNPJ,
		ContratadaAddressDisplay: contratadaAddr,
		ContratadaRepDisplay:     contratadaRep,
		CityForum:                in.CityForum,
		SignatureDateFull:        formatDatePortugueseFull(in.StartDate),
		IncludeWitnesses:         in.EffectiveIncludeWitnesses(),

		Escopo: escopo,

		LeadSubtitle:            leadSubtitle,
		OperationModelText:      operationModelText,
		ObjectServicesText:      objectServicesText,
		HasCanais:               hasCanais,
		CanaisFormatted:         canaisFormatted,
		CicloEstrategicoDias:    dias,
		CicloEstrategicoExtenso: domain.NumberToPortuguese(int64(dias)),
		HorasGravacaoExtenso:    horasExtensoFeminino(escopo.HorasGravacaoSemanal),
		ClientParticipationText: clientParticipation,

		HasSetupFee:             hasSetupFee,
		SetupFeeFormatted:       setupFeeFormatted,
		SetupFeeExtenso:         setupFeeExtenso,
		SetupMarketingCondition: setupMarketingCondition,
		MonthlyFeeFormatted:     monthlyFeeFormatted,
		MonthlyFeeExtenso:       monthlyFeeExtenso,
		DueDay:                  in.DueDay,

		RemuneracaoClauseNum: remuneracaoNum,
		ConteudoOrganicoDesc: conteudoDesc,

		Fase1Periodo: fase1Periodo,
		Fase2Periodo: fase2Periodo,
		Fase3Periodo: fase3Periodo,
		Fase1Desc:    fase1Desc,
		Fase2Desc:    fase2Desc,
		Fase3Desc:    fase3Desc,
	}

	var buf bytes.Buffer
	if err := r.marketingTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute marketing contract template: %w", err)
	}
	return buf.String(), nil
}

func buildConteudoOrganicoDesc(e domain.EscopoMarketing) string {
	parts := []string{"planejamento de pautas, roteiros, ganchos, legendas"}
	if e.HorasGravacaoSemanal > 0 {
		parts = append(parts, fmt.Sprintf("orientação de gravação (%dh/semana em bloco)", e.HorasGravacaoSemanal))
	}
	if e.QtdImagens > 0 {
		parts = append(parts, fmt.Sprintf(
			"criação de %d (%s) imagens/artes %s",
			e.QtdImagens,
			domain.NumberToPortuguese(int64(e.QtdImagens)),
			e.FreqImagens,
		))
	}
	if e.QtdReels > 0 {
		parts = append(parts, fmt.Sprintf(
			"edição de até %d (%s) Reels %s",
			e.QtdReels,
			domain.NumberToPortuguese(int64(e.QtdReels)),
			e.FreqReels,
		))
	}

	var base string
	if len(parts) == 1 {
		base = parts[0]
	} else {
		base = strings.Join(parts[:len(parts)-1], ", ") + " e " + parts[len(parts)-1]
	}

	if e.DirecionamentoStories {
		base += " + direcionamento de Stories"
	}
	return base + "."
}

func joinPortugueseList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " e " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " e " + items[len(items)-1]
	}
}

func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

func horasExtensoFeminino(h int) string {
	switch h {
	case 1:
		return "uma"
	case 2:
		return "duas"
	default:
		return domain.NumberToPortuguese(int64(h))
	}
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
