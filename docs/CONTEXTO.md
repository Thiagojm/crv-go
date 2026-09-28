# Contexto para continuar no Codex

Atualizado em 28/09/2026. Resumo de continuidade, não transcrição literal da conversa.

## Revisão local e planejamento em 28/09/2026

- Checkout confirmado em `D:\Projetos\crv-go`, branch `main`, commit `6ed9d98`, sem alterações locais no início da revisão.
- Conferidos documentação, estrutura e código do protótipo: continuam ausentes backend Go, SQLite e cegamento efetivo.
- Executados neste Windows: `npm.cmd ci`, `npm.cmd run check` (zero erros/avisos) e `npm.cmd run build` (sucesso), com Node 24.18.0. Go disponível: 1.26.5 windows/amd64; não há código Go para testar. O HTML gerado foi restaurado à versão versionada após a verificação.
- Não foi repetido teste interativo de navegador, desenho com mouse ou execução em Linux nesta revisão.
- O usuário solicitou concluir spec e design pelo fluxo brainstorming antes da implementação. Seguir o fluxo completo: esclarecer decisões, consolidar spec em inglês, obter aprovação explícita e só então escrever o plano em inglês. Implementação exige solicitação posterior.
- Preservar a especificação funcional existente onde não foi substituída pelas decisões abaixo. Pontos a fechar: distribuição/importação do acervo e recuperação; transições de abandono/conclusão; concorrência entre abas e falhas de gravação. A interface atual foi escolhida como base com ajustes pontuais.

## Publicação do código

Em 27/09/2026, foi criado o repositório público https://github.com/Thiagojm/crv-go e enviado o branch `main`. O commit inicial (`dfac150`) contém o código, a documentação, capturas e imagens demonstrativas; o acervo privado `archive/farsight` não foi incluído. Isso publica o código e o protótipo, mas não significa que o aplicativo esteja pronto para uso experimental ou que será hospedado como serviço.

## Intenção do usuário

Thiago quer um app de PC simples e padronizado para praticar sozinho os estágios I–III do CRV. Um alvo é definido antes de uma sessão cega. Ao terminar, quatro imagens aparecem; ele escolhe uma e só então recebe o feedback. Quer separar evidência de impressões subjetivas, sem apresentação promocional ou esotérica.

Já possui o banco de imagens e informações complementares usado no repositório privado https://github.com/Thiagojm/SRV. Não gostou do aplicativo SRV e pediu um projeto separado. O repositório anterior não deve ser alterado. Local pretendido no Windows: `C:\Projetos\crv-go`.

## Decisões confirmadas

- Windows e Linux; desenho com mouse.
- Webapp local/offline, backend Go + Chi; frontend Svelte + TypeScript + Vite + Tailwind; Canvas 2D; SQLite no produto final.
- Histórico independente por instalação; sem login, sincronização ou nuvem.
- Uma imagem por alvo elegível.
- Um único modo de sessão, sem separação entre Familiarização e Avaliação (decisão explícita de 28/09/2026).
- Acompanhamento contínuo, sem blocos ou séries de tamanho definido. Histórico e resultados acumulados; abandonos continuam registrados. A regra anterior de análise confirmatória de blocos completos não se aplica mais; os indicadores finais serão descritos na nova spec como acompanhamento descritivo.
- Quatro alternativas ao final: um alvo correto e três distratores, todos distintos dentro da sessão.
- Todas as imagens elegíveis participam de cada novo sorteio. Repetição entre sessões é permitida; não há reservas por bloco nem exclusão por exposição anterior. Alvo, distratores e ordem são persistidos antes da coleta e preservados na retomada.
- Estágio I: ideograma, movimento/forma, sensação/consistência, gestalt e hipóteses/AOL.
- Estágio II: grupos de atributos sensoriais, campos livres e AOL.
- Estágio III: esboço amplo, notas de formas/dimensões/relações espaciais e AOL.
- Checkboxes opcionais com múltipla seleção e textarea em cada grupo. Nenhum campo perceptivo obrigatório. Listas fixas, independentes do alvo.
- Revisão antes de congelar; quatro alternativas depois; escolha definitiva antes de feedback.
- Ajuda Como preencher em cada etapa, orientação breve sempre visível, painel lateral, exemplos fictícios recolhidos e guia inicial Conhecer o fluxo.
- Salvar e encerrar; autosave, retomada da mesma sessão, mesmo alvo e mesma ordem.
- Temas claro/escuro, folha de desenho branca.
- Histórico, estatísticas, exportação PDF/CSV e backup ZIP no produto final.

A especificação anterior está em `docs/design_app_crv.md`; suas regras de modos separados, reservas, ciclos sem repetição e blocos foram substituídas pelas decisões acima e serão consolidadas na nova spec. O usuário escolheu manter a interface atual como base, com ajustes pontuais, e acompanhar os resultados continuamente.

## Banco existente

### Inventário local em 28/09/2026

O usuário adicionou os três pools em `src/assets/farsight/` e solicitou unificação e tratamento de duplicatas. Posteriormente removeu a regra de `.gitignore` e pediu explicitamente que as imagens fiquem no Git. Essa decisão substitui a restrição anterior de versionamento do acervo, mas não constitui solicitação de commit/push nem decisão de embuti-lo no frontend ou executável.

A auditoria local encontrou 714 entradas (238 por pool), 715 referências e 242 arquivos, todos com SHA-256 distintos; nenhum arquivo foi apagado porque não existem duplicatas binárias. `targets.json` já corresponde exatamente à concatenação dos três pools. Há 238 imagens distintas referenciadas por entradas com uma única imagem; duas entradas possuem duas imagens cada e uma não possui imagem. Todas as imagens têm referência. Ainda não foi validada a decodificação nem a equivalência visual entre arquivos diferentes.

Gerado o inventário `src/assets/farsight/catalog-unified.json`, agrupado por hash, preservando os registros de origem e créditos e separando entradas sem imagem ou com múltiplas imagens para revisão. Não é o formato definitivo de importação do app. A pedido do usuário, a pasta foi reduzida a esse catálogo e `images/`: os três JSONs de pools, `targets.json`, `manifest.json` e o utilitário transitório `audit-unify.mjs` foram removidos. Antes da exclusão, foram conferidos os 714 registros completos, metadados de todos os pools, metadados globais (incluindo créditos/restrições de uso), manifesto original e hashes das 242 imagens. Metadados globais e manifesto foram incorporados ao catálogo em `sourceMetadata` e `sourceManifest`; referências a arquivos antigos no manifesto são proveniência histórica. Nenhuma imagem foi removida.

O usuário rejeitou a proposta de reservar imagens por bloco: todas as imagens elegíveis ficam disponíveis em cada nova sessão, independentemente de uso anterior. Confirmou um único modo e um alvo com três distratores.

Arquivos documentados: `archive/farsight/targets-pool-A.json`, pools B/C, `targets.json` e imagens locais. Campos de alvo: id, pool, index, target_specific, description, credit, images (com local_path) e sam_attributes. O parser do SRV também usa source e captured_at no pacote.

O inventário consultado informa 714 entradas, 238 por pool e 242 arquivos de imagem únicos; há sobreposição entre pools. A elegibilidade precisa ser validada na importação. A decisão de permitir repetição entre sessões elimina o requisito anterior de 200 imagens para 50 sessões; cada sessão exige ao menos quatro imagens elegíveis distintas.

A especificação inicial propõe começar pelo pool A; o inventário local agora unifica os três pools e preserva suas origens. O novo sorteio considera o acervo elegível unificado, sem ponderar imagens pelo número de referências nos pools. Exposição anterior não impede reutilização nem torna as imagens inéditas. Atributos SAM e descrições só entram no feedback. O acervo está presente localmente; a publicação de 27/09 não o incluía.

## Entrega realmente existente

Frontend Svelte navegável em `src/` e versão autocontida em `crv_prototipo.html` / `dist/index.html`.

Implementado no protótipo: navegação completa, canvas por mouse, checkboxes e textos, ajuda, revisão, bloqueio pela interface, escolha, feedback, comentários, histórico localStorage, temas, pausa/cronômetro, CSV, backup/restauração JSON e tela de encerramento.

São quatro fotos demonstrativas do Unsplash, repetidas. Alvo e ordem são mantidos em dados do frontend. Não há sigilo experimental. A modalidade Avaliação aparece indisponível. Encerramento apenas mostra a tela final: não existe servidor Go para parar.

Ainda não existem `go.mod`, backend, Chi, banco SQLite, importação do pacote SRV, controle real de repetição/ciclos, blocos experimentais, análise binomial, PDF ou backup ZIP do app final. Não afirmar que esses recursos foram implementados.

## Verificações anteriores

Node 24.19.0; `npm run check` passou sem erros/avisos; build passou. Playwright/Chromium headless abriu o HTML via file:// e percorreu o fluxo. Foram verificadas persistência ao recarregar, preservação dos desenhos, escolha, ajuda, temas e encerramento. Viewports 1536×1024, 1280×720 e 390×844. Ver `docs/VERIFICACAO.md`.

A fonte Inter foi embutida para evitar substituição visual; botão de encerramento ganhou nome acessível em layout estreito. Ajustada a área de desenho em telas de baixa altura. Não houve teste nativo no Windows nem validação de Firefox. O script transitório de QA não foi incluído, apenas o relatório e capturas.

## Próxima conversa

O usuário quer abrir o projeto na aba Codex do ChatGPT desktop e continuar com esse contexto. Estes documentos permitem a continuidade mesmo sem importar a conversa original. Não é necessário refazer a pesquisa de RV ou rediscutir a stack. Primeiro distinguir o que ele deseja fazer na nova tarefa: testar/refinar aparência ou implementar a versão funcional. Se pedir continuar a implementação, seguir o roteiro de pendências e preservar o fluxo acordado.

Este pacote foi organizado no ambiente remoto; não foi gravado diretamente no disco C: do usuário. A extração no Windows é necessária antes de abrir a pasta como projeto local.
