import type { Level } from './orderbook';

const MAX_LEVELS = 15;

function fmtPrice(p: number): string {
  return p.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function fmtQty(q: number): string {
  return q.toFixed(4);
}

function buildRow(
  price: number,
  qty: number,
  cumQty: number,
  side: 'bid' | 'ask',
  changed: boolean,
): HTMLTableRowElement {
  const tr = document.createElement('tr');
  tr.className = side + (changed ? ' flash' : '');

  const tdPrice = document.createElement('td');
  tdPrice.className = 'price';
  tdPrice.textContent = fmtPrice(price);

  const tdQty = document.createElement('td');
  tdQty.textContent = fmtQty(qty);

  const tdCumul = document.createElement('td');
  tdCumul.className = 'cumul';
  tdCumul.textContent = fmtQty(cumQty);

  tr.append(tdPrice, tdQty, tdCumul);
  return tr;
}

export function renderTable(
  asksBody: HTMLTableSectionElement,
  bidsBody: HTMLTableSectionElement,
  spreadEl: HTMLElement,
  bids: Level[],
  asks: Level[],
  changedBids: Set<string>,
  changedAsks: Set<string>,
): void {
  // Asks: sorted ASC (best ask first). Compute cumulative from best ask outward,
  // then reverse for display (worst ask on top, best ask at bottom near spread).
  let askCumQty = 0;
  const asksWithCumul = asks.slice(0, MAX_LEVELS).map(([price, qty]): [number, number, number] => {
    askCumQty += qty;
    return [price, qty, askCumQty];
  });
  const askRows = [...asksWithCumul].reverse().map(([price, qty, cumQty]) =>
    buildRow(price, qty, cumQty, 'ask', changedAsks.has(String(price))),
  );
  asksBody.replaceChildren(...askRows);

  // Bids: sorted DESC (best bid first), best bid at top.
  let bidCumQty = 0;
  const bidRows = bids.slice(0, MAX_LEVELS).map(([price, qty]) => {
    bidCumQty += qty;
    return buildRow(price, qty, bidCumQty, 'bid', changedBids.has(String(price)));
  });
  bidsBody.replaceChildren(...bidRows);

  // Spread bar
  if (bids.length > 0 && asks.length > 0) {
    const bestBid = bids[0][0];
    const bestAsk = asks[0][0];
    const spread  = (bestAsk - bestBid).toFixed(2);
    spreadEl.textContent = `spread  ${spread}  |  bid ${fmtPrice(bestBid)}  ·  ask ${fmtPrice(bestAsk)}`;
  } else {
    spreadEl.textContent = '— spread —';
  }
}
