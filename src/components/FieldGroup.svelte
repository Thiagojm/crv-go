<script lang="ts">
 import type {Group} from '../data';
 export let group:Group;export let values:string[]=[];export let note='';export let onchange:(v:string[],n:string)=>void;
 function change(v:string){onchange(values.includes(v)?values.filter(x=>x!==v):[...values,v],note);}
</script>
<div class="field-group">
 {#if group.collapsed}
 <details><summary>{group.title}<span class="muted">{values.length||note ? 'Preenchido' : 'Opcional'}</span></summary><div class="details-body">{@render controls()}</div></details>
 {:else}<h3>{group.title}</h3>{@render controls()}{/if}
</div>
{#snippet controls()}
 <div class="choices">{#each group.options as opt}<label class:checked={values.includes(opt)}><input type="checkbox" checked={values.includes(opt)} onchange={()=>change(opt)}/><span>{opt}</span></label>{/each}</div>
 <label class="free-label">Outra impressão / descrição livre<textarea aria-label={`${group.title} — escrita livre`} rows="2" value={note} oninput={(e)=>onchange(values,e.currentTarget.value)} placeholder="Escreva livremente, se quiser."></textarea></label>
{/snippet}
