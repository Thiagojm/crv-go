export type SaveStatus = 'idle' | 'saving' | 'saved' | 'error';

export class SaveQueue {
  status: SaveStatus = 'idle';
  lastError = '';
  private chain: Promise<void> = Promise.resolve();
  private timer: ReturnType<typeof setTimeout> | null = null;
  private pending: (() => Promise<void>) | null = null;

  enqueue(task: () => Promise<void>): Promise<void> {
    this.pending = null;
    this.chain = this.chain.catch(() => {}).then(async () => {
      this.status = 'saving';
      try {
        await task();
        this.status = 'saved';
      } catch (e) {
        this.status = 'error';
        this.lastError = e instanceof Error ? e.message : 'Falha ao salvar';
        throw e;
      }
    });
    return this.chain;
  }

  debounce(task: () => Promise<void>, ms = 500): void {
    this.pending = task;
    if (this.timer) clearTimeout(this.timer);
    this.timer = setTimeout(() => {
      this.timer = null;
      const t = this.pending;
      this.pending = null;
      if (t) void this.enqueue(t);
    }, ms);
  }

  async flush(): Promise<boolean> {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    if (this.pending) {
      const t = this.pending;
      this.pending = null;
      try {
        await this.enqueue(t);
      } catch {
        return false;
      }
    }
    try {
      await this.chain;
      return this.status !== 'error';
    } catch {
      return false;
    }
  }
}

export function saveLabel(status: SaveStatus, fallback: string): string {
  if (status === 'saving') return 'Salvando';
  if (status === 'saved') return 'Salvo';
  if (status === 'error') return 'Falha ao salvar';
  return fallback;
}

export function createOpId(): string {
  const key = 'crv-create-op';
  let id = sessionStorage.getItem(key);
  if (!id) {
    id = crypto.randomUUID();
    sessionStorage.setItem(key, id);
  }
  return id;
}

export function clearCreateOp(): void {
  sessionStorage.removeItem('crv-create-op');
}
