/**
 * Push-based queue yang juga berupa AsyncIterable — dipakai sebagai `prompt`
 * (streaming input) buat Agent SDK `query()`, supaya satu `Query` bisa dipakai
 * multi-turn (chat beneran) tanpa spawn ulang tiap prompt baru.
 */
export class AsyncQueue<T> implements AsyncIterable<T> {
  private buffer: T[] = [];
  private pendingResolvers: Array<(v: IteratorResult<T>) => void> = [];
  private closed = false;

  push(item: T): void {
    if (this.closed) return;
    const resolver = this.pendingResolvers.shift();
    if (resolver) {
      resolver({ value: item, done: false });
    } else {
      this.buffer.push(item);
    }
  }

  close(): void {
    this.closed = true;
    for (const resolver of this.pendingResolvers.splice(0)) {
      resolver({ value: undefined as T, done: true });
    }
  }

  [Symbol.asyncIterator](): AsyncIterator<T> {
    return {
      next: (): Promise<IteratorResult<T>> => {
        if (this.buffer.length > 0) {
          return Promise.resolve({ value: this.buffer.shift() as T, done: false });
        }
        if (this.closed) {
          return Promise.resolve({ value: undefined as T, done: true });
        }
        return new Promise((resolve) => this.pendingResolvers.push(resolve));
      },
    };
  }
}
