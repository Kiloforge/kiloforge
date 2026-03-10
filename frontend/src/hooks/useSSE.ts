import { useEffect, useRef, useState } from "react";
import { showToast } from "../api/errorToast";

export type ConnectionState = "connected" | "reconnecting" | "disconnected";

type EventHandlers = Record<string, (data: unknown) => void>;

export function useSSE(url: string, handlers: EventHandlers): ConnectionState {
  const [state, setState] = useState<ConnectionState>("disconnected");
  const handlersRef = useRef(handlers);
  useEffect(() => {
    handlersRef.current = handlers;
  }, [handlers]);
  const esRef = useRef<EventSource | null>(null);
  const retryRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wasConnectedRef = useRef(false);

  useEffect(() => {
    let retryDelay = 1000;
    let closed = false;

    function connect() {
      if (closed) return;
      const es = new EventSource(url);
      esRef.current = es;

      es.onopen = () => {
        setState("connected");
        wasConnectedRef.current = true;
        retryDelay = 1000;
      };

      es.onerror = () => {
        es.close();
        esRef.current = null;
        if (closed) return;
        setState("reconnecting");
        if (wasConnectedRef.current) {
          showToast("warning", "Server connection lost", "Attempting to reconnect...");
          wasConnectedRef.current = false;
        }
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
          } catch (err) {
            console.warn("[SSE] Failed to parse event:", eventType, err);
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
