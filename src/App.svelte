<script lang="ts">
 import {onMount} from 'svelte';
 import Icon from './components/Icon.svelte';
 import DrawingPad from './components/DrawingPad.svelte';
 import FieldGroup from './components/FieldGroup.svelte';
 import Record from './components/Record.svelte';
 import Help from './components/Help.svelte';
 import ConfirmDialog from './components/ConfirmDialog.svelte';
 import {groupsI,groupsII,stages,intro,durationMs,emptyRecord,type RecordData,type Stroke} from './data';
 import {SaveQueue,saveLabel,createOpId,clearCreateOp} from './session';
 import {
  bootstrapFromHash,getReady,getCatalogSummary,shutdown,restoreLease,
  createSession,getCurrent,saveRecord,lockSession,tentativeChoice,confirmChoice,
  abandonSession,saveComment,heartbeat,sendTiming,pauseSession,resumeSession,
  recordExample,recordLoaded,type Readiness,type CatalogSummary,type SessionDTO
 } from './api';

 type Page = 'home'|'session'|'history'|'stats'|'settings'|'closed';
 type Dialog = null | 'lock' | 'choice' | 'abandon' | 'takeover';
 let page: Page = 'home';
 let current: SessionDTO | null = null;
 let draft: RecordData = emptyRecord();
 let dark = false;
 let helpOpen = false;
 let tour = false;
 let helpStep = 0;
 let notice = '';
 let loaded = false;
 let appConnected = false;
 let appShell = false;
 let readiness: Readiness | null = null;
 let catalogSummary: CatalogSummary | null = null;
 let bootError = '';
 let shuttingDown = false;
 let disposition = '';
 let concentration = '';
 let recordOpen = false;
 let dialog: Dialog = null;
 let lightboxUrl: string | null = null;
 let lightboxLabel = '';
 let selected: string | null = null;
 let choiceConf: number | null = null;
 let comment = '';
 let conflict = false;
 let queue = new SaveQueue();
 let timingSeq = 0;
 let draftGen = 0;
 let lastTick = Date.now();
 const KEY = 'crv-design-prototype-v1';

 $: canEdit = !!current && current.state === 'collecting' && !current.paused && current.youHoldLease;
 $: step = !current ? 0 : current.state === 'locked' ? 5 : current.state === 'completed' ? 6 : current.state === 'abandoned' ? draft.step : draft.step;
 $: storeMessage = conflict ? 'Há uma versão mais nova. Recarregue para continuar.' : saveLabel(queue.status, appConnected ? 'Servidor local conectado' : '');
 $: active = current && (current.state === 'collecting' || current.state === 'locked') ? current : null;

 function touchDraft() {
  draftGen++;
  draft = draft;
 }

 function apply(s: SessionDTO, keepDraft = false) {
  const prevID = current?.id;
  const prevState = current?.state;
  current = s;
  timingSeq = s.timingSeq;
  conflict = false;
  if (keepDraft && prevID === s.id && prevState === s.state && (s.state === 'collecting' || s.state === 'locked')) {
   return;
  }
  const r = {...emptyRecord(), ...s.record};
  if (!r.drawings) r.drawings = {ideogram: [], sketch: []};
  if (!r.groups) r.groups = {};
  if (!Array.isArray(r.summary) || r.summary.length < 5) r.summary = [...(r.summary || []), '', '', '', '', ''].slice(0, 5);
  draft = r;
  selected = s.choice || s.tentativeChoice || selected;
  choiceConf = s.tentativeConfidence ?? choiceConf;
  comment = s.comment || comment;
 }

 async function persist(immediate = true) {
  if (!current || !canEdit) return;
  const task = async () => {
   try {
    draft.step = draft.step || 1;
    const gen = draftGen;
    const s = await saveRecord(current!.id, current!.revision, draft);
    apply(s, gen !== draftGen);
   } catch (e) {
    const msg = e instanceof Error ? e.message : '';
    if (msg.includes('versão mais nova') || msg.includes('revision')) conflict = true;
    throw e;
   }
  };
  if (immediate) await queue.enqueue(task);
  else queue.debounce(task);
 }

 function editGroup(key: string, v: string[], n: string) {
  if (!canEdit) return;
  draft.groups = {...draft.groups, [key]: {ids: v, note: n}};
  touchDraft();
  void persist(true);
 }
 function note(key: string, val: string) {
  if (!canEdit) return;
  (draft as {[k: string]: unknown})[key] = val;
  touchDraft();
  void persist(false);
 }
 function field(key: string): string {
  const v = (draft as {[k: string]: unknown})[key];
  return typeof v === 'string' ? v : '';
 }
 function draw(key: 'ideogram' | 'sketch', v: Stroke[]) {
  if (!canEdit) return;
  draft.drawings = {...draft.drawings, [key]: v};
  touchDraft();
  void persist(true);
 }
 async function go(n: number) {
  if (!current || current.state !== 'collecting' || n < 1 || n > 4) return;
  if (!(await queue.flush())) { notice = 'Não foi possível salvar. Corrija antes de mudar de etapa.'; return; }
  const prev = draft.step;
  draft.step = n;
  touchDraft();
  try {
   await persist(true);
   notice = '';
   window.scrollTo(0, 0);
  } catch {
   draft.step = prev;
   touchDraft();
   notice = 'Não foi possível salvar a mudança de etapa.';
  }
 }
 async function start() {
  if (!appConnected || !readiness?.sessionsOpen) { notice = 'Catálogo indisponível.'; return; }
  if (active) { page = 'session'; return; }
  try {
   const s = await createSession(createOpId(), disposition, concentration);
   apply(s);
   page = 'session';
   notice = '';
  } catch (e) {
   notice = e instanceof Error ? e.message : 'Falha ao criar sessão';
  }
 }
 function prepare() {
  if (!appConnected) { notice = bootError || 'Abra pelo executável.'; return; }
  if (active) { page = 'session'; return; }
  current = null;
  draft = emptyRecord();
  page = 'session';
 }
 async function doLock() {
  if (!current || !(await queue.flush())) { notice = 'Salve o registro antes de bloquear.'; dialog = null; return; }
  try {
   apply(await lockSession(current.id, current.revision));
   dialog = null;
   window.scrollTo(0, 0);
  } catch (e) {
   notice = e instanceof Error ? e.message : 'Falha ao bloquear';
   dialog = null;
  }
 }
 async function select(pos: string) {
  if (!current || current.state !== 'locked' || current.paused || !current.youHoldLease) return;
  selected = pos;
  try { apply(await tentativeChoice(current.id, current.revision, pos, choiceConf)); }
  catch (e) { notice = e instanceof Error ? e.message : 'Falha ao selecionar'; }
 }
 async function doChoose() {
  if (!current || !selected) return;
  try {
   apply(await confirmChoice(current.id, current.revision, selected, choiceConf));
   clearCreateOp();
   dialog = null;
   window.scrollTo(0, 0);
  } catch (e) {
   notice = e instanceof Error ? e.message : 'Falha ao confirmar';
   dialog = null;
  }
 }
 async function doAbandon() {
  if (!current || !(await queue.flush())) { dialog = null; return; }
  try {
   apply(await abandonSession(current.id, current.revision));
   clearCreateOp();
   dialog = null;
   page = 'home';
  } catch (e) {
   notice = e instanceof Error ? e.message : 'Falha ao abandonar';
   dialog = null;
  }
 }
 async function doTakeover() {
  if (!current) return;
  try {
   apply(await heartbeat(current.id, true), false);
   dialog = null;
  } catch (e) {
   notice = e instanceof Error ? e.message : 'Falha ao assumir';
   dialog = null;
  }
 }
 async function togglePause() {
  if (!current || !current.youHoldLease) return;
  if (!(await queue.flush())) { notice = 'Não foi possível salvar.'; return; }
  try {
   apply(current.paused ? await resumeSession(current.id) : await pauseSession(current.id), true);
   lastTick = Date.now();
  } catch (e) {
   notice = e instanceof Error ? e.message : 'Falha ao pausar';
  }
 }
 async function reloadConfirmed() {
  if (!current) return;
  try {
   const s = await getCurrent();
   if (s) apply(s);
   conflict = false;
  } catch (e) {
   notice = e instanceof Error ? e.message : 'Falha ao recarregar';
  }
 }
 async function end() {
  if (!appConnected) { notice = 'Reinicie pelo executável para obter um token e poder encerrar o servidor.'; return; }
  if (!(await queue.flush())) { notice = 'Falha ao salvar. O servidor não foi encerrado.'; return; }
  shuttingDown = true;
  const ok = await shutdown();
  if (ok) { page = 'closed'; }
  else { shuttingDown = false; notice = 'Falha ao encerrar o servidor. Tente de novo.'; }
 }
 function toggleTheme() {
  dark = !dark;
  try { localStorage.setItem(KEY, JSON.stringify({dark})); } catch { /* ignore */ }
 }
 function altUrl(pos: string | null | undefined) {
  return current?.alternatives?.find(a => a.position === pos)?.url || '';
 }
 function openLightbox(pos: string) {
  lightboxUrl = altUrl(pos);
  lightboxLabel = `Alternativa ${pos}`;
 }
 async function onComment(val: string) {
  comment = val;
  if (!current || current.state !== 'completed') return;
  queue.debounce(async () => {
   const s = await saveComment(current!.id, comment);
   current = s;
   conflict = false;
  });
 }

 onMount(() => {
  (async () => {
   try {
    const raw = localStorage.getItem(KEY);
    if (raw) dark = !!JSON.parse(raw).dark;
   } catch { /* ignore */ }
   restoreLease();
   const localApp = location.protocol.startsWith('http') && (location.hostname === '127.0.0.1' || location.hostname === 'localhost' || location.hostname === '[::1]');
   try {
    const bootstrapped = await bootstrapFromHash();
    const ready = bootstrapped ? await getReady() : (localApp ? await getReady() : null);
    if (ready) {
     appShell = true;
     appConnected = true;
     readiness = ready;
     catalogSummary = await getCatalogSummary();
     const s = await getCurrent();
     if (s && (s.state === 'collecting' || s.state === 'locked')) {
      try { apply(s.youHoldLease && !s.paused ? await pauseSession(s.id) : s); }
      catch { apply(s); }
      page = 'session';
     } else if (s && s.state === 'completed') {
      apply(s);
     }
    } else if (localApp) {
     appShell = true;
     bootError = 'Abra o aplicativo pelo executável Go (URL com token) para autenticar esta aba.';
    }
   } catch {
    if (localApp) appShell = true;
    bootError = 'Servidor local indisponível.';
   }
   loaded = true;
  })();
  const beat = setInterval(() => {
   if (current && (current.state === 'collecting' || current.state === 'locked') && current.youHoldLease) {
    void heartbeat(current.id).then(s => apply(s, true)).catch(() => { if (current) current.youHoldLease = false; current = current; });
   }
  }, 5000);
  const tick = setInterval(() => {
   if (!current || page !== 'session' || current.paused || !current.youHoldLease || document.visibilityState !== 'visible') {
    lastTick = Date.now();
    return;
   }
   if (current.state !== 'collecting' && current.state !== 'locked') return;
   const now = Date.now();
   const delta = now - lastTick;
   lastTick = now;
   if (delta < 250) return;
   const next = timingSeq + 1;
   void sendTiming(current.id, next, Math.min(delta, 5000)).then(t => {
    if (!current) return;
    current.collectionMs = t.collectionMs;
    current.choiceMs = t.choiceMs;
    timingSeq = t.timingSeq;
    current = current;
   }).catch(() => {});
  }, 1000);
  const vis = () => { lastTick = Date.now(); };
  document.addEventListener('visibilitychange', vis);
  return () => { clearInterval(beat); clearInterval(tick); document.removeEventListener('visibilitychange', vis); };
 });
</script>
<svelte:window onkeydown={(e) => { if (e.key === 'Escape') { lightboxUrl = null; dialog = null; } }}/>
<div class="app" class:dark>
<header class="app-header"><button class="brand" onclick={() => { page = 'home'; }} aria-label="CRV — início">CRV<span class="brand-dot"></span></button><span class="header-divider"></span><span class="demo-label">{appShell ? 'Fase 2 · sessão local' : 'Protótipo de interface'}</span><div class="header-right">{#if page === 'session' && current}<span class="code">{current.code}</span><span class="timer"><Icon name="clock" size={15}/>{durationMs(current.state === 'locked' ? current.choiceMs : current.collectionMs)}</span>{#if current.state === 'collecting' || current.state === 'locked'}<button class="icon-button" onclick={togglePause} disabled={!current.youHoldLease} aria-label={current.paused ? 'Retomar cronômetro' : 'Pausar cronômetro'}><Icon name={current.paused ? 'play' : 'pause'}/></button>{/if}{/if}<button class="icon-button theme-toggle" onclick={toggleTheme} aria-label="Alternar tema"><Icon name={dark ? 'sun' : 'moon'}/></button><button class="exit-button" aria-label="Salvar e encerrar" onclick={end} disabled={shuttingDown}><Icon name="exit" size={16}/><span>{shuttingDown ? 'Encerrando…' : 'Salvar e encerrar'}</span></button></div></header>
{#if page === 'closed'}
 <main class="closed-screen"><Icon name="check" size={38}/><h1>Aplicativo encerrado.</h1><p>O servidor local foi solicitado a fechar. Você pode fechar esta aba.</p></main>
{:else if page === 'session'}
 <nav class="stepper" aria-label="Etapas da sessão">{#each stages as stage, i}<button class:active={step === i} class:complete={i < step} disabled={!(current?.state === 'collecting' && i >= 1 && i <= step && i <= 4)} onclick={() => go(i)}><span class="step-dot">{#if i < step}<Icon name="check" size={11}/>{/if}</span><span>{stage}</span></button>{/each}</nav>
 <main class="session-main">
 <div class="page-title"><div><h1>{stages[step]}</h1><p>{intro[step]}</p></div><button class="secondary" onclick={() => { helpStep = step; tour = false; helpOpen = true; }}><Icon name="help"/>Como preencher</button></div>
 {#if current && current.paused && current.state !== 'completed'}<div class="inline-note" role="status">Sessão pausada. A edição e o cronômetro estão parados. {#if current.youHoldLease}<button class="text-button" onclick={togglePause}>Retomar</button>{/if}</div>{/if}
 {#if current && !current.youHoldLease && (current.state === 'collecting' || current.state === 'locked')}<div class="inline-note" role="status">Outra aba está editando esta sessão. <button class="text-button" onclick={() => dialog = 'takeover'}>Assumir nesta aba</button></div>{/if}
 {#if conflict}<div class="inline-note" role="status">Seu rascunho local foi preservado. <button class="text-button" onclick={reloadConfirmed}>Recarregar dados confirmados</button></div>{/if}
 {#if notice}<div class="inline-note" role="status">{notice}</div>{/if}
 {#if !current}
 <div class="prepare-layout"><section class="surface prep-main"><h2>Uma sessão, três etapas.</h2><p class="lead">Desenhe, descreva e só depois compare.</p><div class="prep-stages">{#each ['Ideograma','Sensorial','Esboço'] as text, i}<div><span>0{i + 1}</span><h3>{text}</h3><p>{['Um gesto e suas primeiras impressões.','Qualidades simples, sem identificar o objeto.','Formas e relações no espaço.'][i]}</p></div>{/each}</div><div class="prep-form"><label>Disposição <span class="muted">opcional</span><input bind:value={disposition} placeholder="Como você está se sentindo?"/></label><label>Concentração <span class="muted">opcional</span><select bind:value={concentration}><option value="">Não informar</option><option>Baixa</option><option>Moderada</option><option>Alta</option></select></label></div></section><aside class="surface prep-aside"><h3>Antes de começar</h3><p>Reserve cerca de 10 minutos. Não há limite obrigatório de tempo.</p><hr/><p class="small muted">{readiness?.eligibleCount ?? 0} imagens elegíveis neste catálogo. O alvo é sorteado ao iniciar e permanece oculto até a escolha.</p><div class="help-note"><strong>Registro cego</strong><p>As imagens só aparecem depois que você finalizar o registro. Todas as imagens elegíveis podem ser sorteadas de novo na próxima sessão.</p></div></aside></div>
 {:else if current.state === 'abandoned'}
 <section class="surface"><h2>Sessão abandonada</h2><p>O registro foi preservado. O alvo continua oculto.</p><Record record={draft}/></section>
 {:else if step === 1}
 <div class="work-grid"><div class="drawing-column"><DrawingPad strokes={draft.drawings.ideogram} readonly={!canEdit} onchange={(v) => draw('ideogram', v)}/><div class="inline-guidance"><Icon name="pen" size={17}/><span>Primeiro o gesto. A interpretação pode ficar em branco.</span></div></div><section class="surface inspector">{#if !draft.attributesOpen}<div class="attribute-intro"><h2>Deixe o gesto vir primeiro.</h2><p>Quando quiser, abra os atributos para registrar movimento, sensação e categoria geral.</p><button class="primary" disabled={!canEdit} onclick={() => { draft.attributesOpen = true; touchDraft(); void persist(true); }}>Registrar impressões <Icon name="arrow"/></button><p class="small muted">Você também pode continuar sem preencher.</p></div>{:else}{#each groupsI as group}<FieldGroup {group} disabled={!canEdit} values={draft.groups[group.key]?.ids || []} note={draft.groups[group.key]?.note || ''} onchange={(v, n) => editGroup(group.key, v, n)}/>{/each}<label class="notes-label">Hipóteses / AOL<textarea rows="3" disabled={!canEdit} value={draft.aol1} oninput={(e) => note('aol1', e.currentTarget.value)} placeholder="Identificações e interpretações que surgirem."></textarea></label>{/if}</section></div>
 {:else if step === 2}
 <div class="sensory-grid">{#each groupsII as group}<section class="surface"><FieldGroup {group} disabled={!canEdit} values={draft.groups[group.key]?.ids || []} note={draft.groups[group.key]?.note || ''} onchange={(v, n) => editGroup(group.key, v, n)}/></section>{/each}</div><div class="two-notes"><label>Outras impressões sensoriais<textarea rows="3" disabled={!canEdit} value={draft.sensory} oninput={(e) => note('sensory', e.currentTarget.value)} placeholder="Algo que não apareceu nas opções?"></textarea></label><label>Hipóteses / AOL<textarea rows="3" disabled={!canEdit} value={draft.aol2} oninput={(e) => note('aol2', e.currentTarget.value)} placeholder="Registre suas interpretações separadamente."></textarea></label></div>
 {:else if step === 3}
 <div class="work-grid sketch-grid"><div class="drawing-column"><DrawingPad strokes={draft.drawings.sketch} readonly={!canEdit} onchange={(v) => draw('sketch', v)}/><div class="inline-guidance"><Icon name="pen" size={17}/><span>Linhas simples são suficientes. Não é um teste de desenho.</span></div></div><section class="surface inspector">{#each [['forms','Formas'],['dimensions','Dimensões / proporções'],['positions','Posições e relações espaciais'],['spatial','Outras anotações'],['aol3','Hipóteses / AOL']] as [key, title]}<label class="notes-label">{title}<textarea rows="2" disabled={!canEdit} value={field(key)} oninput={(e) => note(key, e.currentTarget.value)} placeholder="Escreva livremente, se quiser."></textarea></label>{/each}</section></div>
 {:else if step === 4}
 <div class="review-layout"><section class="surface review-record"><h2>Seu registro</h2><Record record={draft}/></section><aside class="surface sticky-panel"><h2>O que mais se destacou?</h2><p class="muted">Até cinco características. Todas opcionais.</p>{#each draft.summary as val, i}<label class="summary-input"><span>{i + 1}</span><input aria-label={`Característica ${i + 1}`} disabled={!canEdit} value={val} oninput={(e) => { if (!canEdit) return; draft.summary[i] = e.currentTarget.value; touchDraft(); void persist(false); }} placeholder="Uma característica principal"/></label>{/each}<label class="notes-label">Confiança no registro (%)<input type="number" min="0" max="100" disabled={!canEdit} value={draft.confidence ?? ''} oninput={(e) => { if (!canEdit) return; draft.confidence = e.currentTarget.value === '' ? null : Math.max(0, Math.min(100, Number(e.currentTarget.value))); touchDraft(); void persist(false); }} placeholder="Opcional"/></label><div class="help-note"><Icon name="lock"/><p>Ao finalizar, o registro será bloqueado. Você verá as quatro alternativas em seguida.</p></div></aside></div>
 {:else if step === 5}
 <div class="choice-layout"><aside class="surface choice-record"><h2>Seu registro <Icon name="lock" size={16}/></h2><p class="muted small">Somente leitura</p><Record record={draft} compact/></aside><section><div class="target-grid">{#each current.alternatives || [] as alt}<div class="target-card" class:selected={selected === alt.position}><button class="target-select" onclick={() => select(alt.position)} aria-label={`Selecionar alternativa ${alt.position}`} aria-pressed={selected === alt.position}><img src={alt.url} alt={`Alternativa ${alt.position}`} onload={() => { if (current) void recordLoaded(current.id, [alt.position]); }}/><span class="target-label"><b>{alt.position}</b><span>{selected === alt.position ? 'Selecionada' : 'Selecionar imagem'}</span>{#if selected === alt.position}<Icon name="check"/>{/if}</span></button><button class="expand" aria-label={`Ampliar alternativa ${alt.position}`} onclick={() => openLightbox(alt.position)}><Icon name="expand" size={16}/></button></div>{/each}</div><div class="choice-bottom"><p class="muted">Considere o conjunto, inclusive as divergências.</p><label>Confiança na escolha (%)<input type="number" min="0" max="100" disabled={!current.youHoldLease || current.paused} value={choiceConf ?? ''} placeholder="Opcional" oninput={(e) => { choiceConf = e.currentTarget.value === '' ? null : Math.max(0, Math.min(100, Number(e.currentTarget.value))); }}/></label></div></section></div>
 {:else if step === 6}
 <div class="feedback-layout"><section class="surface feedback-photo"><div class="result-line"><span class="result-icon"><Icon name={current.hit ? 'check' : 'arrow'} size={24}/></span><div><h2>{current.hit ? 'Você escolheu o alvo sorteado.' : 'O alvo sorteado era outra imagem.'}</h2><p class="muted">Sua escolha: {current.choice} · Alvo correto: {current.correct}</p></div></div><button class="feedback-image" onclick={() => current?.correct && openLightbox(current.correct)} aria-label="Ampliar alvo correto"><img src={altUrl(current.correct)} alt={current.feedback?.label || 'Alvo correto'}/></button><h2>{current.feedback?.label}</h2><p>{current.feedback?.description}</p><details><summary>Detalhes do alvo</summary><p class="small">Fonte: {current.feedback?.credit || '—'}</p>
{#if current.feedback?.sources?.length}
 <ul>
  {#each current.feedback.sources as src}
   <li class="small muted">{src.pool} · {src.id} · {src.credit}</li>
  {/each}
 </ul>
{/if}
{#if current.feedback?.sam}<p class="small">Atributos SAM disponíveis neste alvo.</p>{/if}
</details></section><aside class="surface sticky-panel"><h2>Depois do feedback</h2><p class="muted">O que correspondeu? O que divergiu?</p><label>Comentário posterior<textarea rows="7" value={comment} oninput={(e) => onComment(e.currentTarget.value)} placeholder="Esta reflexão fica separada do registro original."></textarea></label><div class="timings"><div><span>Coleta</span><strong>{durationMs(current.collectionMs)}</strong></div><div><span>Escolha</span><strong>{durationMs(current.choiceMs)}</strong></div></div><button class="secondary w-full" onclick={() => recordOpen = !recordOpen}>{recordOpen ? 'Ocultar' : 'Consultar'} registro original <Icon name="chevron"/></button></aside></div>{#if recordOpen}<section class="surface mt"><Record record={draft}/></section>{/if}
 {/if}
 </main>
 <footer class="session-footer"><div>{#if current && current.state === 'collecting'}<button class="secondary" onclick={() => step === 1 ? page = 'home' : go(step - 1)}><Icon name="back"/>Voltar</button><button class="text-button abandon" onclick={() => dialog = 'abandon'}>Abandonar</button>{:else}<button class="secondary" onclick={() => page = 'home'}><Icon name="back"/>Início</button>{#if current?.state === 'locked'}<button class="text-button abandon" onclick={() => dialog = 'abandon'}>Abandonar</button>{/if}{/if}</div><span class="footer-hint" role="status">{current ? storeMessage : 'Todas as respostas são opcionais'}</span>{#if !current}<button class="primary" onclick={start} disabled={!readiness?.sessionsOpen}>Iniciar <Icon name="arrow"/></button>{:else if current.state === 'abandoned'}<button class="primary" onclick={() => page = 'home'}>Início <Icon name="arrow"/></button>{:else if step < 4}<button class="primary" onclick={() => go(step + 1)}>Continuar <Icon name="arrow"/></button>{:else if step === 4}<button class="primary" onclick={() => dialog = 'lock'} disabled={!canEdit}>Finalizar registro <Icon name="lock" size={16}/></button>{:else if step === 5}<button class="primary" disabled={selected === null || !current.youHoldLease || current.paused} onclick={() => dialog = 'choice'}>Confirmar escolha <Icon name="arrow"/></button>{:else}<button class="primary" onclick={() => { clearCreateOp(); current = null; page = 'home'; }}>Nova sessão <Icon name="arrow"/></button>{/if}</footer>
{:else}
 <div class="shell"><nav class="sidebar" aria-label="Navegação principal">{#each [['home','Início','home'],['history','Histórico','clock'],['stats','Estatísticas','chart'],['settings','Configurações','settings']] as [id, label, icon]}<button class:nav-active={page === id} onclick={() => page = id as Page}><Icon name={icon}/>{label}</button>{/each}<div class="sidebar-bottom"><button onclick={() => { helpStep = 0; tour = true; helpOpen = true; }}><Icon name="book"/>Conhecer o fluxo</button><p>CRV · Estágios I–III<br/>{appShell ? 'Fase 2 · offline local' : 'Demonstração local'}</p></div></nav>
 <main class="dashboard">
 {#if page === 'home'}
 <div class="page-title"><div><h1>Seu espaço de prática.</h1><p>Um registro por vez, da primeira impressão ao feedback.</p></div></div>
 {#if appConnected}
 <section class="inline-note readiness-banner" role="status">
  {#if readiness?.ready}<strong>Catálogo pronto.</strong> {readiness.eligibleCount} imagens elegíveis · {readiness.excludedCount} exclusões. Sessões cegas estão ativas.
  {:else}<strong>Catálogo indisponível.</strong> {readiness?.message || bootError || 'Aguardando inicialização.'}{/if}
 </section>
 {/if}
 {#if bootError && !appConnected}<section class="inline-note" role="status">{bootError}</section>{/if}
 {#if notice}<section class="inline-note" role="status">{notice}</section>{/if}
 <section class="home-start"><div><h2>{active ? 'Seu registro espera por você.' : 'Comece com uma folha em branco.'}</h2><p>{active ? 'Retome a mesma sessão, com os desenhos e anotações preservados.' : 'Explore os três estágios com calma. As imagens só aparecem depois de finalizar seu registro.'}</p><button class="primary" onclick={prepare} disabled={!appConnected || (!readiness?.sessionsOpen && !active)} title={!appConnected ? 'Abra pelo executável' : ''}>{active ? 'Continuar sessão' : 'Nova sessão'}<Icon name="arrow"/></button></div><div class="home-flow">{#each ['Ideograma','Sensorial','Esboço'] as v, i}<div><span>0{i + 1}</span><div><h3>{v}</h3><p>{['Registre o gesto.','Descreva as qualidades.','Explore as formas.'][i]}</p></div></div>{/each}</div></section>
 <section class="home-bottom"><div><h2>Antes de experimentar</h2><p class="muted">Uma explicação curta em cada etapa. Exemplos disponíveis quando você precisar.</p><button class="secondary" onclick={() => { tour = true; helpStep = 0; helpOpen = true; }}><Icon name="book"/>Conhecer o fluxo</button></div><div class="demo-note"><h3>Fase 2 · sessão persistida</h3><p>O sorteio e as imagens ficam no servidor local. Histórico, estatísticas e exportações chegam nas fases seguintes.</p></div></section>
 {:else if page === 'history'}
 <div class="page-title"><div><h1>Histórico</h1><p>Histórico real chega na Fase 3.</p></div></div>
 <section class="empty-state"><Icon name="book" size={40}/><h2>Ainda sem lista de sessões.</h2><p>A Fase 2 grava o registro cego. A lista e os filtros vêm na fase seguinte.</p></section>
 {:else if page === 'stats'}
 <div class="page-title"><div><h1>Estatísticas</h1><p>Estatísticas contínuas chegam na Fase 3.</p></div></div>
 <section class="empty-state"><Icon name="chart" size={40}/><h2>Sem totais ainda.</h2><p>A taxa de acertos e o gráfico cumulativo usam apenas escolhas confirmadas, a partir da Fase 3.</p></section>
 {:else if page === 'settings'}
 <div class="page-title"><div><h1>Configurações</h1><p>Preferências locais e resumo do catálogo instalado.</p></div></div>
 <section class="settings-section"><div><h2>Aparência</h2><p>O papel de desenho permanece branco nos dois temas.</p></div><button class="secondary" onclick={toggleTheme}><Icon name={dark ? 'sun' : 'moon'}/>{dark ? 'Usar tema claro' : 'Usar tema escuro'}</button></section>
 <section class="settings-section"><div><h2>Banco de alvos</h2>{#if appConnected && catalogSummary}<p>Revisão ativa · {catalogSummary.catalogLabel || 'bundled'}</p><p class="small muted">{catalogSummary.eligibleCount} elegíveis · {catalogSummary.excludedCount} exclusões</p>{#if catalogSummary.exclusions?.length}<details class="catalog-exclusions"><summary>Ver exclusões (amostra)</summary><ul>{#each catalogSummary.exclusions as ex}<li class="small muted">{ex.reason}{#if ex.path} · {ex.path}{/if}</li>{/each}</ul></details>{/if}{:else}<p>Autentique esta aba pela URL do executável para ver o resumo do catálogo.</p>{/if}<p class="small muted">Substituição do catálogo e backup ZIP chegam nas fases 3 e 4.</p></div><span class="muted small">{appConnected ? (catalogSummary?.ready ? 'Pronto' : 'Indisponível') : 'Aguardando autenticação'}</span></section>
 {#if notice}<p class="inline-note" role="status">{notice}</p>{/if}
 <section class="settings-section"><div><h2>Aplicativo local</h2><p>Go + Chi · Svelte · SQLite · Tailwind<br/>Windows e Linux, offline, com históricos independentes.</p></div></section>
 {/if}
 <p class="dashboard-foot">Sem nuvem. Sem interpretação automática. Um registro de cada vez.</p>
 </main></div>
{/if}
{#if helpOpen}<Help step={helpStep} {tour} onclose={() => helpOpen = false} onexample={() => { if (current) void recordExample(current.id, helpStep); }}/>{/if}
{#if dialog === 'lock'}<ConfirmDialog title="Finalizar seu registro?" message="As quatro imagens serão apresentadas. Seus desenhos e anotações não poderão mais ser alterados." confirmLabel="Bloquear e ver imagens" icon="lock" oncancel={() => dialog = null} onconfirm={doLock}/>{/if}
{#if dialog === 'choice'}<ConfirmDialog title="Confirmar esta escolha?" message="Sua escolha será definitiva. O alvo correto será revelado em seguida." confirmLabel="Confirmar e revelar" icon="check" oncancel={() => dialog = null} onconfirm={doChoose}/>{/if}
{#if dialog === 'abandon'}<ConfirmDialog title="Abandonar esta sessão?" message="O registro salvo permanece. O alvo não será revelado e esta sessão não poderá ser retomada." confirmLabel="Abandonar" icon="exit" oncancel={() => dialog = null} onconfirm={doAbandon}/>{/if}
{#if dialog === 'takeover'}<ConfirmDialog title="Assumir esta sessão?" message="A outra aba passará a ser somente leitura. O cronômetro e a edição ficam nesta aba." confirmLabel="Assumir" icon="play" oncancel={() => dialog = null} onconfirm={doTakeover}/>{/if}
{#if lightboxUrl}<div class="lightbox" role="dialog" aria-modal="true" aria-label="Imagem ampliada"><button class="lightbox-close" onclick={() => lightboxUrl = null} aria-label="Fechar imagem"><Icon name="close" size={25}/></button><img src={lightboxUrl} alt={lightboxLabel}/></div>{/if}
</div>
{#if !loaded}<span class="sr-only">Carregando</span>{/if}
