package service

import (
	"html"
	"strings"
	"testing"

	"ia-infinito-contracts/internal/config"
	"ia-infinito-contracts/internal/domain"
)

func TestRenderService_OfficialContractContentAndEscaping(t *testing.T) {
	cfg := &config.Config{
		ContratadaRazaoSocial: "IA Infinito Tecnologia Ltda",
		DefaultCityForum:      "Campo Grande/MS",
	}
	svc, err := NewRenderService(cfg)
	if err != nil {
		t.Fatalf("NewRenderService failed: %v", err)
	}

	in := domain.ContractInput{
		ClientCompanyName: "Cliente <script>alert(1)</script> Ltda",
		ClientCNPJ:        "11222333000181",
		ClientAddress:     "Av. Afonso Pena, 1000, Centro, Campo Grande/MS",
		ClientRepName:     "Carlos Eduardo",
		ClientRepDocument: "52998224725",
		ClientEmail:       "carlos@cliente.com.br",
		ClientPhone:       "(67) 99999-1234",
		SetupFeeCents:     150000,
		MonthlyFeeCents:   250000,
		DueDay:            10,
		StartDate:         "2026-11-17",
		CityForum:         "Campo Grande/MS",
	}

	html, err := svc.RenderHTML(in)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	required := []string{
		"<svg",
		"IA INFINITO",
		"WhatsApp, Facebook, Instagram, Google e TikTok",
		"TRÁFEGO PAGO E ASSUNÇÃO DE RISCO PELA CONTRATADA",
		"todo o risco financeiro e operacional",
		"1.500,00",
		"mil e quinhentos reais",
		"2.500,00",
		"dois mil e quinhentos reais",
		"11.222.333/0001-81",
		"529.982.247-25",
	}
	for _, needle := range required {
		if !strings.Contains(html, needle) {
			t.Errorf("rendered contract missing %q", needle)
		}
	}

	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Errorf("expected HTML escaping of script tag, but raw script tag appeared")
	}

	// Optional CPF omitted: should not render CPF placeholder or CPF text in qualification/signature
	in.ClientRepDocument = ""
	if err := in.Validate(true); err != nil {
		t.Fatalf("expected optional CPF to pass strict validation, got: %v", err)
	}
	htmlNoCPF, err := svc.RenderHTML(in)
	if err != nil {
		t.Fatalf("RenderHTML without CPF failed: %v", err)
	}
	if strings.Contains(htmlNoCPF, "[PREENCHER CPF]") || strings.Contains(htmlNoCPF, "inscrito no CPF sob o nº") {
		t.Errorf("expected no CPF mention in contract when ClientRepDocument is empty")
	}

	// Test: no monthly fee + 4x installments + pay on signature + fixed Contratada data
	cfgOfficial := &config.Config{
		ContratadaRazaoSocial:   "JOEL ACOSTA QUINTANA JUNIOR DESENVOLVIMENTO DE PROGRAMAS LTDA - EPP",
		ContratadaCNPJ:          "63741075000143",
		ContratadaEndereco:      "Campo Grande/MS",
		ContratadaRepresentante: "Joel Acosta Quintana Junior",
		DefaultCityForum:        "Campo Grande/MS",
	}
	svcOfficial, err := NewRenderService(cfgOfficial)
	if err != nil {
		t.Fatalf("NewRenderService official failed: %v", err)
	}
	boolFalse := false
	boolTrue := true
	in4x := domain.ContractInput{
		ClientCompanyName: "Projeto Único Ltda",
		ClientCNPJ:        "11222333000181",
		ClientAddress:     "Rua 14 de Julho, 200, Campo Grande/MS",
		ClientRepName:     "Marcos Lima",
		ClientEmail:       "marcos@projeto.com.br",
		ClientPhone:       "(67) 99999-1111",
		SetupFeeCents:     600000,
		SetupInstallments: 4,
		HasMonthlyFee:     &boolFalse,
		MonthlyFeeCents:   250000,
		PayOnSignature:    &boolTrue,
		DueDay:            15,
		StartDate:         "2026-11-17",
		CityForum:         "Campo Grande/MS",
	}
	html4x, err := svcOfficial.RenderHTML(in4x)
	if err != nil {
		t.Fatalf("RenderHTML 4x failed: %v", err)
	}
	mustContain4x := []string{
		"JOEL ACOSTA QUINTANA JUNIOR DESENVOLVIMENTO DE PROGRAMAS LTDA - EPP",
		"63.741.075/0001-43",
		"Joel Acosta Quintana Junior",
		"Sem mensalidade recorrente",
		"não haverá cobrança de mensalidade recorrente",
		"Valor total de R$ 6.903,68 (seis mil, novecentos e três reais e sessenta e oito centavos), dividido em 4 (quatro) parcelas iguais e sucessivas de R$ 1.725,92 (mil, setecentos e vinte e cinco reais e noventa e dois centavos) cada",
		"Sendo a 1ª (primeira) parcela no valor de R$ 1.725,92 paga no ato da assinatura deste instrumento, e as demais 3 parcelas com vencimento todo dia 15 dos meses subsequentes",
	}
	for _, needle := range mustContain4x {
		if !strings.Contains(html4x, needle) {
			t.Errorf("4x/no-monthly contract missing %q", needle)
		}
	}
	for _, forbidden := range []string{"12,49%", "13,09%", "18,79%", "infinitepay", "acréscimo", "taxa da máquina", "com juros"} {
		if strings.Contains(strings.ToLower(html4x), forbidden) {
			t.Errorf("contract must not mention %q", forbidden)
		}
	}
	if strings.Contains(html4x, "<td>Mensalidade Recorrente</td>") {
		t.Errorf("expected Mensalidade Recorrente table row to be omitted when HasMonthlyFee=false")
	}

	// Test 3x on R$ 1.500,00 (gross-up 12.49% -> R$ 1.714,08 = 3 x R$ 571,36)
	in3x := in4x
	in3x.SetupFeeCents = 150000
	in3x.SetupInstallments = 3
	html3x, err := svcOfficial.RenderHTML(in3x)
	if err != nil {
		t.Fatalf("RenderHTML 3x failed: %v", err)
	}
	if !strings.Contains(html3x, "Valor total de R$ 1.714,08 (mil, setecentos e quatorze reais e oito centavos), dividido em 3 (três) parcelas iguais e sucessivas de R$ 571,36 (quinhentos e setenta e um reais e trinta e seis centavos) cada") {
		t.Errorf("3x rendered text mismatch")
	}
	if strings.Contains(strings.ToLower(html3x), "com juros") || strings.Contains(html3x, "12,49%") || strings.Contains(strings.ToLower(html3x), "infinitepay") {
		t.Errorf("3x contract must not mention installment juros, 12,49%%, or infinitepay")
	}

	// Test: include_witnesses = false removes TESTEMUNHA 1/2 and adjusts closing text
	in4x.IncludeWitnesses = &boolFalse
	htmlNoWitnesses, err := svcOfficial.RenderHTML(in4x)
	if err != nil {
		t.Fatalf("RenderHTML without witnesses failed: %v", err)
	}
	if strings.Contains(htmlNoWitnesses, "TESTEMUNHA 1") || strings.Contains(htmlNoWitnesses, "TESTEMUNHA 2") {
		t.Errorf("expected no witness blocks when IncludeWitnesses=false")
	}
	if !strings.Contains(htmlNoWitnesses, "ou mediante assinatura eletrônica com plena validade jurídica") {
		t.Errorf("expected electronic signature closing text when IncludeWitnesses=false")
	}
	if !strings.Contains(htmlNoWitnesses, "vertical-align: top;") {
		t.Errorf("expected vertical-align: top on signature grid cells")
	}
}

func TestRenderService_UnifiedDynamicScope_ScenarioA_And_ScenarioB(t *testing.T) {
	cfg := &config.Config{
		ContratadaRazaoSocial:   "JOEL ACOSTA QUINTANA JUNIOR DESENVOLVIMENTO DE PROGRAMAS LTDA - EPP",
		ContratadaCNPJ:          "63741075000143",
		ContratadaEndereco:      "Campo Grande/MS",
		ContratadaRepresentante: "Joel Acosta Quintana Junior",
		DefaultCityForum:        "Campo Grande/MS",
	}
	svc, err := NewRenderService(cfg)
	if err != nil {
		t.Fatalf("NewRenderService failed: %v", err)
	}

	boolTrue := true

	// Cenário A: Apenas serviços base/tecnologia sem marketing (sem tráfego, sem Reels, sem link, sem Anexo I)
	baseOnly := domain.EscopoMarketing{
		CicloEstrategicoDias:    90,
		ServicosBaseTecnologia:  true,
		PosicionamentoEditorial: false,
		ConteudoOrganico:        false,
		LinkMulticanal:          false,
		TrafegoPago:             false,
		CaptacaoLeads:           false,
		QtdImagens:              0,
		FreqImagens:             "semanais",
		QtdReels:                0,
		FreqReels:               "semanais",
		DirecionamentoStories:   false,
		HorasGravacaoSemanal:    0,
		Canais:                  nil,
	}
	inA := domain.ContractInput{
		EscopoMarketing:   &baseOnly,
		ClientCompanyName: "Tech Base Ltda",
		ClientTradeName:   "Tech Base",
		ClientCNPJ:        "11222333000181",
		ClientAddress:     "Rua 14 de Julho, 500, Campo Grande/MS",
		ClientRepName:     "Marcos Lima",
		ClientEmail:       "marcos@techbase.com.br",
		ClientPhone:       "(67) 99999-1111",
		SetupFeeCents:     0,
		HasMonthlyFee:     &boolTrue,
		MonthlyFeeCents:   180000,
		DueDay:            15,
		StartDate:         "2026-11-22",
		CityForum:         "Campo Grande/MS",
	}
	htmlA, err := svc.RenderHTML(inA)
	if err != nil {
		t.Fatalf("RenderHTML Scenario A failed: %v", err)
	}
	htmlADecoded := html.UnescapeString(htmlA)

	mustHaveA := []string{
		"counter-reset: clausula;",
		"counter-increment: clausula;",
		"Automação, IA e Sistemas Sob Medida",
		"1.800,00",
		"Cláusula 11",
	}
	for _, needle := range mustHaveA {
		if !strings.Contains(htmlADecoded, needle) {
			t.Errorf("Scenario A missing %q", needle)
		}
	}

	mustOmitA := []string{
		"TRÁFEGO PAGO E ASSUNÇÃO DE RISCO",
		"GESTÃO DE TRÁFEGO PAGO E INVESTIMENTO EM MÍDIA",
		"TRÁFEGO PAGO E RISCO",
		"CANAIS INTEGRADOS",
		"MODELO DE CONTEÚDO E PARTICIPAÇÃO",
		"ANEXO I · ESCOPO OPERACIONAL",
		"Implantação / Setup",
		"Reels",
		"Stories",
	}
	for _, forbidden := range mustOmitA {
		if strings.Contains(htmlADecoded, forbidden) {
			t.Errorf("Scenario A must omit %q", forbidden)
		}
	}

	// Cenário B: Serviços de Marketing habilitados com quantidades no ciclo de 90 dias, canais e Tráfego Pago (risco_contratada)
	fullEscopo := domain.DefaultEscopoMarketing()
	fullEscopo.QtdImagens = 12
	fullEscopo.ObsImagens = "4 por mês"
	fullEscopo.QtdReels = 36
	fullEscopo.ObsGravacao = "1h/semana em bloco"
	fullEscopo.DirecionamentoStories = true
	fullEscopo.ModalidadeTrafego = domain.TrafegoRiscoContratada
	fullEscopo.PlataformasAds = []string{"Meta Ads (Instagram/Facebook)", "Google Ads", "TikTok Ads"}
	fullEscopo.Canais = []string{"WhatsApp", "Instagram", "Facebook", "Google", "TikTok"}

	inB := domain.ContractInput{
		EscopoMarketing:   &fullEscopo,
		ClientCompanyName: "Clínica Exemplo Ltda",
		ClientTradeName:   "Clínica Exemplo",
		ClientCNPJ:        "11222333000181",
		ClientAddress:     "Av. Afonso Pena, 1500, Centro, Campo Grande/MS",
		ClientRepName:     "Dra. Ana Silva",
		ClientEmail:       "ana@clinica.com.br",
		ClientPhone:       "(67) 99999-0000",
		SetupFeeCents:     150000,
		HasMonthlyFee:     &boolTrue,
		MonthlyFeeCents:   250000,
		DueDay:            10,
		StartDate:         "2026-11-22",
		CityForum:         "Campo Grande/MS",
	}
	htmlB, err := svc.RenderHTML(inB)
	if err != nil {
		t.Fatalf("RenderHTML Scenario B failed: %v", err)
	}
	htmlBDecoded := html.UnescapeString(htmlB)

	mustHaveB := []string{
		"counter-reset: clausula;",
		"counter-increment: clausula;",
		"JOEL ACOSTA QUINTANA JUNIOR DESENVOLVIMENTO DE PROGRAMAS LTDA - EPP",
		"63.741.075/0001-43",
		"Joel Acosta Quintana Junior",
		"Clínica Exemplo",
		"TRÁFEGO PAGO E ASSUNÇÃO DE RISCO PELA CONTRATADA",
		"Assunção Integral do Investimento e Risco pela Contratada",
		"Meta Ads (Instagram/Facebook), Google Ads e TikTok Ads",
		"Implantação / Setup",
		"1.500,00",
		"2.500,00",
		"WhatsApp, Instagram, Facebook, Google e TikTok",
		"ANEXO I · ESCOPO OPERACIONAL",
		"Produção e entrega de 12 (doze) imagens/artes estáticas (4 por mês) e 36 (trinta e seis) vídeos/Reels editados ao longo do ciclo estratégico de 90 (noventa) dias, com orientação de gravação (1h/semana em bloco) e direcionamento de Stories.",
		"Campo Grande/MS, 22 de novembro de 2026.",
		"Cláusula 14",
	}
	for _, needle := range mustHaveB {
		if !strings.Contains(htmlBDecoded, needle) {
			t.Errorf("Scenario B missing %q", needle)
		}
	}

	// Cenário C: Tráfego Pago com verba custeada pelo cliente (verba_cliente) + apenas Meta Ads e Google Ads
	verbaClienteEscopo := fullEscopo
	verbaClienteEscopo.ModalidadeTrafego = domain.TrafegoVerbaCliente
	verbaClienteEscopo.PlataformasAds = []string{"Meta Ads (Instagram/Facebook)", "Google Ads"}
	inC := inB
	inC.EscopoMarketing = &verbaClienteEscopo
	htmlC, err := svc.RenderHTML(inC)
	if err != nil {
		t.Fatalf("RenderHTML Scenario C failed: %v", err)
	}
	htmlCDecoded := html.UnescapeString(htmlC)

	mustHaveC := []string{
		"GESTÃO DE TRÁFEGO PAGO E INVESTIMENTO EM MÍDIA",
		"Meta Ads (Instagram/Facebook) e Google Ads",
		"custeada diretamente pela CONTRATANTE às plataformas",
		"responsabilidade exclusiva da CONTRATANTE",
	}
	for _, needle := range mustHaveC {
		if !strings.Contains(htmlCDecoded, needle) {
			t.Errorf("Scenario C missing %q", needle)
		}
	}
	if strings.Contains(htmlCDecoded, "Assunção Integral do Investimento e Risco pela Contratada") {
		t.Errorf("Scenario C (verba_cliente) must not include risco_contratada callout")
	}
}
