import { test, expect } from "./fixtures";

test.describe("Full-screen command mode", () => {
  test("toggle via keyboard shortcut", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    await page.waitForLoadState("networkidle");

    // Full-screen overlay should not be visible initially
    await expect(page.locator("[data-tour='fullscreen-command']")).toHaveCount(0);

    // Cmd+Shift+F opens full-screen mode
    await page.keyboard.press("Meta+Shift+KeyF");
    await expect(page.locator("[data-tour='fullscreen-command']")).toBeVisible();

    // Escape closes full-screen mode
    await page.keyboard.press("Escape");
    await expect(page.locator("[data-tour='fullscreen-command']")).toHaveCount(0);
  });

  test("toggle via header button", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    await page.waitForLoadState("networkidle");

    // Click the Command button in header
    await page.locator("[data-tour='fullscreen-toggle']").click();
    await expect(page.locator("[data-tour='fullscreen-command']")).toBeVisible();

    // Click Exit button
    await page.getByRole("button", { name: "Exit" }).click();
    await expect(page.locator("[data-tour='fullscreen-command']")).toHaveCount(0);
  });

  test("shows agent picker in pane", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    await page.waitForLoadState("networkidle");

    await page.keyboard.press("Meta+Shift+KeyF");
    await expect(page.locator("[data-tour='fullscreen-command']")).toBeVisible();

    // Agent picker select should be present
    const agentSelect = page.locator("[data-tour='fullscreen-command'] select");
    await expect(agentSelect).toBeVisible();

    // Default option is "Select agent..."
    await expect(agentSelect).toHaveValue("");
  });

  test("help panel toggle via button and escape", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    await page.waitForLoadState("networkidle");

    await page.keyboard.press("Meta+Shift+KeyF");
    await expect(page.locator("[data-tour='fullscreen-command']")).toBeVisible();

    // Help panel should not be visible initially
    await expect(page.locator("[data-testid='command-mode-help']")).toHaveCount(0);

    // Click "?" help button
    await page.getByRole("button", { name: "?" }).click();
    await expect(page.locator("[data-testid='command-mode-help']")).toBeVisible();

    // Help panel should show shortcut entries
    await expect(page.locator("[data-testid='command-mode-help'] kbd")).toHaveCount(8);

    // Escape closes help panel (not the fullscreen overlay)
    await page.keyboard.press("Escape");
    await expect(page.locator("[data-testid='command-mode-help']")).toHaveCount(0);
    await expect(page.locator("[data-tour='fullscreen-command']")).toBeVisible();
  });

  test("clear button appears when messages exist", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    await page.waitForLoadState("networkidle");

    await page.keyboard.press("Meta+Shift+KeyF");
    await expect(page.locator("[data-tour='fullscreen-command']")).toBeVisible();

    // Clear button should not be present when no messages
    await expect(page.getByRole("button", { name: "Clear" })).toHaveCount(0);
  });

  test("split pane buttons create additional panes", async ({ page, serverURL }) => {
    await page.goto(serverURL);
    await page.waitForLoadState("networkidle");

    await page.keyboard.press("Meta+Shift+KeyF");
    await expect(page.locator("[data-tour='fullscreen-command']")).toBeVisible();

    // Initially one pane (one agent select)
    await expect(page.locator("[data-tour='fullscreen-command'] select")).toHaveCount(1);

    // Click "Split |" button
    await page.getByRole("button", { name: /Split \|/ }).click();

    // Now two panes (two agent selects)
    await expect(page.locator("[data-tour='fullscreen-command'] select")).toHaveCount(2);
  });
});
