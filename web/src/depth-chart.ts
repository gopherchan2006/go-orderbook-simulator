import type { Level } from './orderbook';

// ─── Layout ────────────────────────────────────────────────────────────────
const PAD = { top: 28, right: 56, bottom: 36, left: 56 };

// ─── Colours ───────────────────────────────────────────────────────────────
const C = {
  bid:    { stroke: '#3fb950', fill: 'rgba(63,185,80,0.15)' },
  ask:    { stroke: '#f85149', fill: 'rgba(248,81,73,0.15)' },
  axis:   '#484f58',
  text:   '#8b949e',
  spread: '#e3b341',
  grid:   'rgba(72,79,88,0.25)',
  bg:     '#0d0f14',
} as const;

// ─── Step-area helper ──────────────────────────────────────────────────────
/**
 * Draws a filled staircase curve.
 *
 * `points` must be sorted ascending by price.
 * For bids the cumulative volume decreases left→right.
 * For asks the cumulative volume increases left→right.
 * Both produce the correct depth-chart shape.
 */
function drawStepArea(
  ctx: CanvasRenderingContext2D,
  points: [number, number][],
  xMap: (p: number) => number,
  yMap: (v: number) => number,
  color: { stroke: string; fill: string },
): void {
  if (points.length === 0) return;

  // ── Fill path ────────────────────────────────────────────────────────────
  ctx.beginPath();
  ctx.moveTo(xMap(points[0][0]), yMap(0));
  ctx.lineTo(xMap(points[0][0]), yMap(points[0][1]));

  for (let i = 0; i < points.length - 1; i++) {
    ctx.lineTo(xMap(points[i + 1][0]), yMap(points[i][1]));      // horizontal
    ctx.lineTo(xMap(points[i + 1][0]), yMap(points[i + 1][1])); // vertical
  }

  ctx.lineTo(xMap(points[points.length - 1][0]), yMap(0));
  ctx.closePath();
  ctx.fillStyle = color.fill;
  ctx.fill();

  // ── Stroke (step line only, no baseline) ─────────────────────────────────
  ctx.beginPath();
  ctx.moveTo(xMap(points[0][0]), yMap(points[0][1]));
  for (let i = 0; i < points.length - 1; i++) {
    ctx.lineTo(xMap(points[i + 1][0]), yMap(points[i][1]));
    ctx.lineTo(xMap(points[i + 1][0]), yMap(points[i + 1][1]));
  }
  ctx.strokeStyle = color.stroke;
  ctx.lineWidth   = 2;
  ctx.stroke();
}

// ─── Public API ────────────────────────────────────────────────────────────
export function drawDepthChart(
  canvas: HTMLCanvasElement,
  bids: Level[],
  asks: Level[],
): void {
  const dpr = window.devicePixelRatio || 1;
  const w   = canvas.clientWidth;
  const h   = canvas.clientHeight;
  if (w === 0 || h === 0) return;

  // Resize backing buffer only when needed (avoids reallocation on every frame)
  const targetW = Math.round(w * dpr);
  const targetH = Math.round(h * dpr);
  if (canvas.width !== targetW || canvas.height !== targetH) {
    canvas.width  = targetW;
    canvas.height = targetH;
  }

  const ctx = canvas.getContext('2d')!;
  ctx.save();
  ctx.scale(dpr, dpr);
  ctx.clearRect(0, 0, w, h);

  // Background
  ctx.fillStyle = C.bg;
  ctx.fillRect(0, 0, w, h);

  if (bids.length === 0 && asks.length === 0) {
    ctx.restore();
    return;
  }

  // ── Cumulative bid points ─────────────────────────────────────────────────
  // Accumulate from best bid (highest price) outward, then sort ASC for drawing.
  const bidsSortedDesc = [...bids].sort((a, b) => b[0] - a[0]);
  let bidCum = 0;
  const bidPoints = bidsSortedDesc
    .map(([p, q]): [number, number] => { bidCum += q; return [p, bidCum]; })
    .sort((a, b) => a[0] - b[0]);

  // ── Cumulative ask points ─────────────────────────────────────────────────
  // Accumulate from best ask (lowest price) outward; already ascending.
  const asksSortedAsc = [...asks].sort((a, b) => a[0] - b[0]);
  let askCum = 0;
  const askPoints = asksSortedAsc.map(([p, q]): [number, number] => {
    askCum += q;
    return [p, askCum];
  });

  const maxCum = Math.max(bidCum, askCum);
  if (maxCum === 0) { ctx.restore(); return; }

  // ── Coordinate mapping ────────────────────────────────────────────────────
  const minPrice = bidPoints.length  > 0 ? bidPoints[0][0]                          : askPoints[0][0];
  const maxPrice = askPoints.length  > 0 ? askPoints[askPoints.length - 1][0]       : bidPoints[bidPoints.length - 1][0];
  const priceRange = maxPrice - minPrice || 1;

  const chartW = w - PAD.left - PAD.right;
  const chartH = h - PAD.top  - PAD.bottom;

  const xMap = (price: number) => PAD.left + ((price - minPrice) / priceRange) * chartW;
  const yMap = (vol:   number) => PAD.top  + chartH - (vol / maxCum) * chartH;

  // ── Horizontal grid ───────────────────────────────────────────────────────
  ctx.strokeStyle = C.grid;
  ctx.lineWidth   = 1;
  ctx.setLineDash([4, 4]);
  for (let i = 1; i <= 4; i++) {
    const y = PAD.top + (chartH / 4) * i;
    ctx.beginPath();
    ctx.moveTo(PAD.left, y);
    ctx.lineTo(PAD.left + chartW, y);
    ctx.stroke();
  }
  ctx.setLineDash([]);

  // ── Step curves ───────────────────────────────────────────────────────────
  drawStepArea(ctx, bidPoints, xMap, yMap, C.bid);
  drawStepArea(ctx, askPoints, xMap, yMap, C.ask);

  // ── Axes ──────────────────────────────────────────────────────────────────
  ctx.strokeStyle = C.axis;
  ctx.lineWidth   = 1;
  ctx.beginPath();
  ctx.moveTo(PAD.left, PAD.top);
  ctx.lineTo(PAD.left, PAD.top + chartH);
  ctx.lineTo(PAD.left + chartW, PAD.top + chartH);
  ctx.stroke();

  // ── X labels (price) ─────────────────────────────────────────────────────
  ctx.fillStyle   = C.text;
  ctx.font        = '10px monospace';
  ctx.textAlign   = 'center';
  for (let i = 0; i <= 5; i++) {
    const price = minPrice + (priceRange * i) / 5;
    ctx.fillText(
      price.toLocaleString('en-US', { maximumFractionDigits: 0 }),
      xMap(price),
      PAD.top + chartH + 18,
    );
  }

  // ── Y labels (cumulative volume) ─────────────────────────────────────────
  ctx.textAlign = 'right';
  for (let i = 0; i <= 4; i++) {
    const vol = (maxCum * i) / 4;
    ctx.fillText(vol.toFixed(1), PAD.left - 6, yMap(vol) + 4);
  }

  // ── Spread marker ─────────────────────────────────────────────────────────
  if (bids.length > 0 && asks.length > 0) {
    const bestBid = bidsSortedDesc[0][0];
    const bestAsk = asksSortedAsc[0][0];
    const midX    = xMap((bestBid + bestAsk) / 2);

    ctx.strokeStyle = 'rgba(227,179,65,0.25)';
    ctx.lineWidth   = 1;
    ctx.setLineDash([4, 4]);
    ctx.beginPath();
    ctx.moveTo(midX, PAD.top + 20);
    ctx.lineTo(midX, PAD.top + chartH);
    ctx.stroke();
    ctx.setLineDash([]);

    ctx.fillStyle   = C.spread;
    ctx.font        = '10px monospace';
    ctx.textAlign   = 'center';
    ctx.fillText(`spread ${(bestAsk - bestBid).toFixed(2)}`, midX, PAD.top + 14);
  }

  // ── Side labels ───────────────────────────────────────────────────────────
  ctx.font      = '10px monospace';
  ctx.fillStyle = C.bid.stroke;
  ctx.textAlign = 'left';
  ctx.fillText('◀ BIDS', PAD.left + 6, PAD.top + 16);
  ctx.fillStyle = C.ask.stroke;
  ctx.textAlign = 'right';
  ctx.fillText('ASKS ▶', PAD.left + chartW - 6, PAD.top + 16);

  ctx.restore();
}
