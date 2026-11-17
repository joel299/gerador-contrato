# Estado atual — Gerador de Contratos IA Infinito

**Última atualização:** 17/11/2026  
**Status:** **v1.6 em produção com CI/CD automatizado (GitHub Actions → GHCR → Coolify).** Pipeline verde de ponta a ponta: testes, build `linux/amd64`, push para `ghcr.io/joel299/gerador-contrato:coolify`, chamada do webhook do Coolify com retorno **HTTP 200** e container `healthy` servindo HTTPS.

## Deploy em produção (verificado)

- **Recurso no Coolify:** projeto `IA Infinito` → ambiente `production` → aplicação **Gerador de Contratos IA Infinito** (`build_pack: dockercompose`, compose `docker-compose.coolify.yml`, branch `main`).
- **Imagem:** `ghcr.io/joel299/gerador-contrato:coolify` (pacote privado no GHCR; o pull a partir da VPS funciona com a credencial já existente no host).
- **Domínio:** `https://contratos-ia-infinito.18.118.25.255.sslip.io` (gerado no padrão `sslip.io` do próprio Coolify, com HTTPS válido e Let's Encrypt via Traefik).
- **Porta interna:** `8080` (serviço `app`, sem proxy Traefik próprio; roteamento gerenciado pelo Coolify).
- **Persistência:** volume nomeado `olnybeujwwk55ukczny7rwax_gerador-contrato-data:/app/data` (SQLite WAL + PDFs locais).
- **Limite de memória:** `mem_limit: 768m`; `pull_policy: always` para sempre buscar a imagem nova no redeploy.
- **Healthcheck:** `GET /health` → `200 {"chromium_pool_ok":true,"sqlite_ok":true,"status":"ok","storage_provider":"local"}`.

### Variáveis configuradas no Coolify

Obrigatórias/preenchidas: `PORT`, `DB_PATH`, `LOCAL_PDF_DIR`, `TZ`, `DEFAULT_PAYMENT_LINK` e o par de domínio do compose (`SERVICE_FQDN_APP`, `SERVICE_URL_APP`).  
Opcionais criadas vazias, prontas para preenchimento: `S3_ENDPOINT`, `S3_BUCKET`, `S3_REGION`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`.

> Nenhum valor de segredo, token ou URL privada de webhook é registrado neste documento.

### Secrets do GitHub Actions

Cadastrados em `joel299/gerador-contrato` → Settings → Secrets and variables → Actions:

- `COOLIFY_DEPLOY_WEBHOOK` — URL de deploy do recurso no Coolify.
- `COOLIFY_DEPLOY_TOKEN` — token de API do Coolify, enviado no header `Authorization: Bearer`.

## Pipeline CI/CD

`.github/workflows/publish-coolify-image.yml` (dispara em `push` na `main` e em `workflow_dispatch`):

1. **Job `test-and-build-image`:** checkout, Go via `go.mod`, instalação do Chrome for Testing (`browser-actions/setup-chrome`, `CHROME_PATH` exportado para o teste de PDF real), `go vet ./...`, `go test ./...`, build e push `linux/amd64` com tags `:coolify` e `:sha-<commit>` + labels de origem do repositório.
2. **Job `trigger-coolify-deploy`:** `POST` no webhook do Coolify com `Authorization: Bearer`; falha explicitamente se o webhook não estiver configurado ou se a resposta não for 2xx.

## Correções aplicadas nesta etapa

1. **`cmd/server/main.go` não estava versionado** — a regra `server` do `.gitignore` ignorava a pasta `cmd/server`, então o build Docker falhava com `stat /app/cmd/server: directory not found`. Regra ajustada para `/server` (apenas binário na raiz) e arquivo versionado.
2. **Teste de PDF falhava no runner** (`chromedp ... starting the browser: context deadline exceeded`) — runner sem Chrome. Adicionado `setup-chrome` no job e o teste passou a respeitar `CHROME_PATH`.
3. **`CHROME_BIN`/`CHROME_PATH` robustos na imagem** — criado symlink estável `/usr/local/bin/chrome` (o nome do binário varia entre versões do pacote `chromium` no Alpine).
4. **Fuso horário do container** — `tzdata` + `TZ=America/Campo_Grande` para não deslocar datas padrão do contrato em UTC.
5. **Compose parametrizado** — `PORT`, `DB_PATH`, `LOCAL_PDF_DIR` e `TZ` agora aceitam override pelas variáveis do Coolify.

## Validações executadas

- `go vet ./...`: aprovado sem alertas (local, 17/11/2026).
- `go test ./...`: aprovado (local e no runner, incluindo geração real de PDF via `chromedp`).
- `docker compose -f docker-compose.coolify.yml config`: sintaxe válida (imagem, volume, `mem_limit`, porta e healthcheck conferidos).
- Workflow run `44151116610`: job de testes/build verde; job de deploy falhou por secrets ausentes (comportamento esperado antes da configuração).
- Workflow run `44155345572` (`workflow_dispatch`): **ambos os jobs verdes** e `Coolify webhook HTTP status: 200`.
- Deploy no Coolify (`u1tctnt28aqhpxluxcpdmvpo`, commit `de1c35a`): `finished`; aplicação `running:healthy`; `GET /health` retornando 200 em HTTP e HTTPS.

## Histórico funcional anterior (preservado)

### Estado confirmado

- **Correção BUG-001 (Preview e PDF sem espaço em branco):**
  - `[hidden] { display: none !important; }` e controle explícito de exibição no `onload` do `iframe` eliminaram o skeleton residual no topo do preview.
  - Margem `@page` ajustada para `19mm 15mm 16mm 15mm` em `web/templates/contract_v1.html` e `pdfHeaderTemplate` refinado em `internal/service/pdf_service.go`.
- **CPF opcional + Validação Matemática (Módulo 11):**
  - Toggle `"Incluir CPF do representante legal"` em `web/index.html` com validação Módulo 11 em JS e Go (`internal/domain/contract.go`).
  - Quando desmarcado ou vazio, nenhuma menção a CPF ou `[PREENCHER CPF]` aparece na qualificação nem na assinatura.
- **PWA Completo (`embed.FS`):**
  - `web/manifest.webmanifest`, `web/sw.js`, `web/assets/icon-192.png` e `web/assets/icon-512.png`, servidos por `internal/handler/router.go`.
  - Responsividade mobile (`<= 900px`) com duas abas fixas e barra inferior fixa de ação.
- **Bloco de Assinaturas e Testemunhas Opcionais (v1.3):**
  - **Alinhamento pelo topo (`BUG-003`):** `table.sig-grid` com `table-layout: fixed`, colunas `50%` e `vertical-align: top`.
  - **Testemunhas opcionais (`include_witnesses`):** checkbox no formulário, coluna no SQLite (migration aditiva) e remoção condicional de `TESTEMUNHA 1/2`.
- **Seção 3 — Condições Comerciais e Vigência (v1.2):**
  - **Mensalidade opcional (`has_monthly_fee`)**, **parcelamento da implantação (`setup_installments` 1x–12x)** e **pagamento no ato da assinatura (`pay_on_signature`)**.
  - **Dados fixos da CONTRATADA:** `JOEL ACOSTA QUINTANA JUNIOR DESENVOLVIMENTO DE PROGRAMAS LTDA - EPP` (`IA INFINITO`), CNPJ `63.741.075/0001-43`, representante `Joel Acosta Quintana Junior` (Campo Grande/MS).
- **Fluxo Integrado PDF + WhatsApp:**
  - Botão **"Exportar PDF e Enviar no WhatsApp"** com `navigator.canShare({ files })` em mobile/PWA, download automático `Contrato_{RazaoSocial}.pdf` e abertura de `https://wa.me/55{numero}?text=...`.

## Pendências e próximos passos

1. **Domínio próprio (opcional):** o app está em `*.sslip.io`. Para usar `contratos.iainfinito.com.br`, criar registro DNS **A** apontando para `18.118.25.255` (hoje `iainfinito.com.br` está em outro host) e trocar o domínio no recurso do Coolify.
2. **Credencial GHCR formal (recomendado):** o pull da imagem privada funciona na VPS, mas convém cadastrar a credencial em **Servers → Registries** (escopo `read:packages`) para não depender de login manual no host.
3. **S3 (opcional):** preencher as cinco variáveis no Coolify para sair do armazenamento local de PDFs.
4. **Repositório e pacote:** o repositório está público; o pacote GHCR segue privado (o que exige credencial no Coolify). Torná-lo público é decisão do Joel.
