import { test, expect } from "./fixtures";

/**
 * Agent Launcher E2E Tests (Playwright)
 *
 * Tests the AgentLauncher dialog — role selection, prompt input,
 * and spawn API integration visible in the browser.
 */

test.describe("Agent Launcher — Overview Page", () => {
  test("New Agent button is visible on dashboard", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);
    const btn = page.getByText("New Agent");
    await expect(btn.first()).toBeVisible({ timeout: 5000 });
  });

  test("clicking New Agent opens launcher dialog", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);
    await page.getByText("New Agent").first().click();

    // Dialog should show role options
    await expect(page.getByText("Architect")).toBeVisible({ timeout: 3000 });
    await expect(page.getByText("Product Advisor")).toBeVisible();
    await expect(page.getByText("Start")).toBeVisible();
    await expect(page.getByText("Cancel")).toBeVisible();
  });

  test("cancel closes launcher dialog", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    await page.getByText("New Agent").first().click();
    await expect(page.getByText("Architect")).toBeVisible({ timeout: 3000 });

    await page.getByText("Cancel").click();
    await expect(page.getByText("Architect")).not.toBeVisible({ timeout: 3000 });
  });

  test("spawn sends role and prompt to API", async ({
    page,
    serverURL,
  }) => {
    await page.goto(serverURL);
    await page.getByText("New Agent").first().click();
    await expect(page.getByText("Architect")).toBeVisible({ timeout: 3000 });

    // Select product-advisor role
    await page.getByText("Product Advisor").click();

    // Enter prompt
    const textarea = page.locator("textarea");
    await textarea.fill("Help me with branding");

    // Intercept the spawn API call
    const requestPromise = page.waitForRequest(
      (req) =>
        req.url().includes("/api/agents/interactive") &&
        req.method() === "POST",
    );

    await page.getByText("Start").click();

    const request = await requestPromise;
    const body = request.postDataJSON();
    expect(body.role).toBe("product-advisor");
    expect(body.prompt).toBe("Help me with branding");
  });
});

test.describe("Agent Launcher — Project Page", () => {
  test("New Agent button replaces Generate Tracks", async ({
    page,
    serverURL,
  }) => {
    // Navigate to a project page — even if project doesn't exist, page loads
    await page.goto(`${serverURL}/projects/test-project`);
    const btn = page.getByText("New Agent");
    // There may be multiple matches if overview also has it in nav
    const visible = await btn.first().isVisible({ timeout: 5000 }).catch(() => false);
    // Project page should have the button in the board section
    expect(visible).toBe(true);
  });
});
