import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { fetcher, FetchError } from "./fetcher";

describe("fetcher", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("returns parsed JSON on success", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ name: "test" }),
    } as Response);

    const result = await fetcher<{ name: string }>("/api/test");
    expect(result).toEqual({ name: "test" });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/test", undefined);
  });

  it("passes init options to fetch", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    } as Response);

    await fetcher("/api/test", { method: "POST", body: "{}" });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/test", { method: "POST", body: "{}" });
  });

  it("throws FetchError on non-ok response", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue({
      ok: false,
      status: 404,
      json: () => Promise.resolve({ error: "Not found" }),
    } as unknown as Response);

    await expect(fetcher("/api/missing")).rejects.toThrow(FetchError);
    try {
      await fetcher("/api/missing");
    } catch (err) {
      expect(err).toBeInstanceOf(FetchError);
      expect((err as FetchError).status).toBe(404);
      expect((err as FetchError).body).toEqual({ error: "Not found" });
    }
  });

  it("handles non-JSON error body gracefully", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.reject(new Error("not json")),
    } as unknown as Response);

    try {
      await fetcher("/api/broken");
    } catch (err) {
      expect(err).toBeInstanceOf(FetchError);
      expect((err as FetchError).status).toBe(500);
      expect((err as FetchError).body).toEqual({ error: "Error 500" });
    }
  });
});
