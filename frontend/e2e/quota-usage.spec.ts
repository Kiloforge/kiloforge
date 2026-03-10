import { test, expect } from "./fixtures";

test.describe("Quota — Stat Card Display", () => {
  test("quota endpoint returns expected fields", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/quota");
    expect(resp.ok).toBe(true);

    const body = await resp.json();
    expect(body).toHaveProperty("input_tokens");
    expect(body).toHaveProperty("output_tokens");
    expect(body).toHaveProperty("cache_read_tokens");
    expect(body).toHaveProperty("cache_creation_tokens");
    expect(body).toHaveProperty("estimated_cost_usd");
    expect(body).toHaveProperty("agent_count");
    expect(body).toHaveProperty("rate_limited");

    expect(typeof body.input_tokens).toBe("number");
    expect(typeof body.output_tokens).toBe("number");
    expect(typeof body.estimated_cost_usd).toBe("number");
    expect(typeof body.rate_limited).toBe("boolean");
  });

  test("stat cards render on overview page", async ({ page, serverURL }) => {
    await page.goto(serverURL);

    // Stat cards section should be visible with known labels.
    await expect(page.getByText("Agents")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("Tokens")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("Rate Limit")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("Est. API Cost")).toBeVisible({
      timeout: 5000,
    });
  });

  test("cost displays in USD format", async ({ page, serverURL }) => {
    await page.goto(serverURL);

    // Cost card should show a dollar sign.
    const costCard = page.locator("text=Est. API Cost").locator("..");
    await expect(costCard).toBeVisible({ timeout: 5000 });

    // The value within the card should contain "$".
    const costValue = costCard.locator("div").nth(1);
    const text = await costValue.textContent();
    expect(text).toContain("$");
  });
});

test.describe("Quota — Rate Limit Display", () => {
  test("rate limit card shows OK when not rate limited", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    // Check current rate limit state.
    const resp = await apiClient.get("/api/quota");
    const quota = await resp.json();

    await page.goto(serverURL);
    await expect(page.getByText("Rate Limit")).toBeVisible({ timeout: 5000 });

    if (!quota.rate_limited) {
      // Should show "OK" when not rate limited.
      await expect(page.getByText("OK")).toBeVisible({ timeout: 5000 });
    }
  });

  test("rate limit state reflected in quota response", async ({
    apiClient,
  }) => {
    const resp = await apiClient.get("/api/quota");
    const quota = await resp.json();

    expect(typeof quota.rate_limited).toBe("boolean");

    if (quota.rate_limited) {
      // When rate limited, retry_after_seconds should be present.
      expect(quota.retry_after_seconds).toBeDefined();
      expect(typeof quota.retry_after_seconds).toBe("number");
    }
  });

  test("rate limit card shows LIMITED when rate limited", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    const resp = await apiClient.get("/api/quota");
    const quota = await resp.json();

    await page.goto(serverURL);
    await expect(page.getByText("Rate Limit")).toBeVisible({ timeout: 5000 });

    if (quota.rate_limited) {
      await expect(page.getByText("LIMITED")).toBeVisible({ timeout: 5000 });
    } else {
      await expect(page.getByText("OK")).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe("Quota — SSE Updates", () => {
  test("SSE events endpoint is accessible", async ({ apiClient }) => {
    // The /events endpoint should be available for SSE connections.
    // We can't fully test SSE via fetch, but verify the endpoint exists.
    const resp = await apiClient.get("/events");
    // SSE endpoint may return 200 with text/event-stream or other status.
    expect([200, 204]).toContain(resp.status);
  });

  test("quota refreshes on page navigation", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    // Get initial quota via API.
    await apiClient.get("/api/quota");

    // Navigate to overview.
    await page.goto(serverURL);
    await expect(page.getByText("Tokens")).toBeVisible({ timeout: 5000 });

    // Navigate away and back to force a refetch.
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/quota-nav.git",
    });
    await page.goto(`${serverURL}/projects/quota-nav`);
    await page.goto(serverURL);

    // Stat cards should still render correctly.
    await expect(page.getByText("Tokens")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("Est. API Cost")).toBeVisible({
      timeout: 5000,
    });
  });

  test("quota data loads on initial page visit", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    // Tokens card should display values (even if zero).
    const tokensCard = page.locator("text=Tokens").locator("..");
    await expect(tokensCard).toBeVisible({ timeout: 5000 });

    // The card should have a value (formatted tokens like "0 / 0" or "1.5k / 800").
    const valueDiv = tokensCard.locator("div").nth(1);
    const text = await valueDiv.textContent();
    expect(text).toBeTruthy();
    // Should contain the separator between input/output.
    expect(text).toContain("/");
  });
});

test.describe("Quota — Aggregation", () => {
  test("quota includes per-agent breakdown when agents exist", async ({
    apiClient,
  }) => {
    const resp = await apiClient.get("/api/quota");
    const quota = await resp.json();

    // agents field is optional — present only if there are tracked agents.
    if (quota.agents) {
      expect(Array.isArray(quota.agents)).toBe(true);
      for (const agent of quota.agents) {
        expect(agent).toHaveProperty("agent_id");
        expect(agent).toHaveProperty("input_tokens");
        expect(agent).toHaveProperty("output_tokens");
        expect(agent).toHaveProperty("estimated_cost_usd");
      }
    }
  });

  test("agent count matches agents array length", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/quota");
    const quota = await resp.json();

    if (quota.agents && quota.agents.length > 0) {
      expect(quota.agent_count).toBeGreaterThanOrEqual(quota.agents.length);
    }
  });

  test("token totals are non-negative", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/quota");
    const quota = await resp.json();

    expect(quota.input_tokens).toBeGreaterThanOrEqual(0);
    expect(quota.output_tokens).toBeGreaterThanOrEqual(0);
    expect(quota.cache_read_tokens).toBeGreaterThanOrEqual(0);
    expect(quota.cache_creation_tokens).toBeGreaterThanOrEqual(0);
    expect(quota.estimated_cost_usd).toBeGreaterThanOrEqual(0);
    expect(quota.agent_count).toBeGreaterThanOrEqual(0);
  });
});

test.describe("Quota — Edge and Failure Cases", () => {
  test("zero quota displays gracefully", async ({ page, serverURL }) => {
    await page.goto(serverURL);

    // Even with zero usage, stat cards should render without errors.
    await expect(page.getByText("Agents")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("Est. API Cost")).toBeVisible({
      timeout: 5000,
    });

    // Cost should show "$0.00" for zero usage.
    const costCard = page.locator("text=Est. API Cost").locator("..");
    const costValue = costCard.locator("div").nth(1);
    const text = await costValue.textContent();
    expect(text).toContain("$");
  });

  test("concurrent quota requests return consistent data", async ({
    apiClient,
  }) => {
    const responses = await Promise.all(
      Array.from({ length: 5 }, () => apiClient.get("/api/quota")),
    );

    for (const resp of responses) {
      expect(resp.ok).toBe(true);
    }

    const bodies = await Promise.all(responses.map((r) => r.json()));

    // All concurrent reads should return the same snapshot.
    const first = bodies[0];
    for (const body of bodies.slice(1)) {
      expect(body.input_tokens).toBe(first.input_tokens);
      expect(body.output_tokens).toBe(first.output_tokens);
      expect(body.estimated_cost_usd).toBe(first.estimated_cost_usd);
    }
  });

  test("quota endpoint handles repeated rapid requests", async ({
    apiClient,
  }) => {
    // Fire 10 requests sequentially to check for rate-limiting or errors.
    for (let i = 0; i < 10; i++) {
      const resp = await apiClient.get("/api/quota");
      expect(resp.ok).toBe(true);
    }
  });
});
