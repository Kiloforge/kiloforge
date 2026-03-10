import { useState, useCallback, useRef } from "react";

export interface WindowEntry {
  agentId: string;
  name?: string;
  role?: string;
  initialX: number;
  initialY: number;
  minimized: boolean;
  unreadCount: number;
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
      if (prev.has(agentId)) return prev;
      const pos = cascadePosition(openCountRef.current);
      openCountRef.current++;
      const next = new Map(prev);
      next.set(agentId, {
        agentId, name, role,
        initialX: pos.x, initialY: pos.y,
        minimized: false, unreadCount: 0,
      });
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

  const minimize = useCallback((agentId: string) => {
    setWindows((prev) => {
      const entry = prev.get(agentId);
      if (!entry || entry.minimized) return prev;
      const next = new Map(prev);
      next.set(agentId, { ...entry, minimized: true });
      return next;
    });
  }, []);

  const restore = useCallback((agentId: string) => {
    setWindows((prev) => {
      const entry = prev.get(agentId);
      if (!entry || !entry.minimized) return prev;
      const next = new Map(prev);
      next.set(agentId, { ...entry, minimized: false, unreadCount: 0 });
      return next;
    });
  }, []);

  const incrementUnread = useCallback((agentId: string) => {
    setWindows((prev) => {
      const entry = prev.get(agentId);
      if (!entry || !entry.minimized) return prev;
      const next = new Map(prev);
      next.set(agentId, { ...entry, unreadCount: entry.unreadCount + 1 });
      return next;
    });
  }, []);

  const has = useCallback((agentId: string): boolean => {
    return windows.has(agentId);
  }, [windows]);

  const getWindows = useCallback((): WindowEntry[] => {
    return Array.from(windows.values());
  }, [windows]);

  const getMinimizedWindows = useCallback((): WindowEntry[] => {
    return Array.from(windows.values()).filter((w) => w.minimized);
  }, [windows]);

  const count = windows.size;

  return {
    windows,
    count,
    open,
    close,
    minimize,
    restore,
    incrementUnread,
    has,
    getWindows,
    getMinimizedWindows,
  };
}
