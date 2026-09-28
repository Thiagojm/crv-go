# CRV Go — projeto e contexto de continuidade

Checkout conferido neste ambiente: `D:\Projetos\crv-go`. Comece por `AGENTS.md`, pela [spec aprovada](docs/specs/2026-09-28-crv-offline-design.md) e pelo [plano de implementação](docs/plans/2026-09-28-crv-offline-plan.md). A implementação ainda não começou.

Repositório público: https://github.com/Thiagojm/crv-go

Este pacote reúne o protótipo, código-fonte, design, capturas, decisões e pendências. O backend Go ainda será implementado.

# CRV — protótipo navegável

Protótipo de interface em português, criado a partir do design aprovado em 26/09/2026. Projeto independente do SRV. Inclui desenho com mouse e fluxo dos estágios I–III até escolha e feedback.

## Abrir sem instalar

Abra `crv_prototipo.html` em um navegador de desktop atualizado, como Edge, Chrome ou Firefox. O arquivo inclui interface, fontes e imagens; não depende de internet. Se a prévia de arquivos do ChatGPT não executar a interface, baixe o HTML e abra-o no navegador do PC.

O mesmo HTML está em `dist/index.html`. Use sempre o mesmo arquivo e navegador para preservar o histórico demonstrativo. Armazenamento de arquivos locais varia entre navegadores; se houver restrição, a interface avisa. A exportação JSON em Configurações permite guardar uma cópia.

## O que experimentar

1. Clique em Nova sessão e Iniciar demonstração.
2. Faça um ideograma com o mouse. Teste caneta, espessura, desfazer/refazer e borracha (remove um traço inteiro).
3. Abra Registrar impressões, marque atributos e escreva livremente.
4. Use Como preencher e Ver exemplo fictício em qualquer etapa.
5. Preencha sensoriais e faça um esboço no estágio III.
6. Revise, confirme o bloqueio e compare as quatro imagens.
7. Confirme uma escolha e registre sua reflexão após o feedback.
8. Consulte histórico, estatísticas, tema escuro e exportações CSV/JSON.
9. Teste Salvar e encerrar e a retomada do registro.

## Funciona nesta entrega

- Fluxo navegável completo, campos opcionais e listas do design aprovado.
- Canvas por mouse, com traços separados para ideograma e esboço.
- Ajuda por etapa, exemplos recolhidos e guia inicial.
- Registro bloqueado pela interface antes das alternativas.
- Ordem e alvo demonstrativos preservados ao recarregar.
- Histórico no armazenamento local do navegador e registros abandonados.
- Comentário de feedback separado do registro original.
- Cronômetro de coleta e escolha, pausa e retomada.
- Temas claro/escuro; layout desktop e adaptação a janela estreita.
- Exportação CSV; backup e restauração JSON da demonstração.

## Limites explícitos

**Não é a versão experimental nem o aplicativo final.** As mesmas quatro fotografias se repetem e a identidade do alvo está no frontend/armazenamento do navegador. Não use os resultados para inferir acurácia de RV.

Ainda não implementados: backend Go + Chi, SQLite, integração real do banco incluído em `farsight/`, cegamento no servidor, estatísticas contínuas, exportação PDF e backup ZIP do aplicativo final. A spec aprovada usa um único modo e permite repetição de imagens entre sessões; as quatro alternativas de cada sessão continuam distintas. O botão de encerramento nesta versão apenas mostra uma tela final; não existe servidor Go a desligar.

O fechamento inesperado pode perder dados ainda não gravados. O controle local é próprio de protótipo e não impede inspeção deliberada nem edição pelo desenvolvedor do navegador. A restauração JSON aceita apenas o formato deste protótipo.

Nenhuma alteração foi feita no repositório SRV. O catálogo unificado e as imagens agora estão versionados em `farsight/` por decisão do usuário; o protótipo continua usando somente as quatro fotos demonstrativas.

## Desenvolvimento

Requisitos: Node.js 22.12+ ou 24 e npm. A versão utilizada na verificação foi Node 24.19.0.

```sh
npm ci
npm run dev
npm run check
npm run build
```

O build gera `dist/index.html` autocontido. Frontend Svelte + TypeScript, Vite, Tailwind e CSS de componentes. A fonte Inter Variable é embutida para consistência offline. O backend planejado continua Go + Chi + SQLite.

### Organização

- `src/App.svelte`: navegação e coordenação da demonstração.
- `src/components/DrawingPad.svelte`: canvas e ferramentas.
- `src/components/FieldGroup.svelte`: atributos e texto livre.
- `src/components/Record.svelte`: registro somente leitura.
- `src/components/Help.svelte`: ajuda contextual.
- `src/components/Icon.svelte`: ícones SVG.
- `src/data.ts`: atributos, ajuda, tipos e dados de exemplo.
- `src/style.css`: tokens, temas e layouts.
- `docs/design_app_crv.md`: especificação funcional de referência.
- `docs/DESIGN-SYSTEM.md`: direção visual e desvios intencionais do conceito.
- `docs/VERIFICACAO.md`: verificações executadas e limites.
- `docs/conceito.png`: conceito visual gerado.
- `docs/tela-ideograma.png`, `docs/tela-escolha.png`, `docs/tela-ajuda.png`: capturas do protótipo implementado.

## Imagens e fonte

As quatro fotografias demonstrativas são do Unsplash, preservadas sem sobrepor texto à fotografia. Não fazem parte do banco SRV. Fontes dos arquivos:

- https://images.unsplash.com/photo-1470770841072-f978cf4d019e
- https://images.unsplash.com/photo-1518837695005-2083093ee35b
- https://images.unsplash.com/photo-1441974231531-c6227db76b6e
- https://images.unsplash.com/photo-1511818966892-d7d671e672a2

Inter Variable, por Rasmus Andersson, distribuída via @fontsource-variable/inter sob SIL Open Font License. Licença incluída em `docs/INTER-OFL.txt`.

As dependências conservam suas licenças. As imagens e a referência de interface não estabelecem qualquer evidência sobre remote viewing.
