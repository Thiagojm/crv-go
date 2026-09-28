# Contexto para continuar no Codex

Atualizado em 27/09/2026. Resumo de continuidade, não transcrição literal da conversa.

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
- Estágio I: ideograma, movimento/forma, sensação/consistência, gestalt e hipóteses/AOL.
- Estágio II: grupos de atributos sensoriais, campos livres e AOL.
- Estágio III: esboço amplo, notas de formas/dimensões/relações espaciais e AOL.
- Checkboxes opcionais com múltipla seleção e textarea em cada grupo. Nenhum campo perceptivo obrigatório. Listas fixas, independentes do alvo.
- Revisão antes de congelar; quatro alternativas depois; escolha definitiva antes de feedback.
- Ajuda Como preencher em cada etapa, orientação breve sempre visível, painel lateral, exemplos fictícios recolhidos e guia inicial Conhecer o fluxo.
- Salvar e encerrar; autosave, retomada da mesma sessão, mesmo alvo e mesma ordem.
- Temas claro/escuro, folha de desenho branca.
- Histórico, estatísticas, exportação PDF/CSV e backup ZIP no produto final.

A especificação completa está em `docs/design_app_crv.md`. O usuário aprovou criar essa especificação e prosseguir ao protótipo. Ainda não enviou avaliação visual ou correções após recebê-lo.

## Banco existente

Arquivos documentados: `archive/farsight/targets-pool-A.json`, pools B/C, `targets.json` e imagens locais. Campos de alvo: id, pool, index, target_specific, description, credit, images (com local_path) e sam_attributes. O parser do SRV também usa source e captured_at no pacote.

O inventário consultado informa 714 entradas, 238 por pool e 242 arquivos de imagem únicos; há sobreposição entre pools. A elegibilidade precisa ser validada na importação. Não prometer 60 sessões sem repetição: quatro imagens inéditas por sessão demandam 240 imagens elegíveis. O design propõe bloco de 50 quando existirem 200 disponíveis; familiarização e reservas reduzem esse total.

Começar pelo pool A; deduplicar arquivos por hash e manter informação de exposição/reservas. Todos os distratores apresentados tornam-se conhecidos. Atributos SAM e descrições só entram no feedback. Este pacote não inclui o acervo privado.

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
