import type {RecordData} from './data';

export type Readiness = {
  ready: boolean;
  phase: string;
  sessionsOpen: boolean;
  eligibleCount: number;
  excludedCount: number;
  catalogLabel?: string;
  error?: string;
  message: string;
};

export type CatalogSummary = Readiness & {
  exclusions?: {sha256?: string; path?: string; reason: string}[];
};

export type Alternative = {position: string; url: string};
export type Feedback = {
  position: string;
  label: string;
  description: string;
  credit: string;
  sam?: unknown;
  sources?: {id: string; pool: string; label: string; description: string; credit: string}[];
};

export type SessionDTO = {
  id: string;
  code: string;
  state: 'collecting' | 'locked' | 'completed' | 'abandoned';
  revision: number;
  paused: boolean;
  youHoldLease: boolean;
  step: number;
  collectionMs: number;
  choiceMs: number;
  timingSeq: number;
  helpVersion: string;
  record: RecordData;
  alternatives?: Alternative[];
  tentativeChoice?: string | null;
  tentativeConfidence?: number | null;
  choice?: string | null;
  correct?: string | null;
  hit?: boolean;
  feedback?: Feedback;
  comment?: string;
  abandonedAfterAlts?: boolean;
  createdAt: string;
};

type Envelope = {session: SessionDTO | null; leaseToken?: string; error?: string; message?: string};

let csrf = '';
let lease = '';
const LEASE_KEY = 'crv-lease';
const CSRF_KEY = 'crv-csrf';

function setCsrf(token: string) {
  if (!token) return;
  csrf = token;
  sessionStorage.setItem(CSRF_KEY, token);
}

export function currentLease(): string {
  return lease;
}

export function restoreLease(): void {
  lease = sessionStorage.getItem(LEASE_KEY) || '';
  csrf = sessionStorage.getItem(CSRF_KEY) || csrf;
}

function headers(mutating = false): Record<string, string> {
  const h: Record<string, string> = {Accept: 'application/json'};
  if (mutating) h['X-CSRF-Token'] = csrf;
  if (lease) h['X-Lease-Token'] = lease;
  return h;
}

async function parse<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = (body as Envelope).message || (body as Envelope).error || `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return body as T;
}

export async function bootstrapFromHash(): Promise<boolean> {
  const hash = location.hash.startsWith('#') ? location.hash.slice(1) : location.hash;
  const token = new URLSearchParams(hash).get('bootstrap');
  if (!token) return false;
  history.replaceState(null, '', location.pathname + location.search);
  const res = await fetch(`/api/bootstrap?token=${encodeURIComponent(token)}`, {credentials: 'same-origin'});
  if (!res.ok) return false;
  const body = await res.json();
  setCsrf(body.csrf || '');
  return !!csrf;
}

export async function getReady(): Promise<Readiness | null> {
  const res = await fetch('/api/ready', {credentials: 'same-origin', headers: headers()});
  if (!res.ok) return null;
  const body = await res.json();
  if (body.csrf) setCsrf(body.csrf);
  return body;
}

export async function getCatalogSummary(): Promise<CatalogSummary | null> {
  const res = await fetch('/api/catalog/summary', {credentials: 'same-origin', headers: headers()});
  if (!res.ok) return null;
  return res.json();
}

export async function shutdown(): Promise<boolean> {
  const res = await fetch('/api/shutdown', {method: 'POST', credentials: 'same-origin', headers: headers(true)});
  return res.ok;
}

function takeLease(env: Envelope): SessionDTO | null {
  if (env.leaseToken) {
    lease = env.leaseToken;
    sessionStorage.setItem(LEASE_KEY, lease);
  }
  if (env.session && (env.session.state === 'completed' || env.session.state === 'abandoned')) {
    sessionStorage.removeItem(LEASE_KEY);
  }
  return env.session;
}

export async function createSession(operationId: string, disposition: string, concentration: string): Promise<SessionDTO> {
  const res = await fetch('/api/sessions', {
    method: 'POST',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({operationId, disposition, concentration}),
  });
  const env = await parse<Envelope>(res);
  const s = takeLease(env);
  if (!s) throw new Error('falha ao criar sessão');
  return s;
}

export async function getCurrent(): Promise<SessionDTO | null> {
  const res = await fetch('/api/sessions/current', {credentials: 'same-origin', headers: headers()});
  const env = await parse<Envelope>(res);
  return env.session;
}

export async function getSession(id: string): Promise<SessionDTO> {
  const res = await fetch(`/api/sessions/${encodeURIComponent(id)}`, {credentials: 'same-origin', headers: headers()});
  const env = await parse<Envelope>(res);
  if (!env.session) throw new Error('sessão não encontrada');
  return env.session;
}

export type HistoryItem = {
  id: string;
  code: string;
  state: SessionDTO['state'];
  createdAt: string;
  collectionMs: number;
  choiceMs: number;
  abandonedAfterAlts?: boolean;
  confirmedChoice?: string | null;
  hit?: boolean | null;
  completedAt?: string;
  abandonedAt?: string;
};

export type HistoryPage = {items: HistoryItem[]; total: number};

export async function fetchHistory(opts: {
  states?: string[];
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
} = {}): Promise<HistoryPage> {
  const q = new URLSearchParams();
  if (opts.states?.length) q.set('state', opts.states.join(','));
  if (opts.from) q.set('from', opts.from);
  if (opts.to) q.set('to', opts.to);
  if (opts.limit != null) q.set('limit', String(opts.limit));
  if (opts.offset != null) q.set('offset', String(opts.offset));
  const res = await fetch(`/api/history?${q}`, {credentials: 'same-origin', headers: headers()});
  return parse(res);
}

export type ChartPoint = {
  n: number;
  hits: number;
  reference: number;
  code: string;
  completedAt: string;
  hit: boolean;
};

export type Statistics = {
  initiated: number;
  active: number;
  completed: number;
  abandonedBefore: number;
  abandonedAfter: number;
  confirmedChoices: number;
  hits: number;
  hitRate: number | null;
  chart: ChartPoint[];
};

export async function fetchStatistics(): Promise<Statistics> {
  const res = await fetch('/api/statistics', {credentials: 'same-origin', headers: headers()});
  return parse(res);
}

export type CatalogImportResult = {
  ready: boolean;
  revisionId?: number;
  report?: CatalogSummary & {
    sourceLabel?: string;
    inventoryImages?: number;
    multiImageSources?: number;
    emptySources?: number;
    error?: string;
  };
  error?: string;
  message?: string;
};

export async function importCatalogFolder(path: string): Promise<CatalogImportResult> {
  const res = await fetch('/api/catalog/import/folder', {
    method: 'POST',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({path}),
  });
  return parse(res);
}

export async function importCatalogZip(file: File): Promise<CatalogImportResult> {
  const body = new FormData();
  body.append('archive', file);
  const res = await fetch('/api/catalog/import/zip', {
    method: 'POST',
    credentials: 'same-origin',
    headers: headers(true),
    body,
  });
  return parse(res);
}

export async function repairCatalog(): Promise<CatalogImportResult> {
  const res = await fetch('/api/catalog/repair', {
    method: 'POST',
    credentials: 'same-origin',
    headers: headers(true),
  });
  return parse(res);
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export async function downloadHistoryCSV(opts: {
  states?: string[];
  from?: string;
  to?: string;
} = {}): Promise<void> {
  const q = new URLSearchParams();
  if (opts.states?.length) q.set('state', opts.states.join(','));
  if (opts.from) q.set('from', opts.from);
  if (opts.to) q.set('to', opts.to);
  const res = await fetch(`/api/exports/csv?${q}`, {credentials: 'same-origin', headers: headers()});
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as {message?: string}).message || `HTTP ${res.status}`);
  }
  const dispo = res.headers.get('Content-Disposition') || '';
  const m = /filename="([^"]+)"/.exec(dispo);
  downloadBlob(await res.blob(), m?.[1] || 'crv-historico.csv');
}

export function openSessionPrint(id: string): void {
  window.open(`/api/exports/sessions/${encodeURIComponent(id)}/print`, '_blank', 'noopener');
}

export async function downloadBackup(): Promise<void> {
  const res = await fetch('/api/backup', {credentials: 'same-origin', headers: headers()});
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as {message?: string}).message || `HTTP ${res.status}`);
  }
  const dispo = res.headers.get('Content-Disposition') || '';
  const m = /filename="([^"]+)"/.exec(dispo);
  downloadBlob(await res.blob(), m?.[1] || 'crv-backup.zip');
}

export type RestoreResult = {
  ok: boolean;
  preBackupPath?: string;
  bootstrapToken?: string;
  message?: string;
  ready?: boolean;
};

export async function restoreBackup(file: File): Promise<RestoreResult> {
  const body = new FormData();
  body.append('archive', file);
  body.append('confirm', 'true');
  const res = await fetch('/api/backup/restore', {
    method: 'POST',
    credentials: 'same-origin',
    headers: headers(true),
    body,
  });
  return parse(res);
}

export async function bootstrapWithToken(token: string): Promise<boolean> {
  const res = await fetch(`/api/bootstrap?token=${encodeURIComponent(token)}`, {credentials: 'same-origin'});
  if (!res.ok) return false;
  const body = await res.json();
  setCsrf(body.csrf || '');
  sessionStorage.removeItem(LEASE_KEY);
  lease = '';
  return !!csrf;
}

export async function saveRecord(id: string, expectedRevision: number, record: RecordData): Promise<SessionDTO> {
  const res = await fetch(`/api/sessions/${id}/record`, {
    method: 'PUT',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({expectedRevision, record}),
  });
  const env = await parse<Envelope>(res);
  const s = takeLease(env);
  if (!s) throw new Error('falha ao salvar');
  return s;
}

export async function lockSession(id: string, expectedRevision: number): Promise<SessionDTO> {
  const res = await fetch(`/api/sessions/${id}/lock`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({expectedRevision}),
  });
  const env = await parse<Envelope>(res);
  const s = takeLease(env);
  if (!s) throw new Error('falha ao bloquear');
  return s;
}

export async function tentativeChoice(id: string, expectedRevision: number, choice: string, confidence: number | null): Promise<SessionDTO> {
  const res = await fetch(`/api/sessions/${id}/choice`, {
    method: 'PUT',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({expectedRevision, choice, confidence}),
  });
  const env = await parse<Envelope>(res);
  const s = takeLease(env);
  if (!s) throw new Error('falha ao registrar escolha');
  return s;
}

export async function confirmChoice(id: string, expectedRevision: number, choice: string, confidence: number | null): Promise<SessionDTO> {
  const res = await fetch(`/api/sessions/${id}/confirm`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({expectedRevision, choice, confidence}),
  });
  const env = await parse<Envelope>(res);
  const s = takeLease(env);
  if (!s) throw new Error('falha ao confirmar');
  return s;
}

export async function abandonSession(id: string, expectedRevision: number): Promise<SessionDTO> {
  const res = await fetch(`/api/sessions/${id}/abandon`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({expectedRevision}),
  });
  const env = await parse<Envelope>(res);
  const s = takeLease(env);
  if (!s) throw new Error('falha ao abandonar');
  return s;
}

export async function saveComment(id: string, comment: string): Promise<SessionDTO> {
  const res = await fetch(`/api/sessions/${id}/comment`, {
    method: 'PUT',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({comment}),
  });
  const env = await parse<Envelope>(res);
  const s = takeLease(env);
  if (!s) throw new Error('falha ao comentar');
  return s;
}

export async function heartbeat(id: string, transfer = false): Promise<SessionDTO> {
  const res = await fetch(`/api/sessions/${id}/lease`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({token: lease, transfer}),
  });
  const env = await parse<Envelope>(res);
  const s = takeLease(env);
  if (!s) throw new Error('falha no lease');
  return s;
}

export async function sendTiming(id: string, seq: number, deltaMs: number): Promise<{collectionMs: number; choiceMs: number; timingSeq: number}> {
  const res = await fetch(`/api/sessions/${id}/timing`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({seq, deltaMs}),
  });
  return parse(res);
}

export async function pauseSession(id: string): Promise<SessionDTO> {
  const res = await fetch(`/api/sessions/${id}/pause`, {method: 'POST', credentials: 'same-origin', headers: headers(true)});
  const env = await parse<Envelope>(res);
  const s = takeLease(env);
  if (!s) throw new Error('falha ao pausar');
  return s;
}

export async function resumeSession(id: string): Promise<SessionDTO> {
  const res = await fetch(`/api/sessions/${id}/resume`, {method: 'POST', credentials: 'same-origin', headers: headers(true)});
  const env = await parse<Envelope>(res);
  const s = takeLease(env);
  if (!s) throw new Error('falha ao retomar');
  return s;
}

export async function recordExample(id: string, step: number): Promise<void> {
  await fetch(`/api/sessions/${id}/example`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({step}),
  });
}

export async function recordLoaded(id: string, positions: string[]): Promise<void> {
  await fetch(`/api/sessions/${id}/loaded`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: {...headers(true), 'Content-Type': 'application/json'},
    body: JSON.stringify({positions}),
  });
}
