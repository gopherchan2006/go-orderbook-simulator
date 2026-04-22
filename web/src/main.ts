import './app.css';
import { OrderBook } from './orderbook';
import { connect } from './ws';
import { renderTable } from './table';
import { drawDepthChart } from './depth-chart';

const WS_URL = 'ws://localhost:8080/ws';

const ob = new OrderBook();

const statusEl = document.getElementById('status')!;
const asksBody = document.getElementById('asks-body') as HTMLTableSectionElement;
const bidsBody = document.getElementById('bids-body') as HTMLTableSectionElement;
const spreadEl = document.getElementById('spread-bar')!;
const canvas   = document.getElementById('depth-canvas') as HTMLCanvasElement;

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
    } else {
      const { changedBids, changedAsks } = ob.applyUpdate(msg.b, msg.a);
      render(changedBids, changedAsks);
    }
  },
  (status) => {
    statusEl.textContent = status === 'connected'
      ? 'connected'
      : 'disconnected — reconnecting…';
    statusEl.className = 'status ' + (status === 'connected' ? 'ok' : 'err');
  },
);

// Redraw chart whenever the canvas container resizes (responsive layout)
const ro = new ResizeObserver(() => drawDepthChart(canvas, ob.sortedBids(), ob.sortedAsks()));
ro.observe(canvas);

