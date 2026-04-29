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

// Rolling CVD (Cumulative Volume Delta) tracker
export class CvdTracker {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;
  private history: number[] = []; // cumulative deltas, one per trade
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

    // zero line
    const zeroY = toY(0);
    ctx.strokeStyle = 'rgba(255,255,255,0.12)';
    ctx.lineWidth = 1;
    ctx.setLineDash([4, 4]);
    ctx.beginPath();
    ctx.moveTo(0, zeroY);
    ctx.lineTo(W, zeroY);
    ctx.stroke();
    ctx.setLineDash([]);

    // fill area under curve
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

    // line
    ctx.beginPath();
    ctx.moveTo(toX(0), toY(history[0]));
    for (let i = 1; i < history.length; i++) {
      ctx.lineTo(toX(i), toY(history[i]));
    }
    ctx.strokeStyle = '#00e5ff';
    ctx.lineWidth = 1.5;
    ctx.stroke();

    // current CVD label
    const last = history[history.length - 1];
    ctx.fillStyle = last >= 0 ? '#3fb950' : '#f85149';
    ctx.font = 'bold 11px monospace';
    ctx.fillText(`CVD ${last >= 0 ? '+' : ''}${last.toFixed(3)}`, 6, 14);
  }
}

export class TapeRenderer {
  private tbody: HTMLTableSectionElement;
  private cvd: CvdTracker;

  constructor(tbody: HTMLTableSectionElement, cvdCanvas: HTMLCanvasElement) {
    this.tbody = tbody;
    this.cvd = new CvdTracker(cvdCanvas);
  }

  push(trade: TradeMsg): void {
    this.cvd.push(trade.side, trade.q);

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

    // prepend so newest is on top
    this.tbody.insertBefore(tr, this.tbody.firstChild);

    // trim to MAX_ROWS
    while (this.tbody.rows.length > MAX_ROWS) {
      this.tbody.deleteRow(this.tbody.rows.length - 1);
    }
  }
}
