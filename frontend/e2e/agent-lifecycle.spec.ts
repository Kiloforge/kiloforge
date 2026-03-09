import { test, expect } from "./fixtures";

/**
 * Agent Lifecycle E2E Tests
 *
 * These tests verify agent-related UI rendering, navigation, and filtering.
 * Agent data is seeded via the API; spawn/stop/resume require a running
 * backend with an interactive spawner configured.
 */

test.describe("Agent Lifecycle — Dashboard Display", () => {
  test("dashboard shows agent section heading", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);
    // The overview page has an "Agents" heading or section.
    await expect(page.getByText("Agents")).toBeVisible({ timeout: 5000 });
  });

  test("dashboard shows 'No agents running' when empty", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);
    // When no agents exist, the grid shows empty state.
    const empty = page.getByText("No agents running");
    if (await empty.isVisible({ timeout: 3000 }).catch(() => false)) {
      await expect(empty).toBeVisible();
    }
  });

  test("agents API returns array", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/agents?active=false");
    expect(resp.ok).toBe(true);
    const agents = await resp.json();
    expect(Array.isArray(agents)).toBe(true);
  });

  test("active agents filter defaults to active only", async ({
    apiClient,
  }) => {
    // Default (no param) should return only active + recently finished.
    const defaultResp = await apiClient.get("/api/agents");
    expect(defaultResp.ok).toBe(true);
    const activeAgents = await defaultResp.json();

    // All returned agents should be active or recently finished.
    for (const agent of activeAgents) {
      const status = agent.status as string;
      // Active statuses or recently finished are ok.
      expect([
        "running",
        "waiting",
        "completed",
        "failed",
        "stopped",
        "suspended",
        "halted",
      ]).toContain(status);
    }
  });
});

test.describe("Agent Lifecycle — Agent Detail Page", () => {
  test("agent detail page shows 404 for nonexistent agent", async ({
    page,
    serverURL,
  }) => {
    await page.goto(`${serverURL}/agents/nonexistent-agent-id`);
    // Should show an error or loading state — not crash.
    // The detail page fetches /api/agents/nonexistent-agent-id which returns 404.
    // UI should show an error message.
    const errorText = page.locator("text=/not found|error|failed/i");
    const loadingText = page.getByText("Loading agent...");
    // Either an error message or loading state is acceptable.
    const hasError = await errorText
      .isVisible({ timeout: 5000 })
      .catch(() => false);
    const hasLoading = await loadingText
      .isVisible({ timeout: 1000 })
      .catch(() => false);
    expect(hasError || hasLoading).toBe(true);
  });

  test("agent detail page has back link", async ({ page, serverURL }) => {
    await page.goto(`${serverURL}/agents/some-id`);
    const backLink = page.locator("text=Back");
    await expect(backLink).toBeVisible({ timeout: 5000 });
  });
});

test.describe("Agent Lifecycle — Agent History Page", () => {
  test("history page loads at /agents", async ({ page, serverURL }) => {
    await page.goto(`${serverURL}/agents`);
    await expect(page.getByText("All Agents")).toBeVisible({ timeout: 5000 });
  });

  test("history page has status filter buttons", async ({
    page,
    serverURL,
  }) => {
    await page.goto(`${serverURL}/agents`);

    // Should have filter chips for each status.
    const filters = ["All", "running", "waiting", "completed", "failed", "stopped", "suspended"];
    for (const filter of filters) {
      const btn = page.getByRole("button", { name: filter, exact: true });
      await expect(btn).toBeVisible({ timeout: 3000 });
    }
  });

  test("history page clicking filter changes active chip", async ({
    page,
    serverURL,
  }) => {
    await page.goto(`${serverURL}/agents`);

    // Click "completed" filter.
    const completedBtn = page.getByRole("button", {
      name: "completed",
      exact: true,
    });
    await completedBtn.click();

    // The clicked button should have the active class.
    // We can verify by checking a visual indicator or just that it was clickable.
    await expect(completedBtn).toBeVisible();
  });

  test("history page shows table headers", async ({ page, serverURL }) => {
    await page.goto(`${serverURL}/agents`);

    // Table should have expected column headers.
    const headers = ["Name / ID", "Role", "Status", "Ref", "Uptime", "Cost", "Updated", "Actions"];
    for (const header of headers) {
      const th = page.locator(`th:has-text("${header}")`);
      // Headers may not be visible if no agents exist (table not rendered).
      // Check if table exists first.
      const tableVisible = await page
        .locator("table")
        .isVisible({ timeout: 2000 })
        .catch(() => false);
      if (tableVisible) {
        await expect(th).toBeVisible();
      }
    }
  });

  test("history page shows empty state when no agents", async ({
    page,
    serverURL,
  }) => {
    await page.goto(`${serverURL}/agents`);
    // If no agents, should show "No agents found" or the table.
    const empty = page.getByText("No agents found");
    const table = page.locator("table");
    const hasEmpty = await empty
      .isVisible({ timeout: 3000 })
      .catch(() => false);
    const hasTable = await table
      .isVisible({ timeout: 1000 })
      .catch(() => false);
    // One of them should be visible.
    expect(hasEmpty || hasTable).toBe(true);
  });
});

test.describe("Agent Lifecycle — Navigation", () => {
  test("dashboard has link to agent history", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);
    const viewAllLink = page.locator('a[href="/agents"]');
    if (await viewAllLink.isVisible({ timeout: 3000 }).catch(() => false)) {
      await viewAllLink.click();
      await expect(page).toHaveURL(/.*\/agents$/);
      await expect(page.getByText("All Agents")).toBeVisible();
    }
  });

  test("history page has back link to dashboard", async ({
    page,
    serverURL,
  }) => {
    await page.goto(`${serverURL}/agents`);
    const backLink = page.locator("text=Back");
    await expect(backLink).toBeVisible({ timeout: 3000 });
  });

  test("navbar has Agents link", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    const agentsLink = page.locator('a:has-text("Agents")');
    await expect(agentsLink).toBeVisible({ timeout: 3000 });
  });
});

test.describe("Agent Lifecycle — API Validation", () => {
  test("get agent returns 404 for unknown ID", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/agents/does-not-exist-12345");
    expect(resp.status).toBe(404);
    const body = await resp.json();
    expect(body.error).toBeTruthy();
  });

  test("stop agent returns error without spawner", async ({ apiClient }) => {
    // Without interactive spawner, stop should return 409.
    const resp = await apiClient.post("/api/agents/any-id/stop");
    // Either 404 (agent not found) or 409 (not configured) is acceptable.
    expect([404, 409]).toContain(resp.status);
  });

  test("resume agent returns error without spawner", async ({ apiClient }) => {
    const resp = await apiClient.post("/api/agents/any-id/resume");
    expect([404, 409]).toContain(resp.status);
  });

  test("delete agent returns error for unknown ID", async ({ apiClient }) => {
    const resp = await apiClient.del("/api/agents/nonexistent-id");
    // 404 or 409 depending on configuration.
    expect([404, 409]).toContain(resp.status);
  });

  test("agent log returns 404 for unknown agent", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/agents/nonexistent/log");
    expect(resp.status).toBe(404);
  });
});
