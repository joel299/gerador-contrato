# Software Design Document (SDD)
**Projeto:** IA Infinito — Motor de Geração e Gestão de Contratos  
**Versão:** 1.0.0  
**Data:** Novembro de 2026  
**Stack Principal:** Go (Golang 1.23+), SQLite (WAL Mode), S3 Compatible Storage, Chromium Headless (`chromedp`), Scalar OpenAPI, Alpine.js + Tailwind CSS.

---

## 1. Visão Geral e Objetivos

### 1.1. Propósito
Construir um sistema web leve, de alta performance e auto-contido em **Go**, capaz de gerar contratos padronizados da **IA Infinito** a partir de um template HTML/CSS fiel à identidade visual da marca. O sistema deve permitir o preenchimento dinâmico de dados cadastrais e financeiros em uma interface dividida (*Split-Screen*), persistir o histórico de contratos em banco de dados local (**SQLite**), armazenar os PDFs gerados em object storage (**S3**) e expor endpoints REST documentados via **Scalar** para integração externa (n8n, CRMs e plataformas de assinatura digital).

### 1.2. Requisitos Não-Funcionais (Performance e UX)
* **Baixo Consumo de Recursos:** Binário único compilado com assets embutidos (`//go:embed`), mantendo consumo base abaixo de 40 MB de RAM em repouso.
* **Respostas Instantâneas e Skeleton UI:** Uso de *Skeleton Screens* durante carregamento de histórico, busca de CNPJ e renderização de PDF, evitando travamentos visuais (*layout shift*).
* **Concorrência Segura:** Pool persistente de contexto do navegador headless (`chromedp`) para evitar o overhead de abrir/fechar processos Chromium a cada requisição.
* **Auditoria e Rastreabilidade:** Todo contrato gerado fica indexado no SQLite com snapshot dos dados JSON e link/chave do arquivo PDF no S3.

---

## 2. Arquitetura do Sistema

### 2.1. Diagrama de Componentes

```text
+-------------------------------------------------------------------+
|                        Cliente Web / API                          |
|  +-----------------------------+  +----------------------------+  |
|  |   Split-Screen UI (/)       |  |   Scalar API Docs (/docs)  |  |
|  |  - Form Lateral + Skeleton  |  |  - OpenAPI 3.1 Spec        |  |
|  |  - Live Preview A4          |  |  - Testes & Integração n8n |  |
|  +--------------+--------------+  +-------------+--------------+  |
+-----------------|-------------------------------|-----------------+
                  | HTTP / JSON                   |
                  v                               v
+-------------------------------------------------------------------+
|                     Servidor HTTP Go (net/http)                   |
|                                                                   |
|  +-------------------+  +------------------+  +----------------+  |
|  |  CNPJ Service     |  | Template Engine  |  | Contract Repo  |  |
|  |  (BrasilAPI +     |  | (html/template)  |  | (SQLite WAL)   |  |
|  |   Cache em Mem.)  |  +--------+---------+  +-------+--------+  |
|  +-------------------+           |                    |           |
|                                  v                    |           |
|                         +--------+---------+          |           |
|                         |   PDF Worker     |          |           |
|                         | (chromedp Pool)  |          |           |
|                         +--------+---------+          |           |
+----------------------------------|--------------------|-----------+
                                   |                    |
                  +----------------+                    |
                  v                                     v
      +-----------------------+             +-----------------------+
      |   S3 Object Storage   |             |   SQLite Database     |
      | (MinIO / AWS S3 / R2) |             |   (./data/contracts.db)|
      +-----------------------+             +-----------------------+
```

### 2.2. Estrutura de Diretórios (Project Skeleton)

```text
ia-infinito-contracts/
├── cmd/
│   └── server/
│       └── main.go                 # Entrypoint, injeção de dependências e graceful shutdown
├── internal/
│   ├── config/
│   │   └── config.go               # Leitura de variáveis de ambiente (.env)
│   ├── database/
│   │   ├── sqlite.go               # Conexão SQLite (WAL, busy_timeout, migrations)
│   │   └── migrations/
│   │       └── 001_init.sql        # Schema inicial de contratos e logs
│   ├── domain/
│   │   └── contract.go             # Structs de domínio, validações e DTOs
│   ├── repository/
│   │   └── contract_repo.go        # CRUD, busca full-text/filtros por CNPJ, Razão Social e Status
│   ├── service/
│   │   ├── cnpj_service.go         # Consulta BrasilAPI com cache TTL em memória
│   │   ├── render_service.go       # Compilação html/template + formatação BRL/extenso
│   │   ├── pdf_service.go          # Pool chromedp para conversão HTML -> PDF A4
│   │   └── storage_service.go      # Upload, download e Presigned URLs via AWS SDK Go v2
│   └── handler/
│       ├── router.go               # Rotas HTTP (Go 1.22+ ServeMux), middlewares e Scalar
│       ├── contract_handler.go     # Endpoints REST de criação, listagem, preview e PDF
│       └── cnpj_handler.go         # Endpoint de autocompletar dados empresariais
├── web/
│   ├── embed.go                    # Diretiva //go:embed para templates, UI e assets
│   ├── index.html                  # Interface Split-Screen com Skeleton UI
│   ├── templates/
│   │   └── contract_v1.html        # Template HTML/CSS oficial do contrato IA Infinito
│   ├── assets/
│   │   └── logo.png                # Logo oficial embutida (ou base64 inline)
│   └── openapi/
│       └── openapi.yaml            # Especificação OpenAPI 3.1 servida no Scalar
├── data/                           # Volume persistente do SQLite (ignorado no Git)
├── Dockerfile                      # Multi-stage build (Go + Alpine + Chromium)
├── docker-compose.yml              # Orquestração da app + MinIO (S3 local opcional)
├── .env.example                    # Variáveis de ambiente documentadas
├── go.mod
└── go.sum
```

---

## 3. Modelagem de Dados (SQLite)

O banco SQLite deve ser inicializado com pragmas otimizados para concorrência e leitura rápida:
* `PRAGMA journal_mode = WAL;`
* `PRAGMA synchronous = NORMAL;`
* `PRAGMA foreign_keys = ON;`
* `PRAGMA busy_timeout = 5000;`
* Driver recomendado: `modernc.org/sqlite` (100% Pure Go, sem necessidade de CGO, facilitando builds multiplataforma) ou `github.com/mattn/go-sqlite3`.

### 3.1. Schema SQL (`001_init.sql`)

```sql
CREATE TABLE IF NOT EXISTS contracts (
    id TEXT PRIMARY KEY,                      -- UUID v7 (ordenável por tempo)
    code TEXT NOT NULL UNIQUE,                -- Código legível ex: CTR-2026-0001
    
    -- Dados da Contratante (Indexados para busca rápida)
    client_company_name TEXT NOT NULL,        -- Razão Social
    client_trade_name TEXT,                   -- Nome Fantasia
    client_cnpj TEXT NOT NULL,                -- Apenas dígitos (14 chars)
    client_address TEXT NOT NULL,
    client_rep_name TEXT NOT NULL,            -- Nome do Responsável Legal
    client_rep_document TEXT NOT NULL,        -- CPF/RG do Responsável
    client_email TEXT NOT NULL,
    client_phone TEXT NOT NULL,

    -- Dados Comerciais e Financeiros (Auditoria)
    setup_fee_cents INTEGER NOT NULL DEFAULT 0,
    monthly_fee_cents INTEGER NOT NULL DEFAULT 0,
    due_day INTEGER NOT NULL,                 -- Dia de vencimento (1 a 31)
    start_date TEXT NOT NULL,                 -- YYYY-MM-DD
    city_forum TEXT NOT NULL,                 -- Foro eleito (ex: Campo Grande/MS)

    -- Snapshot e Armazenamento
    payload_json TEXT NOT NULL,               -- JSON completo usado na renderização
    s3_bucket TEXT,                           -- Nome do bucket S3
    s3_key TEXT,                              -- Caminho do PDF no S3 (ex: 2026/11/CTR-2026-0001.pdf)
    pdf_hash_sha256 TEXT,                     -- Hash de integridade do PDF gerado

    -- Controle e Status
    status TEXT NOT NULL DEFAULT 'DRAFT',     -- DRAFT | GENERATED | SENT_SIGNATURE | SIGNED | CANCELED
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_contracts_cnpj ON contracts(client_cnpj);
CREATE INDEX IF NOT EXISTS idx_contracts_company ON contracts(client_company_name);
CREATE INDEX IF NOT EXISTS idx_contracts_status ON contracts(status);
CREATE INDEX IF NOT EXISTS idx_contracts_created_at ON contracts(created_at DESC);
```

---

## 4. Integração com Object Storage (S3)

Para evitar que o arquivo `.db` cresça indefinidamente com blobs binários e permitir links compartilháveis para assinatura digital, os PDFs gerados são enviados para um bucket compatível com S3 (**AWS S3**, **Cloudflare R2**, **DigitalOcean Spaces** ou **MinIO** auto-hospedado).

### 4.1. Fluxo de Armazenamento
1. O worker `chromedp` gera o buffer `[]byte` do PDF em memória.
2. Calcula-se o hash `SHA-256` do arquivo para garantir integridade jurídica/auditoria.
3. O `StorageService` (usando `github.com/aws/aws-sdk-go-v2/service/s3`) realiza o `PutObject` com `ContentType: "application/pdf"` na chave `contracts/{ano}/{mes}/{codigo}_{cnpj}.pdf`.
4. Os metadados (`s3_bucket`, `s3_key`, `pdf_hash_sha256`) são salvos na linha correspondente do SQLite.
5. Para download ou visualização posterior, a API gera uma **Presigned URL** temporária (válida por 15 minutos) ou faz proxy de stream direto.
6. *Fallback Local:* Caso as credenciais S3 não estejam configuradas no `.env`, o sistema salva automaticamente em `./data/pdfs/` mantendo a mesma interface de código (`StorageProvider`).

---

## 5. Especificação da API & Scalar (`/docs`)

A documentação interativa será servida na rota `/docs` utilizando o **Scalar** apontando para `/openapi.yaml`.

### 5.1. Endpoints Principais

| Método | Rota | Descrição |
| :--- | :--- | :--- |
| `GET` | `/` | Renderiza a aplicação web (Split-Screen Editor + Histórico) |
| `GET` | `/docs` | Interface Scalar OpenAPI 3.1 |
| `GET` | `/openapi.yaml` | Arquivo bruto da especificação OpenAPI |
| `GET` | `/health` | Healthcheck (verifica status do SQLite, S3 e Pool Chromium) |
| `GET` | `/api/v1/cnpj/{cnpj}` | Consulta dados na BrasilAPI, normaliza endereço/sócios e retorna JSON |
| `POST` | `/api/v1/contracts/preview` | Recebe o JSON do formulário e retorna o HTML compilado para o Live Preview |
| `POST` | `/api/v1/contracts` | Salva no SQLite, gera o PDF, envia ao S3 e retorna os dados + URL do PDF |
| `GET` | `/api/v1/contracts` | Lista contratos salvos com paginação e filtro (`?q=razao_ou_cnpj&status=GENERATED`) |
| `GET` | `/api/v1/contracts/{id}` | Retorna detalhes completos e payload JSON para reedição/duplicação |
| `PUT` | `/api/v1/contracts/{id}` | Atualiza dados de um contrato existente, regera o PDF e atualiza o S3 |
| `GET` | `/api/v1/contracts/{id}/pdf` | Redireciona para Presigned URL do S3 ou faz stream do PDF para download |

---

## 6. Design da Interface & Skeleton UI

A interface web (`web/index.html`) será construída sem etapa de compilação pesada (usando **Tailwind CSS** e **Alpine.js** embutidos), dividida em 3 áreas funcionais:

1. **Barra Superior (Header):**
   * Logo da **IA Infinito**.
   * Botão **"Novo Contrato"** e botão **"Contratos Salvos (Histórico)"** (abre drawer lateral com busca instantânea por Razão Social, CNPJ ou Código).
   * Atalho rápido para **API Docs (Scalar)**.
2. **Painel Esquerdo — Editor e Auditoria (40% da largura):**
   * **Bloco 1 — Dados da Empresa:** Campo CNPJ com botão *"Buscar Dados"* (exibe **Skeleton Pulse** nos inputs enquanto consulta a API), Razão Social, Endereço Completo.
   * **Bloco 2 — Responsável Legal e Contato:** Nome do Responsável, CPF/RG, E-mail, Telefone/WhatsApp.
   * **Bloco 3 — Auditoria Financeira e Vigência:** Taxa de Implantação (Setup), Mensalidade (exibe automaticamente o valor por extenso), Dia de Vencimento, Data de Assinatura, Cidade/Foro.
   * **Ações:** *"Salvar Rascunho"* e *"Gerar PDF & Salvar no S3"*.
3. **Painel Direito — Live Preview A4 (60% da largura):**
   * Renderiza a folha A4 com zoom ajustável e paginação visual.
   * Durante a inicialização ou troca de contrato do histórico, exibe um **Skeleton Document** (linhas cinzas animadas simulando cabeçalho, tabela e parágrafos) até o HTML renderizado estar pronto.
   * Debounce de `250ms` nas digitações para atualizar o preview sem gargalos.

---

## 7. Estratégia de Performance (Pool `chromedp`)

Abrir uma instância do Google Chrome/Chromium a cada clique em "Gerar PDF" leva entre 1.5s e 3s e gera picos de CPU. Para garantir geração em **< 300ms**:

1. Na inicialização do `main.go`, cria-se um `ExecAllocator` global do `chromedp` com flags otimizadas para container (`--no-sandbox`, `--disable-gpu`, `--disable-dev-shm-usage`, `--headless`).
2. Um **Worker Pool** (ex: 2 abas/contextos pré-aquecidos protegidos por canal/semáforo) recebe o HTML compilado via `page.SetDocumentContent`.
3. O comando `page.PrintToPDF` é executado diretamente em memória com margens zero (controladas via `@page` no CSS do template) e `PrintBackground: true`.

---

## 8. Plano de Implementação (Fases)

* **Fase 1 — Skeleton do Projeto & Banco de Dados:** Criar estrutura de diretórios, `config.go`, conexão SQLite em modo WAL, execução automática da migration `001_init.sql` e servidor HTTP base com Scalar em `/docs`.
* **Fase 2 — Motor de Template & PDF Pool:** Embutir o HTML oficial do contrato da **IA Infinito** (`contract_v1.html`) e a nova logo, implementar formatadores de moeda/extenso em Go e configurar o pool `chromedp`.
* **Fase 3 — Integração S3 & Consulta CNPJ:** Implementar `StorageService` (S3 + fallback local) e client da BrasilAPI com cache em memória.
* **Fase 4 — Interface Split-Screen & Histórico:** Desenvolver o frontend em `web/index.html` com formulário auditável, estados de Skeleton UI, preview A4 em tempo real e modal de pesquisa de contratos salvos.
* **Fase 5 — Empacotamento Docker:** Criar `Dockerfile` multi-stage (compilando o binário Go estático + instalando `chromium` no Alpine) e `docker-compose.yml` pronto para deploy.