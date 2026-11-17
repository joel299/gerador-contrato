# Bugs e dificuldades — Gerador de Contratos IA Infinito

**Última atualização:** 17/11/2026  
**Status do registro:** 6 ocorrências técnicas registradas e resolvidas (`BUG-001` a `BUG-006`); riscos preventivos mantidos sob controle.

## Como usar este arquivo

Registrar aqui cada bug ou dificuldade técnica encontrada durante o desenvolvimento. Não registrar hipóteses como causas confirmadas. Não apagar problemas antigos após a correção; acrescentar a evidência, a correção aplicada e manter o histórico completo.

Sempre que um bug for registrado ou atualizado aqui, atualizar também `estado-atual.md` e `CONTRATO-PROGRESSO.html`.

---

## Ocorrências registradas

### BUG-006 — WebView do PWA abrindo landing page `wa.me` em vez do app WhatsApp nativo
- **Data:** 17/11/2026
- **Estado:** **Resolvido**
- **Sintoma:** No PWA instalado no celular, ao tentar enviar o contrato no WhatsApp, o sistema abria `https://wa.me/...` dentro de um navegador interno (Custom Tab/WebView) exibindo a tela de "Fazer download do WhatsApp" em vez de abrir o aplicativo instalado. Além disso, chamar `navigator.share({ files })` após `await fetch(...)` perdia o *user gesture*.
- **Evidência:** O esquema `https://wa.me/` dentro de PWA `standalone` permanece em Custom Tab sem repassar para o app nativo em vários dispositivos móveis; `navigator.share` exige clique síncrono direto após o PDF já estar em memória.
- **Causa raiz comprovada:** Uso de `https://wa.me/` no mobile/PWA em vez do protocolo nativo `whatsapp://send` e chamada de `navigator.share` após múltiplos `await` de rede.
- **Correção aplicada:**
  1. `web/index.html`: adicionada função `buildWhatsAppUrl(phoneDigits, text)` que retorna `whatsapp://send?phone=55...&text=...` no mobile/PWA (`window.location.href = waUrl`) e mantém `https://wa.me/...` no desktop.
  2. `web/index.html`: banner pós-geração exibe 3 botões estruturais diretos com *user gesture* preservado: **`Compartilhar Arquivo PDF`** (`navigator.share({ files })` direto no clique), **`Abrir Conversa no WhatsApp`** (`whatsapp://send` no mobile) e **`Baixar PDF`**.
  3. `web/sw.js`: cache incrementado para `ia-infinito-contratos-v3`.
- **Validação:** `go vet ./...`, `go test -v ./...` e `go build` aprovados.

### BUG-005 — Falha na geração/download de PDF em modo PWA mobile por expiração de user gesture e blob standalone
- **Data:** 17/11/2026
- **Estado:** **Resolvido**
- **Sintoma:** No desktop o PDF era gerado normalmente, porém no navegador móvel e no PWA instalado (`display: standalone`) o arquivo não era baixado nem compartilhado ao clicar em **Exportar PDF e Enviar no WhatsApp** ou **Apenas Baixar PDF**.
- **Evidência:**
  1. `navigator.share({ files })` era chamado após múltiplos `await` (`POST /api/contracts`, `POST /pdf`, `fetch` do blob), excedendo a janela de *user gesture* e causando `NotAllowedError` no Android/iOS.
  2. Em modo PWA `standalone`, `<a download>` com URL `blob:` é frequentemente ignorado no iOS Safari / WebViews Android.
  3. `GET /api/contracts/{id}/pdf` retornava 404 se o PDF ainda não existisse previamente e usava `inline` em vez de `attachment` com nome ASCII sanitizado.
- **Causa raiz comprovada:** Expiração de *user gesture* antes de `navigator.share`, dependência exclusiva de `blob:` download em PWA `standalone` e ausência de geração sob demanda + headers mobile-safe no `GET /api/contracts/{id}/pdf`.
- **Correção aplicada:**
  1. `internal/handler/contract_handler.go`: `GET /api/contracts/{id}/pdf` agora gera o PDF sob demanda se ainda não existir no storage e responde com `Content-Type: application/pdf`, `Content-Disposition: attachment; filename="Contrato_IA_Infinito_..."` (sanitizado ASCII), `Content-Length` e `Cache-Control: no-store`.
  2. `web/sw.js`: atualizado para `ia-infinito-contratos-v2` com bypass imediato para `/api/` e métodos diferentes de `GET`.
  3. `web/index.html`: fluxo `exportarPDF(enviarWhatsApp)` com estado de loading imediato, fallback automático de `navigator.share` (`NotAllowedError`) para download direto (`blob:` + `window.location.assign(directUrl)` em modo `standalone`) e banner de ação anti-bloqueio de pop-up com botões diretos **`Baixar PDF Novamente`** e **`Abrir WhatsApp`**.
- **Validação:** `go vet ./...`, `go test -v ./...` (`TestHTTP_EndpointsAndPDFGeneration` validando `GET /api/contracts/{id}/pdf` sob demanda, headers mobile-safe e `sw.js` bypass) e `go build` aprovados.

### BUG-004 — Registros fictícios no banco principal e termos técnicos expostos na interface
- **Data:** 17/11/2026
- **Estado:** **Resolvido**
- **Sintoma:** O banco `./data/contracts.db` continha registros de validações manuais via API e a UI exibia termos técnicos (`SQLite`, `BrasilAPI`, `API Docs` na barra superior) além de backdrop opaco no modal.
- **Evidência:** Testes automatizados já utilizavam `t.TempDir()`, mas testes manuais via `Invoke-RestMethod` gravaram no banco local da aplicação; `web/index.html` exibia jargões de infraestrutura e `rgba(5, 8, 11, 0.65)` no backdrop.
- **Causa raiz comprovada:** Falta de limpeza do diretório `./data` após testes manuais e textos de interface não alinhados à linguagem de produto.
- **Correção aplicada:** Limpeza total de `./data/contracts.db*` e `./data/pdfs/*`; remoção de todos os termos técnicos em `web/index.html` (`Contratos Salvos`, `Rascunho salvo com sucesso.`, link discreto `Documentação de Integração` no rodapé do formulário); backdrop sutil com `blur(4px)` e `rgba(5, 8, 11, 0.35)` preservando a anatomia de 3 camadas (`glass-shell` + `glass-plate`) e botões estruturais com `border-radius: 10px`.
- **Validação:** `go vet ./...`, `go test -v ./...`, `go build` e `GET /api/contracts` retornando `[]` no servidor limpo.

### BUG-003 — Desalinhamento vertical nas linhas de assinatura quando a Razão Social quebra em múltiplas linhas
- **Data:** 17/11/2026
- **Estado:** **Resolvido**
- **Sintoma:** Quando a Razão Social da CONTRATANTE ocupava 2+ linhas na grade de assinaturas (`table.sig-grid`), a linha horizontal de assinatura da esquerda ficava em altura diferente da coluna da direita (`CONTRATADA — IA INFINITO`).
- **Evidência:** Em `web/templates/contract_v1.html`, `table.sig-grid td` utilizava `vertical-align: bottom;`, alinhando o bloco pelo fundo da célula em vez do topo.
- **Causa raiz comprovada:** Uso de `vertical-align: bottom` sem `table-layout: fixed` e sem padronização de `padding-top: 36px` na célula da grade.
- **Correção aplicada:** Alterado `table.sig-grid` para `table-layout: fixed;` e `table.sig-grid td` para `width: 50%; padding: 36px 14px 12px; vertical-align: top;` com `border-top: 1px solid #0f172a` e ordem padronizada de linhas (Razão Social, Representante, CNPJ e CPF opcional).
- **Validação:** `go test -v ./...` (`TestRenderService_OfficialContractContentAndEscaping`), `go build` e inspeção do PDF gerado.

### BUG-001 — Espaço em branco gigante no topo do Live Preview A4 e do PDF exportado
- **Data:** 17/11/2026
- **Estado:** **Resolvido**
- **Sintoma:** Na tela de pré-visualização A4 e no PDF exportado, aparecia um bloco vazio/cinza acima do cabeçalho "IA INFINITO", empurrando o conteúdo do contrato para baixo.
- **Evidência:**
  1. Em `web/index.html`, a classe `.a4-skeleton` definia `display: grid;` após a regra padrão do navegador para `[hidden] { display: none; }`, mantendo o skeleton loader visível mesmo com `previewSkeleton.hidden = true`.
  2. Em `web/templates/contract_v1.html`, a margem superior `@page` de `26mm` excedia a altura necessária para o `HeaderTemplate` do `chromedp`, criando folga excessiva no topo da página 1 do PDF.
- **Causa raiz comprovada:** Conflito de especificidade CSS entre `.a4-skeleton { display: grid; }` e o atributo `[hidden]`, somado à margem superior `@page` superdimensionada no PDF.
- **Correção aplicada:**
  1. Adicionado `[hidden] { display: none !important; }` em `web/index.html` e alternância explícita de `style.display` no `onload` do `iframe`, garantindo que o skeleton desapareça completamente após o carregamento.
  2. Ajustada a margem `@page` em `web/templates/contract_v1.html` para `19mm 15mm 16mm 15mm` e refinado o espaçamento do `pdfHeaderTemplate` em `internal/service/pdf_service.go`, mantendo o documento iniciando exatamente no cabeçalho **IA INFINITO**.
- **Validação:** `go test -v ./...` (`TestHTTP_EndpointsAndPDFGeneration`), `go build` e inspeção visual do PDF gerado.

### BUG-002 — Incompatibilidade de assinatura da API `chromedp v0.23.1` / `cdproto v0.160.4`
- **Data:** 17/11/2026
- **Estado:** **Resolvido**
- **Sintoma:** Erro de compilação em `internal/service/pdf_service.go` durante `go build` / `go test`: `undefined: chromedp.ActionFunc`, `too many arguments in call to chromedp.Run`, e `page.GetFrameTree` / `page.SetDocumentContent` / `page.PrintToPDF` reportados como variáveis `cdp.Command` e não funções.
- **Evidência:** Saída do compilador Go 1.26 / `go doc github.com/chromedp/chromedp.Action` confirmou que a versão `v0.23.1` do `chromedp` utiliza `chromedp.Steps(...)`, `chromedp.Func(func(ctx, *chromedp.Target) error)` e `cdp.Call(ctx, t, page.PrintToPDF, page.PrintToPDFParams{...})`.
- **Causa raiz comprovada:** Mudança de API na versão `v0.23.1` do pacote `github.com/chromedp/chromedp`.
- **Correção aplicada:** Atualizado `internal/service/pdf_service.go` para usar `chromedp.Steps`, `chromedp.Func` e `cdp.Call`.
- **Validação:** `go vet ./...`, `go test -v ./...` e `go build` passaram com código 0.

---

## Riscos e pontos de atenção conhecidos — não são bugs

### RISCO-001 — Consumo de memória e inicialização do Chromium Headless (`chromedp`)
- **Data:** 17/11/2026
- **Estado:** mitigado na implementação (`internal/service/pdf_service.go`).
- **Detalhes:** `ExecAllocator` persistente no boot + semáforo de pool limitado (`PDF_POOL_SIZE=2`) e flags `--no-sandbox`, `--disable-dev-shm-usage`, `--disable-gpu`.

### RISCO-002 — Indisponibilidade ou rate limit da BrasilAPI na consulta de CNPJ
- **Data:** 17/11/2026
- **Estado:** mitigado na implementação (`internal/service/cnpj_service.go`).
- **Detalhes:** Timeout de 5s no `http.Client`, cache em memória (24h), fallback automático em tabela SQLite `cnpj_cache` e edição manual sempre liberada no formulário.

### RISCO-003 — Fidelidade visual entre o Live Preview e o PDF exportado
- **Data:** 17/11/2026
- **Estado:** mitigado na implementação (`web/templates/contract_v1.html` + `pdf_service.go`).
- **Detalhes:** Mesmo template `contract_v1.html` usado no preview e no `chromedp`, com `@page { size: A4; margin: 19mm 15mm 16mm 15mm; }` e `HeaderTemplate` / `FooterTemplate` exibindo a logo oficial da IA Infinito e numeração em todas as páginas do PDF.
