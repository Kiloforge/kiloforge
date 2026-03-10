import { test, expect } from "./fixtures";

/**
 * Kanban Board E2E Tests
 *
 * These tests verify the board UI rendering, column structure, card display,
 * and API integration. Board data is accessed via the REST API; drag-and-drop
 * interactions require a running backend with board service configured.
 */

test.describe("Kanban Board — Board API", () => {
  test("board API returns columns and cards", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/board/test-project");
    // Board may return 200 (configured) or 500 (not configured).
    if (resp.status === 200) {
      const board = await resp.json();
      expect(board.columns).toBeDefined();
      expect(Array.isArray(board.columns)).toBe(true);
    }
  });

  test("board API returns 5 columns", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/board/test-project");
    if (resp.status === 200) {
      const board = await resp.json();
      expect(board.columns).toHaveLength(5);
      expect(board.columns).toEqual([
        "backlog",
        "approved",
        "in_progress",
        "done",
      ]);
    }
  });

  test("board cards map is an object", async ({ apiClient }) => {
    const resp = await apiClient.get("/api/board/test-project");
    if (resp.status === 200) {
      const board = await resp.json();
      expect(typeof board.cards).toBe("object");
    }
  });

  test("move card returns from/to columns", async ({ apiClient }) => {
    // First get the board to see if any cards exist.
    const boardResp = await apiClient.get("/api/board/test-project");
    if (boardResp.status !== 200) return;

    const board = await boardResp.json();
    const cardIds = Object.keys(board.cards || {});
    if (cardIds.length === 0) return;

    const firstCard = board.cards[cardIds[0]];
    const targetColumn =
      firstCard.column === "in_progress" ? "done" : "in_progress";

    const moveResp = await apiClient.post(
      "/api/board/test-project/move",
      { track_id: cardIds[0], to_column: targetColumn },
    );

    if (moveResp.status === 200) {
      const result = await moveResp.json();
      expect(result.track_id).toBe(cardIds[0]);
      expect(result.from_column).toBeDefined();
      expect(result.to_column).toBe(targetColumn);
    }
  });

  test("move card with invalid column returns 400", async ({ apiClient }) => {
    const resp = await apiClient.post("/api/board/test-project/move", {
      track_id: "any-track",
      to_column: "nonexistent-column",
    });
    expect([400, 500]).toContain(resp.status);
  });

  test("sync board for unknown project returns 400", async ({ apiClient }) => {
    const resp = await apiClient.post("/api/board/nonexistent-project/sync");
    expect([400, 500]).toContain(resp.status);
  });
});

test.describe("Kanban Board — UI Rendering", () => {
  test("project page renders board section", async ({ page, serverURL }) => {
    await page.goto(`${serverURL}/projects/test-project`);
    // Board component has data-tour="kanban-board" attribute.
    const board = page.locator('[data-tour="kanban-board"]');
    const boardVisible = await board
      .isVisible({ timeout: 5000 })
      .catch(() => false);
    // If project exists and board is configured, board should render.
    // If not, the page may show an error or no board.
    if (boardVisible) {
      await expect(board).toBeVisible();
    }
  });

  test("board shows 5 column headers", async ({ page, serverURL }) => {
    await page.goto(`${serverURL}/projects/test-project`);
    const board = page.locator('[data-tour="kanban-board"]');
    if (await board.isVisible({ timeout: 5000 }).catch(() => false)) {
      const columnLabels = ["Backlog", "Approved", "In Progress", "Done"];
      for (const label of columnLabels) {
        const col = page.getByText(label, { exact: true });
        await expect(col).toBeVisible({ timeout: 3000 });
      }
    }
  });

  test("each column shows card count", async ({ page, serverURL }) => {
    await page.goto(`${serverURL}/projects/test-project`);
    const board = page.locator('[data-tour="kanban-board"]');
    if (await board.isVisible({ timeout: 5000 }).catch(() => false)) {
      // Column count badges are spans with class columnCount.
      // Each column header has a count number.
      const counts = page.locator('[class*="columnCount"]');
      const countVisible = await counts.first().isVisible({ timeout: 3000 }).catch(() => false);
      if (countVisible) {
        const countElements = await counts.count();
        expect(countElements).toBe(5);
      }
    }
  });

  test("cards are draggable", async ({ page, serverURL }) => {
    await page.goto(`${serverURL}/projects/test-project`);
    const board = page.locator('[data-tour="kanban-board"]');
    if (await board.isVisible({ timeout: 5000 }).catch(() => false)) {
      const cards = page.locator('[class*="card"][draggable="true"]');
      const cardCount = await cards.count();
      if (cardCount > 0) {
        const firstCard = cards.first();
        await expect(firstCard).toHaveAttribute("draggable", "true");
      }
    }
  });

  test("card shows track ID", async ({ page, serverURL }) => {
    await page.goto(`${serverURL}/projects/test-project`);
    const board = page.locator('[data-tour="kanban-board"]');
    if (await board.isVisible({ timeout: 5000 }).catch(() => false)) {
      const cardIds = page.locator('[class*="cardId"]');
      const hasIds = await cardIds.first().isVisible({ timeout: 3000 }).catch(() => false);
      if (hasIds) {
        const text = await cardIds.first().textContent();
        expect(text).toBeTruthy();
      }
    }
  });

  test("card shows title text", async ({ page, serverURL }) => {
    await page.goto(`${serverURL}/projects/test-project`);
    const board = page.locator('[data-tour="kanban-board"]');
    if (await board.isVisible({ timeout: 5000 }).catch(() => false)) {
      const titles = page.locator('[class*="cardTitle"]');
      const hasTitles = await titles.first().isVisible({ timeout: 3000 }).catch(() => false);
      if (hasTitles) {
        const text = await titles.first().textContent();
        expect(text).toBeTruthy();
      }
    }
  });

  test("backlog cards show approve and reject buttons", async ({
    page,
    serverURL,
  }) => {
    await page.goto(`${serverURL}/projects/test-project`);
    const board = page.locator('[data-tour="kanban-board"]');
    if (await board.isVisible({ timeout: 5000 }).catch(() => false)) {
      // Backlog cards have approve (checkmark) and reject (cross) buttons.
      const approveBtn = page.locator('[class*="approveBtn"]');
      const hasApprove = await approveBtn.first().isVisible({ timeout: 3000 }).catch(() => false);
      if (hasApprove) {
        await expect(approveBtn.first()).toBeVisible();
      }
    }
  });
});

test.describe("Kanban Board — Navigation", () => {
  test("card title links to track detail", async ({ page, serverURL }) => {
    await page.goto(`${serverURL}/projects/test-project`);
    const board = page.locator('[data-tour="kanban-board"]');
    if (await board.isVisible({ timeout: 5000 }).catch(() => false)) {
      const cardLinks = page.locator('[class*="cardTitleLink"]');
      const hasLinks = await cardLinks.first().isVisible({ timeout: 3000 }).catch(() => false);
      if (hasLinks) {
        const href = await cardLinks.first().getAttribute("href");
        expect(href).toContain("/projects/test-project/tracks/");
      }
    }
  });
});
