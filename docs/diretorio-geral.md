# Diretório geral — Gerador de Contratos IA Infinito

**Versão:** 1.0  
**Data:** 17/11/2026  
**Status:** Regras iniciais do projeto (pré-implementação)

## Objetivo

O **Gerador de Contratos IA Infinito** é um sistema web leve e performático escrito em **Go (Golang)** que permite preencher dados cadastrais e comerciais de clientes em uma interface dividida (*Split-Screen*), visualizar o contrato em tempo real mantendo o padrão visual oficial da **IA Infinito**, consultar dados automaticamente via CNPJ (BrasilAPI), armazenar e localizar contratos gerados em **SQLite** (com backup opcional de arquivos em **S3**), testar endpoints via **Scalar** e exportar o PDF final pronto para assinatura digital.

Este arquivo define as regras de trabalho para todos os agentes e colaboradores. Ele é a fonte principal para o processo do projeto.

## Estrutura de documentação

- `diretorio-geral.md`: regras do projeto, fluxo de trabalho, responsabilidades, padrão do painel de progresso e decisões de processo.
- `estado-atual.md`: fatos confirmados, etapa atual, trabalho concluído, bloqueios e próximo passo.
- `bugs-e-dificuldades.md`: problemas, erros, causas confirmadas, correções e evidências.
- `SDD.md`: arquitetura técnica, estrutura de pastas (*skeleton*), schema SQLite, armazenamento S3, pool de geração de PDF e especificação da API.
- `SYSTEM_DESIGN.md`: especificação técnica de interface (Liquid Glass estabilizado, tokens IA Infinito, estados e restrições contra excesso de cardificação, tooltips e pill buttons).
- `INDEX.md`: índice das referências visuais e regras de aplicação.
- `COMPONENTS-SKELETON.html`: biblioteca de referência com componentes minimalistas, estados e skeletons prontos.
- `CONTRATO-PROGRESSO.html`: painel visual interativo com etapas, decisões, progresso estimado, prazo e grafo de dependências.

Manter os arquivos na pasta principal de documentação do projeto. Se a pasta ou os nomes mudarem, atualizar este índice.

## Leitura obrigatória no início do trabalho

Todo agente deve seguir esta ordem antes de propor ou executar mudanças:

1. Ler `diretorio-geral.md`.
2. Ler `estado-atual.md` e retomar da próxima ação registrada.
3. Ler `bugs-e-dificuldades.md` para conhecer problemas abertos e correções anteriores.
4. Consultar `SDD.md`, `SYSTEM_DESIGN.md`, `INDEX.md` e `COMPONENTS-SKELETON.html` antes de criar arquivos, rotas, tabelas ou interfaces.
5. Inspecionar os arquivos e configurações envolvidos antes de assumir que algo já existe.

### Regra para subagentes

Todo subagente deve receber e ler este `diretorio-geral.md` antes de trabalhar, além de `estado-atual.md`, `bugs-e-dificuldades.md` e `SDD.md`. O agente que delega deve incluir essas leituras nas instruções da tarefa e confirmar que o subagente teve acesso.

Se um subagente não tiver acesso aos documentos, ele deve registrar o bloqueio e não deve presumir regras ou decisões ausentes.

## Planejamento em tarefas pequenas

Antes de desenvolver, dividir o trabalho em tarefas pequenas, verificáveis e com um resultado claro. Não iniciar uma mudança ampla sem identificar suas partes.

Cada plano deve informar:

- objetivo da tarefa;
- microtarefas em ordem;
- dependências entre microtarefas;
- responsável por cada tarefa, quando houver mais de um agente;
- arquivos ou áreas que podem mudar;
- resultado esperado;
- critério de conclusão;
- validação necessária;
- riscos, dúvidas e bloqueios conhecidos.

As microtarefas devem ser pequenas o bastante para revisão independente. Evitar atribuir o mesmo arquivo a dois agentes ao mesmo tempo. Não delegar trabalho se a tarefa puder ser concluída com segurança e clareza por um único agente.

## Registro das decisões

Cada agente deve registrar as decisões que tomou durante a tarefa. Para cada decisão, documentar:

- decisão tomada;
- motivo da decisão;
- opções relevantes consideradas;
- evidência observada;
- efeito esperado;
- riscos ou limites;
- validação que confirma o resultado.

Separar fatos confirmados, propostas, hipóteses e itens não verificados. Não apresentar hipótese como causa confirmada. Se a evidência não permitir decidir, registrar a dúvida e o próximo passo.

## Atualização dos documentos

### `estado-atual.md`

Atualizar após cada marco significativo ou correção. Registrar a etapa atual, o que foi concluído, arquivos alterados, decisões, evidências, validações, bloqueios e próxima ação exata. Preservar o histórico e permitir que outro agente continue sem depender da conversa.

### `bugs-e-dificuldades.md`

Registrar todo bug novo ou alteração importante de um bug existente. Informar sintoma, impacto, evidência, causa somente quando comprovada, decisão, correção, validação, estado e próximo passo. Fechar um bug somente com evidência da correção.

Dúvidas de produto, tarefas pendentes e riscos não são bugs. Registrá-los como pendências ou riscos, com nome claro.

### `CONTRATO-PROGRESSO.html` (Obrigatório a cada alteração/correção)

**Regra crítica:** O agente **deve atualizar `CONTRATO-PROGRESSO.html` todas as vezes que fizer qualquer implementação, ajuste ou correção** no projeto.

O arquivo `CONTRATO-PROGRESSO.html` deve seguir rigorosamente o mesmo padrão visual e estrutural estabelecido:

- **Documento autocontido (Zero dependências externas):** HTML5 + CSS3 + Vanilla JS embutidos no próprio arquivo, sem CDNs ou frameworks externos.
- **Identidade Visual IA Infinito:** Paleta dark/light alternável (`data-theme="dark"` padrão e `data-theme="light"` persistido em `localStorage`), superfícies translúcidas com glassmorfismo (`backdrop-filter: blur(16px)`) e fallback sólido (`@supports not` e `prefers-reduced-transparency`), respeitando `prefers-reduced-motion`.
- **Estrutura obrigatória das seções:**
  1. **Cabeçalho (`<header>`):** Marca **IA Infinito — Gerador de Contratos** + botão funcional de alternância entre tema claro e escuro.
  2. **Hero (`<section class="hero">`):** Resumo da etapa atual à esquerda + cartão de **Progresso verificado** à direita com percentual real (nunca inventado), barra de progresso (`role="progressbar"`) e estimativa restante.
  3. **Grafo interativo de etapas (`<section class="section">`):** SVG interativo com os nós numerados das etapas (`1` a `5`), estados visuais (`done`, `current`, pendente) e painel de detalhes acessível via clique e teclado (`Enter`/`Space`).
  4. **Colunas de Decisões e Status por Etapa (`<section class="section columns">`):** Lista de decisões técnicas confirmadas (Go, SQLite WAL, S3, Scalar, `chromedp`) e lista vertical de status por fase (`✓` concluída, `●` atual).
  5. **Histórico de Atualizações e Correções:** Blocos cronológicos adicionados a cada entrega/correção descrevendo o que mudou, por que mudou e quais validações (`go test`, `go vet`, `go build`, HTTP) foram executadas.
  6. **Rodapé (`<footer>`):** Data da última atualização, resumo da entrega e referência a `estado-atual.md` e `bugs-e-dificuldades.md`.

Não inventar percentual de progresso com base em atividade não verificada.

## Formato das explicações

- Usar frases curtas, voz ativa e palavras conhecidas.
- Escrever em português claro, seguindo os princípios de linguagem técnica simples do ASD-STE100.
- Explicar o que mudou, por que mudou e como o resultado foi verificado.
- Evitar siglas sem explicação e jargão quando uma palavra simples servir.
- Marcar claramente o que é proposta e o que já funciona no sistema.

## Regras técnicas e de produto

- **Stack obrigatória:** Backend 100% em **Go (Golang 1.22+)** com `embed.FS` para empacotar templates, assets e documentação em um único binário.
- **Padrão do Contrato:** Manter fielmente a identidade visual oficial da **IA Infinito** (logo oficial em todas as páginas, cabeçalho/rodapé padronizados, cláusulas de link personalizado para WhatsApp, Facebook, Instagram, Google e TikTok, e cláusula de tráfego pago com investimento e risco integralmente assumidos pela CONTRATADA).
- **Persistência e Busca:** Salvar todos os contratos gerados em **SQLite** (modo WAL) para consulta, reedição e auditoria rápida; suportar armazenamento de PDFs em **S3** (AWS S3, Cloudflare R2 ou MinIO) com fallback local seguro.
- **Performance e UX:** Interface Split-Screen responsiva com **Skeleton loaders** durante carregamento de listas, consulta de CNPJ e geração de PDF; sem travar a interface.
- **Documentação de API:** Expor especificação OpenAPI 3.1 interativa via **Scalar** (`/docs`).
- **Segurança:** Sanitizar entradas HTML/texto nos templates (`html/template`), validar CNPJ/CPF e valores monetários no servidor e nunca expor chaves S3 ou segredos no frontend.

## Uso de subagentes

Se a tarefa precisar de mais de uma especialidade, o agente responsável pode dividir o trabalho em subtarefas independentes com critérios de aceite, dependências e limite de arquivos claros. O agente responsável deve revisar todas as entregas e atualizar `estado-atual.md`, `bugs-e-dificuldades.md` e `CONTRATO-PROGRESSO.html` após integrar o trabalho.

## Encerramento de cada etapa

Antes de concluir qualquer etapa ou correção:

- executar validações técnicas aplicáveis (`go vet ./...`, `go test ./...`, `go build`);
- atualizar `estado-atual.md`;
- atualizar `bugs-e-dificuldades.md` (se houve bug, bloqueio ou ajuste técnico);
- atualizar `CONTRATO-PROGRESSO.html` refletindo o novo estado;
- apontar o próximo passo exato sem afirmar deploy ou teste que não tenha sido executado.

## Escopo atual do MVP

Inclui: servidor HTTP em Go, interface Split-Screen (formulário lateral + live preview A4), consulta automática de CNPJ via BrasilAPI, conversão automática de valores para extenso em BRL, geração de PDF fiel via `chromedp` / `html/template`, persistência e busca de contratos no SQLite, upload opcional para S3 e documentação interativa via Scalar (`/docs`).

Não inclui: assinatura eletrônica nativa com certificado ICP-Brasil dentro do sistema (o PDF é exportado pronto para plataformas externas como Gov.br, ZapSign, Autentique ou Clicksign) nem gestão multiempresa com controle de permissões complexo nesta primeira versão.
