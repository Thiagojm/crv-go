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
  exclusions?: { sha256?: string; path?: string; reason: string }[];
};

let csrf = '';

function headers(mutating = false): HeadersInit {
  const h: Record<string, string> = { Accept: 'application/json' };
  if (mutating) h['X-CSRF-Token'] = csrf;
  return h;
}

export async function bootstrapFromHash(): Promise<boolean> {
  const hash = location.hash.startsWith('#') ? location.hash.slice(1) : location.hash;
  const params = new URLSearchParams(hash);
  const token = params.get('bootstrap');
  if (!token) return false;
  history.replaceState(null, '', location.pathname + location.search);
  const res = await fetch(`/api/bootstrap?token=${encodeURIComponent(token)}`, { credentials: 'same-origin' });
  if (!res.ok) return false;
  const body = await res.json();
  csrf = body.csrf || '';
  return !!csrf;
}

export async function getReady(): Promise<Readiness | null> {
  const res = await fetch('/api/ready', { credentials: 'same-origin', headers: headers() });
  if (!res.ok) return null;
  return res.json();
}

export async function getCatalogSummary(): Promise<CatalogSummary | null> {
  const res = await fetch('/api/catalog/summary', { credentials: 'same-origin', headers: headers() });
  if (!res.ok) return null;
  return res.json();
}

export async function shutdown(): Promise<boolean> {
  const res = await fetch('/api/shutdown', {
    method: 'POST',
    credentials: 'same-origin',
    headers: headers(true),
  });
  return res.ok;
}
