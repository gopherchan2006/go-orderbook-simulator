package main

import "net/http"

func serveUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(uiHTML))
}

const uiHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Order Book Simulator</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    background: #0d0f14;
    color: #c9d1d9;
    font-family: 'Fira Code', 'Cascadia Code', monospace;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 32px 16px;
    min-height: 100vh;
  }

  h1 { font-size: 18px; color: #8b949e; letter-spacing: 2px; margin-bottom: 6px; }

  #status {
    font-size: 11px;
    margin-bottom: 24px;
    padding: 3px 10px;
    border-radius: 10px;
    background: #161b22;
    transition: color .3s;
  }
  #status.ok   { color: #3fb950; }
  #status.err  { color: #f85149; }
  #status.wait { color: #8b949e; }

  #book { width: 360px; }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  thead th {
    font-size: 11px;
    color: #484f58;
    padding: 4px 10px;
    text-align: right;
    border-bottom: 1px solid #21262d;
  }
  thead th:first-child { text-align: left; }

  td {
    padding: 3px 10px;
    font-size: 13px;
    text-align: right;
    white-space: nowrap;
  }
  td:first-child { text-align: left; }

  tr.ask td { background: rgba(248, 81, 73, 0.07); }
  tr.bid td { background: rgba(63, 185, 80, 0.07); }

  tr.ask td.price { color: #f85149; }
  tr.bid td.price { color: #3fb950; }

  @keyframes flash-ask {
    from { background: rgba(248, 81, 73, 0.55); }
    to   { background: rgba(248, 81, 73, 0.07); }
  }
  @keyframes flash-bid {
    from { background: rgba(63, 185, 80, 0.55); }
    to   { background: rgba(63, 185, 80, 0.07); }
  }
  tr.ask.flash td { animation: flash-ask 0.7s ease-out forwards; }
  tr.bid.flash td { animation: flash-bid 0.7s ease-out forwards; }

  #spread {
    text-align: center;
    font-size: 11px;
    color: #484f58;
    padding: 5px 0;
    border-top: 1px solid #21262d;
    border-bottom: 1px solid #21262d;
    margin: 0;
  }

  #asks-section { margin-bottom: 0; }
</style>
</head>
<body>
  <h1>ORDER BOOK SIMULATOR</h1>
  <div id="status" class="wait">connecting…</div>

  <div id="book">
    <table id="asks-section">
      <thead>
        <tr>
          <th>Price (Ask)</th>
          <th>Qty</th>
        </tr>
      </thead>
      <tbody id="asks-body"></tbody>
    </table>

    <div id="spread">— spread —</div>

    <table id="bids-section">
      <tbody id="bids-body"></tbody>
      <thead>
        <tr>
          <th>Price (Bid)</th>
          <th>Qty</th>
        </tr>
      </thead>
    </table>
  </div>

<script>
(function () {
  'use strict';

  const MAX_LEVELS = 15;

  const bids = new Map();
  const asks = new Map();

  const statusEl  = document.getElementById('status');
  const asksBody  = document.getElementById('asks-body');
  const bidsBody  = document.getElementById('bids-body');
  const spreadEl  = document.getElementById('spread');

  function fmtPrice(p) {
    return Number(p).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  }
  function fmtQty(q) {
    return Number(q).toFixed(4);
  }

  function applyLevels(map, levels) {
    const changed = [];
    (levels || []).forEach(function(level) {
      var k = String(level[0]);
      var q = level[1];
      if (q === 0) { map.delete(k); } else { map.set(k, q); }
      changed.push(k);
    });
    return changed;
  }

  function renderSide(tbody, entries, cls, changedSet) {
    tbody.innerHTML = '';
    var shown = entries.slice(0, MAX_LEVELS);
    shown.forEach(function(entry) {
      var price = entry[0];
      var qty   = entry[1];
      var tr = document.createElement('tr');
      tr.className = cls + (changedSet.has(price) ? ' flash' : '');
      var tdPrice = document.createElement('td');
      tdPrice.className = 'price';
      tdPrice.textContent = fmtPrice(price);
      var tdQty = document.createElement('td');
      tdQty.textContent = fmtQty(qty);
      tr.appendChild(tdPrice);
      tr.appendChild(tdQty);
      tbody.appendChild(tr);
    });
  }

  function render(changedBids, changedAsks) {
    var changedBidSet = new Set(changedBids);
    var changedAskSet = new Set(changedAsks);

    var sortedAsks = Array.from(asks.entries()).sort(function(a, b) {
      return Number(b[0]) - Number(a[0]);
    });

    var sortedBids = Array.from(bids.entries()).sort(function(a, b) {
      return Number(b[0]) - Number(a[0]);
    });

    renderSide(asksBody, sortedAsks, 'ask', changedAskSet);
    renderSide(bidsBody, sortedBids, 'bid', changedBidSet);

    if (sortedAsks.length && sortedBids.length) {
      var bestAsk = Number(sortedAsks[sortedAsks.length - 1][0]);
      var bestBid = Number(sortedBids[0][0]);
      var spread  = (bestAsk - bestBid).toFixed(2);
      spreadEl.textContent = 'spread  ' + spread + '  |  best bid ' + fmtPrice(bestBid) + '  ·  best ask ' + fmtPrice(bestAsk);
    } else {
      spreadEl.textContent = '— spread —';
    }
  }

  function connect() {
    var wsUrl = 'ws://' + location.host + '/ws';
    var ws = new WebSocket(wsUrl);

    ws.onopen = function() {
      statusEl.textContent = 'connected';
      statusEl.className = 'ok';
    };

    ws.onclose = function() {
      statusEl.textContent = 'disconnected — reconnecting in 2s…';
      statusEl.className = 'err';
      setTimeout(connect, 2000);
    };

    ws.onerror = function() {
      ws.close();
    };

    ws.onmessage = function(ev) {
      var msg;
      try { msg = JSON.parse(ev.data); } catch(e) { return; }

      var changedBids = [];
      var changedAsks = [];

      if (msg.e === 'depthSnapshot') {
        bids.clear();
        asks.clear();
        (msg.bids || []).forEach(function(l) {
          bids.set(String(l[0]), l[1]);
          changedBids.push(String(l[0]));
        });
        (msg.asks || []).forEach(function(l) {
          asks.set(String(l[0]), l[1]);
          changedAsks.push(String(l[0]));
        });
      } else if (msg.e === 'depthUpdate') {
        changedBids = applyLevels(bids, msg.b);
        changedAsks = applyLevels(asks, msg.a);
      }

      render(changedBids, changedAsks);
    };
  }

  connect();
}());
</script>
</body>
</html>`
