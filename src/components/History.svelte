<script lang="ts">
 import Icon from './Icon.svelte';
 import RecordView from './Record.svelte';
 import {durationMs, emptyRecord, type RecordData} from '../data';
 import {
  fetchHistory, getSession, saveComment, downloadHistoryCSV, openSessionPrint,
  type HistoryItem, type SessionDTO
 } from '../api';

 export let active = false;

 const pageSize = 50;
 let items: HistoryItem[] = [];
 let total = 0;
 let offset = 0;
 let loading = false;
 let error = '';
 let stateFilter = '';
 let fromDate = '';
 let toDate = '';
 let selected: SessionDTO | null = null;
 let detailRecord: RecordData = emptyRecord();
 let commentDraft = '';
 let commentBusy = false;
 let commentNotice = '';
 let exportBusy = false;

 const stateLabel: {[k: string]: string} = {
  collecting: 'Em coleta',
  locked: 'Aguardando escolha',
  completed: 'Concluída',
  abandoned: 'Abandonada',
 };

 function localDayBounds(ymd: string, endExclusive: boolean): string {
  const [y, m, d] = ymd.split('-').map(Number);
  const local = new Date(y, m - 1, d + (endExclusive ? 1 : 0), 0, 0, 0, 0);
  return local.toISOString();
 }

 async function load(resetOffset = false) {
  if (!active) return;
  if (resetOffset) offset = 0;
  loading = true;
  error = '';
  try {
   const states = stateFilter ? [stateFilter] : [];
   const from = fromDate ? localDayBounds(fromDate, false) : undefined;
   const to = toDate ? localDayBounds(toDate, true) : undefined;
   const page = await fetchHistory({states, from, to, limit: pageSize, offset});
   items = page.items;
   total = page.total;
   if (offset > 0 && items.length === 0 && total > 0) {
    offset = Math.max(0, Math.floor((total - 1) / pageSize) * pageSize);
    const again = await fetchHistory({states, from, to, limit: pageSize, offset});
    items = again.items;
    total = again.total;
   }
  } catch (e) {
   error = e instanceof Error ? e.message : 'Falha ao carregar histórico';
   items = [];
   total = 0;
  } finally {
   loading = false;
  }
 }

 async function openItem(id: string) {
  error = '';
  commentNotice = '';
  try {
   selected = await getSession(id);
   commentDraft = selected.comment || '';
   const r = {...emptyRecord(), ...(selected.record || {})};
   if (!r.drawings) r.drawings = {ideogram: [], sketch: []};
   if (!r.groups) r.groups = {};
   if (!Array.isArray(r.summary) || r.summary.length < 5) r.summary = [...(r.summary || []), '', '', '', '', ''].slice(0, 5);
   detailRecord = r;
  } catch (e) {
   error = e instanceof Error ? e.message : 'Falha ao abrir sessão';
   selected = null;
  }
 }

 async function onSaveComment() {
  if (!selected || selected.state !== 'completed' || commentBusy) return;
  commentBusy = true;
  commentNotice = '';
  try {
   selected = await saveComment(selected.id, commentDraft);
   commentDraft = selected.comment || '';
   commentNotice = 'Comentário salvo.';
  } catch (e) {
   commentNotice = e instanceof Error ? e.message : 'Falha ao salvar comentário';
  } finally {
   commentBusy = false;
  }
 }

 function resultText(it: HistoryItem): string {
  if (it.state === 'completed') {
   if (it.hit === true) return `Acerto · ${it.confirmedChoice || ''}`;
   if (it.hit === false) return `Erro · ${it.confirmedChoice || ''}`;
   return it.confirmedChoice || 'Concluída';
  }
  if (it.state === 'abandoned') {
   return it.abandonedAfterAlts ? 'Abandonada após alternativas' : 'Abandonada antes das alternativas';
  }
  return stateLabel[it.state] || it.state;
 }

 function formatWhen(iso: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString('pt-BR', {dateStyle: 'short', timeStyle: 'short'});
 }

 function pageLabel(): string {
  if (total === 0) return '0 de 0';
  const from = offset + 1;
  const to = Math.min(offset + items.length, total);
  return `${from}–${to} de ${total}`;
 }

 function prevPage() {
  if (offset <= 0 || loading) return;
  offset = Math.max(0, offset - pageSize);
  void load();
 }

 function nextPage() {
  if (offset + pageSize >= total || loading) return;
  offset += pageSize;
  void load();
 }

 async function onExportCSV() {
  if (exportBusy) return;
  exportBusy = true;
  error = '';
  try {
   const states = stateFilter ? [stateFilter] : [];
   const from = fromDate ? localDayBounds(fromDate, false) : undefined;
   const to = toDate ? localDayBounds(toDate, true) : undefined;
   await downloadHistoryCSV({states, from, to});
  } catch (e) {
   error = e instanceof Error ? e.message : 'Falha ao exportar CSV';
  } finally {
   exportBusy = false;
  }
 }

 $: if (active) void load(true);
</script>

{#if selected}
 <div class="page-title">
  <div>
   <h1>{selected.code}</h1>
   <p>{stateLabel[selected.state] || selected.state} · {formatWhen(selected.createdAt)}</p>
  </div>
  <div class="settings-actions">
   <button class="secondary" onclick={() => openSessionPrint(selected!.id)} title="Abre a vista de impressão; use Salvar como PDF no diálogo do navegador">Exportar PDF</button>
   <button class="secondary" onclick={() => { selected = null; }}><Icon name="back"/>Voltar à lista</button>
  </div>
 </div>
 {#if selected.state === 'completed' && selected.feedback}
  <section class="surface feedback-photo" style="margin-bottom:20px">
   <div class="result-line">
    <h2>{selected.hit ? 'Acerto' : 'Erro'} · posição {selected.correct || '—'}</h2>
    <p class="muted">{selected.feedback.label}{#if selected.feedback.credit} · {selected.feedback.credit}{/if}</p>
    <p class="small muted">{selected.feedback.description}</p>
   </div>
   {#if selected.alternatives}
    <div class="target-grid" style="margin-top:16px">
     {#each selected.alternatives as alt}
      <div class="target-label" class:chosen={selected.choice === alt.position} class:correct={selected.correct === alt.position}>
       <img src={alt.url} alt={`Alternativa ${alt.position}`}/>
       <span>{alt.position}</span>
      </div>
     {/each}
    </div>
   {/if}
   <div class="history-comment mt">
    <label>Comentário pós-feedback
     <textarea rows="5" bind:value={commentDraft} placeholder="Esta reflexão fica separada do registro original."></textarea>
    </label>
    <div class="settings-actions" style="margin-top:12px">
     <button class="secondary" disabled={commentBusy} onclick={() => void onSaveComment()}>Salvar comentário</button>
     {#if commentNotice}<span class="small muted">{commentNotice}</span>{/if}
    </div>
   </div>
  </section>
 {:else if selected.state === 'abandoned'}
  <section class="inline-note" role="status">
   Sessão abandonada{#if selected.abandonedAfterAlts} após ver as alternativas{:else} antes das alternativas{/if}. O alvo permanece oculto.
  </section>
 {/if}
 <section class="surface review-record"><RecordView record={detailRecord}/></section>
{:else}
 <div class="page-title">
  <div>
   <h1>Histórico</h1>
   <p>Sessões reais, da mais recente para a mais antiga.</p>
  </div>
  <button class="secondary" disabled={exportBusy || loading} onclick={() => void onExportCSV()}>Exportar CSV</button>
 </div>
 <div class="section-heading">
  <h2>{total} {total === 1 ? 'sessão' : 'sessões'}</h2>
  <div class="settings-actions">
   <label class="filter-label">Estado
    <select bind:value={stateFilter} onchange={() => void load(true)}>
     <option value="">Todos</option>
     <option value="completed">Concluídas</option>
     <option value="abandoned">Abandonadas</option>
     <option value="collecting">Em coleta</option>
     <option value="locked">Aguardando escolha</option>
    </select>
   </label>
   <label class="filter-label">De
    <input type="date" bind:value={fromDate} onchange={() => void load(true)}/>
   </label>
   <label class="filter-label">Até
    <input type="date" bind:value={toDate} onchange={() => void load(true)}/>
   </label>
  </div>
 </div>
 {#if error}<p class="inline-note" role="status">{error}</p>{/if}
 {#if loading}
  <p class="muted">Carregando…</p>
 {:else if items.length === 0}
  <section class="empty-state"><Icon name="book" size={40}/><h2>Nenhuma sessão neste filtro.</h2><p>Conclua ou abandone uma sessão para vê-la aqui.</p></section>
 {:else}
  <div class="table-wrap">
   <table>
    <thead><tr><th>Quando</th><th>Código</th><th>Estado / resultado</th><th>Tempo</th></tr></thead>
    <tbody>
     {#each items as it}
      <tr>
       <td><button class="text-button" onclick={() => void openItem(it.id)}>{formatWhen(it.createdAt)}</button></td>
       <td><button class="text-button" onclick={() => void openItem(it.id)}>{it.code}</button></td>
       <td>{resultText(it)}</td>
       <td>{durationMs((it.collectionMs || 0) + (it.choiceMs || 0))}</td>
      </tr>
     {/each}
    </tbody>
   </table>
  </div>
  {#if total > pageSize}
   <div class="history-pager settings-actions" style="margin-top:16px;justify-content:flex-end">
    <span class="small muted">{pageLabel()}</span>
    <button class="secondary" disabled={offset <= 0 || loading} onclick={prevPage}>Anterior</button>
    <button class="secondary" disabled={offset + pageSize >= total || loading} onclick={nextPage}>Próxima</button>
   </div>
  {/if}
 {/if}
{/if}
