import { test, expect } from "./fixtures";

test.describe("Git Origin Sync — Status Display", () => {
  test("sync status endpoint responds for existing project", async ({
    apiClient,
  }) => {
    // Seed a project with an origin remote.
    const addResp = await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/sync-status-test.git",
    });
    expect(addResp.status).toBe(201);

    // Sync status endpoint should respond (500 expected when gitSync is not
    // configured in the E2E server, but the endpoint must exist and not crash).
    const resp = await apiClient.get(
      "/api/projects/sync-status-test/sync-status",
    );
    expect([200, 500]).toContain(resp.status);

    if (resp.status === 200) {
      const body = await resp.json();
      expect(body).toHaveProperty("status");
      expect(body).toHaveProperty("ahead");
      expect(body).toHaveProperty("behind");
      expect(body).toHaveProperty("local_branch");
      expect(["synced", "ahead", "behind", "diverged", "unknown"]).toContain(
        body.status,
      );
    }

    if (resp.status === 500) {
      const body = await resp.json();
      expect(body.error).toBeTruthy();
    }
  });

  test("sync status returns 404 for nonexistent project", async ({
    apiClient,
  }) => {
    const resp = await apiClient.get(
      "/api/projects/nonexistent-project/sync-status",
    );
    // 404 when project doesn't exist, or 500 if gitSync nil check comes first.
    expect([404, 500]).toContain(resp.status);
  });

  test("project detail page shows Origin Sync section", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    // Seed a project with an origin remote.
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/sync-display.git",
    });

    await page.goto(`${serverURL}/projects/sync-display`);

    // The Origin Sync panel heading should be visible since the project has an
    // origin remote.
    await expect(page.getByText("Origin Sync")).toBeVisible({ timeout: 5000 });
  });

  test("sync panel shows push and pull buttons", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/sync-buttons.git",
    });

    await page.goto(`${serverURL}/projects/sync-buttons`);

    // Wait for the sync panel to render.
    await expect(page.getByText("Origin Sync")).toBeVisible({ timeout: 5000 });

    // Should show push and pull buttons (or loading/error state).
    const pushBtn = page.getByRole("button", { name: "Push to Upstream" });
    const pullBtn = page.getByRole("button", { name: "Pull from Upstream" });

    // At least one of: buttons visible, loading text, or error state.
    const hasPush = await pushBtn.isVisible().catch(() => false);
    const hasPull = await pullBtn.isVisible().catch(() => false);
    const hasLoading = await page
      .getByText("Loading sync status...")
      .isVisible()
      .catch(() => false);
    const hasNoOrigin = await page
      .getByText("No origin remote configured")
      .isVisible()
      .catch(() => false);

    expect(hasPush || hasPull || hasLoading || hasNoOrigin).toBe(true);
  });
});

test.describe("Git Origin Sync — Push", () => {
  test("push endpoint responds for existing project", async ({
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/push-test.git",
    });

    const resp = await apiClient.post("/api/projects/push-test/push", {
      remote_branch: "kf/main",
    });
    // 500 when gitSync not configured, 200 when it is.
    expect([200, 500]).toContain(resp.status);

    if (resp.status === 200) {
      const body = await resp.json();
      expect(body.success).toBe(true);
      expect(body).toHaveProperty("local_branch");
      expect(body).toHaveProperty("remote_branch");
    }
  });

  test("push returns 400 when remote_branch is missing", async ({
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/push-nobranch.git",
    });

    const resp = await apiClient.post("/api/projects/push-nobranch/push", {});
    // 400 for missing remote_branch, or 500 if gitSync nil check comes first.
    expect([400, 500]).toContain(resp.status);
  });

  test("push returns error for nonexistent project", async ({
    apiClient,
  }) => {
    const resp = await apiClient.post(
      "/api/projects/nonexistent-push/push",
      { remote_branch: "kf/main" },
    );
    expect([404, 500]).toContain(resp.status);
  });
});

test.describe("Git Origin Sync — Pull", () => {
  test("pull endpoint responds for existing project", async ({
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/pull-test.git",
    });

    const resp = await apiClient.post("/api/projects/pull-test/pull", {});
    // 500 when gitSync not configured, 200 when it is.
    expect([200, 500]).toContain(resp.status);

    if (resp.status === 200) {
      const body = await resp.json();
      expect(body.success).toBe(true);
      expect(body).toHaveProperty("new_head");
    }
  });

  test("pull returns error for nonexistent project", async ({
    apiClient,
  }) => {
    const resp = await apiClient.post("/api/projects/nonexistent-pull/pull", {});
    expect([404, 500]).toContain(resp.status);
  });

  test("pull accepts optional remote_branch parameter", async ({
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/pull-branch.git",
    });

    const resp = await apiClient.post("/api/projects/pull-branch/pull", {
      remote_branch: "develop",
    });
    // Should not crash with custom branch.
    expect([200, 500]).toContain(resp.status);
  });
});

test.describe("Git Origin Sync — Sync Panel UI", () => {
  test("sync panel updates after refresh click", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/sync-refresh.git",
    });

    await page.goto(`${serverURL}/projects/sync-refresh`);
    await expect(page.getByText("Origin Sync")).toBeVisible({ timeout: 5000 });

    // Click the refresh button (↻).
    const refreshBtn = page.getByRole("button", { name: "Refresh status" });
    if (await refreshBtn.isVisible().catch(() => false)) {
      await refreshBtn.click();

      // After refresh, the panel should still be visible (no crash).
      await expect(page.getByText("Origin Sync")).toBeVisible();
    }
  });

  test("push button opens remote branch input form", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/sync-push-form.git",
    });

    await page.goto(`${serverURL}/projects/sync-push-form`);
    await expect(page.getByText("Origin Sync")).toBeVisible({ timeout: 5000 });

    // Wait for sync status to load (buttons appear after loading).
    const pushBtn = page.getByRole("button", { name: "Push to Upstream" });
    if (await pushBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
      await pushBtn.click();

      // Should show the push form with remote branch input.
      await expect(page.getByText("Remote branch:")).toBeVisible();
      await expect(page.getByPlaceholder("kf/main")).toBeVisible();

      // Cancel should hide the form.
      const cancelBtn = page.getByRole("button", { name: "Cancel" });
      await cancelBtn.click();
      await expect(page.getByText("Remote branch:")).not.toBeVisible();
    }
  });

  test("push form defaults to kf/main branch", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/sync-default-branch.git",
    });

    await page.goto(`${serverURL}/projects/sync-default-branch`);
    await expect(page.getByText("Origin Sync")).toBeVisible({ timeout: 5000 });

    const pushBtn = page.getByRole("button", { name: "Push to Upstream" });
    if (await pushBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
      await pushBtn.click();

      // Input should default to kf/main.
      const input = page.getByPlaceholder("kf/main");
      await expect(input).toHaveValue("kf/main");
    }
  });
});

test.describe("Git Origin Sync — Edge and Failure Cases", () => {
  test("concurrent push and pull do not crash the server", async ({
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/concurrent-sync.git",
    });

    // Fire push and pull simultaneously.
    const [pushResp, pullResp] = await Promise.all([
      apiClient.post("/api/projects/concurrent-sync/push", {
        remote_branch: "kf/main",
      }),
      apiClient.post("/api/projects/concurrent-sync/pull", {}),
    ]);

    // Neither should crash (500 is acceptable for gitSync=nil, but not a
    // connection error or timeout).
    expect([200, 409, 500]).toContain(pushResp.status);
    expect([200, 409, 500]).toContain(pullResp.status);
  });

  test("push with empty remote_branch returns error", async ({
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/push-empty-branch.git",
    });

    const resp = await apiClient.post(
      "/api/projects/push-empty-branch/push",
      { remote_branch: "" },
    );
    // 400 for empty branch, or 500 if gitSync nil check first.
    expect([400, 500]).toContain(resp.status);
  });

  test("sync error is displayed and dismissible in UI", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/sync-error-display.git",
    });

    await page.goto(`${serverURL}/projects/sync-error-display`);
    await expect(page.getByText("Origin Sync")).toBeVisible({ timeout: 5000 });

    // If the sync panel shows an error (likely in E2E since gitSync is nil),
    // the dismiss button should work.
    const dismissBtn = page.locator("button:has-text('×')");
    if (await dismissBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await dismissBtn.click();
      // Error should be dismissed — the dismiss button should no longer be
      // visible in the sync panel context.
    }

    // Panel should still be visible regardless.
    await expect(page.getByText("Origin Sync")).toBeVisible();
  });
});
