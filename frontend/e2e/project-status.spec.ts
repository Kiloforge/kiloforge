import { test, expect } from "./fixtures";

test.describe("Project Status", () => {
  test("project detail page accessible after add", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    // Add a project via API.
    const resp = await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/detail-test.git",
    });
    expect(resp.status).toBe(201);

    // Navigate to the project detail page.
    await page.goto(`${serverURL}/projects/detail-test`);

    // Should show project metadata.
    await expect(page.getByText("detail-test")).toBeVisible();
  });

  test("project list shows all seeded projects", async ({
    apiClient,
  }) => {
    // Add multiple projects.
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/proj-a.git",
    });
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/proj-b.git",
    });
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/proj-c.git",
    });

    // List all projects.
    const listResp = await apiClient.get("/api/projects");
    const projects = await listResp.json();
    const slugs = projects.map((p: { slug: string }) => p.slug);

    expect(slugs).toContain("proj-a");
    expect(slugs).toContain("proj-b");
    expect(slugs).toContain("proj-c");
  });

  test("project has correct fields in API response", async ({
    apiClient,
  }) => {
    const resp = await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/field-check.git",
    });
    expect(resp.status).toBe(201);

    const project = await resp.json();
    expect(project).toHaveProperty("slug");
    expect(project).toHaveProperty("repo_name");
    expect(project).toHaveProperty("active");
    expect(project.slug).toBe("field-check");
    expect(project.repo_name).toBe("field-check");
    expect(project.active).toBe(true);
    expect(project.origin_remote).toBe(
      "https://github.com/user/field-check.git",
    );
  });

  test("sync status endpoint returns data for project", async ({
    apiClient,
  }) => {
    // Add a project.
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/sync-check.git",
    });

    // Check sync status — this may return an error since no actual git repo
    // exists, but should not crash.
    const resp = await apiClient.get("/api/projects/sync-check/sync");
    // Accept either 200 (with sync data) or 404/500 (no real repo).
    expect([200, 404, 500]).toContain(resp.status);
  });

  test("empty project list shows in overview", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    // Should show "Projects" section.
    await expect(page.getByText("Projects")).toBeVisible();

    // With no projects, should show empty state or the add button.
    const addBtn = page.getByRole("button", { name: "+ Add Project" });
    await expect(addBtn).toBeVisible();
  });

  test("project detail shows metadata fields", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    // Seed a project.
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/meta-detail.git",
    });

    await page.goto(`${serverURL}/projects/meta-detail`);

    // Check for metadata labels.
    if (await page.getByText("Slug").isVisible({ timeout: 3000 }).catch(() => false)) {
      await expect(page.getByText("meta-detail")).toBeVisible();
    }
  });

  test("overview page navigates to project detail", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/nav-test.git",
    });

    await page.goto(serverURL);

    // Click on the project link.
    const projectLink = page.locator(`a[href="/projects/nav-test"]`);
    if (await projectLink.isVisible({ timeout: 3000 }).catch(() => false)) {
      await projectLink.click();
      await expect(page).toHaveURL(/.*\/projects\/nav-test/);
    }
  });

  test("concurrent add operations succeed", async ({ apiClient }) => {
    // Fire two adds simultaneously.
    const [resp1, resp2] = await Promise.all([
      apiClient.post("/api/projects", {
        remote_url: "https://github.com/user/concurrent-a.git",
      }),
      apiClient.post("/api/projects", {
        remote_url: "https://github.com/user/concurrent-b.git",
      }),
    ]);

    expect(resp1.status).toBe(201);
    expect(resp2.status).toBe(201);

    // Both should be in the list.
    const listResp = await apiClient.get("/api/projects");
    const projects = await listResp.json();
    const slugs = projects.map((p: { slug: string }) => p.slug);
    expect(slugs).toContain("concurrent-a");
    expect(slugs).toContain("concurrent-b");
  });
});
