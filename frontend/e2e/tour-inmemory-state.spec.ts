import { test, expect } from "./fixtures";

/**
 * Tour In-Memory State E2E Tests
 *
 * Verify that the guided tour injects controlled demo data into the UI
 * so users see populated projects, boards, and tracks during the tour,
 * without requiring any real backend data.
 */

test.describe("Tour — In-Memory Demo State", () => {
  test.beforeEach(async ({ apiClient }) => {
    // Reset tour to pending state before each test
    await apiClient.put("/api/tour", { action: "reset" }).catch(() => {});
  });

  test("full tour flow shows demo data at each step", async ({ page, serverURL }) => {
    await page.goto(serverURL);

    // Tour welcome dialog should appear (pending state)
    const startBtn = page.locator("text=Start Tour");
    await expect(startBtn).toBeVisible({ timeout: 10_000 });

    // Start the tour
    await startBtn.click();

    // Step 0 (welcome): centered dialog with "Let's Go" button
    await expect(page.locator("text=Welcome, Kiloforger")).toBeVisible();
    await expect(page.locator("text=Let's Go")).toBeVisible();

    // Demo project should be visible in the overview
    await expect(page.locator('[data-tour="project-card"]')).toBeVisible({ timeout: 5_000 });
    await expect(page.locator("text=example-project")).toBeVisible();

    // Advance to step 1 (add-project)
    await page.locator("text=Let's Go").click();
    await expect(page.locator("text=Add a Project")).toBeVisible();
    await expect(page.locator('[data-tour="add-project-form"]')).toBeAttached({ timeout: 5_000 });

    // Advance to step 2 (open-project)
    await page.locator("text=Next").click();
    await expect(page.locator("text=Open Your Project")).toBeVisible();
    await expect(page.locator('[data-tour="project-card"]')).toBeVisible();

    // Advance to step 3 (setup-notice) — should navigate to project page
    await page.locator("text=Next").click();
    await expect(page.locator("text=Project Setup")).toBeVisible({ timeout: 5_000 });
    await expect(page).toHaveURL(/\/projects\/example-project/);
    await expect(page.locator('[data-tour="board-section"]')).toBeVisible({ timeout: 5_000 });

    // Advance to step 4 (swarm-capacity)
    await page.locator("text=Next").click();
    await expect(page.locator("text=Swarm Capacity")).toBeVisible();
    await expect(page.locator('[data-tour="swarm-panel"]')).toBeVisible();

    // Advance to step 5 (generate-tracks)
    await page.locator("text=Next").click();
    await expect(page.locator("text=Generate Tracks")).toBeVisible();
    await expect(page.locator('[data-tour="generate-tracks"]')).toBeVisible();

    // Advance to step 6 (board-explanation)
    await page.locator("text=Next").click();
    await expect(page.locator("text=The Kanban Board")).toBeVisible();
    await expect(page.locator('[data-tour="kanban-board"]')).toBeVisible();
    // Verify demo cards are rendered
    await expect(page.locator("text=User authentication with login flow")).toBeVisible();
    await expect(page.locator("text=Password reset via email")).toBeVisible();

    // Advance to step 7 (track-states)
    await page.locator("text=Next").click();
    await expect(page.locator("text=Track Lifecycle")).toBeVisible();

    // Advance to step 8 (move-card) — wait-for-drag with Next + Skip step
    await page.locator("text=Next").click();
    await expect(page.locator("text=Try It: Move a Card")).toBeVisible();
    await expect(page.locator('[data-tour="board-card-first"]')).toBeVisible();

    // Advance to step 9 (deps-conflicts)
    await page.locator("button:has-text('Next')").click();
    await expect(page.locator("text=Dependencies & Conflicts")).toBeVisible();

    // Advance to step 10 (agent-types)
    await page.locator("text=Next").click();
    await expect(page.locator("text=Agent Types")).toBeVisible();

    // Advance to step 11 (notification-center)
    await page.locator("text=Next").click();
    await expect(page.locator("text=Notification Center")).toBeVisible();

    // Advance to step 12 (traces)
    await page.locator("text=Next").click();
    await expect(page.locator("text=Traces & Observability")).toBeVisible();

    // Advance to step 13 (finish)
    await page.locator("text=Next").click();
    await expect(page.locator("text=You're Ready to Forge!")).toBeVisible();
    await page.locator("text=Finish").click();

    // Tour should be complete — toast notification visible
    await expect(page.locator("text=Tour complete")).toBeVisible({ timeout: 5_000 });
  });

  test("tour dismiss clears demo data and restores real state", async ({ page, serverURL }) => {
    await page.goto(serverURL);

    // Start the tour
    const startBtn = page.locator("text=Start Tour");
    await expect(startBtn).toBeVisible({ timeout: 10_000 });
    await startBtn.click();

    // Verify demo project is visible
    await expect(page.locator("text=example-project")).toBeVisible({ timeout: 5_000 });

    // Dismiss the tour
    await page.locator("text=Skip Tour").click();

    // Wait for demo data to be cleared — demo project should disappear
    // The real state should be restored (likely empty for a fresh install)
    await expect(page.locator("text=example-project")).not.toBeVisible({ timeout: 5_000 });
  });

  test("tour restart re-injects demo data correctly", async ({ page, serverURL }) => {
    await page.goto(serverURL);

    // Start and complete the tour
    const startBtn = page.locator("text=Start Tour");
    await expect(startBtn).toBeVisible({ timeout: 10_000 });
    await startBtn.click();

    // Welcome step → skip tour
    await expect(page.locator("text=Welcome, Kiloforger")).toBeVisible();
    await page.locator("text=Skip Tour").click();

    // Wait for dismissal
    await expect(page.locator("text=example-project")).not.toBeVisible({ timeout: 5_000 });

    // Restart the tour via settings menu
    await page.locator("button[title='Settings']").click();
    await page.locator("text=Take Tour").click();

    // Demo data should be re-injected
    await expect(page.locator("text=Welcome, Kiloforger")).toBeVisible({ timeout: 5_000 });
    await expect(page.locator("text=example-project")).toBeVisible({ timeout: 5_000 });
  });
});
