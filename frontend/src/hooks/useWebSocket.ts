import { createSignal, onCleanup, Accessor } from 'solid-js';
import { FlowForgeWebSocket } from '../api/websocket';

/**
 * useWebSocket - WebSocket connection hook with auto-cleanup.
 */
export function useWebSocket(url: string | Accessor<string>) {
  const [connected, setConnected] = createSignal(false);
  const [lastMessage, setLastMessage] = createSignal<any>(null);

  const resolvedUrl = typeof url === 'function' ? url() : url;
  const ws = new FlowForgeWebSocket(resolvedUrl);

  ws.on('connected', () => setConnected(true));
  ws.on('disconnected', () => setConnected(false));
  ws.on('message', (data: any) => setLastMessage(data));

  ws.connect();

  const send = (data: any) => ws.send(data);
  const close = () => ws.close();

  onCleanup(() => ws.close());

  return { connected, lastMessage, send, close, ws };
}
