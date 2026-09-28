# Aplicativo CRV — design funcional e técnico

Versão: 1.0  
Data: 26/09/2026  
Responsável pelo produto: Thiago  
Status: consolidação do escopo aprovado; pronto para protótipo visual e implementação posterior.  
Nome de trabalho: CRV. Nome comercial, ícone e paleta definitiva podem ser definidos no design visual, sem bloquear o projeto.

## 1. Objetivo e limites

Criar um aplicativo pessoal para registrar sessões inspiradas nos estágios I–III do Controlled Remote Viewing, com um alvo oculto e escolha posterior entre quatro imagens. A experiência deve ser simples, rápida, guiada e confortável para desenho com mouse.

O programa será independente do SRV existente: novo projeto, interface e fluxo próprios. O banco de alvos será reaproveitado; componentes antigos só serão reutilizados quando servirem ao novo design, sem impor a experiência anterior.

O aplicativo registra impressões subjetivas e mede a escolha final. Não interpreta ideogramas automaticamente nem apresenta o protocolo como cientificamente comprovado. A adaptação com mouse, checkboxes e autossessão não equivale a um curso completo de CRV.

Este documento especifica o produto; não representa um aplicativo já implementado ou testado. Decisões operacionais adicionais abaixo são padrões de implementação para concretizar o escopo aprovado.

## 2. Decisões fechadas

| Item | Definição |
| --- | --- |
| Sistemas | Windows e Linux |
| Execução | Webapp local, aberto no navegador, inteiramente offline após instalação e importação |
| Usuário | Um usuário por instalação, sem conta ou login |
| Histórico | Independente em cada instalação; sem sincronização |
| Backend | Go + Chi |
| Frontend | Svelte + TypeScript + Vite |
| Estilo | Tailwind CSS |
| Desenho | Canvas 2D com mouse; traços persistidos como dados |
| Dados | SQLite e arquivos locais gerenciados |
| Alvos | Banco existente em Thiagojm/SRV; uma imagem por alvo elegível |
| Sessão | Estágios I, II e III; revisão; escolha entre quatro; feedback |
| Campos | Checkboxes opcionais, seleção múltipla e escrita livre |
| Ajuda | Instruções por etapa, painel lateral e exemplos recolhidos |
| Encerramento | Botão Salvar e encerrar |
| Temas | Claro e escuro; superfície de desenho branca por padrão |
| Exportação | PDF da sessão, CSV dos resultados e ZIP de backup |

## 3. Navegação e organização visual

### Fora da sessão

Menu lateral: Início, Histórico, Estatísticas e Configurações. A gestão do banco de alvos fica em Configurações, evitando uma galeria de imagens na navegação habitual.

Na tela inicial: Nova sessão, Continuar sessão quando houver, Conhecer o fluxo e resumo do bloco atual. Uma única sessão pode estar ativa por instalação. Salvar e encerrar permanece acessível.

### Durante a sessão

Menu principal recolhido. Barra superior com etapa atual, código neutro, indicador de salvamento, cronômetro discreto e Como preencher. Rodapé com Voltar e Continuar; na revisão, o botão se chama Finalizar registro.

O cabeçalho de progresso apresenta: Preparação, Ideograma, Sensorial, Esboço, Revisão, Escolha e Feedback. A navegação só permite estados autorizados. Não é possível pular diretamente para escolha ou feedback.

Layout prioritário para desktop a partir de 1280 × 720, com painéis empilháveis em janelas menores. Textos com boa legibilidade, foco de teclado visível e estados que não dependam apenas de cor. Checkboxes devem ter rótulos clicáveis. Áreas de desenho não podem perder conteúdo quando a janela for redimensionada.

## 4. Fluxo completo

1. Escolher Familiarização ou Avaliação e iniciar a sessão.
2. O backend seleciona quatro alvos elegíveis, sorteia uniformemente um como correto e persiste a definição antes de liberar a coleta.
3. Exibir somente um código aleatório neutro, sem relação com nomes, pool ou identificadores originais.
4. Registrar estágio I, estágio II e estágio III.
5. Revisar, opcionalmente destacar até cinco características principais e finalizar o registro.
6. Bloquear a edição original de forma transacional no backend.
7. Apresentar quatro imagens em posições embaralhadas, preservadas ao reabrir.
8. Selecionar uma imagem, registrar confiança opcional e confirmar a escolha.
9. Persistir a escolha antes de revelar o alvo correto.
10. Exibir feedback, permitir comentário posterior e concluir a sessão.

Não há campos perceptivos obrigatórios. Uma sessão incompleta pode ser finalizada, com os campos vazios preservados e indicados na revisão.

## 5. Telas e campos

### 5.1 Preparação

Mostrar modo, bloco quando aplicável, disponibilidade de alvos, referência de duração e instrução breve. Disposição e concentração são registros opcionais, sem efeito sobre o sorteio.

Referência de duração: dez minutos de coleta, sem interrupção automática ou contagem regressiva alarmante. Contabilizar separadamente tempo ativo de coleta e tempo de escolha. Tempo com o programa fechado ou sessão explicitamente pausada não entra no tempo ativo. Abrir ajuda não pausa automaticamente.

O botão Iniciar cria e persiste a sessão. Se a criação falhar, não apresentar um código como se a sessão estivesse pronta.

### 5.2 Estágio I — Ideograma

Área de desenho à esquerda; atributos e textos à direita. O painel de atributos é aberto pelo botão Registrar impressões, sem obrigar a desenhar primeiro. Não há interpretação automática do traço.

| Grupo | Opções iniciais |
| --- | --- |
| Movimento / forma | Horizontal, vertical, diagonal, curvo, ondulado, angular, circular, sobe, desce |
| Sensação / consistência | Sólido, duro, macio, fluido, gasoso, liso, áspero, granulado |
| Categoria geral / gestalt | Água, terreno/relevo, estrutura construída, vegetação, ser vivo, indefinido |

Cada grupo inclui textarea Outra impressão / descrição livre. Adicionar campo separado Hipóteses / AOL. Todas as opções são desmarcadas inicialmente. Permitir múltiplas seleções; indefinido não deve ser convertido automaticamente em uma categoria.

Uma opção não marcada significa não registrada, e não ausência da característica no alvo. Os rótulos são recursos de registro, não um dicionário universal de ideogramas.

### 5.3 Estágio II — Sensorial

Grupos em cartões simples, com checkbox e textarea própria:

| Grupo | Opções iniciais |
| --- | --- |
| Cores | Branco, preto, cinza, azul, verde, vermelho, amarelo, marrom, laranja, violeta |
| Luminosidade | Claro, escuro, brilhante, fosco, reflexivo, translúcido |
| Textura | Liso, áspero, granulado, fibroso, pegajoso, escorregadio |
| Temperatura | Frio, fresco, morno, quente |
| Umidade | Seco, úmido, molhado |
| Sons | Silencioso, contínuo, intermitente, grave, agudo, rítmico, ruído de água, mecânico, vozes |
| Cheiros | Terroso, vegetal, salino, floral, químico, queimado |
| Sabores | Doce, salgado, amargo, ácido, metálico |

Sons, cheiros e sabores começam recolhidos. Exibir indicação de conteúdo preenchido nos grupos recolhidos. Incluir Outras impressões sensoriais e Hipóteses / AOL.

As listas são fixas e independentes do alvo. Não usar atributos SAM, nomes, descrições ou resultados anteriores para sugerir respostas durante a coleta.

### 5.4 Estágio III — Esboço

Canvas amplo, ocupando a maior parte da tela. Painel lateral com campos livres: Formas, Dimensões/proporções, Posições e relações espaciais, Outras anotações e Hipóteses / AOL.

Ferramentas: caneta, borracha, espessura, desfazer, refazer e limpar com confirmação. A borracha pode remover um traço inteiro na primeira versão, com esse comportamento claro no rótulo de ajuda. Não incluir reconhecimento de objetos, desenho generativo ou correção automática de formas.

Salvar coordenadas em um espaço lógico estável e escalar apenas a apresentação. Preservar traços ao mudar tema, tamanho da janela ou etapa. Uma prévia PNG auxilia histórico e exportação; os dados dos traços permanecem a fonte editável até o bloqueio.

### 5.5 Revisão e bloqueio

Mostrar os três estágios, incluindo checkboxes, textos, desenhos, AOL e campos vazios. Permitir voltar e editar. Campo opcional para até cinco características principais e confiança geral de 0 a 100, sem valor pré-preenchido.

Confirmação: Ao continuar, seu registro será bloqueado e as quatro imagens serão apresentadas. Você poderá comentar depois, mas não alterar o original.

A ação precisa aguardar a confirmação da gravação. Se falhar, manter coleta/revisão e não liberar as alternativas.

### 5.6 Escolha entre quatro

Quatro imagens em grade 2 × 2, identificadas apenas como A, B, C e D, com mesma moldura e proporção original preservada, sem recorte que elimine elementos do alvo. Permitir ampliar cada imagem sem mostrar metadados.

Registro completo disponível em painel ao lado, somente leitura. Selecionar uma alternativa gera um destaque; confirmar exige botão próprio. A seleção pode mudar antes da confirmação, nunca depois. Confiança da escolha é opcional, de 0 a 100, sem sugestão inicial.

Não exibir nomes de arquivos, descrições, créditos ou códigos originais, inclusive em textos alternativos, tooltips e respostas da API. Recursos acessíveis podem usar rótulos neutros como Alternativa A.

### 5.7 Feedback

Mostrar acerto ou erro, alternativa escolhida, alvo correto, fotografia ampliável, identificação, descrição e créditos. Atributos SAM ficam em Detalhes do alvo, recolhidos por padrão.

Campo Comentário após feedback separado do registro original. Exibir tempo de coleta e escolha. A pontuação principal é binária: escolha correta = 1; incorreta = 0. Não atribuir pontos por semelhanças textuais, quantidade de checkboxes ou interpretação retrospectiva de desenhos.

### 5.8 Histórico e estatísticas

Histórico com data, modo, bloco, estado, tempo e resultado quando houver. Filtros por modo, bloco e estado. Abrir sessão concluída mostra registro imutável e feedback. Sessões abandonadas continuam visíveis, sem revelar alvos ainda ocultos.

Indicadores: iniciadas, concluídas, abandonadas antes/depois das alternativas, acertos/total de escolhas confirmadas, taxa de acertos e referência de 25%. Separar Familiarização e Avaliação.

Para blocos completos, apresentar intervalo de confiança binomial de 95% (Wilson) e teste binomial exato unilateral contra p = 0,25. Identificar essas análises como desempenho neste teste; não como comprovação de um mecanismo paranormal. Blocos incompletos mostram contagens descritivas e sua condição de incompletos, sem conclusão confirmatória.

Gráfico simples de acertos acumulados com expectativa de acaso. Não tratar análises exploratórias de confiança, tempo ou atributos como evidência confirmatória.

## 6. Ajuda contextual

Botão Como preencher ao lado do título de cada etapa. Abre painel lateral preservando conteúdo, seleção e posição do desenho. Uma orientação curta fica sempre visível; exemplos começam recolhidos em Ver exemplo.

Cada painel contém Objetivo, Como preencher, Exemplo fictício e Cuidados. O conteúdo é fixo, versionado e independente do alvo. Não usar fotografias do banco nos exemplos. Registrar versão da ajuda e, para auditoria, abertura de exemplos durante a sessão.

| Etapa | Texto-base da orientação | Exemplo ou cuidado |
| --- | --- | --- |
| Preparação | Registre suas impressões antes de ver qualquer imagem. O código identifica a sessão. | O cronômetro orienta; não encerra a sessão. |
| Ideograma | Faça um gesto breve e registre movimento, sensação e uma categoria geral, se surgir. | Movimento: curvo; sensação: fluido; categoria: água? Exemplo fictício, não regra de interpretação. |
| Sensorial | Marque ou escreva qualidades percebidas; deixe em branco o que não surgiu. | Frio, áspero, cinza são descritores. Parece um castelo é uma hipótese que pode ir em AOL. |
| Esboço | Represente formas, proporções e posições. Não é necessário saber desenhar. | Um retângulo alto ao lado de uma área ampla expressa uma relação espacial sem identificar o objeto. |
| Revisão | Confira seu registro antes de bloqueá-lo. | Campos vazios são permitidos; depois de ver as imagens, o original não muda. |
| Escolha | Compare o registro inteiro com as quatro imagens. | Considere contradições e não apenas um detalhe semelhante. |
| Feedback | Confira o resultado e escreva sua reflexão separadamente. | Um acerto isolado não demonstra desempenho acima do acaso. |

Na tela inicial, Conhecer o fluxo permite ler todas as orientações antes de iniciar uma sessão. Explicar AOL como hipótese/interpretação analítica, sem afirmar que o restante do registro esteja livre de imaginação ou erro.

## 7. Banco de alvos e sorteio

Fonte: banco local associado ao repositório privado https://github.com/Thiagojm/SRV, em archive/farsight/. Primeira versão: compatibilidade com targets-pool-A.json e imagens locais referenciadas. Preservar identificadores, descrição, créditos e atributos originais.

O inventário do repositório consultado informa 714 entradas em três pools, 238 por pool e 242 arquivos de imagem únicos, com sobreposição entre pools. Esses números são contexto do acervo, não uma contagem de elegibilidade já validada para o novo app.

Importação:

1. Selecionar um pacote ZIP com JSON e imagens em caminhos relativos; oferecer configuração de pasta local como opção avançada.
2. Validar estrutura, presença e decodificação das imagens, caminhos seguros e campos necessários.
3. Aceitar uma imagem por registro elegível. Entradas incompatíveis são listadas no relatório, sem escolher imagens silenciosamente.
4. Copiar arquivos válidos para área gerenciada, mantendo o acervo original intacto.
5. Calcular SHA-256 por imagem e identificar duplicatas exatas; não alegar que isso detecta todos os recortes ou versões modificadas.
6. Mostrar resumo de entradas válidas, inválidas, duplicadas e quantidade disponível, sem abrir prévias automaticamente.
7. Guardar versão/hash do pacote. Novas importações não alteram sessões ou blocos existentes.

O sorteio usa um conjunto de quatro imagens distintas e elegíveis, seguido da seleção uniforme de uma como alvo correto. Todas as alternativas vêm do mesmo processo. Não selecionar distratores com base no texto da sessão ou escolher manualmente o alvo correto.

Usar aleatoriedade do sistema, com seleção uniforme sem viés de módulo. Registrar internamente conjunto, alvo, ordem, versão do pacote e código neutro antes da coleta. IDs originais e nomes de arquivos permanecem no backend.

As quatro imagens ficam reservadas assim que a sessão começa. Abandono não libera uma nova tentativa com o mesmo conjunto dentro do ciclo. Imagens efetivamente apresentadas são marcadas como vistas; reservas e exposição são registros distintos.

## 8. Familiarização, avaliação e repetição

Familiarização: fluxo idêntico, resultados separados e sugestão de dez sessões iniciais. Não há promessa de aprendizado paranormal em determinado prazo.

Avaliação: bloco padrão de 50 tentativas iniciadas, fixado antes da primeira sessão, apenas se houver ao menos 200 imagens elegíveis. Permitir outro tamanho antes de começar, limitado por floor(imagens elegíveis / 4). Reservas da familiarização reduzem a disponibilidade do mesmo ciclo.

Uma tentativa abandonada ocupa sua posição; não substituí-la até completar uma quantidade desejada de acertos ou escolhas. Um bloco com abandonos não é um bloco confirmatório completo. Não interromper ou prolongar a série com base no resultado observado.

Sem repetição dentro do ciclo, incluindo distratores. Ao esgotar o banco, oferecer iniciar novo ciclo com aviso de que as imagens poderão ser familiares. Preservar o histórico de exposição; não chamar o novo ciclo de imagens inéditas.

A independência dos históricos entre computadores não torna os alvos desconhecidos para o usuário. Registrar familiaridade declarada com o banco, inclusive uso no SRV anterior ou em outra instalação.

Mudanças em atributos, ajuda, regras ou pacote durante um bloco exigem encerrar o bloco como incompleto e iniciar outro. Preferências de aparência não mudam o protocolo.

## 9. Estados, persistência e encerramento

Estados persistentes: coleta, registro_bloqueado, escolha_confirmada, concluida e abandonada. A etapa I/II/III/revisão é um atributo do estado coleta. Pausa é uma condição recuperável, não um novo sorteio.

| Evento | Regra |
| --- | --- |
| Voltar entre I–III/revisão | Permitido enquanto em coleta |
| Finalizar registro | Persiste snapshot e bloqueio antes de liberar alternativas |
| Confirmar escolha | Persiste escolha uma única vez antes de liberar feedback |
| Reabrir programa ou aba | Recupera última gravação confirmada, mesmo alvo e ordem |
| Abandonar | Preserva registros e reservas; guarda fase do abandono |
| Falha de gravação | Mostra erro, mantém trabalho em memória e bloqueia avanço irreversível |
| Falha ao carregar imagem | Não troca alvo; permite corrigir disponibilidade e retomar |
| Salvar e encerrar | Aguarda gravações, fecha banco e encerra servidor |

Autosave após mudança de checkbox, fim de traço e pausa curta na digitação. Navegação deve aguardar operações pendentes. Mostrar Salvando, Salvo ou Falha ao salvar. Não prometer preservação de dados ainda pendentes em caso de queda abrupta.

Fechar a aba não encerra necessariamente o servidor. O botão Salvar e encerrar retorna confirmação e mostra Aplicativo encerrado; você pode fechar esta aba. Se a gravação falhar, não encerrar silenciosamente.

## 10. Arquitetura e dados

Frontend compilado embutido no executável Go; sem dependência de Node ou Go na máquina do usuário. Bibliotecas, fontes e recursos visuais locais, sem CDN. Builds separados para Windows e Linux; selecionar driver SQLite compatível com a estratégia de compilação, preferencialmente sem CGO.

Backend escuta somente em loopback. API valida a etapa em toda leitura e escrita. Bloqueio de coleta e confirmação de escolha devem ser transações atômicas e idempotentes. Não servir o diretório inteiro de alvos como arquivos públicos.

Rotas de imagens liberam apenas as quatro alternativas após bloqueio; feedback libera metadados somente após escolha confirmada. O frontend nunca recebe antecipadamente qual alternativa é correta. Aplicar validação de origem/host e proteção de requisições de mutação, inclusive encerramento, para que páginas externas não controlem o serviço local.

Tabelas ou entidades mínimas:

| Entidade | Conteúdo |
| --- | --- |
| Pacote e alvo | Versão, ID de origem, imagem, hash, descrição, créditos, SAM |
| Ciclo e exposição | Reservas, apresentações e familiaridade declarada |
| Bloco | Modo, tamanho previsto, protocolo, pacote, estado |
| Sessão | Código, estado, etapa, alvo secreto, alternativas, ordem, horários |
| Registro | Checkboxes por IDs estáveis, textos, AOL, traços e resumo |
| Escolha | Alternativa confirmada, confiança, resultado e instante |
| Feedback | Comentário posterior, separado do original |
| Eventos | Criação, bloqueio, exposição, confirmação, abandono e encerramento |
| Preferências | Tema, duração orientativa e localização do pacote |

SQLite e arquivos em diretório de dados próprio, separado do executável e do SRV. Armazenar tempos em UTC e apresentar no horário local. Congelar conteúdo de alvo usado, versão do protocolo e ajuda para que atualizações não reescrevam o passado.

Registros bloqueados não são editáveis pela interface. Essa proteção garante o fluxo do app; não constitui inviolabilidade contra alguém que deliberadamente altere o banco no próprio computador.

## 11. Backup, exportação e erros

PDF: código, datas, modo, protocolo, campos, desenhos, resumo e comentário; incluir identidade e fotografia do alvo somente quando já reveladas. CSV: resultados e estados por sessão, sem segredos de sessões pendentes. ZIP: cópia consistente do banco e arquivos necessários, com versão do formato.

Backup completo inclui dados internos ocultos: avisar para não inspecioná-lo durante uma sessão cega. Restaurar com a sessão encerrada e confirmação de substituição, criando cópia anterior. Não há fusão de históricos na primeira versão.

Estados de interface obrigatórios: primeiro uso; pacote ausente; importação em andamento; imagens inválidas; banco insuficiente; sessão recuperada; salvamento pendente; registro bloqueado; escolha pendente; imagem indisponível; histórico vazio; bloco incompleto; encerramento bem-sucedido ou com erro.

## 12. Critérios de aceitação

- Instalação e execução offline em Windows e Linux, com dependências e versões registradas na implementação.
- Banco importado sem alterar os arquivos originais; duplicatas exatas não entram como alternativas distintas.
- Sessão exibe apenas código neutro durante coleta, inclusive em respostas de API.
- Checkboxes, textos e desenhos são opcionais e sobrevivem à navegação e à recuperação após gravação.
- Listas de atributos e ajuda são iguais para todos os alvos da mesma versão.
- Desenho com mouse mantém coordenadas e proporção após redimensionar.
- Exemplos ficam recolhidos e não usam imagens do banco.
- Registro não pode ser alterado após bloqueio, inclusive por chamada direta à API.
- Alternativas e alvo não mudam ao atualizar, fechar ou reabrir.
- Escolha confirmada é irreversível e persistida antes do feedback.
- Exposição e abandono são preservados; não há substituição silenciosa de tentativas.
- Estatísticas separam modos e tratam corretamente blocos incompletos.
- Salvar e encerrar preserva o progresso e para o servidor.
- Backup/restauração e exportações preservam desenhos e registros sem revelar alvos pendentes nos relatórios usuais.

Testes automatizados devem se concentrar em sorteio e elegibilidade, transições e cegamento, persistência/recuperação, importação e estatística. Validação manual deve cobrir desenho com mouse, temas, janelas menores e execução real nos dois sistemas. Não considerar um sistema validado apenas porque o outro passou.

## 13. Fora da primeira versão

Sincronização, contas, múltiplos usuários, hospedagem pública, mobile, interpretação por IA, reconhecimento de ideogramas, pontuação semântica automática, ARV financeiro, estágios IV–VI, modalidades SRV Basic/Enhanced, importação de sessões antigas e sistema de cursos.

## 14. Referências do banco e continuidade

- [Repositório de origem](https://github.com/Thiagojm/SRV)
- [Formato e inventário do acervo](https://github.com/Thiagojm/SRV/blob/main/archive/README.md)
- [Importador existente](https://github.com/Thiagojm/SRV/blob/main/src-tauri/src/pack.rs)
- [Componente de desenho existente, apenas para avaliação de reaproveitamento](https://github.com/Thiagojm/SRV/blob/main/src/lib/components/DrawingPad.svelte)

O acervo é material de estudo privado, com créditos preservados. O novo executável não redistribui automaticamente essas imagens; o usuário importa seu pacote local.

Não há questão funcional bloqueante para iniciar o protótipo visual. Nome final, ícone, paleta, detalhes de espaçamento e versões exatas de dependências serão decisões da próxima etapa. Este documento permanece a referência do fluxo e das regras aprovadas.
