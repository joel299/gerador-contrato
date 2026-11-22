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

func TestRenderService_MarketingIAInfinito_CompleteAndPartial(t *testing.T) {
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
	// 1. Complete marketing_ia_infinito contract
	fullEscopo := domain.DefaultEscopoMarketing()
	fullEscopo.QtdImagens = 4
	fullEscopo.FreqImagens = "semanais"
	fullEscopo.QtdReels = 3
	fullEscopo.FreqReels = "semanais"
	fullEscopo.DirecionamentoStories = true
	fullEscopo.HorasGravacaoSemanal = 1

	inFull := domain.ContractInput{
		ModeloContrato:    domain.ModeloMarketingIAInfinito,
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
	htmlFull, err := svc.RenderHTML(inFull)
	if err != nil {
		t.Fatalf("RenderHTML marketing full failed: %v", err)
	}

	mustHaveFull := []string{
		"counter-reset: clausula;",
		"counter-increment: clausula;",
		"JOEL ACOSTA QUINTANA JUNIOR DESENVOLVIMENTO DE PROGRAMAS LTDA - EPP",
		"63.741.075/0001-43",
		"Joel Acosta Quintana Junior",
		"Clínica Exemplo",
		"TRÁFEGO PAGO, INVESTIMENTO E ASSUNÇÃO DE RISCO",
		"Assunção Integral do Investimento e Risco pela Contratada",
		"IMPLANTAÇÃO / SETUP",
		"1.500,00",
		"2.500,00",
		"WhatsApp, Instagram, Facebook, Google e TikTok",
		"criação de 4 (quatro) imagens/artes semanais",
		"edição de até 3 (três) Reels semanais + direcionamento de Stories",
		"Campo Grande/MS, 22 de novembro de 2026.",
		"Cláusula 14",
	}
	htmlFullDecoded := html.UnescapeString(htmlFull)
	for _, needle := range mustHaveFull {
		if !strings.Contains(htmlFullDecoded, needle) {
			t.Errorf("full marketing contract missing %q", needle)
		}
	}

	// 2. Partial marketing_ia_infinito contract (no setup, no paid traffic, no link multicanal, 8 monthly images, 0 reels, no stories)
	partialEscopo := domain.EscopoMarketing{
		CicloEstrategicoDias:    90,
		PosicionamentoEditorial: true,
		ConteudoOrganico:        true,
		LinkMulticanal:          false,
		TrafegoPago:             false,
		CaptacaoLeads:           true,
		QtdImagens:              8,
		FreqImagens:             "mensais",
		QtdReels:                0,
		FreqReels:               "semanais",
		DirecionamentoStories:   false,
		HorasGravacaoSemanal:    2,
		Canais:                  []string{"WhatsApp"},
	}
	inPartial := inFull
	inPartial.EscopoMarketing = &partialEscopo
	inPartial.SetupFeeCents = 0

	htmlPartial, err := svc.RenderHTML(inPartial)
	if err != nil {
		t.Fatalf("RenderHTML marketing partial failed: %v", err)
	}

	mustHavePartial := []string{
		"criação de 8 (oito) imagens/artes mensais",
		"orientação de gravação (2h/semana em bloco)",
		"Cláusula 13",
		"2.500,00",
	}
	for _, needle := range mustHavePartial {
		if !strings.Contains(htmlPartial, needle) {
			t.Errorf("partial marketing contract missing %q", needle)
		}
	}

	mustOmitPartial := []string{
		"TRÁFEGO PAGO, INVESTIMENTO E ASSUNÇÃO DE RISCO",
		"TRÁFEGO PAGO E RISCO",
		"CANAIS INTEGRADOS",
		"IMPLANTAÇÃO / SETUP",
		"Implantação / Setup:",
		"com verba de tráfego pago já assumida pela CONTRATADA",
		"Custear integralmente a verba de tráfego pago",
		"direcionamento de Stories",
		"Reels",
	}
	for _, forbidden := range mustOmitPartial {
		if strings.Contains(htmlPartial, forbidden) {
			t.Errorf("partial marketing contract must omit %q", forbidden)
		}
	}
}
