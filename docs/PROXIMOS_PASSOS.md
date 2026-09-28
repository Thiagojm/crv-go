# Próximos passos

## 1. Publicar o código-fonte (concluído em 27/09/2026)

- [x] Revisar o conteúdo versionado; nenhum dado privado do SRV foi incluído.
- [x] Criar o repositório público `Thiagojm/crv-go` e enviar o branch `main`.
- [x] Registrar o endereço remoto e o estado da publicação em `docs/CONTEXTO.md` e `README.md`.
- Esta etapa publica o código e o protótipo; não hospeda o aplicativo nem o torna apto para uso experimental.

## 2. Preparar e conferir localmente

- [x] Checkout local conferido em `D:\Projetos\crv-go` em 28/09/2026.
- [x] `npm.cmd ci`, `npm.cmd run check` e `npm.cmd run build` passaram neste Windows; Node 24.18.0, Go 1.26.5 disponível.
- Ler AGENTS.md e CONTEXTO.md; revisar o protótipo e registrar ajustes solicitados.
- Testar desenho e navegação no Windows; essa validação interativa ainda não foi repetida localmente.
- Registrar versão de Node, Go, navegador e resultados. Não atribuir ao Windows os testes do ambiente anterior.

## 2.1 Concluir spec e design antes da implementação (etapa atual)

- Aplicar brainstorming completo, preservando stack, campos e fluxo já definidos.
- [x] Usuário escolheu manter a interface atual como base, com ajustes pontuais; isso não constitui aprovação de todos os detalhes visuais.
- [x] Auditar os três pools locais e gerar catálogo unificado por hash: 242 arquivos distintos, nenhuma cópia binária a excluir, 238 imagens candidatas provenientes de entradas com uma única imagem.
- [x] Limpar arquivos redundantes a pedido do usuário: manter somente `catalog-unified.json` e `images/`, preservando todos os registros, créditos e metadados no catálogo. Usuário decidiu permitir o acervo no Git; nenhum commit/push foi solicitado nesta etapa.
- Validar decodificação das imagens e definir o contrato final de importação a partir do inventário; manter fora da seleção automática as duas entradas com múltiplas imagens e a entrada sem imagem. O inventário local não é ainda o importador do produto.
- [x] Confirmados modo único, quatro alternativas (um alvo e três distratores) e sorteio do acervo elegível completo, sem reservas ou exclusão por uso anterior. Repetição permitida entre sessões; imagens distintas dentro de cada sessão.
- [x] Confirmado acompanhamento contínuo, sem blocos ou séries de tamanho definido, com histórico e resultados acumulados.
- Fechar contratos de distribuição/importação e recuperação, estados persistentes e concorrência entre abas.
- Consolidar spec em inglês em `docs/specs/`, com critérios observáveis e verificação dos invariantes; solicitar aprovação explícita.
- Após aprovação da spec, escrever plano em inglês em `docs/plans/`, dividido em fases testáveis com validação do usuário ao final de cada fase.
- Não iniciar implementação durante esse fluxo. O roteiro abaixo é referência de pendências, não autorização de execução.

## 3. Implementar núcleo Go

- Criar módulo Go e servidor Chi em loopback; servir build local e abrir navegador.
- Separar handlers HTTP, domínio e persistência; definir schema e migrações SQLite.
- Definir modelos de sessão, registro, pacote e alvo; estatísticas contínuas, sem entidade de bloco.
- Sortear e persistir alvo, três distratores distintos e ordem antes da coleta; usar aleatoriedade uniforme do sistema sobre imagens elegíveis únicas, permitindo reutilização em outras sessões.
- Implementar transições atômicas e idempotentes: coletar, bloquear, escolher, revelar, abandonar.
- Restringir respostas: coleta sem imagem/identidade; escolha com quatro imagens sem marcação; feedback com metadados após confirmação.

## 4. Integrar banco e frontend

- Obter pacote local do banco SRV sem alterar o repositório original.
- Validar caminhos, imagens, metadata, hashes e duplicatas; congelar versão do pacote.
- Substituir localStorage e dados de demonstração pela API, preservando a interface.
- Persistir traços, tempos, campos, versões e retomada; distinguir erro de salvamento de sucesso.
- Implementar shutdown gracioso pelo botão Salvar e encerrar.

## 5. Completar produto

- Um único modo de sessão, sem ciclos de exclusão ou reservas; preservar abandonos e resultados anteriores.
- Estatísticas contínuas: definir indicadores descritivos e critérios de cálculo na nova spec, com testes de casos conhecidos. A análise confirmatória de blocos da especificação anterior foi superada; não implementar blocos ou separação por modo.
- PDF/CSV e backup/restauração ZIP consistentes.
- Builds Windows/Linux; documentação de execução offline e teste real em ambos.

## Prioridade de testes

Antes de uso experimental, verificar cegamento na API, invariantes de transição, recuperação após interrupção, quatro imagens distintas dentro de cada sessão e dados imutáveis após feedback. O protótipo visual não substitui esses controles.

## Fora do escopo atual

IA interpretativa, ARV financeiro, estágios IV–VI, SRV Basic/Enhanced, sincronização, login, multiusuário, publicação pública e migração de registros antigos do SRV.
