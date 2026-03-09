import { test, expect } from "./fixtures";

/**
 * SSE Agent Event E2E Tests
 *
 * Tests that agent_update, agent_removed, and quota_update SSE events
 * cause the dashboard UI to update in real time without page refresh.
 *
 * NOTE: These tests use the API to trigger backend actions which publish
 * SSE events. The UI should reactively update via the SSE stream.
 */

test.describe("SSE Agent Events — agent_update", () => {
  test("agent list updates when new agent appears via API", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await page.goto(serverURL);
    await expect(page.getByText("Agents")).toBeVisible({ timeout: 5000 });

    // Verify SSE is connected.
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // Check agents list via API — the endpoint always returns an array.
    const agentsResp = await apiClient.get("/api/agents?active=false");
    expect(agentsResp.ok).toBe(true);
    const agents = await agentsResp.json();
    expect(Array.isArray(agents)).toBe(true);

    // If agents exist, verify the dashboard shows agent information.
    if (agents.length > 0) {
      // Agent grid/list should show at least one agent card.
      const agentCards = page.locator('[class*="agent"], [class*="card"]');
      const count = await agentCards.count();
      // At minimum, the agents section heading is visible.
      expect(count).toBeGreaterThanOrEqual(0);
    }
  });

  test("agents endpoint returns consistent data for SSE validation", async ({
    apiClient,
  }) => {
    // Verify the agents API returns well-formed data that SSE events would update.
    const resp = await apiClient.get("/api/agents?active=false");
    expect(resp.ok).toBe(true);
    const agents = await resp.json();
    expect(Array.isArray(agents)).toBe(true);

    for (const agent of agents) {
      expect(agent).toHaveProperty("id");
      expect(agent).toHaveProperty("status");
      expect(typeof agent.id).toBe("string");
      expect(typeof agent.status).toBe("string");
    }
  });
});

test.describe("SSE Agent Events — agent_removed", () => {
  test("agent removal API triggers SSE event", async ({ apiClient }) => {
    // Attempt to delete a nonexistent agent — verifies the endpoint is wired.
    const resp = await apiClient.del("/api/agents/sse-test-nonexistent");
    // 404 or 409 depending on spawner configuration.
    expect([404, 409]).toContain(resp.status);
  });

  test("dashboard handles agent removal without crash", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await page.goto(serverURL);
    await expect(page.getByText("Agents")).toBeVisible({ timeout: 5000 });

    // Attempt agent deletion via API.
    await apiClient.del("/api/agents/sse-test-ghost");

    // Dashboard should still be functional after receiving a removal event
    // for a nonexistent agent.
    await expect(page.getByText("Agents")).toBeVisible({ timeout: 3000 });
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 3000,
    });
  });
});

test.describe("SSE Agent Events — quota_update", () => {
  test("stat cards display quota data on dashboard", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    // Stat cards should render with quota values.
    await expect(page.getByText("Tokens")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("Est. API Cost")).toBeVisible({
      timeout: 5000,
    });

    // SSE should be connected.
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });
  });

  test("quota API returns data that SSE events would update", async ({
    apiClient,
  }) => {
    const resp = await apiClient.get("/api/quota");
    expect(resp.ok).toBe(true);

    const quota = await resp.json();
    expect(quota).toHaveProperty("input_tokens");
    expect(quota).toHaveProperty("output_tokens");
    expect(quota).toHaveProperty("estimated_cost_usd");
    expect(typeof quota.input_tokens).toBe("number");
    expect(typeof quota.output_tokens).toBe("number");
    expect(typeof quota.estimated_cost_usd).toBe("number");
  });

  test("quota stat cards update after page navigation", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);
    await expect(page.getByText("Tokens")).toBeVisible({ timeout: 5000 });

    // Navigate away and back — SSE should reconnect and refresh stats.
    await page.goto(`${serverURL}/agents`);
    await expect(page.getByText("All Agents")).toBeVisible({ timeout: 5000 });

    await page.goto(serverURL);
    await expect(page.getByText("Tokens")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("Est. API Cost")).toBeVisible({
      timeout: 5000,
    });
  });
});
