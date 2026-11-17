# Estado atual — Gerador de Contratos IA Infinito

**Última atualização:** 17/11/2026  
**Status:** **v1.6 + Pipeline CI/CD Coolify (GHCR) Publicado em `main`.** Criados `Dockerfile` multi-stage (`linux/amd64`, Alpine + Chromium + fontes pt-BR, usuário não-root), `.dockerignore`, `docker-compose.coolify.yml` (`mem_limit: 768m`, volume `/app/data`) e workflow `.github/workflows/publish-coolify-image.yml` (`go vet` + `go test` + build/push GHCR + webhook Coolify).

## Retomada rápida

O **Gerador de Contratos IA Infinito** está operacional na versão v1.1, atendendo ao modelo oficial em `Contrato de Prestação de Serviços - IA Infinito.pdf`, ao `sdd_sistema_gerador_de_contratos_ia_infinito.md` e ao `SYSTEM_DESIGN.md`.

### Estado confirmado

- **Correção BUG-001 (Preview e PDF sem espaço em branco):**
  - `[hidden] { display: none !important; }` e controle explícito de exibição no `onload` do `iframe` eliminaram o skeleton residual no topo do preview.
  - Margem `@page` ajustada para `19mm 15mm 16mm 15mm` em `web/templates/contract_v1.html` e `pdfHeaderTemplate` refinado em `internal/service/pdf_service.go`.
- **CPF opcional + Validação Matemática (Módulo 11):**
  - Toggle `"Incluir CPF do representante legal"` adicionado em `web/index.html` com validação Módulo 11 em JS e Go (`internal/domain/contract.go`).
  - Quando desmarcado ou vazio, nenhuma menção a CPF ou `[PREENCHER CPF]` aparece na qualificação nem na assinatura (`web/templates/contract_v1.html`).
- **PWA Completo (`embed.FS`):**
  - Criados `web/manifest.webmanifest`, `web/sw.js`, `web/assets/icon-192.png` e `web/assets/icon-512.png`, servidos por `internal/handler/router.go`.
  - Responsividade mobile (`<= 900px`) com duas abas fixas (`1. Preencher Dados` e `2. Visualizar Contrato A4`) e barra inferior fixa de ação.
- **Bloco de Assinaturas e Testemunhas Opcionais (v1.3):**
  - **Alinhamento pelo topo (`BUG-003`):** `table.sig-grid` com `table-layout: fixed`, colunas `50%`, `padding: 36px 14px 12px` e `vertical-align: top`, garantindo linhas de assinatura na mesma altura horizontal e ordem padronizada (Razão Social, Representante, CNPJ e CPF opcional).
  - **Testemunhas opcionais (`include_witnesses`):** checkbox `"Incluir campos de testemunhas no contrato"` no formulário, coluna `include_witnesses` no SQLite (migration aditiva) e remoção completa de `TESTEMUNHA 1` / `TESTEMUNHA 2` do DOM + ajuste da cláusula de encerramento para assinatura eletrônica quando desmarcado.
- **Seção 3 — Condições Comerciais e Vigência (v1.2):**
  - **Mensalidade opcional (`has_monthly_fee`):** toggle `"Incluir mensalidade recorrente"`; quando desmarcado, desabilita o campo (`:disabled`), exibe microcopy explicativo e remove a mensalidade do Quadro Resumo e da Cláusula 14 (adicionando a subcláusula `14.1.1`).
  - **Parcelamento da Implantação (`setup_installments` 1x–12x):** cálculo automático do valor da parcela e exibição por extenso em tempo real no formulário, Quadro Resumo e Cláusula 14.
  - **Pagamento no ato da assinatura (`pay_on_signature`):** quando ativo, ajusta a forma de pagamento na Cláusula 14 (integral à vista ou 1ª parcela na assinatura + demais parcelas no dia de vencimento escolhido).
  - **Dados fixos da CONTRATADA:** `JOEL ACOSTA QUINTANA JUNIOR DESENVOLVIMENTO DE PROGRAMAS LTDA - EPP` (`IA INFINITO`), CNPJ `63.741.075/0001-43`, representante `Joel Acosta Quintana Junior` (Campo Grande/MS).
  - **Migration SQLite aditiva (`ensureCommercialColumns`):** colunas adicionadas com segurança sobre bancos existentes sem perda de dados.
- **Fluxo Integrado PDF + WhatsApp:**
  - Botão principal **"Exportar PDF e Enviar no WhatsApp"** com `navigator.canShare({ files })` em mobile/PWA, download automático `Contrato_{RazaoSocial}.pdf` e abertura de `https://wa.me/55{numero}?text=...`, além de botão secundário **"Apenas Baixar PDF"**.

### Validações executadas

- `go vet ./...`: aprovado sem alertas.
- `go test -v ./...`: 100% aprovado (incluindo CPF opcional, rotas PWA `/manifest.webmanifest`, `/sw.js`, `/assets/icon-*.png` e geração real de PDF via `chromedp`).
- `go build -o ia-infinito-contracts.exe ./cmd/server`: binário compilado com sucesso.

## Próxima ação

- Servidor local em execução em `http://localhost:8080` para teste manual do usuário (Desktop, Mobile/PWA, `/docs` Scalar e exportação PDF + WhatsApp).
