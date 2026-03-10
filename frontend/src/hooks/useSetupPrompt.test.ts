import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useSetupPrompt } from "./useSetupPrompt";

describe("useSetupPrompt", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("calls onConsentRequired on 403 instead of setting error", async () => {
    const onConsentRequired = vi.fn();
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: false,
      status: 403,
      json: () => Promise.resolve({ error: "agent_permissions_not_consented" }),
    } as Response);

    const { result } = renderHook(() =>
      useSetupPrompt({ onConsentRequired }),
    );

    act(() => {
      result.current.requestSetup("test-project", vi.fn());
    });

    await act(async () => {
      await result.current.startSetup();
    });

    expect(onConsentRequired).toHaveBeenCalledTimes(1);
    expect(onConsentRequired).toHaveBeenCalledWith(expect.any(Function));
    expect(result.current.error).toBeNull();
    expect(result.current.starting).toBe(false);
  });

  it("shows error for non-403 failures", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ error: "internal error" }),
    } as Response);

    const { result } = renderHook(() => useSetupPrompt());

    act(() => {
      result.current.requestSetup("test-project", vi.fn());
    });

    await act(async () => {
      await result.current.startSetup();
    });

    expect(result.current.error).toBe("internal error");
  });

  it("sets agentId on successful setup", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ agent_id: "agent-123" }),
    } as Response);

    const { result } = renderHook(() => useSetupPrompt());

    act(() => {
      result.current.requestSetup("test-project", vi.fn());
    });

    await act(async () => {
      await result.current.startSetup();
    });

    expect(result.current.agentId).toBe("agent-123");
    expect(result.current.error).toBeNull();
  });
});
