import type { Level } from './orderbook';

// ─── Buffer ────────────────────────────────────────────────────────────────

interface Snapshot {
  ts:   number;
  bids: Level[];
  asks: Level[];
}

const WINDOW_MS   = 90_000;  // rolling window shown on X-axis
const MAX_ENTRIES = 500;     // hard cap to avoid unbounded growth

export class HeatmapBuffer {
  private snaps: Snapshot[] = [];

  push(bids: Level[], asks: Level[]): void {
    this.snaps.push({ ts: Date.now(), bids: [...bids], asks: [...asks] });
    // trim by time first, then by hard cap
    const cutoff = Date.now() - WINDOW_MS;
    let lo = 0;
    while (lo < this.snaps.length && this.snaps[lo].ts < cutoff) lo++;
    if (lo > 0) this.snaps.splice(0, lo);
    if (this.snaps.length > MAX_ENTRIES)
      this.snaps.splice(0, this.snaps.length - MAX_ENTRIES);
  }

  get entries(): Readonly<Snapshot>[] { return this.snaps; }
  get size(): number { return this.snaps.length; }
}

// ─── Layout / colours ──────────────────────────────────────────────────────

const PAD = { top: 6, right: 8, bottom: 22, left: 64 };

const C = {
  bg:   '#0d0f14',
  axis: '#21262d',
  text: '#484f58',
  bid:  [63, 185, 80]   as [number, number, number],
  ask:  [248, 81,  73]  as [number, number, number],
} as const;

// ─── Public API ────────────────────────────────────────────────────────────

export function drawHeatmap(
  canvas: HTMLCanvasElement,
  buffer: HeatmapBuffer,
): void {
  const snaps = buffer.entries;
  if (snaps.length < 2) return;

  const dpr  = window.devicePixelRatio || 1;
  const cssW = canvas.clientWidth;
  const cssH = canvas.clientHeight;
  if (cssW === 0 || cssH === 0) return;

  const targetW = Math.round(cssW * dpr);
  const targetH = Math.round(cssH * dpr);
  if (canvas.width !== targetW || canvas.height !== targetH) {
    canvas.width  = targetW;
    canvas.height = targetH;
  }

  const ctx = canvas.getContext('2d')!;
  ctx.save();
  ctx.scale(dpr, dpr);
  ctx.fillStyle = C.bg;
  ctx.fillRect(0, 0, cssW, cssH);

  // ── Collect unique price levels ─────────────────────────────────────────
  const priceSet = new Set<number>();
  for (const s of snaps) {
    for (const [p] of s.bids) priceSet.add(p);
    for (const [p] of s.asks) priceSet.add(p);
  }
  const prices = Array.from(priceSet).sort((a, b) => b - a); // top = highest
  const N = prices.length;
  if (N === 0) { ctx.restore(); return; }

  // Row index lookup
  const priceRow = new Map<number, number>();
  prices.forEach((p, i) => priceRow.set(p, i));

  // ── Find max qty for log-normalisation ──────────────────────────────────
  let maxQty = 1;
  for (const s of snaps) {
    for (const [, q] of s.bids) if (q > maxQty) maxQty = q;
    for (const [, q] of s.asks) if (q > maxQty) maxQty = q;
  }
  const logMax = Math.log1p(maxQty);

  // ── Plot area ───────────────────────────────────────────────────────────
  const plotW = cssW - PAD.left - PAD.right;
  const plotH = cssH - PAD.top  - PAD.bottom;
  const colW  = plotW / snaps.length;
  const rowH  = plotH / N;

  // ── Draw cells ──────────────────────────────────────────────────────────
  for (let i = 0; i < snaps.length; i++) {
    const { bids, asks } = snaps[i];
    const x = PAD.left + i * colW;

    for (const [price, qty] of bids) {
      const row = priceRow.get(price);
      if (row === undefined || qty <= 0) continue;
      const alpha = Math.log1p(qty) / logMax;
      const [r, g, b] = C.bid;
      ctx.fillStyle = `rgba(${r},${g},${b},${alpha.toFixed(3)})`;
      ctx.fillRect(x, PAD.top + row * rowH, Math.ceil(colW) + 0.5, Math.ceil(rowH) + 0.5);
    }

    for (const [price, qty] of asks) {
      const row = priceRow.get(price);
      if (row === undefined || qty <= 0) continue;
      const alpha = Math.log1p(qty) / logMax;
      const [r, g, b] = C.ask;
      ctx.fillStyle = `rgba(${r},${g},${b},${alpha.toFixed(3)})`;
      ctx.fillRect(x, PAD.top + row * rowH, Math.ceil(colW) + 0.5, Math.ceil(rowH) + 0.5);
    }
  }

  // ── Y-axis: price labels ────────────────────────────────────────────────
  ctx.fillStyle   = C.text;
  ctx.font        = '10px monospace';
  ctx.textAlign   = 'right';
  ctx.textBaseline = 'middle';
  const minRowPx  = 12; // minimum vertical gap between labels
  let lastLabelY  = -Infinity;
  for (let j = N - 1; j >= 0; j--) {
    const y = PAD.top + j * rowH + rowH / 2;
    if (y - lastLabelY >= minRowPx) {
      ctx.fillText(prices[j].toFixed(0), PAD.left - 4, y);
      lastLabelY = y;
    }
  }

  // ── X-axis baseline + time labels ───────────────────────────────────────
  const axisY = PAD.top + plotH;
  ctx.strokeStyle = C.axis;
  ctx.lineWidth   = 1;
  ctx.beginPath();
  ctx.moveTo(PAD.left, axisY);
  ctx.lineTo(PAD.left + plotW, axisY);
  ctx.stroke();

  ctx.fillStyle    = C.text;
  ctx.textAlign    = 'center';
  ctx.textBaseline = 'top';
  // aim for a label every ~80px, at least one
  const labelStep = Math.max(1, Math.round(80 / colW));
  for (let i = 0; i < snaps.length; i += labelStep) {
    const x  = PAD.left + i * colW + colW / 2;
    const dt = Math.round((snaps[snaps.length - 1].ts - snaps[i].ts) / 1000);
    ctx.fillText(dt === 0 ? 'now' : `-${dt}s`, x, axisY + 4);
  }
  // always label the rightmost column as "now"
  ctx.fillText('now', PAD.left + plotW - colW / 2, axisY + 4);

  ctx.restore();
}
