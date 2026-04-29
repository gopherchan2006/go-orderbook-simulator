export type DepthSnapshotMsg = {
  e: 'depthSnapshot';
  bids: [number, number][];
  asks: [number, number][];
};

export type DepthUpdateMsg = {
  e: 'depthUpdate';
  b: [number, number][];
  a: [number, number][];
};

export type TradeMsg = {
  e: 'trade';
  ts: number;   // unix ms
  p: number;    // price
  q: number;    // qty
  side: 'buy' | 'sell';
  seq: number;
};

export type WSMessage = DepthSnapshotMsg | DepthUpdateMsg | TradeMsg;

export function connect(
  url: string,
  onMessage: (msg: WSMessage) => void,
  onStatus: (s: 'connected' | 'disconnected') => void,
): () => void {
  let stopped = false;
  let ws!: WebSocket;

  function open(): void {
    ws = new WebSocket(url);
    ws.onopen  = () => onStatus('connected');
    ws.onclose = () => {
      onStatus('disconnected');
      if (!stopped) setTimeout(open, 2000);
    };
    ws.onerror = () => ws.close();
    ws.onmessage = (ev: MessageEvent<string>) => {
      try {
        onMessage(JSON.parse(ev.data) as WSMessage);
      } catch {
        // ignore malformed frames
      }
    };
  }

  open();
  return () => { stopped = true; ws?.close(); };
}
