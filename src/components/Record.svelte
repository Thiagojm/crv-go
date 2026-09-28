<script lang="ts">
 import DrawingPad from './DrawingPad.svelte';
 import {groupsI,groupsII,labelsFor,type RecordData} from '../data';
 export let record:RecordData;export let compact=false;
 const extras: [keyof RecordData, string][] = [['aol1','Hipóteses / AOL — I'],['sensory','Outras impressões sensoriais'],['aol2','Hipóteses / AOL — II'],['forms','Formas'],['dimensions','Dimensões / proporções'],['positions','Posições e relações espaciais'],['spatial','Outras anotações'],['aol3','Hipóteses / AOL — III']];
</script>
<div class="record-view">
 {#if record.disposition || record.concentration}
 <div class="record-line"><strong>Preparação</strong><p>{[record.disposition, record.concentration].filter(Boolean).join(' · ') || 'Não registrado'}</p></div>
 {/if}
 <h3>I · Ideograma</h3><DrawingPad strokes={record.drawings.ideogram} readonly {compact}/>
 {#each groupsI as g}<div class="record-line"><strong>{g.title}</strong><p>{labelsFor(g, record.groups[g.key]?.ids)||'Não registrado'}</p>{#if record.groups[g.key]?.note}<p>{record.groups[g.key].note}</p>{/if}</div>{/each}
 <h3>II · Sensorial</h3>{#each groupsII as g}<div class="record-line"><strong>{g.title}</strong><p>{labelsFor(g, record.groups[g.key]?.ids)||'Não registrado'}</p>{#if record.groups[g.key]?.note}<p>{record.groups[g.key].note}</p>{/if}</div>{/each}
 <h3>III · Esboço</h3><DrawingPad strokes={record.drawings.sketch} readonly {compact}/>
 {#each extras as [key,title]}<div class="record-line"><strong>{title}</strong><p>{(record[key] as string)||'Não registrado'}</p></div>{/each}
 {#if record.summary.some(Boolean)}<h3>Características principais</h3><ul>{#each record.summary.filter(Boolean) as v}<li>{v}</li>{/each}</ul>{/if}
 <p class="muted">Confiança do registro: {record.confidence!=null?record.confidence+'%':'não informada'}</p>
</div>
