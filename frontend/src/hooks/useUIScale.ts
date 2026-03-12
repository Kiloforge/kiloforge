import { useState, useCallback, useEffect } from "react";

const STORAGE_KEY = "kf-ui-scale";
const MIN_SCALE = 75;
const MAX_SCALE = 150;
const DEFAULT_SCALE = 100;

function readStoredScale(): number {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw === null) return DEFAULT_SCALE;
    const parsed = Number(raw);
    if (Number.isNaN(parsed)) return DEFAULT_SCALE;
    return clamp(parsed);
  } catch {
    return DEFAULT_SCALE;
  }
}

function clamp(value: number): number {
  return Math.min(MAX_SCALE, Math.max(MIN_SCALE, Math.round(value)));
}

function applyZoom(scale: number) {
  if (scale === DEFAULT_SCALE) {
    document.documentElement.style.zoom = "";
  } else {
    document.documentElement.style.zoom = String(scale / 100);
  }
}

export function useUIScale() {
  const [scale, setScaleState] = useState(readStoredScale);

  useEffect(() => {
    applyZoom(scale);
    return () => {
      document.documentElement.style.zoom = "";
    };
  }, [scale]);

  const setScale = useCallback((value: number) => {
    const clamped = clamp(value);
    setScaleState(clamped);
    try {
      localStorage.setItem(STORAGE_KEY, String(clamped));
    } catch {
      // localStorage may be unavailable
    }
  }, []);

  return { scale, setScale };
}
