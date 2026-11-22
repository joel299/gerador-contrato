package domain

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
)

// ContractStatus represents the lifecycle state of a contract.
type ContractStatus string

const (
	StatusDraft         ContractStatus = "DRAFT"
	StatusGenerated     ContractStatus = "GENERATED"
	StatusSentSignature ContractStatus = "SENT_SIGNATURE"
	StatusSigned        ContractStatus = "SIGNED"
	StatusCanceled      ContractStatus = "CANCELED"
)

const (
	ModeloPadrao              = "padrao"
	ModeloMarketingIAInfinito = "marketing_ia_infinito"
)

// EscopoMarketing holds the dynamic scope configuration for the marketing_ia_infinito contract template.
type EscopoMarketing struct {
	CicloEstrategicoDias int `json:"ciclo_estrategico_dias"`

	PosicionamentoEditorial bool `json:"posicionamento_editorial"`
	ConteudoOrganico        bool `json:"conteudo_organico"`
	LinkMulticanal          bool `json:"link_multicanal"`
	TrafegoPago             bool `json:"trafego_pago"`
	CaptacaoLeads           bool `json:"captacao_leads"`

	QtdImagens            int    `json:"qtd_imagens"`
	FreqImagens           string `json:"freq_imagens"`
	QtdReels              int    `json:"qtd_reels"`
	FreqReels             string `json:"freq_reels"`
	DirecionamentoStories bool   `json:"direcionamento_stories"`
	HorasGravacaoSemanal  int    `json:"horas_gravacao_semanal"`

	Canais []string `json:"canais"`
}

// DefaultEscopoMarketing returns the default full-scope configuration for IA Infinito Marketing & Traffic contracts.
func DefaultEscopoMarketing() EscopoMarketing {
	return EscopoMarketing{
		CicloEstrategicoDias:    90,
		PosicionamentoEditorial: true,
		ConteudoOrganico:        true,
		LinkMulticanal:          true,
		TrafegoPago:             true,
		CaptacaoLeads:           true,
		QtdImagens:              0,
		FreqImagens:             "semanais",
		QtdReels:                3,
		FreqReels:               "semanais",
		DirecionamentoStories:   true,
		HorasGravacaoSemanal:    1,
		Canais:                  []string{"WhatsApp", "Instagram", "Facebook", "Google", "TikTok"},
	}
}

// Normalize sanitizes numeric bounds, frequencies, and channel lists.
func (e *EscopoMarketing) Normalize() {
	if e.CicloEstrategicoDias <= 0 {
		e.CicloEstrategicoDias = 90
	}
	if e.QtdImagens < 0 {
		e.QtdImagens = 0
	}
	if e.FreqImagens != "semanais" && e.FreqImagens != "mensais" {
		e.FreqImagens = "semanais"
	}
	if e.QtdReels < 0 {
		e.QtdReels = 0
	}
	if e.FreqReels != "semanais" && e.FreqReels != "mensais" {
		e.FreqReels = "semanais"
	}
	if e.HorasGravacaoSemanal < 0 {
		e.HorasGravacaoSemanal = 0
	}
	var cleaned []string
	seen := make(map[string]bool)
	for _, ch := range e.Canais {
		t := strings.TrimSpace(ch)
		if t != "" && !seen[t] {
			seen[t] = true
			cleaned = append(cleaned, t)
		}
	}
	e.Canais = cleaned
}

// ContractInput represents the user-submitted payload for previewing, creating, or updating a contract.
type ContractInput struct {
	ModeloContrato  string           `json:"modelo_contrato,omitempty"`
	EscopoMarketing *EscopoMarketing `json:"escopo_marketing,omitempty"`

	ClientCompanyName      string `json:"client_company_name"`
	ClientTradeName        string `json:"client_trade_name,omitempty"`
	ClientCNPJ             string `json:"client_cnpj"`
	ClientAddress          string `json:"client_address"`
	ClientRepName          string `json:"client_rep_name"`
	ClientRepDocument      string `json:"client_rep_document"`
	ClientRepNationality   string `json:"client_rep_nationality,omitempty"`
	ClientRepMaritalStatus string `json:"client_rep_marital_status,omitempty"`
	ClientRepProfession    string `json:"client_rep_profession,omitempty"`
	ClientEmail            string `json:"client_email"`
	ClientPhone            string `json:"client_phone"`

	SetupFeeCents      int64  `json:"setup_fee_cents"`
	SetupInstallments  int    `json:"setup_installments,omitempty"`
	SetupPaymentMethod string `json:"setup_payment_method,omitempty"`
	SetupPaymentForm   string `json:"setup_payment_form,omitempty"`
	HasMonthlyFee      *bool  `json:"has_monthly_fee,omitempty"`
	MonthlyFeeCents    int64  `json:"monthly_fee_cents"`
	MonthlyPaymentForm string `json:"monthly_payment_form,omitempty"`
	PayOnSignature     *bool  `json:"pay_on_signature,omitempty"`
	IncludeWitnesses   *bool  `json:"include_witnesses,omitempty"`
	DueDay             int    `json:"due_day"`
	StartDate          string `json:"start_date"`
	BillingStartDate   string `json:"billing_start_date,omitempty"`
	CityForum          string `json:"city_forum"`

	// Optional Contratada overrides (defaults come from server config)
	ContratadaCompanyName string `json:"contratada_company_name,omitempty"`
	ContratadaCNPJ        string `json:"contratada_cnpj,omitempty"`
	ContratadaAddress     string `json:"contratada_address,omitempty"`
	ContratadaRepName     string `json:"contratada_rep_name,omitempty"`
	ContratadaRepCPF      string `json:"contratada_rep_cpf,omitempty"`
}

// Contract represents a persisted contract record in SQLite.
type Contract struct {
	ID                string         `json:"id"`
	Code              string         `json:"code"`
	ClientCompanyName string         `json:"client_company_name"`
	ClientTradeName   string         `json:"client_trade_name"`
	ClientCNPJ        string         `json:"client_cnpj"`
	ClientAddress     string         `json:"client_address"`
	ClientRepName     string         `json:"client_rep_name"`
	ClientRepDocument string         `json:"client_rep_document"`
	ClientEmail       string         `json:"client_email"`
	ClientPhone       string         `json:"client_phone"`
	SetupFeeCents              int64          `json:"setup_fee_cents"`
	SetupInstallments          int            `json:"setup_installments"`
	SetupPaymentMethod         string         `json:"setup_payment_method"`
	SetupTotalFinalCents       int64          `json:"setup_total_final_cents"`
	SetupInstallmentValueCents int64          `json:"setup_installment_value_cents"`
	HasMonthlyFee             bool           `json:"has_monthly_fee"`
	MonthlyFeeCents           int64          `json:"monthly_fee_cents"`
	PayOnSignature            bool           `json:"pay_on_signature"`
	IncludeWitnesses          bool           `json:"include_witnesses"`
	DueDay                    int            `json:"due_day"`
	StartDate                 string         `json:"start_date"`
	CityForum                 string         `json:"city_forum"`
	ModeloContrato            string         `json:"modelo_contrato"`
	EscopoJSON                string         `json:"escopo_json"`
	PayloadJSON               string         `json:"payload_json"`
	S3Bucket          string         `json:"s3_bucket,omitempty"`
	S3Key             string         `json:"s3_key,omitempty"`
	PDFHashSHA256     string         `json:"pdf_hash_sha256,omitempty"`
	Status            ContractStatus `json:"status"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// Validate checks required fields and formats on ContractInput.
func (in *ContractInput) Validate(requireStrict bool) error {
	cnpjDigits := OnlyDigits(in.ClientCNPJ)
	if requireStrict {
		if strings.TrimSpace(in.ClientCompanyName) == "" {
			return errors.New("razão social da contratante é obrigatória")
		}
		if !IsValidCNPJ(cnpjDigits) {
			return errors.New("CNPJ da contratante inválido")
		}
		if strings.TrimSpace(in.ClientAddress) == "" {
			return errors.New("endereço completo da contratante é obrigatório")
		}
		if strings.TrimSpace(in.ClientRepName) == "" {
			return errors.New("nome do representante legal é obrigatório")
		}
		if strings.TrimSpace(in.ClientEmail) == "" {
			return errors.New("e-mail de contato é obrigatório")
		}
		if strings.TrimSpace(in.ClientPhone) == "" {
			return errors.New("telefone/WhatsApp é obrigatório")
		}
	} else {
		// Draft mode allows partial data, only validating non-empty CNPJ checksums if provided.
		if cnpjDigits != "" && len(cnpjDigits) == 14 && !IsValidCNPJ(cnpjDigits) {
			return errors.New("CNPJ da contratante inválido")
		}
	}

	// CPF do representante legal é opcional; quando informado, exige validação matemática (Módulo 11).
	if strings.TrimSpace(in.ClientRepDocument) != "" {
		cpfDigits := OnlyDigits(in.ClientRepDocument)
		if !IsValidCPF(cpfDigits) {
			return errors.New("CPF do representante legal inválido")
		}
	}

	if in.SetupFeeCents < 0 || in.MonthlyFeeCents < 0 {
		return errors.New("valores financeiros não podem ser negativos")
	}
	if in.SetupInstallments != 0 && (in.SetupInstallments < 1 || in.SetupInstallments > 12) {
		return errors.New("parcelamento da implantação deve estar entre 1x e 12x")
	}
	if in.DueDay != 0 && (in.DueDay < 1 || in.DueDay > 31) {
		return errors.New("dia de vencimento deve estar entre 1 e 31")
	}
	if requireStrict && in.ModeloContrato == ModeloMarketingIAInfinito {
		if !in.EffectiveHasMonthlyFee() || in.MonthlyFeeCents <= 0 {
			return errors.New("valor da mensalidade recorrente é obrigatório para o modelo Marketing IA Infinito")
		}
	}
	return nil
}

// EffectiveHasMonthlyFee returns whether recurring monthly fee is enabled.
func (in *ContractInput) EffectiveHasMonthlyFee() bool {
	if in.HasMonthlyFee != nil {
		return *in.HasMonthlyFee && in.MonthlyFeeCents > 0
	}
	return in.MonthlyFeeCents > 0
}

// EffectivePayOnSignature returns whether the first installment / setup is paid upon signature.
func (in *ContractInput) EffectivePayOnSignature() bool {
	if in.PayOnSignature != nil {
		return *in.PayOnSignature
	}
	return true
}

// EffectiveIncludeWitnesses returns whether witness signature blocks are included in the contract.
func (in *ContractInput) EffectiveIncludeWitnesses() bool {
	if in.IncludeWitnesses != nil {
		return *in.IncludeWitnesses
	}
	return true
}

// installmentRates maps installment count (1..12) to the net gross-up rate.
var installmentRates = map[int]float64{
	1:  0.0599,
	2:  0.1139,
	3:  0.1249,
	4:  0.1309,
	5:  0.1379,
	6:  0.1449,
	7:  0.1549,
	8:  0.1609,
	9:  0.1669,
	10: 0.1739,
	11: 0.1839,
	12: 0.1879,
}

// CalculateSetupInstallments computes the total final amount and per-installment value in cents
// using gross-up division (total = base / (1 - rate)), rounding each installment to whole cents
// so N * installmentCents == totalFinalCents. For 1x, defaults to PIX (0%); use CalculateSetupInstallmentsWithMethod for 1x card.
func CalculateSetupInstallments(baseCents int64, installments int) (totalFinalCents int64, installmentCents int64) {
	method := "pix"
	if installments >= 2 {
		method = "card"
	}
	return CalculateSetupInstallmentsWithMethod(baseCents, installments, method)
}

// CalculateSetupInstallmentsWithMethod computes grossed-up total and installment cents for PIX ("pix") or Credit Card ("card").
func CalculateSetupInstallmentsWithMethod(baseCents int64, installments int, paymentMethod string) (totalFinalCents int64, installmentCents int64) {
	if baseCents <= 0 {
		return 0, 0
	}
	if installments < 1 {
		installments = 1
	} else if installments > 12 {
		installments = 12
	}

	rate := 0.0
	if installments >= 2 || paymentMethod == "card" {
		rate = installmentRates[installments]
	}

	totalCents := int64(math.Round(float64(baseCents) / (1.0 - rate)))
	installmentCents = int64(math.Round(float64(totalCents) / float64(installments)))
	totalFinalCents = installmentCents * int64(installments)
	return totalFinalCents, installmentCents
}

// SetupInstallmentValueCents returns the per-installment value in cents.
func (in *ContractInput) SetupInstallmentValueCents() int64 {
	_, inst := CalculateSetupInstallmentsWithMethod(in.SetupFeeCents, in.SetupInstallments, in.EffectiveSetupPaymentMethod())
	return inst
}

// SetupTotalFinalCents returns the total final setup amount in cents after installment calculation.
func (in *ContractInput) SetupTotalFinalCents() int64 {
	total, _ := CalculateSetupInstallmentsWithMethod(in.SetupFeeCents, in.SetupInstallments, in.EffectiveSetupPaymentMethod())
	return total
}

// EffectiveSetupPaymentMethod returns "card" or "pix" (default for 1x).
func (in *ContractInput) EffectiveSetupPaymentMethod() string {
	if in.SetupInstallments >= 2 || in.SetupPaymentMethod == "card" {
		return "card"
	}
	return "pix"
}

// Normalize cleans and applies defaults to ContractInput.
func (in *ContractInput) Normalize(defaultForum string) {
	if strings.TrimSpace(in.ModeloContrato) != ModeloMarketingIAInfinito {
		in.ModeloContrato = ModeloPadrao
	} else {
		in.ModeloContrato = ModeloMarketingIAInfinito
		if in.EscopoMarketing == nil {
			def := DefaultEscopoMarketing()
			in.EscopoMarketing = &def
		}
		in.EscopoMarketing.Normalize()
	}
	in.ClientCompanyName = strings.TrimSpace(in.ClientCompanyName)
	in.ClientTradeName = strings.TrimSpace(in.ClientTradeName)
	in.ClientCNPJ = OnlyDigits(in.ClientCNPJ)
	in.ClientAddress = strings.TrimSpace(in.ClientAddress)
	in.ClientRepName = strings.TrimSpace(in.ClientRepName)
	in.ClientRepDocument = strings.TrimSpace(in.ClientRepDocument)
	in.ClientEmail = strings.TrimSpace(in.ClientEmail)
	in.ClientPhone = strings.TrimSpace(in.ClientPhone)
	in.CityForum = strings.TrimSpace(in.CityForum)
	if in.CityForum == "" {
		in.CityForum = defaultForum
	}
	if in.SetupInstallments < 1 {
		in.SetupInstallments = 1
	} else if in.SetupInstallments > 12 {
		in.SetupInstallments = 12
	}
	if in.HasMonthlyFee != nil && !*in.HasMonthlyFee {
		in.MonthlyFeeCents = 0
	}
	if in.DueDay == 0 {
		in.DueDay = 10
	}
	if strings.TrimSpace(in.StartDate) == "" {
		in.StartDate = time.Now().Format("2006-01-02")
	}
	if strings.TrimSpace(in.SetupPaymentForm) == "" {
		in.SetupPaymentForm = "À vista na assinatura"
	}
	if strings.TrimSpace(in.MonthlyPaymentForm) == "" {
		in.MonthlyPaymentForm = "Boleto / PIX / Transferência"
	}
}

// OnlyDigits strips all non-numeric characters from a string.
func OnlyDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// FormatCNPJ formats a 14-digit string as 00.000.000/0000-00.
func FormatCNPJ(s string) string {
	d := OnlyDigits(s)
	if len(d) != 14 {
		return s
	}
	return fmt.Sprintf("%s.%s.%s/%s-%s", d[0:2], d[2:5], d[5:8], d[8:12], d[12:14])
}

// FormatCPF formats an 11-digit string as 000.000.000-00.
func FormatCPF(s string) string {
	d := OnlyDigits(s)
	if len(d) != 11 {
		return s
	}
	return fmt.Sprintf("%s.%s.%s-%s", d[0:3], d[3:6], d[6:9], d[9:11])
}

// IsValidCNPJ validates a Brazilian CNPJ number including check digits.
func IsValidCNPJ(cnpj string) bool {
	d := OnlyDigits(cnpj)
	if len(d) != 14 || allSameDigits(d) {
		return false
	}

	w1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	w2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	if calcCheckDigit(d[:12], w1) != int(d[12]-'0') {
		return false
	}
	return calcCheckDigit(d[:13], w2) == int(d[13]-'0')
}

// IsValidCPF validates a Brazilian CPF number including check digits.
func IsValidCPF(cpf string) bool {
	d := OnlyDigits(cpf)
	if len(d) != 11 || allSameDigits(d) {
		return false
	}

	w1 := []int{10, 9, 8, 7, 6, 5, 4, 3, 2}
	w2 := []int{11, 10, 9, 8, 7, 6, 5, 4, 3, 2}

	if calcCheckDigit(d[:9], w1) != int(d[9]-'0') {
		return false
	}
	return calcCheckDigit(d[:10], w2) == int(d[10]-'0')
}

func allSameDigits(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}
	return true
}

func calcCheckDigit(digits string, weights []int) int {
	sum := 0
	for i, w := range weights {
		sum += int(digits[i]-'0') * w
	}
	rem := sum % 11
	if rem < 2 {
		return 0
	}
	return 11 - rem
}

// FormatBRL formats integer cents as Brazilian currency string without the "R$" prefix (e.g., "2.500,00").
func FormatBRL(cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	reais := cents / 100
	cent := cents % 100

	reaisStr := fmt.Sprintf("%d", reais)
	var grouped strings.Builder
	n := len(reaisStr)
	for i, ch := range reaisStr {
		if i > 0 && (n-i)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(ch)
	}
	res := fmt.Sprintf("%s,%02d", grouped.String(), cent)
	if neg {
		return "-" + res
	}
	return res
}

// CentsToExtenso converts an integer amount in BRL cents into Portuguese words.
func CentsToExtenso(cents int64) string {
	if cents < 0 {
		return "menos " + CentsToExtenso(-cents)
	}
	if cents == 0 {
		return "zero reais"
	}

	reais := cents / 100
	centavos := cents % 100

	var parts []string
	if reais > 0 {
		reaisWords := numberToPortuguese(reais)
		if reais == 1 {
			parts = append(parts, reaisWords+" real")
		} else if reais >= 1000000 && reais%1000000 == 0 {
			parts = append(parts, reaisWords+" de reais")
		} else {
			parts = append(parts, reaisWords+" reais")
		}
	}

	if centavos > 0 {
		centWords := numberToPortuguese(centavos)
		if centavos == 1 {
			parts = append(parts, centWords+" centavo")
		} else {
			parts = append(parts, centWords+" centavos")
		}
	}

	return strings.Join(parts, " e ")
}

// NumberToPortuguese converts a non-negative integer into Portuguese words (e.g., 90 -> "noventa").
func NumberToPortuguese(n int64) string {
	return numberToPortuguese(n)
}

func numberToPortuguese(n int64) string {
	if n == 0 {
		return "zero"
	}
	if n == 100 {
		return "cem"
	}

	unidades := []string{"", "um", "dois", "três", "quatro", "cinco", "seis", "sete", "oito", "nove",
		"dez", "onze", "doze", "treze", "quatorze", "quinze", "dezesseis", "dezessete", "dezoito", "dezenove"}
	dezenas := []string{"", "", "vinte", "trinta", "quarenta", "cinquenta", "sessenta", "setenta", "oitenta", "noventa"}
	centenas := []string{"", "cento", "duzentos", "trezentos", "quatrocentos", "quinhentos", "seiscentos", "setecentos", "oitocentos", "novecentos"}

	threeDigits := func(v int64) string {
		if v == 0 {
			return ""
		}
		if v == 100 {
			return "cem"
		}
		var out []string
		c := v / 100
		rem := v % 100
		if c > 0 {
			out = append(out, centenas[c])
		}
		if rem > 0 {
			if rem < 20 {
				out = append(out, unidades[rem])
			} else {
				d := rem / 10
				u := rem % 10
				if u > 0 {
					out = append(out, dezenas[d]+" e "+unidades[u])
				} else {
					out = append(out, dezenas[d])
				}
			}
		}
		return strings.Join(out, " e ")
	}

	type scale struct {
		divisor  int64
		singular string
		plural   string
	}
	scales := []scale{
		{1000000000, "bilhão", "bilhões"},
		{1000000, "milhão", "milhões"},
		{1000, "mil", "mil"},
		{1, "", ""},
	}

	type group struct {
		val  int64
		text string
	}
	var groups []group
	rem := n
	for _, sc := range scales {
		q := rem / sc.divisor
		rem = rem % sc.divisor
		if q == 0 {
			continue
		}
		w := threeDigits(q)
		if sc.divisor == 1000 && q == 1 {
			groups = append(groups, group{val: q * sc.divisor, text: "mil"})
		} else if sc.singular != "" {
			name := sc.plural
			if q == 1 {
				name = sc.singular
			}
			groups = append(groups, group{val: q * sc.divisor, text: w + " " + name})
		} else {
			groups = append(groups, group{val: q, text: w})
		}
	}

	if len(groups) == 1 {
		return groups[0].text
	}

	var result strings.Builder
	for i, g := range groups {
		if i > 0 {
			isLast := i == len(groups)-1
			lastRaw := g.val
			if isLast && (lastRaw < 100 || lastRaw%100 == 0) {
				result.WriteString(" e ")
			} else {
				result.WriteString(", ")
			}
		}
		result.WriteString(g.text)
	}
	return result.String()
}

// NewUUIDv7 generates a time-ordered UUID v7 string.
func NewUUIDv7() string {
	var u [16]byte
	ms := uint64(time.Now().UnixMilli())
	binary.BigEndian.PutUint64(u[0:8], ms<<16)
	_, _ = rand.Read(u[6:])
	u[6] = (u[6] & 0x0f) | 0x70 // version 7
	u[8] = (u[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}
