import { useState, useEffect, useCallback, useRef } from 'react';
import { Platform } from 'react-native';

const WS_URL = __DEV__
  ? Platform.select({
      ios: 'ws://localhost:8007/ws',
      android: 'ws://10.0.2.2:8007/ws',
      default: 'ws://localhost:8007/ws',
    })
  : 'wss://api.equishare.com/ws';

export interface QuoteData {
  symbol: string;
  bid_price: number;
  bid_size: number;
  ask_price: number;
  ask_size: number;
  mid_price: number;
  spread: number;
  timestamp: string;
}

interface WebSocketMessage {
  type: 'quote' | 'trade' | 'bar' | 'error' | 'pong';
  symbol?: string;
  data?: QuoteData;
  error?: string;
  timestamp: string;
}

type QuoteCallback = (quote: QuoteData) => void;

interface UseWebSocketReturn {
  isConnected: boolean;
  subscribe: (symbols: string[]) => void;
  unsubscribe: (symbols: string[]) => void;
  quotes: Map<string, QuoteData>;
  onQuote: (callback: QuoteCallback) => () => void;
}

export function useWebSocket(): UseWebSocketReturn {
  const [isConnected, setIsConnected] = useState(false);
  const [quotes, setQuotes] = useState<Map<string, QuoteData>>(new Map());
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const callbacksRef = useRef<Set<QuoteCallback>>(new Set());
  const subscribedSymbolsRef = useRef<Set<string>>(new Set());

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    const ws = new WebSocket(WS_URL!);
    wsRef.current = ws;

    ws.onopen = () => {
      setIsConnected(true);
      if (subscribedSymbolsRef.current.size > 0) {
        ws.send(JSON.stringify({
          type: 'subscribe',
          symbols: Array.from(subscribedSymbolsRef.current),
        }));
      }
    };

    ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);

        if (message.type === 'quote' && message.data) {
          const quote = message.data;
          setQuotes((prev) => {
            const next = new Map(prev);
            next.set(quote.symbol, quote);
            return next;
          });

          callbacksRef.current.forEach((callback) => {
            callback(quote);
          });
        }
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    ws.onclose = () => {
      setIsConnected(false);
      reconnectTimeoutRef.current = setTimeout(() => {
        connect();
      }, 3000);
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }, []);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  useEffect(() => {
    connect();
    return () => disconnect();
  }, [connect, disconnect]);

  const subscribe = useCallback((symbols: string[]) => {
    symbols.forEach((s) => subscribedSymbolsRef.current.add(s.toUpperCase()));

    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: 'subscribe',
        symbols: symbols.map((s) => s.toUpperCase()),
      }));
    }
  }, []);

  const unsubscribe = useCallback((symbols: string[]) => {
    symbols.forEach((s) => subscribedSymbolsRef.current.delete(s.toUpperCase()));

    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: 'unsubscribe',
        symbols: symbols.map((s) => s.toUpperCase()),
      }));
    }
  }, []);

  const onQuote = useCallback((callback: QuoteCallback) => {
    callbacksRef.current.add(callback);
    return () => {
      callbacksRef.current.delete(callback);
    };
  }, []);

  return {
    isConnected,
    subscribe,
    unsubscribe,
    quotes,
    onQuote,
  };
}

export default useWebSocket;
