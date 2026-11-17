CREATE TABLE IF NOT EXISTS contracts (
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

CREATE INDEX IF NOT EXISTS idx_contracts_cnpj ON contracts(client_cnpj);
CREATE INDEX IF NOT EXISTS idx_contracts_company ON contracts(client_company_name);
CREATE INDEX IF NOT EXISTS idx_contracts_status ON contracts(status);
CREATE INDEX IF NOT EXISTS idx_contracts_created_at ON contracts(created_at DESC);

CREATE TABLE IF NOT EXISTS cnpj_cache (
    cnpj TEXT PRIMARY KEY,
    response_json TEXT NOT NULL,
    cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
