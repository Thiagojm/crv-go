# Sistema visual do protótipo

## Phase 5 distribution baseline (2026-10-01)

The historical visual reference below remains the baseline. Phase 5 packages the existing Go-backed interface; it does not redesign the application. Windows packaged-browser checks and their exact limits are recorded in `docs/VERIFICACAO.md`. Linux runtime and native browser printing require separate platform evidence.

Referência: conceito gerado em `conceito.png`, pela ferramenta de geração de imagens integrada. O conceito é uma proposta desta etapa, não uma aprovação visual já recebida do usuário.

## Direção

Caderno de pesquisa contemporâneo, interface discreta, fundo cinza neutro levemente esverdeado, papel branco e verde floresta. Conteúdo em português, sem referências místicas. Tipografia Inter Variable embutida, com alternativas de sistema, para uso offline e aparência consistente.

- Fundo: #f5f6f3; superfície: #ffffff; texto: #202b27.
- Destaque: #254e43; seleção: #e6eeea; borda: #dce1db.
- Títulos: 29–42 px; corpo: 13–16 px; rótulos: 11–15 px.
- Espaçamento: 8, 12, 16, 24, 36, 48 px.
- Raios: 4–8 px; divisórias finas; nenhuma sombra decorativa no conteúdo principal.
- Ícones próprios em SVG, traço de 1,65 px, 16–24 px.
- Folha de desenho branca mesmo no tema escuro; ferramentas em superfície temática.
- Painel da etapa I em proporção aproximada 60/40; etapa III privilegia o canvas.
- Botão primário verde e secundário com contorno; atributos como checkboxes em retângulos compactos.
- Ajuda lateral e confirmações modais; conteúdo longo rola sem sobrepor o rodapé.

## Composição e escopo

Cabeçalho global, stepper durante sessões, navegação lateral fora delas. Componentes separados para desenho, grupos de atributos, registro somente leitura, ajuda e ícones. O shell coordena demonstração, histórico e navegação.

As telas adicionais usam o mesmo sistema: preparação, sensorial, esboço, revisão, escolha, feedback, histórico, estatísticas, configurações e encerramento. Não há backend nesta entrega.

## Desvios intencionais em relação à imagem-conceito

A especificação funcional aprovada tem precedência sobre abreviações da imagem: listas completas, textarea por grupo e AOL; atributos recolhidos inicialmente; nome do botão “Salvar e encerrar”; ausência de desenho predefinido. O canvas mantém proporção lógica estável em vez de esticar o desenho para preencher uma altura arbitrária. A indicação “Protótipo de interface” evita confundir esta entrega com um experimento válido.

## Brief da geração

Tela desktop CRV, estágio Ideograma, português, canvas branco à esquerda, atributos opcionais à direita, cabeçalho com código/cronômetro, sete etapas, ajuda contextual e rodapé Voltar/Continuar. Paleta neutra e verde, tipografia sans legível, sem gráficos decorativos nem métricas fictícias. A imagem-conceito é referência visual; todos os controles reais são HTML/Svelte.
