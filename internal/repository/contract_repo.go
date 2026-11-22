package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"ia-infinito-contracts/internal/domain"
)

var ErrNotFound = errors.New("contrato não encontrado")

// ContractRepository handles CRUD and search queries on the SQLite contracts table.
type ContractRepository struct {
	db *sql.DB
}

func NewContractRepository(db *sql.DB) *ContractRepository {
	return &ContractRepository{db: db}
}

// NextCode generates a sequential human-readable contract code like CTR-2026-0001.
func (r *ContractRepository) NextCode(ctx context.Context) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("CTR-%d-", year)
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM contracts WHERE code LIKE ?`,
		prefix+"%",
	).Scan(&count)
	if err != nil {
		return "", fmt.Errorf("count contracts for code: %w", err)
	}
	return fmt.Sprintf("%s%04d", prefix, count+1), nil
}

func (r *ContractRepository) Create(ctx context.Context, c *domain.Contract) error {
	now := time.Now().UTC()
	if c.ID == "" {
		c.ID = domain.NewUUIDv7()
	}
	if c.Code == "" {
		code, err := r.NextCode(ctx)
		if err != nil {
			return err
		}
		c.Code = code
	}
	if c.Status == "" {
		c.Status = domain.StatusDraft
	}
	c.CreatedAt = now
	c.UpdatedAt = now

	if c.SetupInstallments < 1 {
		c.SetupInstallments = 1
	}
	if c.SetupPaymentMethod == "" {
		c.SetupPaymentMethod = "pix"
	}
	if c.ModeloContrato == "" {
		c.ModeloContrato = domain.ModeloPadrao
	}
	if c.EscopoJSON == "" {
		c.EscopoJSON = "{}"
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO contracts (
			id, code, client_company_name, client_trade_name, client_cnpj,
			client_address, client_rep_name, client_rep_document, client_email, client_phone,
			setup_fee_cents, setup_installments, setup_payment_method, setup_total_final_cents, setup_installment_value_cents,
			has_monthly_fee, monthly_fee_cents, pay_on_signature, include_witnesses,
			due_day, start_date, city_forum,
			modelo_contrato, escopo_json,
			payload_json, s3_bucket, s3_key, pdf_hash_sha256, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Code, c.ClientCompanyName, c.ClientTradeName, c.ClientCNPJ,
		c.ClientAddress, c.ClientRepName, c.ClientRepDocument, c.ClientEmail, c.ClientPhone,
		c.SetupFeeCents, c.SetupInstallments, c.SetupPaymentMethod, c.SetupTotalFinalCents, c.SetupInstallmentValueCents,
		boolToInt(c.HasMonthlyFee), c.MonthlyFeeCents, boolToInt(c.PayOnSignature), boolToInt(c.IncludeWitnesses),
		c.DueDay, c.StartDate, c.CityForum,
		c.ModeloContrato, c.EscopoJSON,
		c.PayloadJSON, c.S3Bucket, c.S3Key, c.PDFHashSHA256, string(c.Status), c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert contract: %w", err)
	}
	return nil
}

func (r *ContractRepository) Update(ctx context.Context, c *domain.Contract) error {
	c.UpdatedAt = time.Now().UTC()
	if c.SetupInstallments < 1 {
		c.SetupInstallments = 1
	}
	if c.ModeloContrato == "" {
		c.ModeloContrato = domain.ModeloPadrao
	}
	if c.EscopoJSON == "" {
		c.EscopoJSON = "{}"
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE contracts SET
			client_company_name = ?,
			client_trade_name = ?,
			client_cnpj = ?,
			client_address = ?,
			client_rep_name = ?,
			client_rep_document = ?,
			client_email = ?,
			client_phone = ?,
			setup_fee_cents = ?,
			setup_installments = ?,
			setup_payment_method = ?,
			setup_total_final_cents = ?,
			setup_installment_value_cents = ?,
			has_monthly_fee = ?,
			monthly_fee_cents = ?,
			pay_on_signature = ?,
			include_witnesses = ?,
			due_day = ?,
			start_date = ?,
			city_forum = ?,
			modelo_contrato = ?,
			escopo_json = ?,
			payload_json = ?,
			status = ?,
			updated_at = ?
		WHERE id = ?`,
		c.ClientCompanyName, c.ClientTradeName, c.ClientCNPJ, c.ClientAddress,
		c.ClientRepName, c.ClientRepDocument, c.ClientEmail, c.ClientPhone,
		c.SetupFeeCents, c.SetupInstallments, c.SetupPaymentMethod, c.SetupTotalFinalCents, c.SetupInstallmentValueCents,
		boolToInt(c.HasMonthlyFee), c.MonthlyFeeCents, boolToInt(c.PayOnSignature), boolToInt(c.IncludeWitnesses),
		c.DueDay, c.StartDate, c.CityForum,
		c.ModeloContrato, c.EscopoJSON,
		c.PayloadJSON, string(c.Status), c.UpdatedAt, c.ID,
	)
	if err != nil {
		return fmt.Errorf("update contract: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ContractRepository) UpdatePDFMetadata(ctx context.Context, id, bucket, key, sha256Hex string, status domain.ContractStatus) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE contracts SET
			s3_bucket = ?,
			s3_key = ?,
			pdf_hash_sha256 = ?,
			status = ?,
			updated_at = ?
		WHERE id = ?`,
		bucket, key, sha256Hex, string(status), time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("update pdf metadata: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ContractRepository) GetByID(ctx context.Context, id string) (*domain.Contract, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, code, client_company_name, COALESCE(client_trade_name, ''), client_cnpj,
		       client_address, client_rep_name, client_rep_document, client_email, client_phone,
		       setup_fee_cents, setup_installments, COALESCE(setup_payment_method, 'pix'), setup_total_final_cents, setup_installment_value_cents,
		       has_monthly_fee, monthly_fee_cents, pay_on_signature, include_witnesses,
		       due_day, start_date, city_forum,
		       COALESCE(modelo_contrato, 'padrao'), COALESCE(escopo_json, '{}'),
		       payload_json, COALESCE(s3_bucket, ''), COALESCE(s3_key, ''), COALESCE(pdf_hash_sha256, ''),
		       status, created_at, updated_at
		FROM contracts WHERE id = ?`, id)

	c, err := scanContract(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get contract by id: %w", err)
	}
	return c, nil
}

// List returns contracts filtered by optional search query (Razão Social, CNPJ, or Code) and status.
func (r *ContractRepository) List(ctx context.Context, query, status string, limit, offset int) ([]domain.Contract, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var where []string
	var args []any

	q := strings.TrimSpace(query)
	if q != "" {
		digits := domain.OnlyDigits(q)
		if digits != "" {
			where = append(where, "(client_company_name LIKE ? OR client_trade_name LIKE ? OR code LIKE ? OR client_cnpj LIKE ?)")
			likeQ := "%" + q + "%"
			likeD := "%" + digits + "%"
			args = append(args, likeQ, likeQ, likeQ, likeD)
		} else {
			where = append(where, "(client_company_name LIKE ? OR client_trade_name LIKE ? OR code LIKE ?)")
			likeQ := "%" + q + "%"
			args = append(args, likeQ, likeQ, likeQ)
		}
	}

	if st := strings.TrimSpace(status); st != "" {
		where = append(where, "status = ?")
		args = append(args, st)
	}

	sqlStr := `
		SELECT id, code, client_company_name, COALESCE(client_trade_name, ''), client_cnpj,
		       client_address, client_rep_name, client_rep_document, client_email, client_phone,
		       setup_fee_cents, setup_installments, COALESCE(setup_payment_method, 'pix'), setup_total_final_cents, setup_installment_value_cents,
		       has_monthly_fee, monthly_fee_cents, pay_on_signature, include_witnesses,
		       due_day, start_date, city_forum,
		       COALESCE(modelo_contrato, 'padrao'), COALESCE(escopo_json, '{}'),
		       payload_json, COALESCE(s3_bucket, ''), COALESCE(s3_key, ''), COALESCE(pdf_hash_sha256, ''),
		       status, created_at, updated_at
		FROM contracts`
	if len(where) > 0 {
		sqlStr += " WHERE " + strings.Join(where, " AND ")
	}
	sqlStr += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close()

	var list []domain.Contract
	for rows.Next() {
		c, err := scanContract(rows)
		if err != nil {
			return nil, fmt.Errorf("scan contract: %w", err)
		}
		list = append(list, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate contracts: %w", err)
	}
	if list == nil {
		list = []domain.Contract{}
	}
	return list, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanContract(s scanner) (*domain.Contract, error) {
	var c domain.Contract
	var status string
	var hasMonthly, payOnSig, includeWitnesses int
	err := s.Scan(
		&c.ID, &c.Code, &c.ClientCompanyName, &c.ClientTradeName, &c.ClientCNPJ,
		&c.ClientAddress, &c.ClientRepName, &c.ClientRepDocument, &c.ClientEmail, &c.ClientPhone,
		&c.SetupFeeCents, &c.SetupInstallments, &c.SetupPaymentMethod, &c.SetupTotalFinalCents, &c.SetupInstallmentValueCents,
		&hasMonthly, &c.MonthlyFeeCents, &payOnSig, &includeWitnesses,
		&c.DueDay, &c.StartDate, &c.CityForum,
		&c.ModeloContrato, &c.EscopoJSON,
		&c.PayloadJSON, &c.S3Bucket, &c.S3Key, &c.PDFHashSHA256,
		&status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.HasMonthlyFee = hasMonthly != 0
	c.PayOnSignature = payOnSig != 0
	c.IncludeWitnesses = includeWitnesses != 0
	c.Status = domain.ContractStatus(status)
	return &c, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
