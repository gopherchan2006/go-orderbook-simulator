import './app.css';
import { OrderBook } from './orderbook';
import { connect } from './ws';
import { renderTable } from './table';
import { drawDepthChart } from './depth-chart';
import { TapeRenderer } from './tape';

const WS_URL = 'ws://localhost:8080/ws';

const ob = new OrderBook();

const statusEl  = document.getElementById('status')!;
const asksBody  = document.getElementById('asks-body') as HTMLTableSectionElement;
const bidsBody  = document.getElementById('bids-body') as HTMLTableSectionElement;
const spreadEl  = document.getElementById('spread-bar')!;
const canvas    = document.getElementById('depth-canvas') as HTMLCanvasElement;
const tapeBody  = document.getElementById('tape-body') as HTMLTableSectionElement;
const cvdCanvas = document.getElementById('cvd-canvas') as HTMLCanvasElement;
const tape      = new TapeRenderer(tapeBody, cvdCanvas);

function render(
  changedBids = new Set<string>(),
  changedAsks = new Set<string>(),
): void {
  const bids = ob.sortedBids();
  const asks = ob.sortedAsks();
  renderTable(asksBody, bidsBody, spreadEl, bids, asks, changedBids, changedAsks);
  drawDepthChart(canvas, bids, asks);
}

connect(
  WS_URL,
  (msg) => {
    if (msg.e === 'depthSnapshot') {
      ob.applySnapshot(msg.bids, msg.asks);
      render();
    } else if (msg.e === 'depthUpdate') {
      const { changedBids, changedAsks } = ob.applyUpdate(msg.b, msg.a);
      render(changedBids, changedAsks);
    } else if (msg.e === 'trade') {
      tape.push(msg);
    }
  },
  (status) => {
    statusEl.textContent = status === 'connected'
      ? 'connected'
      : 'disconnected — reconnecting…';
    statusEl.className = 'status ' + (status === 'connected' ? 'ok' : 'err');
  },
);

const ro = new ResizeObserver(() => {
  drawDepthChart(canvas, ob.sortedBids(), ob.sortedAsks());
});
ro.observe(canvas);
