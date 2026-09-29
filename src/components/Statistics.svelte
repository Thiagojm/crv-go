<script lang="ts">
 import Icon from './Icon.svelte';
 import {fetchStatistics, type Statistics} from '../api';

 export let active = false;

 let stats: Statistics | null = null;
 let loading = false;
 let error = '';

 async function load() {
  if (!active) return;
  loading = true;
  error = '';
  try {
   stats = await fetchStatistics();
  } catch (e) {
   error = e instanceof Error ? e.message : 'Falha ao carregar estatísticas';
   stats = null;
  } finally {
   loading = false;
  }
 }

 function pct(rate: number | null | undefined): string {
  if (rate == null || Number.isNaN(rate)) return '—';
  return `${(rate * 100).toFixed(rate * 100 % 1 === 0 ? 0 : 1)}%`;
 }

 function hitFrac(s: Statistics): string {
  if (!s.confirmedChoices) return '—';
  return `${s.hits}/${s.confirmedChoices}`;
 }

 /** SVG polyline points for cumulative hits and 25% reference. */
 function series(points: Statistics['chart'], key: 'hits' | 'reference'): string {
  if (!points.length) return '';
  const w = 640, h = 220, pad = 28;
  const maxX = Math.max(points.length, 1);
  const maxY = Math.max(
   ...points.map(p => Math.max(p.hits, p.reference)),
   1,
  );
  return points.map((p, i) => {
   const x = pad + ((i + 1) / maxX) * (w - pad * 2);
   const yVal = key === 'hits' ? p.hits : p.reference;
   const y = h - pad - (yVal / maxY) * (h - pad * 2);
   return `${x},${y}`;
  }).join(' ');
 }

 $: if (active) void load();
</script>

<div class="page-title">
 <div>
  <h1>Estatísticas</h1>
  <p>Totais acumulados de toda a instalação. Filtros do histórico não alteram estes números.</p>
 </div>
</div>

{#if error}<p class="inline-note" role="status">{error}</p>{/if}
{#if loading}
 <p class="muted">Carregando…</p>
{:else if !stats || stats.initiated === 0}
 <section class="empty-state"><Icon name="chart" size={40}/><h2>Sem totais ainda.</h2><p>A taxa de acertos usa apenas escolhas confirmadas.</p></section>
{:else}
 <div class="stat-strip" role="group" aria-label="Totais">
  <div><span>Iniciadas</span><strong>{stats.initiated}</strong></div>
  <div><span>Ativas</span><strong>{stats.active}</strong></div>
  <div><span>Concluídas</span><strong>{stats.completed}</strong></div>
  <div><span>Abandonadas</span><strong>{stats.abandonedBefore + stats.abandonedAfter}</strong>
   <p class="small muted">antes {stats.abandonedBefore} · depois {stats.abandonedAfter}</p></div>
 </div>

 <section class="surface stats-panel">
  <h2>Acertos confirmados</h2>
  <div class="bar-row"><span>Escolhas</span><div class="bar-track"><div style={`width:${stats.confirmedChoices ? 100 : 0}%`}></div></div><strong>{stats.confirmedChoices}</strong></div>
  <div class="bar-row"><span>Acertos</span><div class="bar-track"><div style={`width:${stats.confirmedChoices ? (stats.hits / stats.confirmedChoices) * 100 : 0}%`}></div></div><strong>{stats.hits}</strong></div>
  <div class="bar-row"><span>Referência 25%</span><div class="bar-track"><div class="chance" style="width:25%"></div></div><strong>¼</strong></div>
  <p>Taxa de acertos: <strong>{hitFrac(stats)}</strong> ({pct(stats.hitRate)}). Abandonos e sessões ativas não entram no denominador.</p>
 </section>

 <section class="surface stats-panel mt">
  <h2>Acumulado ao longo das conclusões</h2>
  {#if stats.chart.length === 0}
   <p class="muted">Ainda não há sessões concluídas para o gráfico.</p>
  {:else}
   <svg class="chart-svg" viewBox="0 0 640 220" role="img" aria-label="Acertos acumulados versus referência de 25%">
    <polyline fill="none" stroke="var(--subtle)" stroke-width="2" stroke-dasharray="6 4" points={series(stats.chart, 'reference')}/>
    <polyline fill="none" stroke="var(--accent)" stroke-width="3" points={series(stats.chart, 'hits')}/>
   </svg>
   <div class="table-wrap" style="margin-top:18px">
    <table>
     <thead><tr><th>#</th><th>Código</th><th>Resultado</th><th>Acertos acum.</th><th>Ref. 25%</th></tr></thead>
     <tbody>
      {#each stats.chart as p}
       <tr>
        <td>{p.n}</td>
        <td>{p.code}</td>
        <td>{p.hit ? 'Acerto' : 'Erro'}</td>
        <td>{p.hits}</td>
        <td>{p.reference.toFixed(2)}</td>
       </tr>
      {/each}
     </tbody>
    </table>
   </div>
  {/if}
 </section>
{/if}
