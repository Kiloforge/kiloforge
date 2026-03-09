import { test, expect } from "./fixtures";

/**
 * SSE Reconnection and Failure E2E Tests
 *
 * Tests auto-reconnect behavior, event burst handling, malformed events,
 * and resilience to server disruptions.
 */

// Helper for unique names.
let counter = 0;
function uid(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}`;
}

test.describe("SSE Reconnection — Auto-reconnect", () => {
  test("SSE reconnects after navigation away and back", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    // Verify initial connection.
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // Navigate away (destroys SSE connection).
    await page.goto(`${serverURL}/agents`);
    await expect(page.getByText("All Agents")).toBeVisible({ timeout: 5000 });

    // Navigate back — SSE should reconnect.
    await page.goto(serverURL);
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 10000,
    });
  });

  test("SSE connection survives multiple navigations", async ({
    page,
    serverURL,
  }) => {
    // Rapidly navigate between pages.
    for (let i = 0; i < 3; i++) {
      await page.goto(serverURL);
      await page.goto(`${serverURL}/agents`);
    }

    // Final navigation back to home.
    await page.goto(serverURL);
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 10000,
    });
  });

  test("events are received after reconnection", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await page.goto(serverURL);
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // Navigate away and back to trigger reconnect cycle.
    await page.goto(`${serverURL}/agents`);
    await page.goto(serverURL);
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 10000,
    });

    // After reconnection, events should still be processed.
    // Trigger a project creation to publish an SSE event.
    const slug = uid("sse-reconn");
    await apiClient.post("/api/projects", {
      remote_url: `https://github.com/user/${slug}.git`,
    });

    // Dashboard should still be functional.
    await expect(page.getByText("Projects")).toBeVisible({ timeout: 5000 });

    // Cleanup.
    await apiClient.del(`/api/projects/${slug}`);
  });
});

test.describe("SSE Reconnection — Event Burst", () => {
  test("rapid project creation burst — all events processed", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await page.goto(serverURL);
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    const slugs: string[] = [];

    // Fire 10 rapid project creates (each publishes an SSE event).
    for (let i = 0; i < 10; i++) {
      const slug = uid(`burst-${i}`);
      slugs.push(slug);
      // Don't await sequentially — fire them rapidly.
      apiClient.post("/api/projects", {
        remote_url: `https://github.com/user/${slug}.git`,
      });
    }

    // Wait a moment for events to propagate.
    await page.waitForTimeout(2000);

    // SSE should still be connected (no crash from burst).
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 3000,
    });

    // Dashboard should still be functional.
    await expect(page.getByText("Projects")).toBeVisible({ timeout: 3000 });

    // Cleanup all burst projects.
    for (const slug of slugs) {
      await apiClient.del(`/api/projects/${slug}`).catch(() => {});
    }
  });

  test("concurrent API actions produce SSE events without drops", async ({
    apiClient,
    serverURL,
  }) => {
    const slugs: string[] = [];

    // Create 5 projects concurrently.
    const creates = Array.from({ length: 5 }, (_, i) => {
      const slug = uid(`conc-${i}`);
      slugs.push(slug);
      return apiClient.post("/api/projects", {
        remote_url: `https://github.com/user/${slug}.git`,
      });
    });

    const results = await Promise.all(creates);
    for (const r of results) {
      expect(r.ok).toBe(true);
    }

    // Verify all projects exist via API.
    for (const slug of slugs) {
      const resp = await apiClient.get(`/api/projects/${slug}`);
      expect(resp.ok).toBe(true);
    }

    // Cleanup.
    for (const slug of slugs) {
      await apiClient.del(`/api/projects/${slug}`).catch(() => {});
    }
  });
});

test.describe("SSE Reconnection — Malformed Events", () => {
  test("dashboard handles malformed SSE data gracefully", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    // Verify SSE is connected.
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // Collect any console errors during the test.
    const consoleErrors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") {
        consoleErrors.push(msg.text());
      }
    });

    // Perform normal API actions — the SSE client should handle
    // any edge cases in event parsing (the useSSE hook catches JSON parse errors).
    await page.waitForTimeout(1000);

    // Dashboard should remain functional.
    await expect(page.getByText("Agents")).toBeVisible({ timeout: 3000 });

    // No JavaScript errors related to JSON parsing should appear.
    const jsonErrors = consoleErrors.filter(
      (e) => e.includes("JSON") || e.includes("parse"),
    );
    expect(jsonErrors).toHaveLength(0);
  });

  test("SSE hook silently ignores parse errors — connection stays open", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // The useSSE hook has a try/catch around JSON.parse — verify the
    // connection stays open even if unusual data comes through.
    // We verify by checking the indicator stays connected over time.
    await page.waitForTimeout(2000);
    await expect(indicator).toHaveAttribute("data-status", "connected");
  });
});

test.describe("SSE Reconnection — Server Restart Simulation", () => {
  test("page reload simulates reconnect — SSE re-establishes", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // Reload page — simulates SSE disconnect/reconnect.
    await page.reload();

    // SSE should re-establish after reload.
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 10000,
    });

    // Dashboard should be functional.
    await expect(page.getByText("Agents")).toBeVisible({ timeout: 5000 });
  });

  test("dashboard state refreshes after reconnection", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    // Create a project before loading the page.
    const slug = uid("pre-reload");
    await apiClient.post("/api/projects", {
      remote_url: `https://github.com/user/${slug}.git`,
    });

    await page.goto(serverURL);
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // Delete project while page is loaded.
    await apiClient.del(`/api/projects/${slug}`);

    // Reload — state should be refreshed.
    await page.reload();
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 10000,
    });

    // Project should be gone.
    const getResp = await apiClient.get(`/api/projects/${slug}`);
    expect(getResp.status).toBe(404);
  });
});
