import { useState, useEffect, useRef, useCallback } from "react";

export type WSConnectionState = "connecting" | "connected" | "disconnected" | "reconnecting";

export interface WSMessage {
  type: "output" | "input" | "status" | "error";
  text: string;
  timestamp: Date;
}

interface ServerMessage {
  type: string;
  text?: string;
  status?: string;
  message?: string;
  exit_code?: number;
}

export function useAgentWebSocket(agentId: string | null) {
  const [messages, setMessages] = useState<WSMessage[]>([]);
  const [status, setStatus] = useState<WSConnectionState>("disconnected");
  const [agentStatus, setAgentStatus] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const retryRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const retryDelayRef = useRef(1000);

  const connect = useCallback(() => {
    if (!agentId) return;

    setStatus("connecting");
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws/agent/${encodeURIComponent(agentId)}`);
    wsRef.current = ws;

    ws.onopen = () => {
      setStatus("connected");
      retryDelayRef.current = 1000;
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data as string) as ServerMessage;
        const now = new Date();

        switch (msg.type) {
          case "output":
            setMessages((prev) => [...prev, { type: "output", text: msg.text ?? "", timestamp: now }]);
            break;
          case "status":
            setAgentStatus(msg.status ?? null);
            if (msg.status === "completed" || msg.status === "failed") {
              setMessages((prev) => [
                ...prev,
                {
                  type: "status",
                  text: msg.status === "completed"
                    ? `Agent exited (code ${msg.exit_code ?? 0})`
                    : "Agent failed",
                  timestamp: now,
                },
              ]);
            }
            break;
          case "error":
            setMessages((prev) => [...prev, { type: "error", text: msg.message ?? "Unknown error", timestamp: now }]);
            break;
        }
      } catch {
        // ignore malformed messages
      }
    };

    ws.onclose = () => {
      wsRef.current = null;
      if (agentStatus === "completed" || agentStatus === "failed") {
        setStatus("disconnected");
        return;
      }
      setStatus("reconnecting");
      const delay = Math.min(retryDelayRef.current, 10000);
      retryDelayRef.current = Math.min(retryDelayRef.current * 2, 10000);
      retryRef.current = setTimeout(connect, delay);
    };

    ws.onerror = () => {
      // onclose will fire after onerror
    };
  }, [agentId, agentStatus]);

  useEffect(() => {
    if (!agentId) return;

    setMessages([]);
    setAgentStatus(null);
    connect();

    return () => {
      if (retryRef.current) clearTimeout(retryRef.current);
      wsRef.current?.close();
      wsRef.current = null;
      setStatus("disconnected");
    };
  }, [agentId]); // eslint-disable-line react-hooks/exhaustive-deps

  const sendMessage = useCallback(
    (text: string) => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
      wsRef.current.send(JSON.stringify({ type: "input", text }));
      setMessages((prev) => [...prev, { type: "input", text, timestamp: new Date() }]);
    },
    [],
  );

  return { messages, sendMessage, status, agentStatus };
}
