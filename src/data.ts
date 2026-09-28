export type Option = {id: string; label: string};
export type Group = {key: string; title: string; options: Option[]; collapsed?: boolean};
export type Point = {x: number; y: number};
export type Stroke = {points: Point[]; width: number};
export type GroupValue = {ids: string[]; note: string};
export type RecordData = {
  version: string;
  disposition: string;
  concentration: string;
  attributesOpen: boolean;
  groups: Record<string, GroupValue>;
  aol1: string;
  sensory: string;
  aol2: string;
  forms: string;
  dimensions: string;
  positions: string;
  spatial: string;
  aol3: string;
  drawings: {ideogram: Stroke[]; sketch: Stroke[]};
  summary: string[];
  confidence: number | null;
  step: number;
};

function g(key: string, title: string, pairs: string[], collapsed = false): Group {
  const options: Option[] = [];
  for (let i = 0; i < pairs.length; i += 2) options.push({id: pairs[i], label: pairs[i + 1]});
  return {key, title, options, collapsed: collapsed || undefined};
}

export const groupsI: Group[] = [
  g('movement', 'Movimento / forma', ['horizontal','Horizontal','vertical','Vertical','diagonal','Diagonal','curved','Curvo','wavy','Ondulado','angular','Angular','circular','Circular','up','Sobe','down','Desce']),
  g('feel', 'Sensação / consistência', ['solid','Sólido','hard','Duro','soft','Macio','fluid','Fluido','gaseous','Gasoso','smooth','Liso','rough','Áspero','granular','Granulado']),
  g('gestalt', 'Categoria geral', ['water','Água','terrain','Terreno / relevo','structure','Estrutura construída','vegetation','Vegetação','living','Ser vivo','undefined','Indefinido']),
];
export const groupsII: Group[] = [
  g('colors', 'Cores', ['white','Branco','black','Preto','gray','Cinza','blue','Azul','green','Verde','red','Vermelho','yellow','Amarelo','brown','Marrom','orange','Laranja','violet','Violeta']),
  g('light', 'Luminosidade', ['light','Claro','dark','Escuro','bright','Brilhante','matte','Fosco','reflective','Reflexivo','translucent','Translúcido']),
  g('texture', 'Textura', ['smooth','Liso','rough','Áspero','granular','Granulado','fibrous','Fibroso','sticky','Pegajoso','slippery','Escorregadio']),
  g('temp', 'Temperatura', ['cold','Frio','cool','Fresco','warm','Morno','hot','Quente']),
  g('wet', 'Umidade', ['dry','Seco','damp','Úmido','wet','Molhado']),
  g('sound', 'Sons', ['silent','Silencioso','continuous','Contínuo','intermittent','Intermitente','low','Grave','high','Agudo','rhythmic','Rítmico','water','Ruído de água','mechanical','Mecânico','voices','Vozes'], true),
  g('smell', 'Cheiros', ['earthy','Terroso','vegetal','Vegetal','saline','Salino','floral','Floral','chemical','Químico','burnt','Queimado'], true),
  g('taste', 'Sabores', ['sweet','Doce','salty','Salgado','bitter','Amargo','sour','Ácido','metallic','Metálico'], true),
];
export const allGroups = [...groupsI, ...groupsII];
export const stages = ['Preparação','Ideograma','Sensorial','Esboço','Revisão','Escolha','Feedback'];
export const intro = [
  'Reserve alguns minutos para registrar suas impressões.',
  'Um gesto breve. Depois, registre suas primeiras impressões.',
  'Descreva qualidades. Não é preciso identificar o alvo.',
  'Dê espaço às formas e às relações entre elas.',
  'Confira seu registro antes de conhecer as alternativas.',
  'Compare o registro completo e escolha uma imagem.',
  'Observe o resultado. Preserve o registro original.',
];
export const help = [
  {goal:'Preparar uma sessão com um alvo oculto.',how:'Encontre um ambiente confortável. O código identifica a sessão, sem indicar o conteúdo do alvo. O tempo é apenas uma referência.',example:'Você pode deixar disposição e concentração em branco e iniciar quando estiver pronto.',care:'O cronômetro orienta; não encerra a sessão.'},
  {goal:'Registrar um gesto breve e uma impressão geral.',how:'Desenhe com o mouse. Em Registrar impressões, anote movimento, sensação e uma categoria, se surgir. Checkboxes e textos são opcionais.',example:'Movimento: curvo. Sensação: fluido. Categoria: água? É um exemplo de preenchimento, não uma regra de interpretação.',care:'Um traço ondulado não significa obrigatoriamente água. Se não surgir uma interpretação, deixe indefinido.'},
  {goal:'Registrar características sensoriais simples.',how:'Marque atributos, escreva livremente ou combine os dois. Os campos vazios significam não registrado, não ausente.',example:'Frio, áspero e cinza são descritores. Parece um castelo é uma identificação: registre em Hipóteses / AOL.',care:'Não preencha para completar a lista. As sugestões são sempre iguais, independentemente do alvo.'},
  {goal:'Representar formas, dimensões e posições.',how:'Use linhas simples. Anote tamanho relativo e localização dos elementos. A borracha remove um traço inteiro.',example:'Um retângulo alto à esquerda de uma área ampla comunica uma relação espacial sem identificar o objeto.',care:'Não precisa saber desenhar. O esboço é separado do ideograma inicial.'},
  {goal:'Conferir e finalizar o registro original.',how:'Revise também as hipóteses e contradições. Você pode voltar aos estágios anteriores enquanto o registro está aberto.',example:'Você pode destacar até cinco características, deixando as restantes em branco.',care:'Depois de mostrar as alternativas, a edição original será bloqueada. Comentários posteriores ficam separados.'},
  {goal:'Escolher a imagem mais compatível com seu registro.',how:'Leia o registro completo e examine as quatro alternativas. A seleção pode mudar até clicar em Confirmar escolha.',example:'Se uma imagem combina com uma cor, mas contradiz seu desenho, considere as duas informações.',care:'Não reinterprete o registro para ajustá-lo à imagem. Sua escolha ficará definitiva após a confirmação.'},
  {goal:'Comparar e registrar uma reflexão posterior.',how:'Confira a escolha e o alvo correto. Escreva o que observou no comentário pós-feedback, sem mudar o original.',example:'Anote correspondências e divergências específicas, preservando o que escreveu antes.',care:'Um acerto isolado não demonstra desempenho acima do acaso.'},
];
export function emptyRecord(): RecordData {
  return {version:'crv-record-v1',disposition:'',concentration:'',attributesOpen:false,groups:{},aol1:'',sensory:'',aol2:'',forms:'',dimensions:'',positions:'',spatial:'',aol3:'',drawings:{ideogram:[],sketch:[]},summary:['','','','',''],confidence:null,step:1};
}
export function labelsFor(group: Group, ids: string[] = []): string {
  const map = new Map(group.options.map(o => [o.id, o.label]));
  return ids.map(id => map.get(id) || id).join(', ');
}
export const duration = (s: number) => `${Math.floor(s / 60).toString().padStart(2, '0')}:${(s % 60).toString().padStart(2, '0')}`;
export const durationMs = (ms: number) => duration(Math.max(0, Math.floor(ms / 1000)));
