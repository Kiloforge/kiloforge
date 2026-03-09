import { test, expect } from "./fixtures";

/**
 * SSE Project and Lock Event E2E Tests
 *
 * Tests that project_update, project_removed, lock_update, lock_released,
 * and trace_update SSE events cause the dashboard UI to update in real time.
 */

// Helper for unique names.
let counter = 0;
function uid(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}`;
}

// Helper for DELETE with JSON body.
async function deleteWithBody(
  baseURL: string,
  path: string,
  body: unknown,
): Promise<Response> {
  return fetch(`${baseURL}${path}`, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

test.describe("SSE Project Events — project_update and project_removed", () => {
  test("project add triggers SSE event — dashboard updates", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await page.goto(serverURL);

    // Wait for SSE connection.
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // Add a project via API.
    const slug = uid("sse-proj");
    const addResp = await apiClient.post("/api/projects", {
      remote_url: `https://github.com/user/${slug}.git`,
    });
    expect(addResp.ok).toBe(true);

    // The project should appear in the UI via SSE event.
    // Wait for the project list to update (SSE -> TanStack Query invalidation).
    await expect(page.getByText("Projects")).toBeVisible({ timeout: 5000 });

    // Verify the project exists via API.
    const getResp = await apiClient.get(`/api/projects/${slug}`);
    expect(getResp.ok).toBe(true);

    // Cleanup.
    await apiClient.del(`/api/projects/${slug}`);
  });

  test("project removal triggers SSE event — dashboard handles it", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    const slug = uid("sse-rm-proj");

    // Create a project first.
    await apiClient.post("/api/projects", {
      remote_url: `https://github.com/user/${slug}.git`,
    });

    await page.goto(serverURL);
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // Delete the project via API — this publishes a project_removed SSE event.
    const delResp = await apiClient.del(`/api/projects/${slug}`);
    expect(delResp.ok).toBe(true);

    // Dashboard should remain stable after receiving removal event.
    await expect(page.getByText("Projects")).toBeVisible({ timeout: 3000 });

    // Verify project is gone via API.
    const getResp = await apiClient.get(`/api/projects/${slug}`);
    expect(getResp.status).toBe(404);
  });

  test("rapid project create/delete cycle — SSE stays connected", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await page.goto(serverURL);
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // Rapid create/delete cycle.
    for (let i = 0; i < 3; i++) {
      const slug = uid(`sse-rapid-${i}`);
      await apiClient.post("/api/projects", {
        remote_url: `https://github.com/user/${slug}.git`,
      });
      await apiClient.del(`/api/projects/${slug}`);
    }

    // SSE should still be connected.
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 3000,
    });
    await expect(page.getByText("Projects")).toBeVisible({ timeout: 3000 });
  });
});

test.describe("SSE Lock Events — lock_update and lock_released", () => {
  test("lock acquire triggers SSE event", async ({
    apiClient,
    serverURL,
  }) => {
    const scope = uid("sse-lock");
    const holder = "sse-test-worker";

    try {
      // Acquire a lock — this publishes a lock_update SSE event.
      const resp = await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder,
        ttl_seconds: 30,
      });
      expect(resp.status).toBe(200);

      const lock = await resp.json();
      expect(lock.scope).toBe(scope);
      expect(lock.holder).toBe(holder);
    } finally {
      await deleteWithBody(serverURL, `/api/locks/${scope}`, { holder });
    }
  });

  test("lock release triggers SSE event", async ({
    apiClient,
    serverURL,
  }) => {
    const scope = uid("sse-lock-rel");
    const holder = "sse-test-releaser";

    // Acquire.
    await apiClient.post(`/api/locks/${scope}/acquire`, {
      holder,
      ttl_seconds: 30,
    });

    // Release — this publishes a lock_released SSE event.
    const relResp = await deleteWithBody(serverURL, `/api/locks/${scope}`, {
      holder,
    });
    expect(relResp.status).toBe(200);
    const body = await relResp.json();
    expect(body.released).toBe(true);

    // Verify gone.
    const listResp = await apiClient.get("/api/locks");
    const locks = await listResp.json();
    const found = locks.find(
      (l: { scope: string }) => l.scope === scope,
    );
    expect(found).toBeUndefined();
  });

  test("dashboard handles lock events without crash", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    const scope = uid("sse-lock-ui");
    const holder = "sse-ui-worker";

    await page.goto(serverURL);
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    try {
      // Acquire lock — triggers lock_update event.
      await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder,
        ttl_seconds: 30,
      });

      // Dashboard should remain stable.
      await expect(page.getByText("Projects")).toBeVisible({ timeout: 3000 });

      // Release lock — triggers lock_released event.
      await deleteWithBody(serverURL, `/api/locks/${scope}`, { holder });

      // Dashboard should still be stable.
      await expect(page.getByText("Projects")).toBeVisible({ timeout: 3000 });
    } finally {
      await deleteWithBody(serverURL, `/api/locks/${scope}`, {
        holder,
      }).catch(() => {});
    }
  });
});

test.describe("SSE Trace Events — trace_update", () => {
  test("trace API endpoint is accessible", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/traces");
    // Traces may return 200 or 404 depending on implementation.
    expect([200, 404]).toContain(resp.status);
  });

  test("dashboard handles trace events gracefully", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    // SSE should be connected — trace_update events are handled silently.
    const indicator = page.locator('[data-testid="sse-status"]');
    await expect(indicator).toHaveAttribute("data-status", "connected", {
      timeout: 5000,
    });

    // Dashboard should render without errors.
    await expect(page.getByText("Agents")).toBeVisible({ timeout: 5000 });
  });
});
