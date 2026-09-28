<script lang="ts">
 import DrawingPad from './DrawingPad.svelte';import {groupsI,groupsII,type Session} from '../data';
 export let session:Session;export let compact=false;
 const extras=[['aol1','Hipóteses / AOL — I'],['sensory','Outras impressões sensoriais'],['aol2','Hipóteses / AOL — II'],['forms','Formas'],['dimensions','Dimensões / proporções'],['positions','Posições e relações espaciais'],['spatial','Outras anotações'],['aol3','Hipóteses / AOL — III']];
</script>
<div class="record-view">
 <h3>I · Ideograma</h3><DrawingPad strokes={session.drawings.ideogram} readonly {compact}/>
 {#each groupsI as g}<div class="record-line"><strong>{g.title}</strong><p>{session.record[g.key]?.join(', ')||'Não registrado'}</p>{#if session.notes[g.key]}<p>{session.notes[g.key]}</p>{/if}</div>{/each}
 <h3>II · Sensorial</h3>{#each groupsII as g}<div class="record-line"><strong>{g.title}</strong><p>{session.record[g.key]?.join(', ')||'Não registrado'}</p>{#if session.notes[g.key]}<p>{session.notes[g.key]}</p>{/if}</div>{/each}
 <h3>III · Esboço</h3><DrawingPad strokes={session.drawings.sketch} readonly {compact}/>
 {#each extras as [key,title]}<div class="record-line"><strong>{title}</strong><p>{session.notes[key]||'Não registrado'}</p></div>{/each}
 {#if session.summary.some(Boolean)}<h3>Características principais</h3><ul>{#each session.summary.filter(Boolean) as v}<li>{v}</li>{/each}</ul>{/if}
 <p class="muted">Confiança do registro: {session.confidence?session.confidence+'%':'não informada'}</p>
</div>
