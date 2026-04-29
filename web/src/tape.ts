import type { TradeMsg } from './ws';

const MAX_ROWS = 80;

function fmtPrice(p: number): string {
  return p.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function fmtQty(q: number): string {
  return q.toFixed(4);
}

function fmtTime(ms: number): string {
  const d = new Date(ms);
  const hh = String(d.getHours()).padStart(2, '0');
  const mm = String(d.getMinutes()).padStart(2, '0');
  const ss = String(d.getSeconds()).padStart(2, '0');
  const ms3 = String(d.getMilliseconds()).padStart(3, '0');
  return `${hh}:${mm}:${ss}.${ms3}`;
}

// ── CVD chart ────────────────────────────────────────────────────────────────

export class CvdTracker {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;
  private history: number[] = [];
  private cvd = 0;

  constructor(canvas: HTMLCanvasElement) {
    this.canvas = canvas;
    this.ctx = canvas.getContext('2d')!;
  }

  push(side: 'buy' | 'sell', qty: number): void {
    this.cvd += side === 'buy' ? qty : -qty;
    this.history.push(this.cvd);
    if (this.history.length > 400) this.history.shift();
    this.draw();
  }

  draw(): void {
    const { canvas, ctx, history } = this;
    const W = canvas.width = canvas.offsetWidth;
    const H = canvas.height = canvas.offsetHeight;
    ctx.clearRect(0, 0, W, H);

    if (history.length < 2) return;

    const min = Math.min(...history);
    const max = Math.max(...history);
    const range = max - min || 1;

    const toY = (v: number) => H - ((v - min) / range) * (H - 20) - 10;
    const toX = (i: number) => (i / (history.length - 1)) * W;

    const zeroY = toY(0);
    ctx.strokeStyle = 'rgba(255,255,255,0.12)';
    ctx.lineWidth = 1;
    ctx.setLineDash([4, 4]);
    ctx.beginPath();
    ctx.moveTo(0, zeroY);
    ctx.lineTo(W, zeroY);
    ctx.stroke();
    ctx.setLineDash([]);

    ctx.beginPath();
    ctx.moveTo(toX(0), toY(history[0]));
    for (let i = 1; i < history.length; i++) {
      ctx.lineTo(toX(i), toY(history[i]));
    }
    ctx.lineTo(toX(history.length - 1), H);
    ctx.lineTo(0, H);
    ctx.closePath();
    const grad = ctx.createLinearGradient(0, 0, 0, H);
    grad.addColorStop(0, 'rgba(0,229,255,0.25)');
    grad.addColorStop(1, 'rgba(0,229,255,0.02)');
    ctx.fillStyle = grad;
    ctx.fill();

    ctx.beginPath();
    ctx.moveTo(toX(0), toY(history[0]));
    for (let i = 1; i < history.length; i++) {
      ctx.lineTo(toX(i), toY(history[i]));
    }
    ctx.strokeStyle = '#00e5ff';
    ctx.lineWidth = 1.5;
    ctx.stroke();

    const last = history[history.length - 1];
    ctx.fillStyle = last >= 0 ? '#3fb950' : '#f85149';
    ctx.font = 'bold 11px monospace';
    ctx.fillText(`CVD ${last >= 0 ? '+' : ''}${last.toFixed(3)}`, 6, 14);
  }
}

// ── Tape bubbles chart ────────────────────────────────────────────────────────

interface Bubble {
  ts: number;
  price: number;
  qty: number;
  side: 'buy' | 'sell';
}

const PAD = { top: 14, right: 56, bottom: 28, left: 10 };
const MAX_BUBBLES = 400;
const WINDOW_MS = 60_000;

export class BubblesChart {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;
  private data: Bubble[] = [];
  private rafId = 0;
  private dirty = false;

  constructor(canvas: HTMLCanvasElement) {
    this.canvas = canvas;
    this.ctx = canvas.getContext('2d')!;

    const ro = new ResizeObserver(() => { this.dirty = true; this.schedule(); });
    ro.observe(canvas);
  }

  push(trade: TradeMsg): void {
    this.data.push({ ts: trade.ts, price: trade.p, qty: trade.q, side: trade.side });
    if (this.data.length > MAX_BUBBLES) this.data.shift();
    this.dirty = true;
    this.schedule();
  }

  private schedule(): void {
    if (this.rafId) return;
    this.rafId = requestAnimationFrame(() => {
      this.rafId = 0;
      if (this.dirty) { this.dirty = false; this.draw(); }
    });
  }

  draw(): void {
    const { canvas, ctx, data } = this;
    const dpr = window.devicePixelRatio || 1;
    const cssW = canvas.offsetWidth;
    const cssH = canvas.offsetHeight;
    if (cssW === 0 || cssH === 0) return;

    const tW = Math.round(cssW * dpr);
    const tH = Math.round(cssH * dpr);
    if (canvas.width !== tW || canvas.height !== tH) {
      canvas.width = tW;
      canvas.height = tH;
    }

    ctx.save();
    ctx.scale(dpr, dpr);
    ctx.clearRect(0, 0, cssW, cssH);

    ctx.fillStyle = '#0d1117';
    ctx.fillRect(0, 0, cssW, cssH);

    if (data.length === 0) {
      ctx.fillStyle = '#484f58';
      ctx.font = '11px monospace';
      ctx.textAlign = 'center';
      ctx.fillText('waiting for trades…', cssW / 2, cssH / 2);
      ctx.restore();
      return;
    }

    const now = data[data.length - 1].ts;
    const tMin = now - WINDOW_MS;
    const visible = data.filter(b => b.ts >= tMin);
    if (visible.length === 0) { ctx.restore(); return; }

    const plotW = cssW - PAD.left - PAD.right;
    const plotH = cssH - PAD.top - PAD.bottom;

    const prices = visible.map(b => b.price);
    let priceMin = Math.min(...prices);
    let priceMax = Math.max(...prices);
    const priceRange = priceMax - priceMin || 1;
    const padPct = 0.12;
    priceMin -= priceRange * padPct;
    priceMax += priceRange * padPct;
    const pRange = priceMax - priceMin;

    const toX = (ts: number) => PAD.left + ((ts - tMin) / WINDOW_MS) * plotW;
    const toY = (p: number) => PAD.top + (1 - (p - priceMin) / pRange) * plotH;
    const toR = (q: number) => Math.min(18, Math.max(3, 2.5 + Math.log1p(q * 80) * 3.2));

    // grid lines
    ctx.strokeStyle = 'rgba(255,255,255,0.05)';
    ctx.lineWidth = 1;
    const gridSteps = 4;
    for (let i = 0; i <= gridSteps; i++) {
      const y = PAD.top + (plotH / gridSteps) * i;
      ctx.beginPath();
      ctx.moveTo(PAD.left, y);
      ctx.lineTo(PAD.left + plotW, y);
      ctx.stroke();
    }

    // time grid every 10s
    ctx.strokeStyle = 'rgba(255,255,255,0.04)';
    for (let dt = 0; dt <= WINDOW_MS; dt += 10_000) {
      const x = toX(tMin + dt);
      ctx.beginPath();
      ctx.moveTo(x, PAD.top);
      ctx.lineTo(x, PAD.top + plotH);
      ctx.stroke();
    }

    // price axis labels (right)
    ctx.fillStyle = '#484f58';
    ctx.font = '10px monospace';
    ctx.textAlign = 'left';
    ctx.textBaseline = 'middle';
    for (let i = 0; i <= gridSteps; i++) {
      const p = priceMax - (pRange / gridSteps) * i;
      const y = PAD.top + (plotH / gridSteps) * i;
      ctx.fillText(p.toFixed(1), PAD.left + plotW + 4, y);
    }

    // time labels (bottom)
    ctx.textAlign = 'center';
    ctx.textBaseline = 'top';
    for (let dt = 0; dt <= WINDOW_MS; dt += 15_000) {
      const x = toX(tMin + dt);
      const sec = Math.round((WINDOW_MS - dt) / 1000);
      ctx.fillText(sec === 0 ? 'now' : `-${sec}s`, x, PAD.top + plotH + 4);
    }

    // price line (connect mid-prices)
    if (visible.length > 1) {
      ctx.beginPath();
      ctx.moveTo(toX(visible[0].ts), toY(visible[0].price));
      for (let i = 1; i < visible.length; i++) {
        ctx.lineTo(toX(visible[i].ts), toY(visible[i].price));
      }
      ctx.strokeStyle = 'rgba(255,255,255,0.10)';
      ctx.lineWidth = 1;
      ctx.stroke();
    }

    // bubbles — draw sells first, then buys (buys on top)
    for (const side of ['sell', 'buy'] as const) {
      for (const b of visible) {
        if (b.side !== side) continue;
        const x = toX(b.ts);
        const y = toY(b.price);
        const r = toR(b.qty);

        const isBuy = b.side === 'buy';
        const color = isBuy ? '63,185,80' : '248,81,73';

        ctx.beginPath();
        ctx.arc(x, y, r, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(${color},0.65)`;
        ctx.fill();
        ctx.strokeStyle = `rgba(${color},0.90)`;
        ctx.lineWidth = 1;
        ctx.stroke();
      }
    }

    ctx.restore();
  }
}

// ── Table renderer ────────────────────────────────────────────────────────────

export class TapeRenderer {
  private tbody: HTMLTableSectionElement;
  private cvd: CvdTracker;
  private bubbles: BubblesChart;

  constructor(
    tbody: HTMLTableSectionElement,
    cvdCanvas: HTMLCanvasElement,
    bubblesCanvas: HTMLCanvasElement,
  ) {
    this.tbody = tbody;
    this.cvd = new CvdTracker(cvdCanvas);
    this.bubbles = new BubblesChart(bubblesCanvas);
  }

  push(trade: TradeMsg): void {
    this.cvd.push(trade.side, trade.q);
    this.bubbles.push(trade);

    const tr = document.createElement('tr');
    tr.className = 'tape-row ' + trade.side;

    const tdTime = document.createElement('td');
    tdTime.textContent = fmtTime(trade.ts);
    tdTime.className = 'tape-time';

    const tdPrice = document.createElement('td');
    tdPrice.textContent = fmtPrice(trade.p);
    tdPrice.className = 'tape-price';

    const tdQty = document.createElement('td');
    tdQty.textContent = fmtQty(trade.q);
    tdQty.className = 'tape-qty';

    const tdSide = document.createElement('td');
    tdSide.textContent = trade.side === 'buy' ? '▲ BUY' : '▼ SELL';
    tdSide.className = 'tape-side ' + trade.side;

    tr.append(tdTime, tdPrice, tdQty, tdSide);
    this.tbody.insertBefore(tr, this.tbody.firstChild);

    while (this.tbody.rows.length > MAX_ROWS) {
      this.tbody.deleteRow(this.tbody.rows.length - 1);
    }
  }
}
