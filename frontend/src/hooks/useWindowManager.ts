import { useState, useCallback, useRef } from "react";

export interface WindowEntry {
  agentId: string;
  name?: string;
  role?: string;
  initialX: number;
  initialY: number;
}

const CASCADE_OFFSET = 30;
const CASCADE_WRAP = 5;
const DEFAULT_WIDTH = 720;
const DEFAULT_HEIGHT = 500;

function cascadePosition(index: number): { x: number; y: number } {
  const slot = index % CASCADE_WRAP;
  const baseX = Math.max(0, (window.innerWidth - DEFAULT_WIDTH) / 2);
  const baseY = Math.max(0, (window.innerHeight - DEFAULT_HEIGHT) / 2);
  return {
    x: baseX + slot * CASCADE_OFFSET,
    y: baseY + slot * CASCADE_OFFSET,
  };
}

export function useWindowManager() {
  const [windows, setWindows] = useState<Map<string, WindowEntry>>(new Map());
  const openCountRef = useRef(0);

  const open = useCallback((agentId: string, name?: string, role?: string) => {
    setWindows((prev) => {
      if (prev.has(agentId)) return prev; // already open — caller should use focus
      const pos = cascadePosition(openCountRef.current);
      openCountRef.current++;
      const next = new Map(prev);
      next.set(agentId, { agentId, name, role, initialX: pos.x, initialY: pos.y });
      return next;
    });
  }, []);

  const close = useCallback((agentId: string) => {
    setWindows((prev) => {
      if (!prev.has(agentId)) return prev;
      const next = new Map(prev);
      next.delete(agentId);
      return next;
    });
  }, []);

  const has = useCallback((agentId: string): boolean => {
    return windows.has(agentId);
  }, [windows]);

  const getWindows = useCallback((): WindowEntry[] => {
    return Array.from(windows.values());
  }, [windows]);

  const count = windows.size;

  return {
    windows,
    count,
    open,
    close,
    has,
    getWindows,
  };
}
