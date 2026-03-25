import { getAccessToken } from './client';

// ---------------------------------------------------------------------------
// WebSocket event types
// ---------------------------------------------------------------------------
export interface WsMessage {
  type: string;
  payload: unknown;
}

export type WsEventHandler = (payload: unknown) => void;

// ---------------------------------------------------------------------------
// FlowForge WebSocket Client
// ---------------------------------------------------------------------------
export class FlowForgeWebSocket {
  private url: string;
  private ws: WebSocket | null = null;
  private handlers = new Map<string, Set<WsEventHandler>>();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private intentionallyClosed = false;

  constructor(path: string) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    this.url = `${protocol}//${window.location.host}${path}`;
  }

  connect(): void {
    this.intentionallyClosed = false;
    this.reconnectAttempts = 0;
    this.doConnect();
  }

  private doConnect(): void {
    const token = getAccessToken();
    const urlWithAuth = token ? `${this.url}?token=${encodeURIComponent(token)}` : this.url;

    this.ws = new WebSocket(urlWithAuth);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.emit('_connected', null);
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        // Backend sends flat JSON with a "type" field, not nested {type, payload}
        if (data.type) {
          const { type, ...rest } = data;
          this.emit(type, rest);
        } else {
          this.emit('message', data);
        }
      } catch {
        // Raw text messages (e.g., log lines)
        this.emit('message', event.data);
      }
    };

    this.ws.onerror = () => {
      this.emit('_error', null);
    };

    this.ws.onclose = () => {
      this.emit('_disconnected', null);
      if (!this.intentionallyClosed) {
        this.scheduleReconnect();
      }
    };
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.emit('_max_reconnect', null);
      return;
    }

    // Exponential backoff: 1s, 2s, 4s, 8s... up to 30s
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;

    this.reconnectTimer = setTimeout(() => {
      this.doConnect();
    }, delay);
  }

  disconnect(): void {
    this.intentionallyClosed = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  send(type: string, payload: unknown): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, payload }));
    }
  }

  on(event: string, handler: WsEventHandler): () => void {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, new Set());
    }
    this.handlers.get(event)!.add(handler);

    // Return unsubscribe function
    return () => {
      this.handlers.get(event)?.delete(handler);
    };
  }

  off(event: string, handler: WsEventHandler): void {
    this.handlers.get(event)?.delete(handler);
  }

  private emit(event: string, payload: unknown): void {
    const handlers = this.handlers.get(event);
    if (handlers) {
      handlers.forEach((h) => h(payload));
    }
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// ---------------------------------------------------------------------------
// Convenience: create a log-streaming WebSocket for a run
// ---------------------------------------------------------------------------
export function createRunLogSocket(runId: string): FlowForgeWebSocket {
  return new FlowForgeWebSocket(`/ws/runs/${runId}/logs`);
}

// ---------------------------------------------------------------------------
// Global event socket singleton
// ---------------------------------------------------------------------------
let eventSocket: FlowForgeWebSocket | null = null;

export function getEventSocket(): FlowForgeWebSocket {
  if (!eventSocket) {
    eventSocket = new FlowForgeWebSocket('/ws/events');
  }
  return eventSocket;
}

export function connectEventSocket(): void {
  getEventSocket().connect();
}

export function disconnectEventSocket(): void {
  if (eventSocket) {
    eventSocket.disconnect();
    eventSocket = null;
  }
}
