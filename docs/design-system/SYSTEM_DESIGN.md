# System Design — Gerador de Contratos IA Infinito

**Versão:** 1.0  
**Data:** 17/11/2026  
**Referências visuais:** `REF-001` a `REF-006` e `SUP-007` a `SUP-009` (indexadas em `INDEX.md`)  
**Paleta oficial:** Preto profundo (`#0d1114`), Verde Esmeralda (`#22c58b` / `#147a55`), Ciano (`#1ea7e1` / `#0b7eb0`) e Dourado discreto (`#e2c987` / `#8c6a24`), com suporte nativo aos temas **Escuro (padrão)** e **Claro**.

---

## 1. Princípios Arquiteturais da Interface

Este documento traduz as referências visuais (`reference-01` a `reference-09`) para a engenharia de frontend do **Gerador de Contratos IA Infinito**, estabelecendo regras estritas de composição, materiais, estados e restrições de UI.

### Regras Proibitivas (Bloqueio de Anti-Padrões)
1. **Proibição terminante de excesso de cardificação:** Não envolver cada campo, grupo de formulário ou item de lista em cards independentes com sombras e bordas próprias. O formulário lateral usa agrupamento plano separado por linhas divisórias sutis (`1px solid var(--border-subtle)`) e espaçamento vertical consistente. Apenas **superfícies estruturais primárias** (shell do painel, modais/drawers, barra de navegação flutuante e folha A4 de preview) recebem elevação ou camada de vidro.
2. **Proibição de excesso de tooltips:** Não adicionar ícones de interrogação (`?`) ou balões flutuantes para explicar campos óbvios. Instruções e regras de negócio (como formato de CNPJ, cálculo por extenso ou cláusula de tráfego pago) devem aparecer como microcopy estático direto abaixo do input (`12px`, `var(--text-muted)`), conforme `SUP-007` e `REF-003`.
3. **Proibição de pill buttons em excesso:** Botões de ação (primários, secundários, ghost e controles de formulário) devem usar raio de borda estrutural (`--radius-control: 10px` ou `12px`), alinhado às referências `REF-001`, `REF-002` e `SUP-008`. O formato pílula (`9999px`) é restrito exclusivamente a indicadores pontuais de status ou tags compactas de filtro ativo, nunca em botões de ação ou submissão.
4. **Minimalismo funcional:** Nenhum brilho neon difuso, gradiente decorativo animado, borda arco-íris ou micro-animação supérflua. Cada elemento visual deve comunicar hierarquia, estado de interação ou legibilidade.

---

## 2. Anatomia do Material (Liquid Glass Estabilizado)

Baseado diretamente em `REF-004` (*Glass Anatomy*), `REF-005` (*Edge Construction*), `REF-006` (*Glass Depth Tiers*) e `SUP-009` (*Pass vs Fix*).

### 2.1 Ordem Obrigatória de Camadas (`REF-004` e `SUP-009`)
Nunca renderizar texto ou inputs diretamente sobre uma camada translúcida sem estabilização (`SUP-009` — *background-dependent fail*). Toda superfície de vidro segue exatamente 3 camadas empilhadas:

1. **Camada 1 — Outer Glass Shell (Casca Externa):**
   - Contém o `backdrop-filter: blur(16px) saturate(130%)`, a borda física (`border-edge`), o brilho especular superior (`highlight-band`) e a sombra externa (`shadow-cue`).
   - Padding interno de `6px` a `8px` atuando como moldura estrutural.
2. **Camada 2 — Inner Stabilized Plate / Scrim (Placa Interna Estabilizadora):**
   - Superfície interna de alta opacidade (`88%` a `94%` no tema escuro; `90%` a `96%` no tema claro) com raio interno `calc(var(--radius-shell) - 6px)`.
   - Isola completamente o conteúdo contra manchas de luz, formas geométricas ou contrastes abruptos no fundo (`REF-001` — *hotspot*, *shape*, *complex*, *dark*).
3. **Camada 3 — Content Layer (Camada de Conteúdo):**
   - Tipografia, inputs, tabelas e controles apoiados sobre a placa estabilizada, garantindo contraste mínimo WCAG AA (`>= 4.5:1` para texto normal e `>= 3:1` para bordas de controle).

### 2.2 Construção de Borda Equilibrada (`REF-005` — *Balanced vs Flat/Overdone*)
O acabamento da borda deve evitar o aspecto lavado (*flat*) e o reflexo metálico exagerado (*overdone*):

- **Border Edge:** `1px solid var(--glass-border-outer)`
- **Highlight Band (luz superior):** `inset 0 1px 0 0 var(--glass-highlight-top)`
- **Inner Stroke (contorno interno sutil):** `inset 0 0 0 1px var(--glass-inner-stroke)`
- **Shadow Cue (profundidade controlada):** `0 16px 40px -8px rgba(0, 0, 0, 0.36)` (dark) / `0 14px 34px -8px rgba(15, 23, 30, 0.10)` (light).

### 2.3 Zonas Adequadas vs Zonas de Risco (`REF-003` e `SUP-007`)
- **Zonas de Vidro (`Works`):** Barra superior de navegação (`Nav`), barra flutuante de ações do PDF, modais de confirmação e gaveta (Drawer) de histórico de contratos.
- **Zonas Estabilizadas de Alta Densidade (`Risk` -> `Controlled`):** O formulário cadastral/comercial, listas/tabelas de contratos salvos (`SUP-007`) e o documento A4 em si devem operar sobre placas estabilizadas sólidas ou quase opacas, preservando leitura rápida linha a linha (`rows must scan`) e microcopy legível (`microcopy survives`).

### 2.4 Fallback Obrigatório (`SUP-008` — *Fallback Stress Test*)
Quando o navegador não suportar `backdrop-filter` ou o usuário ativar `prefers-reduced-transparency: reduce`:
- O layout, espaçamento, bordas e estados permanecem **100% idênticos**.
- A casca externa e a placa interna assumem cores sólidas equivalentes (`var(--surface-solid-shell)` e `var(--surface-solid-plate)`), sem transparência e sem perda de hierarquia.

---

## 3. Tokens de Design (Paleta IA Infinito)

Não utilizar tons cinza-azulados genéricos das imagens de referência. Aplicar exclusivamente os tokens semânticos abaixo:

```css
:root {
  color-scheme: dark;

  /* Fundos estruturais */
  --bg-canvas: #0d1114;
  --bg-subtle: #13191e;

  /* Camadas Liquid Glass (Dark) */
  --glass-shell-bg: rgba(20, 27, 33, 0.62);
  --glass-plate-bg: rgba(22, 29, 36, 0.92);
  --surface-solid-shell: #141b21;
  --surface-solid-plate: #182027;
  --surface-input: #10161b;
  --surface-input-disabled: #141a20;

  /* Bordas e luz especular (REF-005) */
  --glass-border-outer: rgba(255, 255, 255, 0.11);
  --glass-highlight-top: rgba(255, 255, 255, 0.16);
  --glass-inner-stroke: rgba(255, 255, 255, 0.05);
  --border-subtle: #26313b;
  --border-control: #32404d;

  /* Identidade IA Infinito */
  --accent-primary: #22c58b;        /* Verde Esmeralda */
  --accent-primary-hover: #1eb07c;
  --accent-primary-fg: #071912;
  --accent-secondary: #1ea7e1;      /* Ciano Técnico */
  --accent-gold: #e2c987;           /* Dourado Institucional */

  /* Tipografia */
  --text-primary: #f0f4f8;
  --text-secondary: #c2cdd6;
  --text-muted: #8b99a6;
  --text-disabled: #586673;

  /* Estados Semânticos (REF-002) */
  --state-focus-ring: rgba(30, 167, 225, 0.35);
  --state-focus-border: #1ea7e1;
  --state-error-bg: rgba(239, 68, 68, 0.10);
  --state-error-border: #ef4444;
  --state-error-text: #fca5a5;

  /* Geometria e Elevação */
  --radius-shell: 20px;
  --radius-plate: 14px;
  --radius-control: 10px;
  --glass-blur: 16px;
  --shadow-shell: 0 18px 44px -10px rgba(0, 0, 0, 0.45);
}

html[data-theme="light"] {
  color-scheme: light;

  --bg-canvas: #f2f5f7;
  --bg-subtle: #e7edf1;

  /* Camadas Liquid Glass (Light) */
  --glass-shell-bg: rgba(255, 255, 255, 0.66);
  --glass-plate-bg: rgba(255, 255, 255, 0.93);
  --surface-solid-shell: #eef2f5;
  --surface-solid-plate: #ffffff;
  --surface-input: #f8fafb;
  --surface-input-disabled: #edf1f4;

  --glass-border-outer: rgba(15, 23, 30, 0.10);
  --glass-highlight-top: rgba(255, 255, 255, 0.85);
  --glass-inner-stroke: rgba(15, 23, 30, 0.04);
  --border-subtle: #dce4eb;
  --border-control: #c5d1dc;

  --accent-primary: #147a55;
  --accent-primary-hover: #106345;
  --accent-primary-fg: #ffffff;
  --accent-secondary: #0b7eb0;
  --accent-gold: #8c6a24;

  --text-primary: #131d24;
  --text-secondary: #364550;
  --text-muted: #5d6d7a;
  --text-disabled: #96a3ae;

  --state-focus-ring: rgba(11, 126, 176, 0.25);
  --state-focus-border: #0b7eb0;
  --state-error-bg: rgba(220, 38, 38, 0.07);
  --state-error-border: #dc2626;
  --state-error-text: #b91c1c;

  --shadow-shell: 0 16px 38px -10px rgba(19, 29, 36, 0.12);
}
```

---

## 4. Especificação de Estados de Interação (`REF-002`)

Conforme `REF-002` (*states must be obvious without labels*), todo controle interativo deve distinguir seus estados por **estrutura e contraste**, nunca apenas por mudança sutil de opacidade:

1. **Default:** Borda `1px solid var(--border-control)`, fundo `var(--surface-input)`, rótulo nítido em `var(--text-primary)`.
2. **Focus (`:focus-visible`):** Borda `1px solid var(--state-focus-border)` + anel externo `box-shadow: 0 0 0 3px var(--state-focus-ring)`.
3. **Pressed (`:active`):** Deslocamento vertical sutil (`transform: translateY(1px)`) e escurecimento interno controlado sem alterar layout vizinho.
4. **Disabled (`:disabled`):** Fundo `var(--surface-input-disabled)`, texto em `var(--text-disabled)`, borda tracejada ou de baixo contraste, `cursor: not-allowed` (nunca apenas `opacity: 0.5` sobre texto brilhante).
5. **Error (`[aria-invalid="true"]`):** Borda `1px solid var(--state-error-border)`, fundo `var(--state-error-bg)`, ícone de alerta alinhado à direita dentro do campo e mensagem objetiva abaixo em `var(--state-error-text)`.
6. **Loading (`[aria-busy="true"]`):** Substituição do conteúdo dinâmico por **Skeleton Loader** com dimensões reservadas (evitando *Layout Shift* / CLS) ou spinner integrado ao botão de exportação de PDF com texto de progresso real.

---

## 5. Layout Split-Screen e Contrato A4

### 5.1 Estrutura da Tela Principal (`/`)
- **Top Bar (`height: 60px`):** Superfície `glass-shell` fixa no topo com:
  - Esquerda: Logo vetorial **IA Infinito** + título `Gerador de Contratos`.
  - Direita: Botão `Contratos Salvos` (abre gaveta/modal de busca SQLite), link `API Docs` (`/docs` Scalar) e alternador de tema (`Escuro`/`Claro`).
- **Workspace Split-Screen (`calc(100vh - 60px)`):**
  - **Coluna Esquerda (`440px` fixo em desktop, `100%` em mobile — Painel de Dados):**
    - Placa estabilizada única (`surface-solid-plate`) com scroll interno próprio.
    - Seções separadas apenas por divisores horizontais (`1px solid var(--border-subtle)`) e subtítulos (`11px`, uppercase, `letter-spacing: 0.08em`, `var(--accent-secondary)`):
      1. *Identificação da Contratante* (CNPJ com botão integrado `Buscar CNPJ`, Razão Social, Endereço completo).
      2. *Responsável Legal e Contato* (Nome completo, CPF, Telefone/WhatsApp, E-mail).
      3. *Condições Comerciais* (Valor Setup, Valor Mensalidade, Dia de Vencimento, Data de Assinatura).
    - Rodapé fixo da coluna esquerda com dois botões estruturais (`border-radius: 10px`): **Salvar Rascunho** (secundário) e **Gerar e Exportar PDF** (primário `var(--accent-primary)`).
  - **Coluna Direita (`1fr` — Palco de Pré-visualização A4):**
    - Fundo neutro (`--bg-canvas`) com centralização da folha A4 (`width: 210mm`, `min-height: 297mm`, fundo branco `#ffffff`, texto `#111827`).
    - Renderiza o template oficial do contrato da **IA Infinito** atualizado em tempo real conforme os inputs da esquerda.
    - Mantém fielmente a logo oficial da **IA Infinito** no cabeçalho de todas as páginas, a cláusula de link personalizado (**WhatsApp, Facebook, Instagram, Google e TikTok**) e a cláusula de **Tráfego Pago com investimento e risco integralmente assumidos pela CONTRATADA**.

### 5.2 Drawer/Modal de Histórico e Busca (SQLite)
- Segue o padrão `SUP-007` (*controlled density*) e `SUP-008` (*glass + fallback*):
  - Campo de busca instantânea por Razão Social ou CNPJ.
  - Tabela limpa com colunas: `Empresa`, `CNPJ`, `Mensalidade`, `Atualizado em` e ações `Carregar` / `Baixar PDF`.
  - Durante o carregamento da consulta (`GET /api/contracts`), exibe exatamente 5 linhas de **Skeleton Loader** com a mesma altura das linhas reais (`48px`).
