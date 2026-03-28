import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock FlowForgeWebSocket
vi.mock('../api/websocket', () => {
  const mockWs = {
    on: vi.fn().mockReturnValue(vi.fn()),
    connect: vi.fn(),
    close: vi.fn(),
    send: vi.fn(),
    disconnect: vi.fn(),
    isConnected: false,
  };
  return {
    FlowForgeWebSocket: vi.fn(() => mockWs),
  };
});

describe('useWebSocket', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('creates a WebSocket connection', async () => {
    const { FlowForgeWebSocket } = await import('../api/websocket');
    const { useWebSocket } = await import('./useWebSocket');
    const { createRoot } = await import('solid-js');

    createRoot((dispose) => {
      const result = useWebSocket('/ws/test');
      expect(FlowForgeWebSocket).toHaveBeenCalledWith('/ws/test');
      expect(result).toHaveProperty('connected');
      expect(result).toHaveProperty('lastMessage');
      expect(result).toHaveProperty('send');
      expect(result).toHaveProperty('close');
      expect(result).toHaveProperty('ws');
      dispose();
    });
  });

  it('starts disconnected', async () => {
    const { useWebSocket } = await import('./useWebSocket');
    const { createRoot } = await import('solid-js');

    createRoot((dispose) => {
      const { connected } = useWebSocket('/ws/test');
      expect(connected()).toBe(false);
      dispose();
    });
  });

  it('starts with null lastMessage', async () => {
    const { useWebSocket } = await import('./useWebSocket');
    const { createRoot } = await import('solid-js');

    createRoot((dispose) => {
      const { lastMessage } = useWebSocket('/ws/test');
      expect(lastMessage()).toBeNull();
      dispose();
    });
  });

  it('registers event handlers', async () => {
    const { FlowForgeWebSocket } = await import('../api/websocket');
    const { useWebSocket } = await import('./useWebSocket');
    const { createRoot } = await import('solid-js');

    createRoot((dispose) => {
      useWebSocket('/ws/test');
      const mockInstance = (FlowForgeWebSocket as any).mock.results[0]?.value;
      if (mockInstance) {
        expect(mockInstance.on).toHaveBeenCalledWith('connected', expect.any(Function));
        expect(mockInstance.on).toHaveBeenCalledWith('disconnected', expect.any(Function));
        expect(mockInstance.on).toHaveBeenCalledWith('message', expect.any(Function));
      }
      dispose();
    });
  });

  it('calls connect on initialization', async () => {
    const { FlowForgeWebSocket } = await import('../api/websocket');
    const { useWebSocket } = await import('./useWebSocket');
    const { createRoot } = await import('solid-js');

    createRoot((dispose) => {
      useWebSocket('/ws/events');
      const mockInstance = (FlowForgeWebSocket as any).mock.results[0]?.value;
      if (mockInstance) {
        expect(mockInstance.connect).toHaveBeenCalled();
      }
      dispose();
    });
  });
});
