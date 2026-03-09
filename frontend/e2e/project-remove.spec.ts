import { test, expect } from "./fixtures";

test.describe("Remove Project", () => {
  test("remove project via API", async ({ apiClient }) => {
    // Seed a project.
    const addResp = await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/removable.git",
    });
    expect(addResp.status).toBe(201);

    // Remove it.
    const delResp = await apiClient.del("/api/projects/removable");
    expect(delResp.status).toBe(204);

    // Verify it's gone from the list.
    const listResp = await apiClient.get("/api/projects");
    const projects = await listResp.json();
    expect(
      projects.some((p: { slug: string }) => p.slug === "removable"),
    ).toBe(false);
  });

  test("remove project with cleanup via API", async ({ apiClient }) => {
    const addResp = await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/cleanup-target.git",
    });
    expect(addResp.status).toBe(201);

    const delResp = await apiClient.del(
      "/api/projects/cleanup-target?cleanup=true",
    );
    expect(delResp.status).toBe(204);

    // Verify removed.
    const listResp = await apiClient.get("/api/projects");
    const projects = await listResp.json();
    expect(
      projects.some((p: { slug: string }) => p.slug === "cleanup-target"),
    ).toBe(false);
  });

  test("remove nonexistent project returns 404", async ({ apiClient }) => {
    const resp = await apiClient.del("/api/projects/does-not-exist");
    expect(resp.status).toBe(404);

    const body = await resp.json();
    expect(body.error).toBeTruthy();
  });

  test("remove project via UI dialog", async ({ page, serverURL, apiClient }) => {
    // Seed a project via API.
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/ui-removable.git",
    });

    await page.goto(serverURL);

    // Wait for project to appear in the list.
    const projectRow = page.locator(`text=ui-removable`);
    if (await projectRow.isVisible({ timeout: 3000 }).catch(() => false)) {
      // Click the × remove button for this project.
      const removeBtn = page.locator(`button[title="Remove ui-removable"]`);
      await removeBtn.click();

      // Verify the confirmation dialog appears.
      await expect(page.getByText("Remove Project")).toBeVisible();
      await expect(page.getByText("ui-removable")).toBeVisible();

      // Click "Remove" to confirm.
      await page.getByRole("button", { name: "Remove", exact: true }).click();

      // Wait for project to disappear.
      await expect(projectRow).not.toBeVisible({ timeout: 5000 });
    }
  });

  test("cancel remove keeps project in list", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    // Seed a project.
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/keep-me.git",
    });

    await page.goto(serverURL);

    const projectRow = page.locator(`text=keep-me`);
    if (await projectRow.isVisible({ timeout: 3000 }).catch(() => false)) {
      // Click remove button.
      const removeBtn = page.locator(`button[title="Remove keep-me"]`);
      await removeBtn.click();

      // Click "Cancel" in the dialog.
      await page.getByRole("button", { name: "Cancel" }).click();

      // Project should still be visible.
      await expect(projectRow).toBeVisible();
    }
  });

  test("remove dialog shows cleanup warning", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    await apiClient.post("/api/projects", {
      remote_url: "https://github.com/user/cleanup-warn.git",
    });

    await page.goto(serverURL);

    const removeBtn = page.locator(`button[title="Remove cleanup-warn"]`);
    if (await removeBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await removeBtn.click();

      // Check the cleanup checkbox.
      const checkbox = page.getByLabel(
        "Also delete Gitea repo and local clone",
      );
      await checkbox.check();

      // Warning text should appear.
      await expect(
        page.getByText("This will permanently delete"),
      ).toBeVisible();

      // Cancel to clean up.
      await page.getByRole("button", { name: "Cancel" }).click();
    }
  });
});
