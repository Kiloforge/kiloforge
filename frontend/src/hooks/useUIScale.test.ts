import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useUIScale } from "./useUIScale";

const STORAGE_KEY = "kf-ui-scale";

describe("useUIScale", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.style.zoom = "";
  });

  afterEach(() => {
    document.documentElement.style.zoom = "";
    vi.restoreAllMocks();
  });

  it("defaults to 150 when no stored value", () => {
    const { result } = renderHook(() => useUIScale());
    expect(result.current.scale).toBe(150);
  });

  it("reads persisted scale from localStorage", () => {
    localStorage.setItem(STORAGE_KEY, "120");
    const { result } = renderHook(() => useUIScale());
    expect(result.current.scale).toBe(120);
  });

  it("applies zoom style to document.documentElement on mount", () => {
    localStorage.setItem(STORAGE_KEY, "140");
    renderHook(() => useUIScale());
    expect(document.documentElement.style.zoom).toBe("1.4");
  });

  it("applies zoom when default scale is 150", () => {
    renderHook(() => useUIScale());
    expect(document.documentElement.style.zoom).toBe("1.5");
  });

  it("does not set zoom when scale is explicitly set to 100", () => {
    localStorage.setItem(STORAGE_KEY, "100");
    renderHook(() => useUIScale());
    expect(document.documentElement.style.zoom).toBe("");
  });

  it("persists new scale to localStorage on setScale", () => {
    const { result } = renderHook(() => useUIScale());
    act(() => result.current.setScale(120));
    expect(localStorage.getItem(STORAGE_KEY)).toBe("120");
    expect(result.current.scale).toBe(120);
  });

  it("applies zoom style when scale changes", () => {
    const { result } = renderHook(() => useUIScale());
    act(() => result.current.setScale(85));
    expect(document.documentElement.style.zoom).toBe("0.85");
  });

  it("clears zoom when reset to 100", () => {
    localStorage.setItem(STORAGE_KEY, "120");
    const { result } = renderHook(() => useUIScale());
    expect(document.documentElement.style.zoom).toBe("1.2");
    act(() => result.current.setScale(100));
    expect(document.documentElement.style.zoom).toBe("");
  });

  it("clamps values to valid range", () => {
    const { result } = renderHook(() => useUIScale());
    act(() => result.current.setScale(200));
    expect(result.current.scale).toBe(150);
    act(() => result.current.setScale(50));
    expect(result.current.scale).toBe(75);
  });
});
