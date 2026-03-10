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

export interface WindowControls {
  setRect: (x: number, y: number, width: number, height: number) => void;
  getRect: () => { x: number; y: number; width: number; height: number; zIndex: number };
  bringToFront: () => void;
}

const CASCADE_OFFSET = 30;
const CASCADE_WRAP = 5;
const DEFAULT_WIDTH = 720;
const DEFAULT_HEIGHT = 500;
const TILE_GAP = 8;
const HEADER_HEIGHT = 60;

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
  const controlsRef = useRef<Map<string, WindowControls>>(new Map());
  const preMaximizeRef = useRef<Map<string, { x: number; y: number; width: number; height: number }>>(new Map());
  const maximizedRef = useRef<Set<string>>(new Set());

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
    controlsRef.current.delete(agentId);
    preMaximizeRef.current.delete(agentId);
    maximizedRef.current.delete(agentId);
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

  // --- Controls registry ---

  const registerControls = useCallback((agentId: string, controls: WindowControls) => {
    controlsRef.current.set(agentId, controls);
  }, []);

  const unregisterControls = useCallback((agentId: string) => {
    controlsRef.current.delete(agentId);
    preMaximizeRef.current.delete(agentId);
    maximizedRef.current.delete(agentId);
  }, []);

  // --- Focused window (highest z-index among non-minimized) ---

  const getFocusedId = useCallback((): string | null => {
    let best: string | null = null;
    let bestZ = -1;
    for (const [id, entry] of windows) {
      if (entry.minimized) continue;
      const ctrl = controlsRef.current.get(id);
      if (!ctrl) continue;
      const z = ctrl.getRect().zIndex;
      if (z > bestZ) {
        bestZ = z;
        best = id;
      }
    }
    return best;
  }, [windows]);

  // --- Tiling ---

  const tileAll = useCallback(() => {
    const visible = Array.from(windows.entries()).filter(([, e]) => !e.minimized);
    const n = visible.length;
    if (n === 0) return;

    const vw = window.innerWidth;
    const vh = window.innerHeight - HEADER_HEIGHT;
    const top = HEADER_HEIGHT;

    const layouts: Record<number, Array<{ x: number; y: number; w: number; h: number }>> = {
      1: [{ x: vw * 0.1, y: top + vh * 0.05, w: vw * 0.8, h: vh * 0.9 }],
      2: [
        { x: TILE_GAP, y: top + TILE_GAP, w: (vw - TILE_GAP * 3) / 2, h: vh - TILE_GAP * 2 },
        { x: (vw + TILE_GAP) / 2, y: top + TILE_GAP, w: (vw - TILE_GAP * 3) / 2, h: vh - TILE_GAP * 2 },
      ],
      3: [
        { x: TILE_GAP, y: top + TILE_GAP, w: (vw - TILE_GAP * 3) / 2, h: vh - TILE_GAP * 2 },
        { x: (vw + TILE_GAP) / 2, y: top + TILE_GAP, w: (vw - TILE_GAP * 3) / 2, h: (vh - TILE_GAP * 3) / 2 },
        { x: (vw + TILE_GAP) / 2, y: top + (vh + TILE_GAP) / 2, w: (vw - TILE_GAP * 3) / 2, h: (vh - TILE_GAP * 3) / 2 },
      ],
      4: [
        { x: TILE_GAP, y: top + TILE_GAP, w: (vw - TILE_GAP * 3) / 2, h: (vh - TILE_GAP * 3) / 2 },
        { x: (vw + TILE_GAP) / 2, y: top + TILE_GAP, w: (vw - TILE_GAP * 3) / 2, h: (vh - TILE_GAP * 3) / 2 },
        { x: TILE_GAP, y: top + (vh + TILE_GAP) / 2, w: (vw - TILE_GAP * 3) / 2, h: (vh - TILE_GAP * 3) / 2 },
        { x: (vw + TILE_GAP) / 2, y: top + (vh + TILE_GAP) / 2, w: (vw - TILE_GAP * 3) / 2, h: (vh - TILE_GAP * 3) / 2 },
      ],
    };

    let positions: Array<{ x: number; y: number; w: number; h: number }>;
    if (n <= 4) {
      positions = layouts[n];
    } else {
      // Auto-grid for 5+
      const cols = Math.ceil(Math.sqrt(n));
      const rows = Math.ceil(n / cols);
      const cellW = (vw - TILE_GAP * (cols + 1)) / cols;
      const cellH = (vh - TILE_GAP * (rows + 1)) / rows;
      positions = [];
      for (let i = 0; i < n; i++) {
        const col = i % cols;
        const row = Math.floor(i / cols);
        positions.push({
          x: TILE_GAP + col * (cellW + TILE_GAP),
          y: top + TILE_GAP + row * (cellH + TILE_GAP),
          w: cellW,
          h: cellH,
        });
      }
    }

    visible.forEach(([id], i) => {
      const ctrl = controlsRef.current.get(id);
      if (!ctrl || !positions[i]) return;
      maximizedRef.current.delete(id);
      preMaximizeRef.current.delete(id);
      ctrl.setRect(positions[i].x, positions[i].y, positions[i].w, positions[i].h);
    });
  }, [windows]);

  // --- Snap left/right ---

  const snapFocused = useCallback((side: "left" | "right") => {
    const id = getFocusedId();
    if (!id) return;
    const ctrl = controlsRef.current.get(id);
    if (!ctrl) return;

    const vw = window.innerWidth;
    const vh = window.innerHeight - HEADER_HEIGHT;
    const top = HEADER_HEIGHT;
    const w = (vw - TILE_GAP * 3) / 2;
    const h = vh - TILE_GAP * 2;
    const x = side === "left" ? TILE_GAP : (vw + TILE_GAP) / 2;

    maximizedRef.current.delete(id);
    preMaximizeRef.current.delete(id);
    ctrl.setRect(x, top + TILE_GAP, w, h);
  }, [getFocusedId]);

  // --- Maximize / restore ---

  const toggleMaximizeFocused = useCallback(() => {
    const id = getFocusedId();
    if (!id) return;
    const ctrl = controlsRef.current.get(id);
    if (!ctrl) return;

    if (maximizedRef.current.has(id)) {
      // Restore
      const prev = preMaximizeRef.current.get(id);
      if (prev) {
        ctrl.setRect(prev.x, prev.y, prev.width, prev.height);
      }
      maximizedRef.current.delete(id);
      preMaximizeRef.current.delete(id);
    } else {
      // Maximize
      const rect = ctrl.getRect();
      preMaximizeRef.current.set(id, { x: rect.x, y: rect.y, width: rect.width, height: rect.height });
      maximizedRef.current.add(id);
      const vw = window.innerWidth;
      const vh = window.innerHeight - HEADER_HEIGHT;
      ctrl.setRect(0, HEADER_HEIGHT, vw, vh);
    }
  }, [getFocusedId]);

  // --- Minimize / restore focused ---

  const toggleMinimizeFocused = useCallback(() => {
    const id = getFocusedId();
    if (!id) return;
    const entry = windows.get(id);
    if (!entry) return;
    if (entry.minimized) {
      restore(id);
    } else {
      minimize(id);
    }
  }, [getFocusedId, windows, minimize, restore]);

  // --- Cycle focus ---

  const cycleFocus = useCallback((direction: 1 | -1) => {
    const visible = Array.from(windows.entries())
      .filter(([, e]) => !e.minimized)
      .map(([id]) => id);
    if (visible.length < 2) return;

    // Sort by z-index to find current focus
    const sorted = visible
      .map((id) => {
        const ctrl = controlsRef.current.get(id);
        return { id, z: ctrl ? ctrl.getRect().zIndex : 0 };
      })
      .sort((a, b) => a.z - b.z);

    const currentIdx = sorted.length - 1; // highest z = focused
    const nextIdx = ((currentIdx + direction) % sorted.length + sorted.length) % sorted.length;
    const nextId = sorted[nextIdx].id;

    const ctrl = controlsRef.current.get(nextId);
    if (ctrl) ctrl.bringToFront();
  }, [windows]);

  // --- Close focused ---

  const closeFocused = useCallback(() => {
    const id = getFocusedId();
    if (!id) return;
    close(id);
  }, [getFocusedId, close]);

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
    // Controls registry
    registerControls,
    unregisterControls,
    // Tiling & arrangement
    tileAll,
    snapFocused,
    toggleMaximizeFocused,
    toggleMinimizeFocused,
    cycleFocus,
    closeFocused,
    getFocusedId,
  };
}
