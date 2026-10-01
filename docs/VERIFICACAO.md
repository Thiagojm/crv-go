# Verificação do protótipo

## Resultado

- `npm run check`: zero erros e zero avisos.
- `npm run build`: concluído; HTML único com JS, CSS, fonte e imagens embutidos.
- Fluxo real verificado por Playwright com Chromium headless, abrindo o HTML por `file://`.
- Browser integrado não estava disponível; o download padrão de Chromium falhou. Foi usado Chromium distribuído via pacote npm, com renderização por software, apenas para QA.
- Viewports: 1536 × 1024 (mesma dimensão do conceito) e 390 × 844; revisão adicional em 1280 × 720.
- Sem erros JavaScript de página durante o fluxo completo.

## Interações verificadas

Início → preparação → ideograma com mouse → checkbox e escrita livre → ajuda e exemplo → sensorial → esboço com mouse → revisão → confirmação de bloqueio → quatro alternativas → recarregamento e recuperação da mesma sessão → escolha → confirmação → feedback e comentário → histórico → estatísticas → configurações/tema → nova sessão em janela estreita → salvar e encerrar.

Asserções verificaram preservação de traços, atributos, ordem das imagens e alvo, imutabilidade do registro no fluxo pós-bloqueio, estado concluído, ausência de overflow horizontal da página em janela estreita e abertura da tela de encerramento. Isso não valida um backend, pois ele ainda não existe.

## Revisão visual

Conceito `docs/conceito.png` e capturas finais foram inspecionados como imagens. Comparação em tamanho nativo:

| Ponto | Conceito / requisito | Resultado e ajuste |
| --- | --- | --- |
| Layout | Canvas à esquerda e atributos à direita, aproximadamente 60/40 | Preservado; canvas com proporção estável, painel longo com rolagem da página |
| Tipografia | Sans legível e hierarquia clara | Fonte Inter embutida após detectar substituição por fonte inadequada no ambiente de QA |
| Paleta | Fundo #f5f6f3, papel branco, verde #254e43 | Preservada; tema escuro mantém papel branco |
| Controles | Ferramentas compactas, ajuda e rodapé de navegação | Preservados; adicionado nome acessível ao encerramento quando o texto fica oculto em telas estreitas |
| Conteúdo | Ideograma, descrição breve, atributos e texto livre | Lista completa da especificação; termos abreviados da imagem substituídos pelos aprovados |
| Imagens | Quatro alternativas padronizadas sem recorte destrutivo | Grade 2 × 2, proporção preservada com object-fit: contain, ampliação disponível |
| Responsividade | Trabalho desktop, adaptação a janela estreita | Painéis empilhados sem overflow horizontal da página; stepper possui rolagem própria |

### Conferência de texto acima da dobra

Título Ideograma, descrição, ajuda, navegação das sete etapas e ferramentas conferidos. Diferenças intencionais: “Protótipo de interface” em lugar de “Sessão de demonstração”; “Salvar e encerrar” conforme especificação; cronômetro/código reais de demonstração; opções completas e textarea por grupo; controles de tema, pausa e espessura. Nenhum conteúdo do alvo aparece na coleta visual.

O resultado mantém a direção do conceito com essas adaptações funcionais documentadas. O conceito foi gerado nesta etapa; a aprovação visual do usuário ainda será obtida pelo uso do protótipo. Não se reivindica identidade pixel a pixel nem teste nativo em Windows.

## Limites

Verificação realizada em Chromium no ambiente Linux de execução. Compatibilidade com Windows, Firefox e diferentes políticas de armazenamento de arquivos locais ainda requer teste no dispositivo do usuário. A seção seguinte registra a Fase 1 do aplicativo Go; sessões cegas, PDF e backup ZIP ainda não existem.

## Fase 1 — servidor local e catálogo (2026-09-28, Windows)

Ambiente: Windows, Go 1.26.5, Node 24.18.0. Comandos: `npm.cmd run check`, `npm.cmd run build`, `go vet ./...`, `go test ./...`, `go build -o bin/crv.exe .`.

- Catálogo bundled decodificado: **192 elegíveis**, **4 exclusões** (imagens sem fonte única); após remoção de duplicatas/semelhantes; relatório preserva summary **multi=2 / empty=1**, provenance e `source_url`.
- Correções pós-revisão: ativação só após arquivos verificados; retomada sem pasta `farsight/` se a revisão estiver íntegra; rejeição de formato/hashes duplicados/links em ancestrais; `/api/*` inexistente → 404.
- Testes HTTP: Host/Origin rejeitados, bootstrap/CSRF obrigatórios, rotas `/farsight/` e `/images/` em 404, `sessionsOpen=false`.
- Trava de instância: segunda execução no mesmo `--data-dir` falha e reporta a URL existente.
- Reinício reutiliza a revisão ativa sem duplicar nem alterar `farsight/`.
- Banco ausente: `ready=false` com mensagem de reparo.
- Navegador (Playwright nesta sessão): banner de prontidão, Nova sessão desabilitada, tema, Configurações com contagens, Conhecer o fluxo, Histórico vazio honesto, Salvar e encerrar encerra o processo.

Não verificado nesta fase: empacotamento offline completo, Linux runtime, abertura automática do navegador quando `rundll32` falha (apenas log da URL). Sessões reais: ver Fase 2 abaixo.

## Fase 2 — sessão cega persistida (2026-09-28, Windows)

Ambiente: Windows, Go 1.26.5, Node 24.18.0. Comandos: `npm.cmd run check`, `npm.cmd run build`, `go vet ./...`, `go test ./...`, `npm.cmd run test:e2e`.

- Criação sorteia alvo + 3 distratores distintos com `crypto/rand` antes de devolver o código; retries com o mesmo `operationId` reutilizam a sessão; um banco de 4 imagens permite sessões sucessivas.
- DTOs de coleta/escolha não incluem hash, caminho nem créditos; `/api/sessions/{id}/images/{A-D}` responde 404 antes do bloqueio e após abandono anterior às alternativas.
- Lock, escolha tentada, confirmação (idempotente / conflito se outra letra), abandono e comentário pós-feedback são transições no servidor com revisão. Tempo não incrementa a revisão. Lease 15 s / heartbeat 5 s; retomada após recarga fica pausada.
- Playwright (Chromium, 1280×720): desenho, checkbox, texto, ajuda, bloqueio, escolha, feedback; recarga pausa e preserva “Sólido”; segunda aba mostra Assumir; abandono e Salvar e encerrar.

## Fase 3 — histórico, estatísticas e catálogo (2026-09-28, Windows)

Ambiente: Windows, Go 1.26.5, Node 24.18.0. Comandos: `npm.cmd run check`, `npm.cmd run build`, `go vet ./...`, `go test ./...`, `npm.cmd run test:e2e` (5 testes), `go build -o bin/crv.exe .`.

- `GET /api/history` e `GET /api/statistics`: lista newest-first com filtros de estado/data; totais cumulativos independentes dos filtros; `hitRate` ausente sem escolhas confirmadas; gráfico com referência 0,25·n.
- Exemplo A9 no store: 1 acerto + 1 erro + 1 abandono → 2 escolhas, 1 acerto, 50%.
- Import ZIP/pasta e reparo: 409 com sessão ativa; ZIP corrompido/travessia rejeitados; revisão antiga permanece no disco.
- UI: Histórico (lista + detalhe com Record/feedback), Estatísticas (faixa + SVG + tabela), Configurações (Importar ZIP, pasta absoluta, Reparar). Assets demo removidos do frontend; `dist` sem banco.
- Browser MCP: fluxo vazio → 1 sessão concluída na lista/detalhe; estatísticas 0/1; import visível; viewport 390×844.

## Fase 4 — exportações e backup (2026-09-28, Windows)

Ambiente: Windows, Go 1.26.5, Node 24.18.0. Comandos: `npm.cmd run check`, `npm.cmd run build`, `go vet ./...`, `go test ./...`, `npm.cmd run test:e2e`, `go build -o bin/crv.exe .`.

- CSV (`GET /api/exports/csv`): colunas permitidas; strings tipo fórmula neutralizadas; sem SHA/caminhos/créditos.
- PDF: vista HTML imprimível (`GET /api/exports/sessions/{id}/print`) com desenhos SVG e orientação “Salvar como PDF”; alvo via `template.URL` (data URI); Esboço em página própria com título+desenho; e2e exige `naturalWidth>0`, imagem embutida no PDF e headings com vetores (pymupdf). Diálogo nativo do navegador fica para validação manual.
- Backup ZIP `crv-backup-v1` com snapshot `VACUUM INTO`, revisões referenciadas, preferências e manifesto SHA-256; restauração com cópia prévia, marcador durável e `ResolveInterrupted` no startup.
- Endurecimento: marcador `live_moved` antes de mover o payload; `dataMu` RWMutex drena handlers em voo antes do restore; undo falho preserva `OldDir`; ValidateArchive exige revisões ativas/referenciadas pelo DB estagiado; Create/Validate exigem arquivos de imagem; download de backup faz `queue.flush()`.
- UI: Exportar CSV / Exportar PDF no Histórico; Baixar/Restaurar backup nas Configurações.

Não verificado nesta fase: empacotamento Linux/Windows (Fase 5), diálogo nativo Salvar como PDF pelo usuário, `go test -race` (CGO_ENABLED=0 neste host), validação manual completa do restore na instalação real.

## Phase 5 — local distribution (2026-10-01)

Host: Windows 11 Pro 10.0.26300, build 26300. Toolchain: Go 1.27.1 windows/amd64, Node 24.21.0, npm 12.2.0; browser: Playwright Chromium 153.0.8010.12.

Final `npm.cmd run test:e2e` with `CRV_PACKAGE_DIR` set to the extracted final Windows archive: **8 passed**. Without it, the distribution test is intentionally skipped (7 passed / 1 skipped).

`go test ./...`, `go vet ./...`, `npm.cmd run check` (0 errors/warnings), `npm.cmd run build` and `npm.cmd run package` passed. Archives: `packages/crv-go-windows-amd64.tar.gz` and `packages/crv-go-linux-amd64.tar.gz` (approximately 21 MiB each); actual hashes are in `packages/SHA256SUMS.txt`. Tar listings confirm the sibling bank/licenses/instructions, no user data or development outputs, and Linux executable mode 0755. Source catalog/credits are unchanged; the frontend remains a single embedded HTML without the bank.

The extracted Windows executable passed `tests/package.spec.ts`: unrelated working directory, paths with spaces, empty child PATH, bootstrap, real catalog (192 eligible / 4 excluded), mouse drawing, shutdown/restart, preserved drawing, editing takeover/resume, complete session, CSV, headless PDF with decoded target image, ZIP backup/restore, missing-bank and insufficient-bank (three eligible images) repair states and graceful shutdown. External browser requests were blocked; none were attempted. This is browser-network evidence, not a physically disconnected host test. Both home themes at 1280 pixels and the dark home at 1000 pixels were visually inspected; no horizontal overflow at 1000 pixels. The modal closes with Escape. Full keyboard traversal and image/drawing review remain manual requirements.

To include the distribution test in the normal suite on Windows (extract first):

```powershell
$env:CRV_PACKAGE_DIR = 'D:\path with spaces\crv-go-windows-amd64'
npm.cmd run test:e2e
```

On Linux, point `CRV_PACKAGE_DIR` to the extracted Linux directory and use `npm run test:e2e` on a development test host. The executable itself requires neither Node nor Go; the browser test harness does. The opt-in test never launches Node/Go children to build or run the application. Test data and screenshots remain in isolated OS temporary directories printed by the test; PDF fixture evidence remains under gitignored `tmp-data/phase4-pdf-evidence/`.

Phase 5 remains **incomplete**: real Linux runtime (WSL is not installed), host-network-disabled operation, default-browser automatic launch, native Save as PDF, complete manual keyboard/drawing checks have no new evidence. No commit/push, signing or publication was performed.

## Drawing corrections and commit verification (2026-10-01)

Pointer-captured points outside the canvas are clamped before saving, preserving backend coordinate validation. Removed short-window canvas sizing that left part of the dashed sketch area unusable; the drawing surface fills the area at its 1000:620 aspect ratio. The session test verifies accepted saves at boundary coordinates for ideogram/sketch and canvas geometry at 1000×720, 1280×720 and 1440×900. `npm.cmd run check` passed with zero errors/warnings; the full source `npm.cmd run test:e2e` was rerun before commit: 7 passed / 1 skipped (distribution opt-in). Frontend build and `go build -o bin/crv-canvas-fix.exe .` passed. The corrected binary started and served HTTP 200 using isolated test data.

The user accepted the canvas correction and separately requested memory synchronization, commit and push. Existing Phase-5 archives predate these corrections; regenerate after closing the previous packaged executable. The historical eight-test package result above does not validate the corrected archives. Linux/native platform gates and signing/publication remain open.

