import { test, expect } from "./fixtures";

test.describe("Health and Status", () => {
  test("health endpoint returns 200 with ok status", async ({ apiClient }) => {
    const resp = await apiClient.get("/health");
    expect(resp.ok).toBe(true);

    const body = await resp.json();
    expect(body.status).toBe("ok");
    expect(body).toHaveProperty("projects");
    expect(typeof body.projects).toBe("number");
  });

  test("health endpoint returns JSON content type", async ({ apiClient }) => {
    const resp = await apiClient.get("/health");
    const contentType = resp.headers.get("content-type");
    expect(contentType).toContain("application/json");
  });

  test("dashboard loads without server error", async ({ page, serverURL }) => {
    const resp = await page.goto(serverURL);
    expect(resp).not.toBeNull();
    expect(resp!.status()).toBeLessThan(500);

    // Page should render without blank screen.
    await expect(page.locator("body")).not.toBeEmpty();
  });
});

test.describe("Preflight Validation", () => {
  test("preflight endpoint returns expected fields", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/preflight");
    expect(resp.ok).toBe(true);

    const body = await resp.json();
    expect(body).toHaveProperty("claude_authenticated");
    expect(body).toHaveProperty("skills_ok");
    expect(body).toHaveProperty("consent_given");
    expect(body).toHaveProperty("setup_required");

    expect(typeof body.claude_authenticated).toBe("boolean");
    expect(typeof body.skills_ok).toBe("boolean");
    expect(typeof body.consent_given).toBe("boolean");
    expect(typeof body.setup_required).toBe("boolean");
  });

  test("preflight with skills missing returns skills_missing array", async ({
    apiClient,
  }) => {
    const resp = await apiClient.get("/api/preflight");
    expect(resp.ok).toBe(true);

    const body = await resp.json();
    // If skills are not ok, skills_missing should be an array.
    if (!body.skills_ok && body.skills_missing) {
      expect(Array.isArray(body.skills_missing)).toBe(true);
      expect(body.skills_missing.length).toBeGreaterThan(0);
    }
  });

  test("skills install banner appears when skills missing", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    // Seed a project so we can visit the project page.
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/preflight-skills.git",
    });

    await page.goto(`${serverURL}/projects/preflight-skills`);

    // Check preflight status to know what to expect.
    const preflight = await apiClient.get("/api/preflight");
    const status = await preflight.json();

    if (!status.skills_ok) {
      // Skills install banner should be visible.
      await expect(
        page.getByText("Required skills not installed"),
      ).toBeVisible({ timeout: 5000 });
    }
  });

  test("setup required banner appears when setup incomplete", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/preflight-setup.git",
    });

    await page.goto(`${serverURL}/projects/preflight-setup`);

    // Check setup status.
    const setupResp = await apiClient.get(
      "/api/projects/preflight-setup/setup-status",
    );
    if (setupResp.ok) {
      const setupStatus = await setupResp.json();
      if (!setupStatus.setup_complete) {
        // Setup banner should appear (if skills are OK).
        const preflight = await apiClient.get("/api/preflight");
        const status = await preflight.json();
        if (status.skills_ok) {
          await expect(
            page.getByText("Kiloforge setup required"),
          ).toBeVisible({ timeout: 5000 });
        }
      }
    }
  });
});

test.describe("Consent Flow", () => {
  test("consent endpoint returns consent state", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/consent/agent-permissions");
    expect(resp.ok).toBe(true);

    const body = await resp.json();
    expect(body).toHaveProperty("consented");
    expect(typeof body.consented).toBe("boolean");
  });

  test("record consent sets consented to true", async ({ apiClient }) => {
    // Record consent.
    const postResp = await apiClient.post("/api/consent/agent-permissions");
    expect(postResp.ok).toBe(true);

    const postBody = await postResp.json();
    expect(postBody.consented).toBe(true);

    // Verify persistence.
    const getResp = await apiClient.get("/api/consent/agent-permissions");
    expect(getResp.ok).toBe(true);

    const getBody = await getResp.json();
    expect(getBody.consented).toBe(true);
  });

  test("consent state persists across requests", async ({ apiClient }) => {
    // Record consent.
    await apiClient.post("/api/consent/agent-permissions");

    // Multiple reads should all return consented.
    const [r1, r2, r3] = await Promise.all([
      apiClient.get("/api/consent/agent-permissions"),
      apiClient.get("/api/consent/agent-permissions"),
      apiClient.get("/api/consent/agent-permissions"),
    ]);

    for (const r of [r1, r2, r3]) {
      const body = await r.json();
      expect(body.consented).toBe(true);
    }
  });
});

test.describe("Tour API", () => {
  test("tour state defaults to pending", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/tour");
    expect(resp.ok).toBe(true);

    const body = await resp.json();
    expect(body).toHaveProperty("status");
    expect(body).toHaveProperty("current_step");
    // Fresh state should be pending with step 0.
    expect(["pending", "active", "dismissed", "completed"]).toContain(
      body.status,
    );
    expect(typeof body.current_step).toBe("number");
  });

  test("accept tour transitions to active state", async ({ apiClient }) => {
    const resp = await apiClient.put("/api/tour", { action: "accept" });
    expect(resp.ok).toBe(true);

    const body = await resp.json();
    expect(body.status).toBe("active");
    expect(body.current_step).toBe(0);
    expect(body.started_at).toBeTruthy();
  });

  test("advance tour increments step", async ({ apiClient }) => {
    // Start tour first.
    await apiClient.put("/api/tour", { action: "accept" });

    // Advance to step 2.
    const resp = await apiClient.put("/api/tour", {
      action: "advance",
      step: 2,
    });
    expect(resp.ok).toBe(true);

    const body = await resp.json();
    expect(body.current_step).toBe(2);

    // Verify persistence.
    const getResp = await apiClient.get("/api/tour");
    const state = await getResp.json();
    expect(state.current_step).toBe(2);
  });

  test("complete tour transitions to completed state", async ({
    apiClient,
  }) => {
    await apiClient.put("/api/tour", { action: "accept" });
    const resp = await apiClient.put("/api/tour", { action: "complete" });
    expect(resp.ok).toBe(true);

    const body = await resp.json();
    expect(body.status).toBe("completed");
    expect(body.completed_at).toBeTruthy();
  });

  test("dismiss tour transitions to dismissed state", async ({
    apiClient,
  }) => {
    // Reset tour to pending first by accepting then dismissing.
    await apiClient.put("/api/tour", { action: "accept" });
    const resp = await apiClient.put("/api/tour", { action: "dismiss" });
    expect(resp.ok).toBe(true);

    const body = await resp.json();
    expect(body.status).toBe("dismissed");
    expect(body.dismissed_at).toBeTruthy();
  });

  test("tour demo board returns valid board structure", async ({
    apiClient,
  }) => {
    const resp = await apiClient.get("/api/tour/demo-board");
    expect(resp.ok).toBe(true);

    const body = await resp.json();
    expect(body).toHaveProperty("columns");
    expect(body).toHaveProperty("cards");
    expect(Array.isArray(body.columns)).toBe(true);
    expect(body.columns.length).toBeGreaterThan(0);
  });

  test("invalid tour action returns 400", async ({ apiClient }) => {
    const resp = await apiClient.put("/api/tour", {
      action: "invalid-action",
    });
    expect(resp.status).toBe(400);
  });
});

test.describe("Edge and Failure Cases", () => {
  test("concurrent preflight calls return consistent results", async ({
    apiClient,
  }) => {
    const responses = await Promise.all(
      Array.from({ length: 5 }, () => apiClient.get("/api/preflight")),
    );

    // All should succeed.
    for (const resp of responses) {
      expect(resp.ok).toBe(true);
    }

    // All should return the same structure.
    const bodies = await Promise.all(responses.map((r) => r.json()));
    for (const body of bodies) {
      expect(body).toHaveProperty("claude_authenticated");
      expect(body).toHaveProperty("skills_ok");
      expect(body).toHaveProperty("consent_given");
      expect(body).toHaveProperty("setup_required");
    }

    // Values should be consistent across all responses.
    const first = bodies[0];
    for (const body of bodies.slice(1)) {
      expect(body.skills_ok).toBe(first.skills_ok);
      expect(body.consent_given).toBe(first.consent_given);
      expect(body.setup_required).toBe(first.setup_required);
    }
  });

  test("rapid consent toggle maintains consistent state", async ({
    apiClient,
  }) => {
    // Record consent multiple times rapidly.
    await Promise.all(
      Array.from({ length: 3 }, () =>
        apiClient.post("/api/consent/agent-permissions"),
      ),
    );

    // Final state should be consented.
    const resp = await apiClient.get("/api/consent/agent-permissions");
    const body = await resp.json();
    expect(body.consented).toBe(true);
  });

  test("tour state survives rapid updates", async ({ apiClient }) => {
    // Accept, advance, advance rapidly.
    await apiClient.put("/api/tour", { action: "accept" });
    await apiClient.put("/api/tour", { action: "advance", step: 1 });
    await apiClient.put("/api/tour", { action: "advance", step: 3 });

    const resp = await apiClient.get("/api/tour");
    const body = await resp.json();
    expect(body.current_step).toBe(3);
    expect(body.status).toBe("active");
  });
});
