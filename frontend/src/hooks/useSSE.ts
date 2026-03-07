import { useEffect, useRef, useState } from "react";

export type ConnectionState = "connected" | "reconnecting" | "disconnected";

type EventHandlers = Record<string, (data: unknown) => void>;

export function useSSE(url: string, handlers: EventHandlers): ConnectionState {
  const [state, setState] = useState<ConnectionState>("disconnected");
  const handlersRef = useRef(handlers);
  handlersRef.current = handlers;
  const esRef = useRef<EventSource | null>(null);
  const retryRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    let retryDelay = 1000;
    let closed = false;

    function connect() {
      if (closed) return;
      const es = new EventSource(url);
      esRef.current = es;

      es.onopen = () => {
        setState("connected");
        retryDelay = 1000;
      };

      es.onerror = () => {
        es.close();
        esRef.current = null;
        if (closed) return;
        setState("reconnecting");
        const delay = Math.min(retryDelay, 30000);
        retryDelay = Math.min(retryDelay * 2, 30000);
        retryRef.current = setTimeout(connect, delay);
      };

      const eventTypes = Object.keys(handlersRef.current);
      for (const eventType of eventTypes) {
        es.addEventListener(eventType, (e: MessageEvent) => {
          try {
            const parsed: unknown = JSON.parse(e.data as string);
            handlersRef.current[eventType]?.(parsed);
          } catch {
            // ignore malformed events
          }
        });
      }
    }

    connect();

    return () => {
      closed = true;
      esRef.current?.close();
      if (retryRef.current) clearTimeout(retryRef.current);
      setState("disconnected");
    };
  }, [url]);

  return state;
}
