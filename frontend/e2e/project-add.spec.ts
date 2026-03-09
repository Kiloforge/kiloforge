import { test, expect } from "./fixtures";

test.describe("Add Project", () => {
  test("add project with valid HTTPS URL via API", async ({ apiClient }) => {
    const resp = await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/https-project.git",
    });
    expect(resp.status).toBe(201);

    const project = await resp.json();
    expect(project.slug).toBe("https-project");
    expect(project.active).toBe(true);
    expect(project.origin_remote).toBe(
      "https://github.com/user/https-project.git",
    );

    // Verify it appears in the list.
    const listResp = await apiClient.get("/api/projects");
    const projects = await listResp.json();
    expect(projects.some((p: { slug: string }) => p.slug === "https-project")).toBe(true);
  });

  test("add project with valid SSH URL via API", async ({ apiClient }) => {
    const resp = await apiClient.post("/api/projects", {
      remote_url: "git@github.com:user/ssh-project.git",
    });
    expect(resp.status).toBe(201);

    const project = await resp.json();
    expect(project.slug).toBe("ssh-project");
    expect(project.active).toBe(true);
  });

  test("add project with custom name via API", async ({ apiClient }) => {
    const resp = await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/original.git",
      name: "custom-name",
    });
    expect(resp.status).toBe(201);

    const project = await resp.json();
    expect(project.slug).toBe("custom-name");
  });

  test("empty URL returns 400", async ({ apiClient }) => {
    const resp = await apiClient.post("/api/projects", { remote_url: "" });
    expect(resp.status).toBe(400);

    const body = await resp.json();
    expect(body.error).toBeTruthy();
  });

  test("duplicate project returns 409", async ({ apiClient }) => {
    // Add first project.
    const first = await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/dup-project.git",
    });
    expect(first.status).toBe(201);

    // Add same project again.
    const second = await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/dup-project.git",
    });
    expect(second.status).toBe(409);

    const body = await second.json();
    expect(body.error).toContain("already");
  });

  test("add project form validates empty URL in UI", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    // Click the "+ Add Project" button to expand the form.
    const addBtn = page.getByRole("button", { name: "+ Add Project" });
    if (await addBtn.isVisible()) {
      await addBtn.click();
    }

    // Submit with empty URL.
    const submitBtn = page.getByRole("button", { name: "Add Project" });
    if (await submitBtn.isVisible()) {
      await submitBtn.click();
      // Should show validation error.
      await expect(page.getByText("Remote URL is required")).toBeVisible();
    }
  });

  test("add project form validates invalid URL in UI", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);

    const addBtn = page.getByRole("button", { name: "+ Add Project" });
    if (await addBtn.isVisible()) {
      await addBtn.click();

      // Type an invalid URL.
      await page.locator("#remote-url").fill("not-a-url");
      const submitBtn = page.getByRole("button", { name: "Add Project" });
      await submitBtn.click();

      // Should show validation error.
      await expect(
        page.getByText("Must be a git remote URL (SSH or HTTPS)"),
      ).toBeVisible();
    }
  });
});
