<script lang="ts">
 import {onMount} from 'svelte';
 import Icon from './Icon.svelte';
 export let title = '';
 export let message = '';
 export let confirmLabel = 'Confirmar';
 export let icon = 'lock';
 export let onconfirm: () => void;
 export let oncancel: () => void;
 let panel: HTMLElement;
 let first: HTMLButtonElement;
 onMount(() => {
  const previous = document.activeElement as HTMLElement;
  first.focus();
  return () => previous?.focus();
 });
 function key(e: KeyboardEvent) {
  if (e.key === 'Escape') oncancel();
  if (e.key === 'Tab') {
   const items = Array.from(panel.querySelectorAll<HTMLElement>('button'));
   const a = items[0], b = items.at(-1);
   if (e.shiftKey && document.activeElement === a) { e.preventDefault(); b?.focus(); }
   else if (!e.shiftKey && document.activeElement === b) { e.preventDefault(); a?.focus(); }
  }
 }
</script>
<svelte:window onkeydown={key}/>
<div class="modal-backdrop">
 <div class="modal" role="dialog" aria-modal="true" aria-labelledby="confirm-title" bind:this={panel}>
  <Icon name={icon} size={28}/>
  <h2 id="confirm-title">{title}</h2>
  <p>{message}</p>
  <div class="modal-actions">
   <button bind:this={first} class="secondary" onclick={oncancel}>Voltar</button>
   <button class="primary" onclick={onconfirm}>{confirmLabel}</button>
  </div>
 </div>
</div>
