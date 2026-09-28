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

Não verificado nesta fase: empacotamento offline completo, Linux runtime, abertura automática do navegador quando `rundll32` falha (apenas log da URL), sessões reais.

