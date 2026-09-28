import image0 from './assets/demo-0.jpg';
import image1 from './assets/demo-1.jpg';
import image2 from './assets/demo-2.jpg';
import image3 from './assets/demo-3.jpg';
export type Group={key:string;title:string;options:string[];collapsed?:boolean};
export const groupsI:Group[]=[
 {key:'movement',title:'Movimento / forma',options:['Horizontal','Vertical','Diagonal','Curvo','Ondulado','Angular','Circular','Sobe','Desce']},
 {key:'feel',title:'Sensação / consistência',options:['Sólido','Duro','Macio','Fluido','Gasoso','Liso','Áspero','Granulado']},
 {key:'gestalt',title:'Categoria geral',options:['Água','Terreno / relevo','Estrutura construída','Vegetação','Ser vivo','Indefinido']}];
export const groupsII:Group[]=[
 {key:'colors',title:'Cores',options:['Branco','Preto','Cinza','Azul','Verde','Vermelho','Amarelo','Marrom','Laranja','Violeta']},
 {key:'light',title:'Luminosidade',options:['Claro','Escuro','Brilhante','Fosco','Reflexivo','Translúcido']},
 {key:'texture',title:'Textura',options:['Liso','Áspero','Granulado','Fibroso','Pegajoso','Escorregadio']},
 {key:'temp',title:'Temperatura',options:['Frio','Fresco','Morno','Quente']},
 {key:'wet',title:'Umidade',options:['Seco','Úmido','Molhado']},
 {key:'sound',title:'Sons',options:['Silencioso','Contínuo','Intermitente','Grave','Agudo','Rítmico','Ruído de água','Mecânico','Vozes'],collapsed:true},
 {key:'smell',title:'Cheiros',options:['Terroso','Vegetal','Salino','Floral','Químico','Queimado'],collapsed:true},
 {key:'taste',title:'Sabores',options:['Doce','Salgado','Amargo','Ácido','Metálico'],collapsed:true}];
export const stages=['Preparação','Ideograma','Sensorial','Esboço','Revisão','Escolha','Feedback'];
export const intro=[
 'Reserve alguns minutos para registrar suas impressões.',
 'Um gesto breve. Depois, registre suas primeiras impressões.',
 'Descreva qualidades. Não é preciso identificar o alvo.',
 'Dê espaço às formas e às relações entre elas.',
 'Confira seu registro antes de conhecer as alternativas.',
 'Compare o registro completo e escolha uma imagem.',
 'Observe o resultado. Preserve o registro original.'
];
export const help=[
 {goal:'Preparar uma sessão com um alvo oculto.',how:'Encontre um ambiente confortável. O código identifica a sessão, sem indicar o conteúdo do alvo. O tempo é apenas uma referência.',example:'Você pode deixar disposição e concentração em branco e iniciar quando estiver pronto.',care:'Esta demonstração usa quatro fotografias repetidas. Não é um teste experimental.'},
 {goal:'Registrar um gesto breve e uma impressão geral.',how:'Desenhe com o mouse. Em Registrar impressões, anote movimento, sensação e uma categoria, se surgir. Checkboxes e textos são opcionais.',example:'Movimento: curvo. Sensação: fluido. Categoria: água? É um exemplo de preenchimento, não uma regra de interpretação.',care:'Um traço ondulado não significa obrigatoriamente água. Se não surgir uma interpretação, deixe indefinido.'},
 {goal:'Registrar características sensoriais simples.',how:'Marque atributos, escreva livremente ou combine os dois. Os campos vazios significam não registrado, não ausente.',example:'Frio, áspero e cinza são descritores. Parece um castelo é uma identificação: registre em Hipóteses / AOL.',care:'Não preencha para completar a lista. As sugestões são sempre iguais, independentemente do alvo.'},
 {goal:'Representar formas, dimensões e posições.',how:'Use linhas simples. Anote tamanho relativo e localização dos elementos. A borracha remove um traço inteiro.',example:'Um retângulo alto à esquerda de uma área ampla comunica uma relação espacial sem identificar o objeto.',care:'Não precisa saber desenhar. O esboço é separado do ideograma inicial.'},
 {goal:'Conferir e finalizar o registro original.',how:'Revise também as hipóteses e contradições. Você pode voltar aos estágios anteriores enquanto o registro está aberto.',example:'Você pode destacar até cinco características, deixando as restantes em branco.',care:'Depois de mostrar as alternativas, a edição original será bloqueada. Comentários posteriores ficam separados.'},
 {goal:'Escolher a imagem mais compatível com seu registro.',how:'Leia o registro completo e examine as quatro alternativas. A seleção pode mudar até clicar em Confirmar escolha.',example:'Se uma imagem combina com uma cor, mas contradiz seu desenho, considere as duas informações.',care:'Não reinterprete o registro para ajustá-lo à imagem. Sua escolha ficará definitiva após a confirmação.'},
 {goal:'Comparar e registrar uma reflexão posterior.',how:'Confira a escolha e o alvo correto. Escreva o que observou no comentário pós-feedback, sem mudar o original.',example:'Anote correspondências e divergências específicas, preservando o que escreveu antes.',care:'Um acerto isolado não demonstra desempenho acima do acaso. Este protótipo não gera evidência experimental.'}
];
/** Prototype-only demo photos. The real bank is served by Go from farsight/; never import that folder here. */
export const demoTargets=[
 {image:image0,name:'Paisagem de montanha',description:'Fotografia de paisagem com relevo, água e construções. Imagem demonstrativa; não pertence ao banco SRV.',credit:'Unsplash · photo-1470770841072-f978cf4d019e'},
 {image:image1,name:'Oceano',description:'Fotografia marítima com ondas e horizonte. Imagem demonstrativa; não pertence ao banco SRV.',credit:'Unsplash · photo-1518837695005-2083093ee35b'},
 {image:image2,name:'Floresta',description:'Fotografia de vegetação e árvores. Imagem demonstrativa; não pertence ao banco SRV.',credit:'Unsplash · photo-1441974231531-c6227db76b6e'},
 {image:image3,name:'Arquitetura',description:'Fotografia de uma estrutura arquitetônica. Imagem demonstrativa; não pertence ao banco SRV.',credit:'Unsplash · photo-1511818966892-d7d671e672a2'}
];
export const phase1SessionMessage='Sessões reais com cegamento no servidor abrem na Fase 2.';
export type Point={x:number;y:number};
export type Stroke={points:Point[];width:number};
export type Session={id:string;code:string;step:number;status:'active'|'locked'|'done'|'abandoned';mode:string;created:string;seconds:number;choiceSeconds:number;record:Record<string,string[]>;notes:Record<string,string>;drawings:Record<string,Stroke[]>;summary:string[];confidence:string;choiceConfidence:string;selected:number|null;target:number;order:number[];attributes:boolean;comment:string;openedExamples:number[]};
export const duration=(s:number)=>`${Math.floor(s/60).toString().padStart(2,'0')}:${(s%60).toString().padStart(2,'0')}`;
export function newSession(mode:string):Session {const a=new Uint32Array(4);crypto.getRandomValues(a);const order=[0,1,2,3];for(let i=3;i>0;i--){const j=a[i]%(i+1);[order[i],order[j]]=[order[j],order[i]];}return {id:crypto.randomUUID(),code:`${1000+a[1]%9000}–${1000+a[2]%9000}`,step:1,status:'active',mode,created:new Date().toISOString(),seconds:0,choiceSeconds:0,record:{},notes:{},drawings:{ideogram:[],sketch:[]},summary:['','','','',''],confidence:'',choiceConfidence:'',selected:null,target:a[0]%4,order,attributes:false,comment:'',openedExamples:[]};}
