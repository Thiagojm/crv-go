<script lang="ts">
 import {onMount} from 'svelte';import Icon from './Icon.svelte';import type {Stroke,Point} from '../data';
 export let strokes:Stroke[]=[];export let onchange:(v:Stroke[])=>void=()=>{};export let readonly=false;export let compact=false;
 let canvas:HTMLCanvasElement;let tool='pen';let width=3;let active:Point[]=[];let redo:Stroke[]=[];let drawing=false;
 function paint(){if(!canvas)return;const ctx=canvas.getContext('2d')!;ctx.clearRect(0,0,1000,620);ctx.fillStyle='#fff';ctx.fillRect(0,0,1000,620);for(const stroke of [...strokes,...(active.length?[{points:active,width}]:[])]){ctx.beginPath();ctx.lineWidth=stroke.width;ctx.strokeStyle='#26342f';ctx.lineCap='round';ctx.lineJoin='round';stroke.points.forEach((p,i)=>i?ctx.lineTo(p.x,p.y):ctx.moveTo(p.x,p.y));if(stroke.points.length===1)ctx.lineTo(stroke.points[0].x+.1,stroke.points[0].y+.1);ctx.stroke();}}
 $: if(canvas&&strokes)paint();
 function point(e:PointerEvent):Point {const r=canvas.getBoundingClientRect();return{x:Math.max(0,Math.min(1000,(e.clientX-r.left)/r.width*1000)),y:Math.max(0,Math.min(620,(e.clientY-r.top)/r.height*620))};}
 function start(e:PointerEvent){if(readonly||e.button!==0)return;canvas.setPointerCapture(e.pointerId);const p=point(e);if(tool==='eraser'){const index=strokes.findLastIndex(s=>s.points.some(q=>Math.hypot(q.x-p.x,q.y-p.y)<24));if(index>=0){onchange(strokes.filter((_,i)=>i!==index));redo=[];}return;}drawing=true;active=[p];paint();}
 function move(e:PointerEvent){if(drawing){active=[...active,point(e)];paint();}}
 function finish(){if(!drawing)return;drawing=false;const next=[...strokes,{points:active,width}];active=[];redo=[];onchange(next);}
 function undo(){if(strokes.length){redo=[...redo,strokes.at(-1)!];onchange(strokes.slice(0,-1));}}
 function forward(){if(redo.length){onchange([...strokes,redo.at(-1)!]);redo=redo.slice(0,-1);}}
 onMount(paint);
</script>
<div class="drawing-panel" class:compact>
 {#if !readonly}<div class="drawing-toolbar">
 <button class:tool-active={tool==='pen'} onclick={()=>tool='pen'} title="Caneta"><Icon name="pen"/> <span>Caneta</span></button>
 <button class:tool-active={tool==='eraser'} onclick={()=>tool='eraser'} title="Borracha: remove um traço inteiro"><Icon name="eraser"/><span>Borracha</span></button>
 <span class="divider"></span><button onclick={undo} disabled={!strokes.length} aria-label="Desfazer"><Icon name="undo"/></button><button onclick={forward} disabled={!redo.length} aria-label="Refazer"><Icon name="redo"/></button>
 <label class="width-control" title="Espessura"><span>Traço</span><select aria-label="Espessura do traço" bind:value={width}><option value={2}>Fino</option><option value={3}>Médio</option><option value={6}>Grosso</option></select></label>
 <button class="clear-tool" onclick={()=>{if(strokes.length&&confirm('Limpar este desenho?')){redo=[];onchange([]);}}} aria-label="Limpar desenho"><Icon name="trash"/></button>
 </div>{/if}
 <div class="canvas-wrap"><canvas bind:this={canvas} width="1000" height="620" onpointerdown={start} onpointermove={move} onpointerup={finish} onpointercancel={finish} aria-label={readonly?'Desenho registrado':'Área de desenho livre com mouse'} class:eraser={tool==='eraser'}></canvas>{#if !strokes.length&&!active.length}<span class="canvas-label">{readonly?'SEM DESENHO REGISTRADO':'DESENHO LIVRE'}</span>{/if}</div>
 {#if !readonly}<p class="canvas-caption">Não precisa representar um objeto. <span>{strokes.length} {strokes.length===1?'traço':'traços'}</span></p>{/if}
</div>
