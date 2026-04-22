export type Level = [number, number]; // [price, qty]

export class OrderBook {
  readonly bids = new Map<string, number>();
  readonly asks = new Map<string, number>();

  applySnapshot(bids: Level[], asks: Level[]): void {
    this.bids.clear();
    this.asks.clear();
    for (const [p, q] of bids) this.bids.set(String(p), q);
    for (const [p, q] of asks) this.asks.set(String(p), q);
  }

  applyUpdate(
    bids: Level[],
    asks: Level[],
  ): { changedBids: Set<string>; changedAsks: Set<string> } {
    const changedBids = new Set<string>();
    const changedAsks = new Set<string>();

    for (const [p, q] of (bids ?? [])) {
      const k = String(p);
      q === 0 ? this.bids.delete(k) : this.bids.set(k, q);
      changedBids.add(k);
    }
    for (const [p, q] of (asks ?? [])) {
      const k = String(p);
      q === 0 ? this.asks.delete(k) : this.asks.set(k, q);
      changedAsks.add(k);
    }

    return { changedBids, changedAsks };
  }

  /** Sorted highest price first (best bid on top). */
  sortedBids(): Level[] {
    return Array.from(this.bids.entries())
      .map(([p, q]): Level => [Number(p), q])
      .sort((a, b) => b[0] - a[0]);
  }

  /** Sorted lowest price first (best ask on top). */
  sortedAsks(): Level[] {
    return Array.from(this.asks.entries())
      .map(([p, q]): Level => [Number(p), q])
      .sort((a, b) => a[0] - b[0]);
  }
}
