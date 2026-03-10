import { renderHook } from "@testing-library/react";
import { describe, it, expect, vi, afterEach } from "vitest";
import { usePlatform } from "./usePlatform";

describe("usePlatform", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("detects Mac via navigator.platform", () => {
    vi.stubGlobal("navigator", { platform: "MacIntel", userAgent: "" });
    const { result } = renderHook(() => usePlatform());
    expect(result.current.isMac).toBe(true);
    expect(result.current.mod).toBe("⌘");
    expect(result.current.shift).toBe("⇧");
    expect(result.current.modKey).toBe("Meta");
  });

  it("detects Windows via navigator.platform", () => {
    vi.stubGlobal("navigator", { platform: "Win32", userAgent: "" });
    const { result } = renderHook(() => usePlatform());
    expect(result.current.isMac).toBe(false);
    expect(result.current.mod).toBe("Ctrl");
    expect(result.current.shift).toBe("Shift+");
    expect(result.current.modKey).toBe("Control");
  });

  it("detects Linux via navigator.platform", () => {
    vi.stubGlobal("navigator", { platform: "Linux x86_64", userAgent: "" });
    const { result } = renderHook(() => usePlatform());
    expect(result.current.isMac).toBe(false);
    expect(result.current.mod).toBe("Ctrl");
    expect(result.current.shift).toBe("Shift+");
    expect(result.current.modKey).toBe("Control");
  });

  it("falls back to userAgent for Mac detection", () => {
    vi.stubGlobal("navigator", {
      platform: "",
      userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
    });
    const { result } = renderHook(() => usePlatform());
    expect(result.current.isMac).toBe(true);
  });

  it("returns non-Mac when navigator is unavailable", () => {
    vi.stubGlobal("navigator", undefined);
    const { result } = renderHook(() => usePlatform());
    expect(result.current.isMac).toBe(false);
    expect(result.current.mod).toBe("Ctrl");
  });
});
