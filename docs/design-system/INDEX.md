# Índice de referências visuais — Gerador de Contratos IA Infinito

**Status:** 9 ativos visuais únicos conferidos e indexados para orientar a implementação do Design System (`SYSTEM_DESIGN.md`) e dos componentes skeleton (`web/components/`).

| ID | Painel | Arquivo | O que observar e aplicar no projeto |
|---|---|---|---|
| REF-001 | Background Stress Board | [reference-01.jpg](./reference-01.jpg) | O mesmo componente sobre fundos *calm*, *texture*, *hotspot*, *shape*, *complex* e *dark*. Garante que cabeçalhos e modais mantenham leitura estável em qualquer fundo. |
| REF-002 | State Stress Test | [reference-02.jpg](./reference-02.jpg) | Define os 5 estados obrigatórios sem depender de rótulos: *focus* (anel ciano), *pressed*, *disabled* (baixo contraste estrutural), *error* (borda/fundo vermelho + ícone) e *loading* (skeleton/spinner). |
| REF-003 | Works vs Risk | [reference-03.jpg](./reference-03.jpg) | Separa zonas adequadas para vidro (*Overlay* e *Nav*) das zonas de risco (*Data dense* e *Long copy*), onde o formulário de contrato e a folha A4 exigem placa estabilizada. |
| REF-004 | Glass Anatomy | [reference-04.png](./reference-04.png) | Define a hierarquia obrigatória de 3 camadas: `outer glass shell` -> `inner plate (scrim)` -> `content sits here`. |
| REF-005 | Edge Construction | [reference-05.png](./reference-05.png) | Especifica a borda equilibrada (*balanced*): `border edge` + `highlight band` superior + `inner stroke` + `shadow cue`, evitando o aspecto *flat* ou *overdone*. |
| REF-006 | Glass Depth Tiers | [reference-06.jpg](./reference-06.jpg) | Demonstra níveis de profundidade em sobreposição (`Glass-1`, `Glass-2`, `Glass-3`). Usar apenas como referência de material/profundidade, mantendo a paleta IA Infinito. |
| SUP-007 | Density Stress Board | [reference-07.jpg](./reference-07.jpg) | Compara tabela/formulário denso em estado de risco vs controlado (`rows must scan`, `microcopy survives`), base para a lista de contratos salvos no SQLite e o formulário lateral. |
| SUP-008 | Fallback Stress | [reference-08.jpg](./reference-08.jpg) | Valida que modais e painéis mantêm exatamente o mesmo layout e estados quando renderizados em superfície sólida de fallback (`prefers-reduced-transparency` ou ausência de `backdrop-filter`). |
| SUP-009 | Pass vs Fix | [reference-09.jpg](./reference-09.jpg) | Demonstra a falha de texto dependente do fundo (*fail*) contra a correção com placa estabilizada (*fix*). |

## Regras de Aplicação das Referências
1. **Material vs Paleta:** Utilizar das imagens apenas a engenharia de vidro estabilizado, construção de bordas, hierarquia tipográfica e estados de controle. **Não** importar tons cinza-azulados genéricos; aplicar exclusivamente os tokens da **IA Infinito** (`#0d1114`, `#22c58b`, `#1ea7e1`, `#e2c987`).
2. **Restrições Estritas:** Proibido excesso de cardificação (não encaixotar cada campo em cards), proibido excesso de tooltips (preferir microcopy inline) e proibido excesso de *pill buttons* (usar botões retangulares com `border-radius: 10px`).
