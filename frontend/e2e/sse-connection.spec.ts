import { test, expect } from "./fixtures";

/**
 * SSE Connection E2E Tests
 *
 * Tests SSE connection establishment, connection indicator UI,
 * and correct request/response headers.
 */

test.describe("SSE Connection — Establishment", () => {
  test("SSE connects on dashboard load — indicator shows connected", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    // The connection indicator should show "connected" after SSE handshake.
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toBeVisible({ timeout: 5000 });
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });
    await expect(indicator).toHaveText("connected");
  });

  test("SSE endpoint returns text/event-stream content type", async ({
    serverURL,
  }) => {
    // Open a raw fetch to /events and check headers.
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 3000);

    try {
      const resp = await fetch(`${serverURL}/events`, {
        headers: { Accept: "text/event-stream" },
        signal: controller.signal,
      });
      expect(resp.status).toBe(200);
      expect(resp.headers.get("content-type")).toContain("text/event-stream");
      expect(resp.headers.get("cache-control")).toContain("no-cache");
    } catch (e: unknown) {
      // AbortError is expected — we just needed the headers.
      if (e instanceof Error && e.name !== "AbortError") throw e;
    } finally {
      clearTimeout(timeout);
    }
  });

  test("/events endpoint is accessible via apiClient", async ({
    apiClient,
  }) => {
    const resp = await apiClient.get("/events");
    // The endpoint should be reachable; returns 200 for SSE streams.
    expect([200, 204]).toContain(resp.status);
  });
});

test.describe("SSE Connection — Indicator UI", () => {
  test("connection indicator has correct visual state when connected", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toBeVisible({ timeout: 5000 });
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // Should have the connected CSS class (green styling).
    const classList = await indicator.getAttribute("class");
    expect(classList).toContain("connected");
  });

  test("connection indicator renders in header", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    // Indicator should be inside the header.
    const headerIndicator = page.locator(
      'header [data-testid="sse-status"]',
    );
    await expect(headerIndicator).toBeVisible({ timeout: 5000 });
  });
});
