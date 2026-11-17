package domain

import "testing"

func TestIsValidCNPJ(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"11.222.333/0001-81", true},
		{"11222333000181", true},
		{"00.000.000/0000-00", false},
		{"11.111.111/1111-11", false},
		{"12.345.678/0001-99", false},
		{"123", false},
	}
	for _, tc := range tests {
		if got := IsValidCNPJ(tc.input); got != tc.want {
			t.Errorf("IsValidCNPJ(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestIsValidCPF(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"529.982.247-25", true},
		{"52998224725", true},
		{"111.111.111-11", false},
		{"123.456.789-00", false},
	}
	for _, tc := range tests {
		if got := IsValidCPF(tc.input); got != tc.want {
			t.Errorf("IsValidCPF(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestFormatBRL(t *testing.T) {
	tests := []struct {
		cents int64
		want  string
	}{
		{0, "0,00"},
		{150, "1,50"},
		{250000, "2.500,00"},
		{123456789, "1.234.567,89"},
	}
	for _, tc := range tests {
		if got := FormatBRL(tc.cents); got != tc.want {
			t.Errorf("FormatBRL(%d) = %q, want %q", tc.cents, got, tc.want)
		}
	}
}

func TestCentsToExtenso(t *testing.T) {
	tests := []struct {
		cents int64
		want  string
	}{
		{0, "zero reais"},
		{100, "um real"},
		{150, "um real e cinquenta centavos"},
		{250000, "dois mil e quinhentos reais"},
		{123456, "mil, duzentos e trinta e quatro reais e cinquenta e seis centavos"},
		{100000000, "um milhão de reais"},
		{10050, "cem reais e cinquenta centavos"},
	}
	for _, tc := range tests {
		if got := CentsToExtenso(tc.cents); got != tc.want {
			t.Errorf("CentsToExtenso(%d) = %q, want %q", tc.cents, got, tc.want)
		}
	}
}

func TestCalculateSetupInstallments(t *testing.T) {
	tests := []struct {
		baseCents   int64
		installment int
		method      string
		wantTotal   int64
		wantPart    int64
	}{
		{150000, 1, "pix", 150000, 150000},   // 1x PIX: 0.00% -> 1500.00
		{150000, 1, "card", 159557, 159557},  // 1x Card: 5.99% -> 1500 / 0.9401 = 1595.57
		{150000, 2, "card", 169282, 84641},   // 2x: 11.39% -> 1500 / 0.8861 = 1692.81 -> 2 x 846.41
		{150000, 3, "card", 171408, 57136},   // 3x: 12.49% -> 1500 / 0.8751 = 1714.08 -> 3 x 571.36
		{600000, 4, "card", 690368, 172592},  // 4x: 13.09% -> 6000 / 0.8691 = 6903.69 -> 4 x 1725.92
		{150000, 12, "card", 184704, 15392},  // 12x: 18.79% -> 1500 / 0.8121 = 1847.06 -> 12 x 153.92
	}
	for _, tc := range tests {
		gotTotal, gotPart := CalculateSetupInstallmentsWithMethod(tc.baseCents, tc.installment, tc.method)
		if gotTotal != tc.wantTotal || gotPart != tc.wantPart {
			t.Errorf("CalculateSetupInstallmentsWithMethod(%d, %d, %q) = (%d, %d), want (%d, %d)",
				tc.baseCents, tc.installment, tc.method, gotTotal, gotPart, tc.wantTotal, tc.wantPart)
		}
		if gotTotal != gotPart*int64(tc.installment) {
			t.Errorf("invariant failed: total %d != part %d * %d", gotTotal, gotPart, tc.installment)
		}
	}
}
