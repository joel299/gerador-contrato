package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"ia-infinito-contracts/internal/database"
	"ia-infinito-contracts/internal/domain"
	_ "modernc.org/sqlite"
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
	if updated.ModeloContrato != domain.ModeloPadrao || updated.EscopoJSON != "{}" {
		t.Errorf("expected default modelo_contrato='padrao' and escopo_json='{}', got modelo=%q escopo=%q", updated.ModeloContrato, updated.EscopoJSON)
	}

	// Create and reload a unified contract with custom escopo_json
	escopoJSON := `{"ciclo_estrategico_dias":90,"posicionamento_editorial":true,"conteudo_organico":true,"link_multicanal":true,"trafego_pago":true,"modalidade_trafego":"verba_cliente","plataformas_ads":["Meta Ads (Instagram/Facebook)","Google Ads"],"captacao_leads":true,"qtd_imagens":24,"obs_imagens":"8 por mês","qtd_reels":12,"obs_gravacao":"1h/semana em bloco","direcionamento_stories":true,"horas_gravacao_semanal":1,"canais":["WhatsApp","Instagram"]}`
	c2 := &domain.Contract{
		ClientCompanyName: "Beta Marketing Ltda",
		ClientTradeName:   "Beta Digital",
		ClientCNPJ:        "11222333000181",
		ClientAddress:     "Av. Afonso Pena, 2000, Campo Grande/MS",
		ClientRepName:     "Maria Oliveira",
		ClientEmail:       "maria@beta.com.br",
		ClientPhone:       "(67) 99999-2222",
		SetupFeeCents:     0,
		HasMonthlyFee:     true,
		MonthlyFeeCents:   300000,
		DueDay:            10,
		StartDate:         "2026-11-22",
		CityForum:         "Campo Grande/MS",
		EscopoJSON:        escopoJSON,
		PayloadJSON:       `{"client_company_name":"Beta Marketing Ltda"}`,
		Status:            domain.StatusDraft,
	}
	if err := repo.Create(ctx, c2); err != nil {
		t.Fatalf("Create c2 failed: %v", err)
	}
	got2, err := repo.GetByID(ctx, c2.ID)
	if err != nil {
		t.Fatalf("GetByID c2 failed: %v", err)
	}
	if got2.EscopoJSON != escopoJSON {
		t.Errorf("unexpected contract escopo persistence: escopo=%q", got2.EscopoJSON)
	}
}

func TestOpenSQLite_SafeMigrationPreservesLegacyContracts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	// Simulate pre-existing production DB without modelo_contrato / escopo_json
	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw sqlite: %v", err)
	}
	legacyDDL := `
	CREATE TABLE contracts (
		id TEXT PRIMARY KEY,
		code TEXT NOT NULL UNIQUE,
		client_company_name TEXT NOT NULL,
		client_trade_name TEXT,
		client_cnpj TEXT NOT NULL,
		client_address TEXT NOT NULL,
		client_rep_name TEXT NOT NULL,
		client_rep_document TEXT NOT NULL,
		client_email TEXT NOT NULL,
		client_phone TEXT NOT NULL,
		setup_fee_cents INTEGER NOT NULL DEFAULT 0,
		monthly_fee_cents INTEGER NOT NULL DEFAULT 0,
		due_day INTEGER NOT NULL,
		start_date TEXT NOT NULL,
		city_forum TEXT NOT NULL,
		payload_json TEXT NOT NULL,
		s3_bucket TEXT,
		s3_key TEXT,
		pdf_hash_sha256 TEXT,
		status TEXT NOT NULL DEFAULT 'DRAFT',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	INSERT INTO contracts (
		id, code, client_company_name, client_cnpj, client_address,
		client_rep_name, client_rep_document, client_email, client_phone,
		setup_fee_cents, monthly_fee_cents, due_day, start_date, city_forum, payload_json, status
	) VALUES (
		'legacy-001', 'CTR-2026-0001', 'Empresa Legada Ltda', '11222333000181', 'Rua A, 10',
		'Carlos', '', 'carlos@legada.com', '67999990000',
		150000, 250000, 10, '2026-11-01', 'Campo Grande/MS', '{"client_company_name":"Empresa Legada Ltda"}', 'GENERATED'
	);`
	if _, err := rawDB.Exec(legacyDDL); err != nil {
		_ = rawDB.Close()
		t.Fatalf("seed legacy db: %v", err)
	}
	_ = rawDB.Close()

	// Now open via OpenSQLite (idempotent migration)
	db, err := database.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite on legacy db failed: %v", err)
	}
	defer db.Close()

	repo := NewContractRepository(db)
	got, err := repo.GetByID(context.Background(), "legacy-001")
	if err != nil {
		t.Fatalf("GetByID legacy row failed: %v", err)
	}
	if got.ClientCompanyName != "Empresa Legada Ltda" || got.ModeloContrato != domain.ModeloPadrao || got.EscopoJSON != "{}" {
		t.Errorf("legacy row corrupted after migration: %+v", got)
	}
}
