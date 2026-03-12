import { useState, useCallback, useRef, useEffect } from "react";

export type ResizeEdge =
  | "n" | "s" | "e" | "w"
  | "nw" | "ne" | "sw" | "se"
  | null;

interface FloatingWindowState {
  x: number;
  y: number;
  width: number;
  height: number;
  zIndex: number;
}

interface UseFloatingWindowOptions {
  defaultWidth?: number;
  defaultHeight?: number;
  minWidth?: number;
  minHeight?: number;
  initialX?: number;
  initialY?: number;
}

const EDGE_ZONE = 8;

let globalZIndex = 100;

/** Read current CSS zoom factor (1.0 when unset). */
function getZoom(): number {
  return parseFloat(document.documentElement.style.zoom || "1") || 1;
}

function clampPosition(
  x: number,
  y: number,
  width: number,
  height: number,
): { x: number; y: number } {
  const vw = window.innerWidth;
  const vh = window.innerHeight;
  return {
    x: Math.max(0, Math.min(x, vw - width)),
    y: Math.max(0, Math.min(y, vh - height)),
  };
}

function clampSize(
  width: number,
  height: number,
  minWidth: number,
  minHeight: number,
): { width: number; height: number } {
  const vw = window.innerWidth;
  const vh = window.innerHeight;
  return {
    width: Math.max(minWidth, Math.min(width, vw)),
    height: Math.max(minHeight, Math.min(height, vh)),
  };
}

function centerPosition(width: number, height: number): { x: number; y: number } {
  return {
    x: Math.max(0, (window.innerWidth - width) / 2),
    y: Math.max(0, (window.innerHeight - height) / 2),
  };
}

export function detectEdge(
  clientX: number,
  clientY: number,
  rect: DOMRect,
): ResizeEdge {
  const zone = EDGE_ZONE * getZoom();
  const left = clientX - rect.left < zone;
  const right = rect.right - clientX < zone;
  const top = clientY - rect.top < zone;
  const bottom = rect.bottom - clientY < zone;

  if (top && left) return "nw";
  if (top && right) return "ne";
  if (bottom && left) return "sw";
  if (bottom && right) return "se";
  if (top) return "n";
  if (bottom) return "s";
  if (left) return "w";
  if (right) return "e";
  return null;
}

export function cursorForEdge(edge: ResizeEdge): string {
  switch (edge) {
    case "n": case "s": return "ns-resize";
    case "e": case "w": return "ew-resize";
    case "nw": case "se": return "nwse-resize";
    case "ne": case "sw": return "nesw-resize";
    default: return "";
  }
}

export function useFloatingWindow(options: UseFloatingWindowOptions = {}) {
  const {
    defaultWidth = 720,
    defaultHeight = 500,
    minWidth = 400,
    minHeight = 300,
    initialX,
    initialY,
  } = options;

  const [state, setState] = useState<FloatingWindowState>(() => {
    const size = clampSize(defaultWidth, defaultHeight, minWidth, minHeight);
    const centered = centerPosition(size.width, size.height);
    const pos = clampPosition(
      initialX ?? centered.x,
      initialY ?? centered.y,
      size.width,
      size.height,
    );
    return { ...pos, ...size, zIndex: ++globalZIndex };
  });

  const [isDragging, setIsDragging] = useState(false);
  const [isResizing, setIsResizing] = useState(false);
  const [resizeEdge, setResizeEdge] = useState<ResizeEdge>(null);
  const [hoverEdge, setHoverEdge] = useState<ResizeEdge>(null);

  const dragStart = useRef({ mouseX: 0, mouseY: 0, winX: 0, winY: 0 });
  const resizeStart = useRef({
    mouseX: 0, mouseY: 0,
    winX: 0, winY: 0,
    winW: 0, winH: 0,
    edge: null as ResizeEdge,
  });

  const bringToFront = useCallback(() => {
    setState((prev) => {
      const next = ++globalZIndex;
      if (prev.zIndex === next - 1) return prev; // already on top
      return { ...prev, zIndex: next };
    });
  }, []);

  const onDragStart = useCallback(
    (e: React.PointerEvent) => {
      e.preventDefault();
      (e.target as HTMLElement).setPointerCapture(e.pointerId);
      dragStart.current = {
        mouseX: e.clientX,
        mouseY: e.clientY,
        winX: state.x,
        winY: state.y,
      };
      setIsDragging(true);
      bringToFront();
    },
    [state.x, state.y, bringToFront],
  );

  const onDragMove = useCallback(
    (e: React.PointerEvent) => {
      if (!isDragging) return;
      const zoom = getZoom();
      const dx = (e.clientX - dragStart.current.mouseX) / zoom;
      const dy = (e.clientY - dragStart.current.mouseY) / zoom;
      const newX = dragStart.current.winX + dx;
      const newY = dragStart.current.winY + dy;
      const clamped = clampPosition(newX, newY, state.width, state.height);
      setState((prev) => ({ ...prev, x: clamped.x, y: clamped.y }));
    },
    [isDragging, state.width, state.height],
  );

  const onDragEnd = useCallback(() => {
    setIsDragging(false);
  }, []);

  const onResizeStart = useCallback(
    (e: React.PointerEvent, edge: ResizeEdge) => {
      if (!edge) return;
      e.preventDefault();
      e.stopPropagation();
      (e.target as HTMLElement).setPointerCapture(e.pointerId);
      resizeStart.current = {
        mouseX: e.clientX,
        mouseY: e.clientY,
        winX: state.x,
        winY: state.y,
        winW: state.width,
        winH: state.height,
        edge,
      };
      setResizeEdge(edge);
      setIsResizing(true);
      bringToFront();
    },
    [state.x, state.y, state.width, state.height, bringToFront],
  );

  const onResizeMove = useCallback(
    (e: React.PointerEvent) => {
      if (!isResizing) return;
      const { mouseX, mouseY, winX, winY, winW, winH, edge } = resizeStart.current;
      const zoom = getZoom();
      const dx = (e.clientX - mouseX) / zoom;
      const dy = (e.clientY - mouseY) / zoom;

      let newX = winX;
      let newY = winY;
      let newW = winW;
      let newH = winH;

      if (edge?.includes("e")) newW = winW + dx;
      if (edge?.includes("w")) { newW = winW - dx; newX = winX + dx; }
      if (edge?.includes("s")) newH = winH + dy;
      if (edge?.includes("n")) { newH = winH - dy; newY = winY + dy; }

      // Enforce minimums — if shrinking below min, snap position back
      if (newW < minWidth) {
        if (edge?.includes("w")) newX = winX + winW - minWidth;
        newW = minWidth;
      }
      if (newH < minHeight) {
        if (edge?.includes("n")) newY = winY + winH - minHeight;
        newH = minHeight;
      }

      // Clamp to viewport
      const vw = window.innerWidth;
      const vh = window.innerHeight;
      newW = Math.min(newW, vw);
      newH = Math.min(newH, vh);
      newX = Math.max(0, Math.min(newX, vw - newW));
      newY = Math.max(0, Math.min(newY, vh - newH));

      setState((prev) => ({
        ...prev,
        x: newX,
        y: newY,
        width: newW,
        height: newH,
      }));
    },
    [isResizing, minWidth, minHeight],
  );

  const onResizeEnd = useCallback(() => {
    setIsResizing(false);
    setResizeEdge(null);
  }, []);

  // Handle viewport resize — clamp window if it goes out of bounds
  useEffect(() => {
    const handleResize = () => {
      setState((prev) => {
        const size = clampSize(prev.width, prev.height, minWidth, minHeight);
        const pos = clampPosition(prev.x, prev.y, size.width, size.height);
        if (
          pos.x === prev.x && pos.y === prev.y &&
          size.width === prev.width && size.height === prev.height
        ) return prev;
        return { ...prev, ...pos, ...size };
      });
    };
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, [minWidth, minHeight]);

  const reset = useCallback(() => {
    const size = clampSize(defaultWidth, defaultHeight, minWidth, minHeight);
    const pos = centerPosition(size.width, size.height);
    setState({ ...pos, ...size, zIndex: ++globalZIndex });
  }, [defaultWidth, defaultHeight, minWidth, minHeight]);

  const setRect = useCallback(
    (x: number, y: number, width: number, height: number) => {
      const size = clampSize(width, height, minWidth, minHeight);
      const pos = clampPosition(x, y, size.width, size.height);
      setState((prev) => ({ ...prev, ...pos, ...size }));
    },
    [minWidth, minHeight],
  );

  const getRect = useCallback(
    () => ({ x: state.x, y: state.y, width: state.width, height: state.height, zIndex: state.zIndex }),
    [state.x, state.y, state.width, state.height, state.zIndex],
  );

  return {
    // State
    x: state.x,
    y: state.y,
    width: state.width,
    height: state.height,
    zIndex: state.zIndex,
    isDragging,
    isResizing,
    resizeEdge,
    hoverEdge,
    setHoverEdge,

    // Actions
    bringToFront,
    reset,

    // Drag handlers (for header)
    onDragStart,
    onDragMove,
    onDragEnd,

    // Resize handlers (for window container)
    onResizeStart,
    onResizeMove,
    onResizeEnd,

    // External control (for tiling / snapping)
    setRect,
    getRect,
  };
}
