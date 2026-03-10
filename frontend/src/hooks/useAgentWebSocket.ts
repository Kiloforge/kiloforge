import { useState, useEffect, useRef, useCallback } from "react";

export type WSConnectionState = "connecting" | "connected" | "disconnected" | "reconnecting";

export type WSMessageType =
  | "output" | "input" | "status" | "error"
  | "turn_start" | "text" | "tool_use" | "thinking" | "turn_end" | "system";

export interface WSUsageInfo {
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
}

export interface WSMessage {
  type: WSMessageType;
  text: string;
  timestamp: Date;
  turnId?: string;
  toolName?: string;
  toolId?: string;
  toolInput?: Record<string, unknown>;
  thinking?: string;
  costUsd?: number;
  usage?: WSUsageInfo;
  subtype?: string;
  data?: Record<string, unknown>;
}

interface ServerMessage {
  type: string;
  text?: string;
  status?: string;
  message?: string;
  exit_code?: number;
  turn_id?: string;
  tool_name?: string;
  tool_id?: string;
  input?: Record<string, unknown>;
  thinking?: string;
  cost_usd?: number;
  usage?: WSUsageInfo;
  subtype?: string;
  data?: Record<string, unknown>;
}

/** Terminal agent statuses that should stop reconnection attempts. */
const TERMINAL_STATUSES = new Set([
  "completed", "failed", "stopped", "force-killed", "resume-failed", "replaced",
]);

/** Maximum number of consecutive reconnect attempts before giving up. */
const MAX_RECONNECT_ATTEMPTS = 10;

export function useAgentWebSocket(agentId: string | null) {
  const [messages, setMessages] = useState<WSMessage[]>([]);
  const [status, setStatus] = useState<WSConnectionState>("disconnected");
  const [agentStatus, setAgentStatus] = useState<string | null>(null);
  const agentStatusRef = useRef(agentStatus);
  const wsRef = useRef<WebSocket | null>(null);
  const retryRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const retryDelayRef = useRef(1000);
  const retryCountRef = useRef(0);

  useEffect(() => {
    agentStatusRef.current = agentStatus;
  }, [agentStatus]);

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
        // Successfully received a message — reset reconnect counter.
        retryCountRef.current = 0;
        // Defensive: coerce text fields to string in case server sends unexpected types
        const safeText = typeof msg.text === "string" ? msg.text : String(msg.text ?? "");
        const safeThinking = typeof msg.thinking === "string" ? msg.thinking : String(msg.thinking ?? "");

        switch (msg.type) {
          case "output":
            // Backward compat: treat legacy output as text
            setMessages((prev) => [...prev, { type: "text", text: safeText, timestamp: now }]);
            break;
          case "text":
            setMessages((prev) => [...prev, {
              type: "text",
              text: safeText,
              turnId: msg.turn_id,
              timestamp: now,
            }]);
            break;
          case "turn_start":
            setMessages((prev) => [...prev, {
              type: "turn_start",
              text: "",
              turnId: msg.turn_id,
              timestamp: now,
            }]);
            break;
          case "turn_end":
            setMessages((prev) => [...prev, {
              type: "turn_end",
              text: "",
              turnId: msg.turn_id,
              costUsd: msg.cost_usd,
              usage: msg.usage,
              timestamp: now,
            }]);
            break;
          case "tool_use":
            setMessages((prev) => [...prev, {
              type: "tool_use",
              text: msg.tool_name ?? "",
              turnId: msg.turn_id,
              toolName: msg.tool_name,
              toolId: msg.tool_id,
              toolInput: msg.input,
              timestamp: now,
            }]);
            break;
          case "thinking":
            setMessages((prev) => [...prev, {
              type: "thinking",
              text: "",
              thinking: safeThinking,
              turnId: msg.turn_id,
              timestamp: now,
            }]);
            break;
          case "system":
            setMessages((prev) => [...prev, {
              type: "system",
              text: "",
              subtype: msg.subtype,
              data: msg.data,
              timestamp: now,
            }]);
            break;
          case "status":
            setAgentStatus(msg.status ?? null);
            if (msg.status && TERMINAL_STATUSES.has(msg.status)) {
              setMessages((prev) => [
                ...prev,
                {
                  type: "status",
                  text: msg.status === "completed"
                    ? `Agent exited (code ${msg.exit_code ?? 0})`
                    : `Agent ${msg.status}`,
                  timestamp: now,
                },
              ]);
            }
            break;
          case "error":
            setMessages((prev) => [...prev, { type: "error", text: msg.message ?? "Unknown error", timestamp: now }]);
            break;
        }
      } catch (err) {
        console.warn("[WebSocket] Failed to parse message:", err);
      }
    };

    ws.onclose = () => {
      wsRef.current = null;

      // If we already know the agent is in a terminal state, stop.
      if (agentStatusRef.current && TERMINAL_STATUSES.has(agentStatusRef.current)) {
        setStatus("disconnected");
        return;
      }

      // Max retry limit reached — give up.
      if (retryCountRef.current >= MAX_RECONNECT_ATTEMPTS) {
        console.warn(`[WebSocket] Max reconnect attempts (${MAX_RECONNECT_ATTEMPTS}) reached for agent ${agentId}`);
        setStatus("disconnected");
        return;
      }

      // If agentStatus is null (never received a status message), the WS
      // may have closed before the status arrived. Check via REST API.
      if (agentStatusRef.current == null && agentId) {
        fetch(`/api/agents/${encodeURIComponent(agentId)}`)
          .then((res) => (res.ok ? res.json() : null))
          .then((data: { status?: string } | null) => {
            if (data?.status && TERMINAL_STATUSES.has(data.status)) {
              setAgentStatus(data.status);
              setStatus("disconnected");
              setMessages((prev) => [
                ...prev,
                { type: "status", text: `Agent ${data.status}`, timestamp: new Date() },
              ]);
              return;
            }
            // Not terminal — schedule reconnect.
            scheduleReconnect();
          })
          .catch(() => {
            // REST also failed — schedule reconnect anyway.
            scheduleReconnect();
          });
        return;
      }

      // Agent is in a non-terminal state — schedule reconnect.
      scheduleReconnect();
    };

    ws.onerror = () => {
      // onclose will fire after onerror
    };

    function scheduleReconnect() {
      setStatus("reconnecting");
      retryCountRef.current += 1;
      const delay = Math.min(retryDelayRef.current, 10000);
      retryDelayRef.current = Math.min(retryDelayRef.current * 2, 10000);
      retryRef.current = setTimeout(connect, delay);
    }
  }, [agentId]);

  useEffect(() => {
    if (!agentId) return;

    setMessages([]);
    setAgentStatus(null);
    retryCountRef.current = 0;
    connect();

    return () => {
      if (retryRef.current) clearTimeout(retryRef.current);
      wsRef.current?.close();
      wsRef.current = null;
      setStatus("disconnected");
    };
  }, [agentId, connect]);

  const sendMessage = useCallback(
    (text: string) => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
      wsRef.current.send(JSON.stringify({ type: "input", text }));
      setMessages((prev) => [...prev, { type: "input", text, timestamp: new Date() }]);
    },
    [],
  );

  const clearMessages = useCallback(() => {
    setMessages([]);
  }, []);

  return { messages, sendMessage, clearMessages, status, agentStatus };
}
