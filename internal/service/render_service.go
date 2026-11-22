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
	ClientCompanyName         string
	ClientTradeName           string
	ClientCNPJFormatted       string
	ClientAddress             string
	ClientRepName             string
	ClientRepQualif           string
	ClientRepDocument         string
	HasSetupFee               bool
	SetupFeeFormatted         string
	SetupFeeExtenso           string
	SetupInstallments         int
	SetupInstallmentFormatted string
	SetupInstallmentExtenso   string
	SetupSummaryText          string
	SetupClauseValueText      string
	SetupPaymentForm          string
	HasMonthlyFee             bool
	MonthlyFeeFormatted       string
	MonthlyFeeExtenso         string
	MonthlySummaryText        string
	MonthlyPaymentForm        string
	PayOnSignature            bool
	IncludeWitnesses          bool
	DueDay                    int
	BillingStartDateDisplay   string
	CityForum                 string
	SignatureDateFull         string

	ContratadaCompanyName    string
	ContratadaCNPJDisplay    string
	ContratadaAddressDisplay string
	ContratadaRepDisplay     string
	ContratadaRepCPFDisplay  string

	Escopo                  domain.EscopoMarketing
	LeadSubtitle            string
	OperationModelText      string
	ObjectServicesText      string
	HasCanais               bool
	CanaisFormatted         string
	IsTrafegoRiscoContratada bool
	IsTrafegoVerbaCliente    bool
	PlataformasAdsFormatted  string
	HasLeadsClause          bool
	HasAnexoI               bool
	CicloEstrategicoDias    int
	CicloEstrategicoExtenso string
	HorasGravacaoExtenso    string
	GravacaoTexto           string
	ClientParticipationText string
	RemuneracaoClauseNum    int
	ConteudoOrganicoDesc    string

	Fase1Periodo string
	Fase2Periodo string
	Fase3Periodo string
	Fase1Desc    string
	Fase2Desc    string
	Fase3Desc    string
}

// RenderService compiles and renders the unified official IA Infinito contract HTML template.
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

// RenderHTML renders the unified official contract HTML using html/template (auto-escaping all user inputs).
func (r *RenderService) RenderHTML(in domain.ContractInput) (string, error) {
	in.Normalize(r.cfg.DefaultCityForum)

	escopo := domain.DefaultEscopoMarketing()
	if in.EscopoMarketing != nil {
		escopo = *in.EscopoMarketing
		escopo.Normalize()
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
	hasSetupFee := totalFinalCents > 0
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

	// Dynamic scope derivations
	hasCanais := escopo.LinkMulticanal && len(escopo.Canais) > 0
	canaisFormatted := joinPortugueseList(escopo.Canais)
	isTrafegoRisco := escopo.TrafegoPago && escopo.ModalidadeTrafego != domain.TrafegoVerbaCliente
	isTrafegoVerbaCliente := escopo.TrafegoPago && escopo.ModalidadeTrafego == domain.TrafegoVerbaCliente
	plataformasAdsFormatted := joinPortugueseList(escopo.PlataformasAds)
	if plataformasAdsFormatted == "" {
		plataformasAdsFormatted = "Meta Ads (Instagram/Facebook), Google Ads e TikTok Ads"
	}
	hasLeadsClause := hasCanais || escopo.CaptacaoLeads || escopo.TrafegoPago
	hasAnexoI := escopo.PosicionamentoEditorial || escopo.ConteudoOrganico || hasCanais || escopo.TrafegoPago || escopo.CaptacaoLeads

	var leadParts, objParts, opParts []string
	if escopo.ServicosBaseTecnologia {
		leadParts = append(leadParts, "automação, inteligência artificial e sistemas sob medida")
		objParts = append(objParts, "automação, soluções com inteligência artificial, integração de sistemas e suporte tecnológico")
		opParts = append(opParts, "automação, IA e sistemas")
	}
	if escopo.PosicionamentoEditorial {
		leadParts = append(leadParts, "estratégia de posicionamento de marca")
		objParts = append(objParts, "estratégia de posicionamento de marca")
		opParts = append(opParts, "posicionamento estratégico")
	}
	if escopo.ConteudoOrganico {
		leadParts = append(leadParts, "produção de conteúdo orgânico")
		objParts = append(objParts, "planejamento e edição de conteúdo digital")
		opParts = append(opParts, "conteúdo orgânico")
	}
	if hasCanais {
		leadParts = append(leadParts, "link personalizado multicanal")
		objParts = append(objParts, "criação e manutenção de link personalizado multicanal")
		opParts = append(opParts, "link personalizado multicanal")
	}
	if isTrafegoRisco {
		leadParts = append(leadParts, "gestão e investimento em mídia paga")
		objParts = append(objParts, "gestão e investimento direto em campanhas de tráfego pago")
		opParts = append(opParts, "operação de tráfego pago")
	} else if isTrafegoVerbaCliente {
		leadParts = append(leadParts, "gestão estratégica de tráfego pago")
		objParts = append(objParts, "gestão estratégica, criação e otimização de campanhas de tráfego pago")
		opParts = append(opParts, "gestão de tráfego pago")
	}
	if escopo.CaptacaoLeads {
		leadParts = append(leadParts, "geração de oportunidades comerciais")
		objParts = append(objParts, "estruturação da jornada de captação de oportunidades comerciais")
	}
	if len(leadParts) == 0 {
		leadParts = []string{"prestação de serviços especializados de estratégia, tecnologia e suporte operacional"}
	}
	if len(objParts) == 0 {
		objParts = []string{"serviços especializados de estratégia, tecnologia e suporte operacional"}
	}
	if len(opParts) == 0 {
		opParts = []string{"serviços técnicos e estratégicos sob demanda"}
	}

	leadSubtitle := capitalizeFirst(joinPortugueseList(leadParts)) + "."
	objectServicesText := "serviços especializados de " + joinPortugueseList(objParts)
	operationModelText := "Prestação de serviços de " + joinPortugueseList(opParts) + "."
	if hasMonthly {
		operationModelText = "Prestação recorrente de serviços de " + joinPortugueseList(opParts) + "."
	}

	gravacaoTexto := escopo.ObsGravacao
	if gravacaoTexto == "" && escopo.HorasGravacaoSemanal > 0 {
		gravacaoTexto = fmt.Sprintf("%dh/semana em bloco", escopo.HorasGravacaoSemanal)
	}

	var clientParticipation string
	if escopo.ConteudoOrganico && gravacaoTexto != "" {
		clientParticipation = fmt.Sprintf(
			"Disponibilidade conforme rotina combinada (%s) para gravação, validação de materiais e atendimento comercial aos interessados.",
			gravacaoTexto,
		)
	} else {
		clientParticipation = "Disponibilidade para fornecimento de informações, acessos técnicos, validações de entregas e acompanhamento operacional."
	}

	// Calculate exact clause number of REMUNERAÇÃO E CONDIÇÕES DE PAGAMENTO matching CSS counter(clausula)
	// Fixed clauses before Remuneração:
	// 1. Objeto, 2. Escopo, 3. Aprovação e Revisões, 4. Obrigações Contratada, 5. Obrigações Contratante,
	// 6. Itens Não Incluídos, 7. Indicadores e Relatórios, 8. LGPD, 9. Confidencialidade, 10. Propriedade Intelectual -> 10 fixed + 1 (Remuneração itself) = 11
	remuneracaoNum := 11
	if escopo.ConteudoOrganico {
		remuneracaoNum++
	}
	if escopo.TrafegoPago {
		remuneracaoNum++
	}
	if hasLeadsClause {
		remuneracaoNum++
	}

	// Anexo I — Conteúdo Orgânico dynamic text (90-day cycle)
	conteudoDesc := buildConteudoOrganicoDesc(escopo, gravacaoTexto)

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
	fase1Periodo := fmt.Sprintf("DIAS 01 A %02d — ESTRUTURAÇÃO", p1End)
	fase2Periodo := fmt.Sprintf("DIAS %02d A %02d — TESTES E VALIDAÇÃO", p1End+1, p2End)
	fase3Periodo := fmt.Sprintf("DIAS %02d A %02d — CONSOLIDAÇÃO E ESCALA", p2End+1, dias)

	var f1Parts []string
	f1Parts = append(f1Parts, "Organização da comunicação e dos perfis")
	if hasCanais {
		f1Parts = append(f1Parts, fmt.Sprintf("publicação do link personalizado (%s)", canaisFormatted))
	}
	if escopo.ConteudoOrganico {
		f1Parts = append(f1Parts, "definição de roteiros e início das gravações")
	}
	if isTrafegoRisco {
		f1Parts = append(f1Parts, fmt.Sprintf("ativação das primeiras campanhas em %s custeadas pela IA Infinito", plataformasAdsFormatted))
	} else if isTrafegoVerbaCliente {
		f1Parts = append(f1Parts, fmt.Sprintf("configuração e ativação das primeiras campanhas em %s com verba da CONTRATANTE", plataformasAdsFormatted))
	}
	fase1Desc := joinPortugueseList(f1Parts) + "."

	var fase2Desc, fase3Desc string
	if isTrafegoRisco {
		fase2Desc = "Veiculação de conteúdos e campanhas de tráfego pago custeadas pela CONTRATADA para validar abordagens que geram conversas qualificadas."
		fase3Desc = "Concentração de esforços e verba de mídia nos criativos, canais e públicos com maior taxa de conversão comercial para a CONTRATANTE."
	} else if isTrafegoVerbaCliente {
		fase2Desc = "Veiculação de conteúdos e otimização das campanhas de tráfego pago (com verba direta da CONTRATANTE) para validar abordagens que geram conversas qualificadas."
		fase3Desc = "Concentração de esforços nos criativos, canais e públicos validados com maior taxa de conversão comercial para a CONTRATANTE."
	} else {
		fase2Desc = "Análise de métricas reais de engajamento e conversão, otimização de pautas e criativos e validação das abordagens que geram conversas qualificadas."
		fase3Desc = "Concentração de esforços nos formatos e canais validados com maior taxa de conversão comercial para a CONTRATANTE."
	}

	data := contractTemplateData{
		ClientCompanyName:         firstNonEmpty(in.ClientCompanyName, "[RAZÃO SOCIAL DA EMPRESA CONTRATANTE]"),
		ClientTradeName:           strings.TrimSpace(in.ClientTradeName),
		ClientCNPJFormatted:       firstNonEmpty(domain.FormatCNPJ(in.ClientCNPJ), "[PREENCHER CNPJ]"),
		ClientAddress:             firstNonEmpty(in.ClientAddress, "[ENDEREÇO COMPLETO, BAIRRO, CIDADE/UF, CEP]"),
		ClientRepName:             firstNonEmpty(in.ClientRepName, "[NOME DO REPRESENTANTE LEGAL]"),
		ClientRepQualif:           strings.Join(qualifParts, ", "),
		ClientRepDocument:         formatDoc(in.ClientRepDocument),
		HasSetupFee:               hasSetupFee,
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

		Escopo:                   escopo,
		LeadSubtitle:             leadSubtitle,
		OperationModelText:       operationModelText,
		ObjectServicesText:       objectServicesText,
		HasCanais:                hasCanais,
		CanaisFormatted:          canaisFormatted,
		IsTrafegoRiscoContratada: isTrafegoRisco,
		IsTrafegoVerbaCliente:    isTrafegoVerbaCliente,
		PlataformasAdsFormatted:  plataformasAdsFormatted,
		HasLeadsClause:           hasLeadsClause,
		HasAnexoI:                hasAnexoI,
		CicloEstrategicoDias:     dias,
		CicloEstrategicoExtenso:  domain.NumberToPortuguese(int64(dias)),
		HorasGravacaoExtenso:     horasExtensoFeminino(escopo.HorasGravacaoSemanal),
		GravacaoTexto:            gravacaoTexto,
		ClientParticipationText:  clientParticipation,
		RemuneracaoClauseNum:     remuneracaoNum,
		ConteudoOrganicoDesc:     conteudoDesc,

		Fase1Periodo: fase1Periodo,
		Fase2Periodo: fase2Periodo,
		Fase3Periodo: fase3Periodo,
		Fase1Desc:    fase1Desc,
		Fase2Desc:    fase2Desc,
		Fase3Desc:    fase3Desc,
	}

	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute contract template: %w", err)
	}
	return buf.String(), nil
}

func buildConteudoOrganicoDesc(e domain.EscopoMarketing, gravacaoTexto string) string {
	dias := e.CicloEstrategicoDias
	if dias <= 0 {
		dias = 90
	}
	diasExt := domain.NumberToPortuguese(int64(dias))

	var entregas []string
	if e.QtdImagens > 0 {
		imgPart := fmt.Sprintf("%d (%s) imagens/artes estáticas", e.QtdImagens, domain.NumberToPortuguese(int64(e.QtdImagens)))
		if e.ObsImagens != "" {
			imgPart += fmt.Sprintf(" (%s)", e.ObsImagens)
		}
		entregas = append(entregas, imgPart)
	}
	if e.QtdReels > 0 {
		vidPart := fmt.Sprintf("%d (%s) vídeos/Reels editados", e.QtdReels, domain.NumberToPortuguese(int64(e.QtdReels)))
		entregas = append(entregas, vidPart)
	}

	var base string
	if len(entregas) > 0 {
		base = fmt.Sprintf(
			"Produção e entrega de %s ao longo do ciclo estratégico de %d (%s) dias",
			joinPortugueseList(entregas),
			dias,
			diasExt,
		)
	} else {
		base = fmt.Sprintf(
			"Planejamento editorial, pautas, roteiros, ganchos e legendas ao longo do ciclo estratégico de %d (%s) dias",
			dias,
			diasExt,
		)
	}

	var extras []string
	if gravacaoTexto != "" {
		extras = append(extras, fmt.Sprintf("orientação de gravação (%s)", gravacaoTexto))
	}
	if e.DirecionamentoStories {
		extras = append(extras, "direcionamento de Stories")
	}
	if len(extras) > 0 {
		base += ", com " + joinPortugueseList(extras)
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
