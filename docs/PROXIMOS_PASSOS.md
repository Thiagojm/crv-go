# Próximos passos

## 1. Publicar o código-fonte (concluído em 27/09/2026)

- [x] Revisar o conteúdo versionado; nenhum dado privado do SRV foi incluído.
- [x] Criar o repositório público `Thiagojm/crv-go` e enviar o branch `main`.
- [x] Registrar o endereço remoto e o estado da publicação em `docs/CONTEXTO.md` e `README.md`.
- Esta etapa publica o código e o protótipo; não hospeda o aplicativo nem o torna apto para uso experimental.

## 2. Preparar e conferir localmente

- Abrir `C:\Projetos\crv-go` como projeto local no Codex.
- Ler AGENTS.md e CONTEXTO.md; revisar o protótipo e registrar ajustes solicitados.
- Executar npm ci, npm run check e npm run build; testar desenho e navegação no Windows.
- Registrar versão de Node, Go, navegador e resultados. Não atribuir ao Windows os testes do ambiente anterior.

## 3. Implementar núcleo Go

- Criar módulo Go e servidor Chi em loopback; servir build local e abrir navegador.
- Separar handlers HTTP, domínio e persistência; definir schema e migrações SQLite.
- Definir modelos de sessão, registro, pacote, alvo, ciclo, bloco e exposição.
- Reservar conjunto e sortear alvo antes da coleta; usar aleatoriedade uniforme do sistema.
- Implementar transições atômicas e idempotentes: coletar, bloquear, escolher, revelar, abandonar.
- Restringir respostas: coleta sem imagem/identidade; escolha com quatro imagens sem marcação; feedback com metadados após confirmação.

## 4. Integrar banco e frontend

- Obter pacote local do banco SRV sem alterar o repositório original.
- Validar caminhos, imagens, metadata, hashes e duplicatas; congelar versão do pacote.
- Substituir localStorage e dados de demonstração pela API, preservando a interface.
- Persistir traços, tempos, campos, versões e retomada; distinguir erro de salvamento de sucesso.
- Implementar shutdown gracioso pelo botão Salvar e encerrar.

## 5. Completar produto

- Ciclos sem repetição, controle dos distratores vistos e familiaridade declarada.
- Familiarização e blocos de avaliação fixos; abandonos e blocos incompletos.
- Estatísticas binomiais e IC, com testes de casos conhecidos e rótulos apropriados.
- PDF/CSV e backup/restauração ZIP consistentes.
- Builds Windows/Linux; documentação de execução offline e teste real em ambos.

## Prioridade de testes

Antes de uso experimental, verificar cegamento na API, invariantes de transição, recuperação após interrupção, não repetição e dados imutáveis após feedback. O protótipo visual não substitui esses controles.

## Fora do escopo atual

IA interpretativa, ARV financeiro, estágios IV–VI, SRV Basic/Enhanced, sincronização, login, multiusuário, publicação pública e migração de registros antigos do SRV.
