import { test, expect } from "./fixtures";

test.describe("Settings Menu", () => {
  test.beforeEach(async ({ apiClient }) => {
    await apiClient.put("/api/tour", { action: "complete" }).catch(() => {});
  });

  test("gear icon opens/closes dropdown", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    const settingsBtn = page.locator("button[title='Settings']");
    await expect(settingsBtn).toBeVisible({ timeout: 10_000 });

    await expect(page.locator("text=Take Tour")).not.toBeVisible();

    await settingsBtn.click();
    await expect(page.locator("text=Take Tour")).toBeVisible();

    await settingsBtn.click();
    await expect(page.locator("text=Take Tour")).not.toBeVisible();
  });

  test("dropdown closes when clicking outside", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    const settingsBtn = page.locator("button[title='Settings']");
    await expect(settingsBtn).toBeVisible({ timeout: 10_000 });

    await settingsBtn.click();
    await expect(page.locator("text=Take Tour")).toBeVisible();

    await page.locator("main").click();
    await expect(page.locator("text=Take Tour")).not.toBeVisible();
  });

  test("Take Tour restarts tour and shows welcome dialog", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    const settingsBtn = page.locator("button[title='Settings']");
    await expect(settingsBtn).toBeVisible({ timeout: 10_000 });

    await settingsBtn.click();
    await page.locator("text=Take Tour").click();

    await expect(page.locator("text=Welcome to Kiloforge")).toBeVisible({ timeout: 5_000 });
  });

  test("analytics toggle is visible and functional", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    const settingsBtn = page.locator("button[title='Settings']");
    await expect(settingsBtn).toBeVisible({ timeout: 10_000 });

    await settingsBtn.click();
    await expect(page.locator("text=Anonymous usage data")).toBeVisible();
    await expect(page.locator("text=Help improve kiloforge")).toBeVisible();

    const toggle = page.locator("button[role='switch']");
    await expect(toggle).toBeVisible();

    // Toggle off and back on
    await toggle.click();
    await toggle.click();
  });

  test("analytics toggle persists state via API", async ({ page, serverURL, apiClient }) => {
    await page.goto(serverURL);
    const settingsBtn = page.locator("button[title='Settings']");
    await expect(settingsBtn).toBeVisible({ timeout: 10_000 });

    await settingsBtn.click();
    const toggle = page.locator("button[role='switch']");
    await expect(toggle).toBeVisible();

    // Toggle off and verify API state
    await toggle.click();
    await page.waitForTimeout(500);
    const resp = await apiClient.get("/api/config");
    const config = await resp.json();
    expect(config.analytics_enabled).toBe(false);

    // Toggle back on
    await toggle.click();
    await page.waitForTimeout(500);
    const resp2 = await apiClient.get("/api/config");
    const config2 = await resp2.json();
    expect(config2.analytics_enabled).toBe(true);
  });

  test("completion toast appears after finishing tour", async ({ page, serverURL, apiClient }) => {
    // Reset tour to pending so we can complete it through the UI
    await apiClient.put("/api/tour", { action: "reset" }).catch(() => {});
    await page.goto(serverURL);

    // Start the tour
    const startBtn = page.locator("text=Start Tour");
    await expect(startBtn).toBeVisible({ timeout: 10_000 });
    await startBtn.click();

    // Advance through welcome step
    await expect(page.locator("text=Let's Go")).toBeVisible();
    await page.locator("text=Let's Go").click();

    // Skip and finish
    await expect(page.locator("text=Next")).toBeVisible();
    await page.locator("text=Next").click();
    await page.locator("text=Next").click();
    await page.locator("text=Next").click();
    await page.locator("text=Next").click();
    await page.locator("text=Next").click();

    // Last step — skip and finish
    await page.locator("text=Skip and finish tour").click();

    // Toast should appear
    await expect(page.locator("text=Tour complete")).toBeVisible({ timeout: 5_000 });
  });
});
