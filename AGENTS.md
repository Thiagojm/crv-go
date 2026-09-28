# CRV Go — instruções para continuidade

## Leia antes de trabalhar

1. `docs/CONTEXTO.md`: decisões da conversa e estado real da entrega.
2. `docs/design_app_crv.md`: especificação funcional aprovada.
3. `docs/PROXIMOS_PASSOS.md`: pendências e sequência sugerida.
4. `README.md` e `docs/VERIFICACAO.md`: execução e verificações anteriores.

## Regras do projeto

- Responda em português. Preserve decisões já tomadas; resolva detalhes rotineiros sem reabrir a discussão da stack.
- Este é um aplicativo novo, separado de Thiagojm/SRV. Não alterar aquele repositório nem recriar sua interface. O usuário não gostou do app SRV.
- Stack final: Go + Chi, Svelte + TypeScript + Vite, Tailwind, Canvas 2D e SQLite. Windows e Linux, local/offline, históricos independentes, mouse.
- O código atual é apenas um protótipo frontend. Não há backend Go, SQLite, importação real ou cegamento efetivo. Não apresentar a demonstração como aplicativo experimental pronto.
- Campos opcionais, checkboxes e texto livre, ajuda contextual e exemplos recolhidos fazem parte dos requisitos aprovados.
- No produto final: sortear alvo antes da coleta; guardar segredo no backend; bloquear registro antes das quatro alternativas; persistir escolha antes do feedback. Nunca adaptar distratores ao conteúdo da sessão.
- Preservar desenhos, hipóteses/AOL, versões do protocolo, estados, abandonos e feedback separado. Não reconstruir o original a partir de comentários posteriores.
- Banco de estudo originado de `archive/farsight` em https://github.com/Thiagojm/SRV, agora presente em `src/assets/farsight/`. Em 28/09/2026 o usuário decidiu manter imagens e catálogo no Git, substituindo a restrição anterior de versionamento. Preservar créditos. Isso não autoriza commit/push automaticamente nem decide embutir imagens no frontend ou executável; a integração por pacote local continua sendo a referência até revisão do design.
- Não confundir 714 entradas dos três pools com 714 imagens únicas. Importador precisa tratar sobreposição e imagens iguais. Começar pelo pool A.
- O aspecto visual foi proposto e entregue para teste, mas não houve aprovação explícita posterior da aparência. Não alegar aprovação pixel a pixel.
- Atualize `docs/CONTEXTO.md` e `docs/PROXIMOS_PASSOS.md` ao concluir etapas materiais.

## Comandos existentes

- `npm ci`
- `npm run dev`
- `npm run check`
- `npm run build`

Em PowerShell, `npm.cmd` é alternativa se a política de execução bloquear `npm.ps1`. Não mudar políticas globais para isso.

Não há comando de testes Go ou suíte E2E versionada neste pacote ainda. O relatório registra a verificação manual/automatizada feita no ambiente anterior, não uma garantia de que foi repetida aqui. Introduza testes necessários para os invariantes reais ao implementar o backend.
