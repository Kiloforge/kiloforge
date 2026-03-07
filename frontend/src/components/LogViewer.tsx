import { useState, useEffect, useRef, useCallback } from "react";
import type { LogResponse } from "../types/api";
import styles from "./LogViewer.module.css";

interface Props {
  agentId: string;
  onClose: () => void;
}

export function LogViewer({ agentId, onClose }: Props) {
  const [lines, setLines] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [following, setFollowing] = useState(false);
  const preRef = useRef<HTMLPreElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    setLoading(true);
    fetch(`/-/api/agents/${encodeURIComponent(agentId)}/log?lines=200`)
      .then((r) => r.json())
      .then((data: LogResponse) => {
        setLines(data.lines || []);
        setLoading(false);
        requestAnimationFrame(() => {
          if (preRef.current) {
            preRef.current.scrollTop = preRef.current.scrollHeight;
          }
        });
      })
      .catch(() => {
        setLines(["Failed to load log."]);
        setLoading(false);
      });
  }, [agentId]);

  useEffect(() => {
    if (!following) {
      eventSourceRef.current?.close();
      eventSourceRef.current = null;
      return;
    }

    const es = new EventSource(
      `/-/api/agents/${encodeURIComponent(agentId)}/log?lines=200&follow=true`,
    );
    eventSourceRef.current = es;

    es.onmessage = (e) => {
      setLines((prev) => [...prev, e.data as string]);
      requestAnimationFrame(() => {
        if (preRef.current) {
          preRef.current.scrollTop = preRef.current.scrollHeight;
        }
      });
    };

    es.onerror = () => {
      es.close();
      setFollowing(false);
    };

    return () => {
      es.close();
    };
  }, [following, agentId]);

  const handleBackdropClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === e.currentTarget) onClose();
    },
    [onClose],
  );

  useEffect(() => {
    return () => {
      eventSourceRef.current?.close();
    };
  }, []);

  return (
    <div className={styles.overlay} onClick={handleBackdropClick}>
      <div className={styles.modal}>
        <div className={styles.header}>
          <h3>
            Log: <span className={styles.agentId}>{agentId}</span>
          </h3>
          <div className={styles.controls}>
            <label className={styles.followToggle}>
              <input
                type="checkbox"
                checked={following}
                onChange={(e) => setFollowing(e.target.checked)}
              />
              Follow
            </label>
            <button className={styles.closeBtn} onClick={onClose}>
              &times;
            </button>
          </div>
        </div>
        <pre ref={preRef} className={styles.viewer}>
          {loading ? "Loading..." : lines.join("\n") || "No log data available."}
        </pre>
      </div>
    </div>
  );
}
