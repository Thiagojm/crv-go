# Abrir no Codex com contexto

Guia conferido em 27/09/2026. Os rótulos podem variar conforme versão e disponibilidade da interface.

1. Extraia o pacote de modo que `AGENTS.md` fique em `C:\Projetos\crv-go\AGENTS.md`.
2. No ChatGPT desktop, selecione **Codex** no seletor de produto.
3. Em **Projects / Projetos**, crie ou abra um **projeto local** e selecione `C:\Projetos\crv-go` como pasta principal.
4. Inicie uma conversa dentro desse projeto.
5. Cole o texto abaixo.

> Continue o projeto desta pasta. Leia primeiro AGENTS.md, docs/CONTEXTO.md, docs/design_app_crv.md e docs/PROXIMOS_PASSOS.md. Esta pasta contém um protótipo frontend, não o aplicativo final. Preserve o fluxo CRV I–III, os checkboxes opcionais, a ajuda contextual e a escolha entre quatro imagens. A stack final é Go + Chi + Svelte + TypeScript + SQLite + Tailwind. É um app separado do SRV. Confira o estado atual e prossiga com a implementação do backend e integração do banco conforme a especificação, sem refazer a discussão de requisitos.

Se quiser somente revisar a interface primeiro, substitua a última frase por: “Abra o protótipo e me ajude a revisar a interface antes de implementar o backend.”

A documentação também descreve adicionar uma conversa ChatGPT existente a uma conversa Codex a partir de New chat / Quick chat, quando disponível. Use isso como contexto adicional, não como substituto dos arquivos. Não foi confirmada uma migração automática específica desta conversa Work para a sua interface.

Um projeto local fornece acesso à pasta do computador; um projeto ChatGPT com arquivos anexados não implica acesso ao seu disco C:. O ambiente remoto em que este pacote foi preparado também não tem esse acesso.

Fontes oficiais:

- https://learn.chatgpt.com/docs/projects
- https://learn.chatgpt.com/docs/use-chatgpt

Alternativa, se você já usa Codex CLI: abra PowerShell nesta pasta e execute `codex --cd "C:\Projetos\crv-go"`. Isso é uma alternativa ao aplicativo, não um passo necessário para usar sua aba Codex.
