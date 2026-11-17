package repository

import (
	"context"
	"path/filepath"
	"testing"

	"ia-infinito-contracts/internal/database"
	"ia-infinito-contracts/internal/domain"
)

func TestContractRepository_WAL_CRUD_Search(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite failed: %v", err)
	}
	defer db.Close()

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("expected WAL journal_mode, got %q", journalMode)
	}

	repo := NewContractRepository(db)
	ctx := context.Background()

	c1 := &domain.Contract{
		ClientCompanyName: "Alpha Tecnologia Ltda",
		ClientTradeName:   "Alpha Tech",
		ClientCNPJ:        "11222333000181",
		ClientAddress:     "Rua das Flores, 100, Centro, Campo Grande/MS, CEP 79000-000",
		ClientRepName:     "João Silva",
		ClientRepDocument: "529.982.247-25",
		ClientEmail:       "joao@alpha.com.br",
		ClientPhone:       "(67) 99999-0001",
		SetupFeeCents:              600000,
		SetupInstallments:          4,
		SetupPaymentMethod:         "card",
		SetupTotalFinalCents:       690368,
		SetupInstallmentValueCents: 172592,
		HasMonthlyFee:              false,
		MonthlyFeeCents:            0,
		PayOnSignature:             true,
		IncludeWitnesses:           false,
		DueDay:                     15,
		StartDate:         "2026-11-17",
		CityForum:         "Campo Grande/MS",
		PayloadJSON:       `{"client_company_name":"Alpha Tecnologia Ltda"}`,
		Status:            domain.StatusDraft,
	}
	if err := repo.Create(ctx, c1); err != nil {
		t.Fatalf("Create c1 failed: %v", err)
	}
	if c1.ID == "" || c1.Code == "" {
		t.Fatalf("expected generated ID and Code, got ID=%q Code=%q", c1.ID, c1.Code)
	}

	got, err := repo.GetByID(ctx, c1.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.ClientCompanyName != c1.ClientCompanyName || got.SetupInstallments != 4 || got.SetupPaymentMethod != "card" || got.HasMonthlyFee || !got.PayOnSignature || got.IncludeWitnesses || got.SetupTotalFinalCents != 690368 || got.SetupInstallmentValueCents != 172592 {
		t.Errorf("unexpected loaded contract fields: %+v", got)
	}

	// Update company name
	got.ClientCompanyName = "Alpha Tecnologia S.A."
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Search by Razão Social
	byName, err := repo.List(ctx, "Alpha", "", 10, 0)
	if err != nil || len(byName) != 1 {
		t.Fatalf("List by name failed: len=%d err=%v", len(byName), err)
	}

	// Search by formatted CNPJ
	byCNPJ, err := repo.List(ctx, "11.222.333/0001-81", "", 10, 0)
	if err != nil || len(byCNPJ) != 1 {
		t.Fatalf("List by CNPJ failed: len=%d err=%v", len(byCNPJ), err)
	}

	// Update PDF metadata
	if err := repo.UpdatePDFMetadata(ctx, got.ID, "local", "2026/11/test.pdf", "abc123", domain.StatusGenerated); err != nil {
		t.Fatalf("UpdatePDFMetadata failed: %v", err)
	}
	updated, _ := repo.GetByID(ctx, got.ID)
	if updated.Status != domain.StatusGenerated || updated.PDFHashSHA256 != "abc123" {
		t.Errorf("unexpected updated metadata: status=%s hash=%s", updated.Status, updated.PDFHashSHA256)
	}
}
