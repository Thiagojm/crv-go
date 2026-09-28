<script lang="ts">
 import {onMount} from 'svelte';import Icon from './Icon.svelte';import {help,stages} from '../data';
 export let step=0;export let onclose:()=>void;export let onexample:()=>void=()=>{};export let tour=false;
 let panel:HTMLElement;let first:HTMLButtonElement;
 onMount(()=>{const previous=document.activeElement as HTMLElement;first.focus();return()=>previous?.focus();});
 function key(e:KeyboardEvent){if(e.key==='Escape')onclose();if(e.key==='Tab'){const items=Array.from(panel.querySelectorAll<HTMLElement>('button,summary,a,input,select,textarea,[tabindex="0"]'));const a=items[0],b=items.at(-1);if(e.shiftKey&&document.activeElement===a){e.preventDefault();b?.focus();}else if(!e.shiftKey&&document.activeElement===b){e.preventDefault();a?.focus();}}}
</script>
<svelte:window onkeydown={key}/>
<div class="drawer-scrim" role="presentation" onclick={onclose}></div>
<div class="help-drawer" role="dialog" aria-modal="true" aria-label="Como preencher" tabindex="-1" bind:this={panel}>
 <header><span class="eyeline">GUIA DE PREENCHIMENTO</span><button bind:this={first} class="icon-button" onclick={onclose} aria-label="Fechar ajuda"><Icon name="close"/></button></header>
 {#if tour}<label>Etapa<select bind:value={step}>{#each stages as stage,i}<option value={i}>{stage}</option>{/each}</select></label>{/if}
 <h2>{stages[step]}</h2><p class="lead">{help[step].goal}</p>
 <h3>Como preencher</h3><p>{help[step].how}</p>
 <details ontoggle={(e)=>{if(e.currentTarget.open)onexample();}}><summary>Ver exemplo fictício</summary><p>{help[step].example}</p></details>
 <div class="help-note"><h3>Cuidados</h3><p>{help[step].care}</p></div>
 <p class="muted small">As sugestões não dependem do alvo. Você pode deixar qualquer campo em branco.</p>
 <button class="primary w-full" onclick={onclose}>Entendi, voltar à tela <Icon name="arrow"/></button>
</div>
