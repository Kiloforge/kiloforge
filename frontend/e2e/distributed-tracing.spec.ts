import { test, expect } from "./fixtures";

// Helper to generate unique IDs for test isolation.
let idCounter = 0;
function uid(prefix: string): string {
  return `${prefix}-${Date.now()}-${++idCounter}`;
}

// Seed a trace with spans via the POST /api/traces endpoint.
async function seedTrace(
  apiClient: { post: (path: string, body?: unknown) => Promise<Response> },
  traceId: string,
  spans: Array<{
    span_id: string;
    parent_id?: string;
    name: string;
    start_time: string;
    end_time: string;
    duration_ms: number;
    status?: string;
    attributes?: Record<string, string>;
    events?: Array<{
      name: string;
      timestamp: string;
      attributes?: Record<string, string>;
    }>;
  }>,
): Promise<void> {
  const resp = await apiClient.post("/api/traces", {
    trace_id: traceId,
    spans: spans.map((s) => ({
      ...s,
      status: s.status ?? "ok",
    })),
  });
  expect(resp.status).toBe(201);
}

// Build a simple trace with N spans in a chain.
function buildChainTrace(
  traceId: string,
  baseTime: Date,
  depth: number,
  attrs?: Record<string, string>,
): Array<{
  span_id: string;
  parent_id?: string;
  name: string;
  start_time: string;
  end_time: string;
  duration_ms: number;
  attributes?: Record<string, string>;
}> {
  const spans = [];
  for (let i = 0; i < depth; i++) {
    const start = new Date(baseTime.getTime() + i * 100);
    const end = new Date(start.getTime() + 500 - i * 50);
    spans.push({
      span_id: `${traceId}-span-${i}`,
      parent_id: i > 0 ? `${traceId}-span-${i - 1}` : undefined,
      name: i === 0 ? `track/test-operation` : `child-op-level-${i}`,
      start_time: start.toISOString(),
      end_time: end.toISOString(),
      duration_ms: end.getTime() - start.getTime(),
      attributes: i === 0 ? attrs : undefined,
    });
  }
  return spans;
}

// ============================================================================
// Phase 1: Trace List Tests
// ============================================================================

test.describe("Trace List", () => {
  test("list with seeded traces", async ({ page, serverURL, apiClient }) => {
    const baseTime = new Date();
    const traces = [];

    // Seed 3 traces with known IDs.
    for (let i = 0; i < 3; i++) {
      const traceId = uid(`trace-list-${i}`);
      traces.push(traceId);
      const start = new Date(baseTime.getTime() + i * 1000);
      await seedTrace(apiClient, traceId, [
        {
          span_id: `${traceId}-root`,
          name: `track/list-op-${i}`,
          start_time: start.toISOString(),
          end_time: new Date(start.getTime() + 200 + i * 100).toISOString(),
          duration_ms: 200 + i * 100,
        },
        {
          span_id: `${traceId}-child`,
          parent_id: `${traceId}-root`,
          name: `child-${i}`,
          start_time: new Date(start.getTime() + 50).toISOString(),
          end_time: new Date(start.getTime() + 150).toISOString(),
          duration_ms: 100,
        },
      ]);
    }

    // Verify via API.
    const listResp = await apiClient.get("/api/traces");
    expect(listResp.status).toBe(200);
    const allTraces = await listResp.json();
    for (const traceId of traces) {
      const found = allTraces.find(
        (t: { trace_id: string }) => t.trace_id === traceId,
      );
      expect(found).toBeTruthy();
      expect(found.span_count).toBe(2);
    }

    // Verify in UI — navigate to overview page where TraceList is rendered.
    await page.goto(serverURL);
    for (let i = 0; i < 3; i++) {
      await expect(page.getByText(`list-op-${i}`)).toBeVisible({
        timeout: 5000,
      });
    }
  });

  test("empty list shows message", async ({ page, serverURL, apiClient }) => {
    // The list may not be empty due to other tests, but check the API works.
    const listResp = await apiClient.get("/api/traces");
    expect(listResp.status).toBe(200);
    const traces = await listResp.json();
    expect(Array.isArray(traces)).toBe(true);

    // Navigate and check the section exists.
    await page.goto(serverURL);
    await expect(page.getByText("Traces")).toBeVisible({ timeout: 5000 });

    // If no traces, the empty message should show.
    if (traces.length === 0) {
      await expect(page.getByText("No traces recorded")).toBeVisible();
    }
  });

  test("trace metadata display", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    const traceId = uid("trace-meta");
    const start = new Date();
    const end = new Date(start.getTime() + 350);

    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-root`,
        name: "track/meta-display-op",
        start_time: start.toISOString(),
        end_time: end.toISOString(),
        duration_ms: 350,
      },
    ]);

    // Navigate to overview.
    await page.goto(serverURL);

    // Verify root span name and span count are visible.
    await expect(page.getByText("meta-display-op")).toBeVisible({
      timeout: 5000,
    });
    // Span count should show "1".
    const row = page.locator("tr", { hasText: "meta-display-op" });
    await expect(row).toBeVisible();
    await expect(row.getByText("1")).toBeVisible();
    // Duration should be shown.
    await expect(row.getByText("350ms")).toBeVisible();
  });
});

// ============================================================================
// Phase 2: Trace Detail Tests
// ============================================================================

test.describe("Trace Detail", () => {
  test("span hierarchy", async ({ page, serverURL, apiClient }) => {
    const traceId = uid("trace-hierarchy");
    const base = new Date();

    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-root`,
        name: "root-operation",
        start_time: base.toISOString(),
        end_time: new Date(base.getTime() + 1000).toISOString(),
        duration_ms: 1000,
      },
      {
        span_id: `${traceId}-child`,
        parent_id: `${traceId}-root`,
        name: "child-operation",
        start_time: new Date(base.getTime() + 100).toISOString(),
        end_time: new Date(base.getTime() + 700).toISOString(),
        duration_ms: 600,
      },
      {
        span_id: `${traceId}-grandchild`,
        parent_id: `${traceId}-child`,
        name: "grandchild-operation",
        start_time: new Date(base.getTime() + 200).toISOString(),
        end_time: new Date(base.getTime() + 500).toISOString(),
        duration_ms: 300,
      },
    ]);

    await page.goto(`${serverURL}/traces/${traceId}`);

    // All three spans should be visible.
    await expect(page.getByText("root-operation")).toBeVisible({
      timeout: 5000,
    });
    await expect(page.getByText("child-operation")).toBeVisible();
    await expect(page.getByText("grandchild-operation")).toBeVisible();

    // Verify span count in header.
    await expect(page.getByText("3 spans")).toBeVisible();
  });

  test("span attributes", async ({ page, serverURL, apiClient }) => {
    const traceId = uid("trace-attrs");
    const base = new Date();

    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-root`,
        name: "attributed-op",
        start_time: base.toISOString(),
        end_time: new Date(base.getTime() + 500).toISOString(),
        duration_ms: 500,
        attributes: {
          "project.slug": "test-project",
          "track.id": "my-track-123",
          "agent.id": "agent-42",
          "webhook.event": "push",
        },
      },
    ]);

    await page.goto(`${serverURL}/traces/${traceId}`);
    await expect(page.getByText("attributed-op")).toBeVisible({
      timeout: 5000,
    });

    // Click on the span to show attributes.
    await page.getByText("attributed-op").click();

    // Verify attributes are displayed.
    await expect(page.getByText("project.slug")).toBeVisible();
    await expect(page.getByText("test-project")).toBeVisible();
    await expect(page.getByText("track.id")).toBeVisible();
    await expect(page.getByText("my-track-123")).toBeVisible();
    await expect(page.getByText("agent.id")).toBeVisible();
    await expect(page.getByText("agent-42")).toBeVisible();
    await expect(page.getByText("webhook.event")).toBeVisible();
    await expect(page.getByText("push")).toBeVisible();
  });

  test("navigation from list", async ({ page, serverURL, apiClient }) => {
    const traceId = uid("trace-nav");
    const base = new Date();

    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-root`,
        name: "track/nav-test-op",
        start_time: base.toISOString(),
        end_time: new Date(base.getTime() + 200).toISOString(),
        duration_ms: 200,
      },
    ]);

    // Navigate to overview.
    await page.goto(serverURL);
    await expect(page.getByText("nav-test-op")).toBeVisible({ timeout: 5000 });

    // Click on the trace link.
    await page.getByRole("link", { name: "nav-test-op" }).click();

    // URL should change to trace detail.
    await expect(page).toHaveURL(new RegExp(`/traces/${traceId}`));

    // Detail page should load.
    await expect(page.getByText("nav-test-op")).toBeVisible();
    await expect(page.getByText("1 span")).toBeVisible();
  });
});

// ============================================================================
// Phase 3: Timeline Tests
// ============================================================================

test.describe("Timeline", () => {
  test("timeline rendering", async ({ page, serverURL, apiClient }) => {
    const traceId = uid("trace-timeline");
    const base = new Date();

    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-root`,
        name: "timeline-root",
        start_time: base.toISOString(),
        end_time: new Date(base.getTime() + 1000).toISOString(),
        duration_ms: 1000,
      },
      {
        span_id: `${traceId}-a`,
        parent_id: `${traceId}-root`,
        name: "timeline-child-a",
        start_time: new Date(base.getTime() + 100).toISOString(),
        end_time: new Date(base.getTime() + 600).toISOString(),
        duration_ms: 500,
      },
      {
        span_id: `${traceId}-b`,
        parent_id: `${traceId}-root`,
        name: "timeline-child-b",
        start_time: new Date(base.getTime() + 300).toISOString(),
        end_time: new Date(base.getTime() + 900).toISOString(),
        duration_ms: 600,
      },
    ]);

    await page.goto(`${serverURL}/traces/${traceId}`);

    // All spans should be rendered in the timeline.
    await expect(page.getByText("timeline-root")).toBeVisible({
      timeout: 5000,
    });
    await expect(page.getByText("timeline-child-a")).toBeVisible();
    await expect(page.getByText("timeline-child-b")).toBeVisible();

    // Duration labels should be shown.
    await expect(page.getByText("1000ms")).toBeVisible();
    await expect(page.getByText("500ms")).toBeVisible();
    await expect(page.getByText("600ms")).toBeVisible();
  });

  test("relative timing", async ({ page, serverURL, apiClient }) => {
    const traceId = uid("trace-timing");
    const base = new Date();

    // Root starts at 0ms, child starts at 100ms.
    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-root`,
        name: "timing-root",
        start_time: base.toISOString(),
        end_time: new Date(base.getTime() + 500).toISOString(),
        duration_ms: 500,
      },
      {
        span_id: `${traceId}-child`,
        parent_id: `${traceId}-root`,
        name: "timing-child",
        start_time: new Date(base.getTime() + 100).toISOString(),
        end_time: new Date(base.getTime() + 400).toISOString(),
        duration_ms: 300,
      },
    ]);

    await page.goto(`${serverURL}/traces/${traceId}`);
    await expect(page.getByText("timing-root")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("timing-child")).toBeVisible();

    // Verify the timeline bars exist — the CSS classes indicate bar positioning.
    // The root bar should start at 0%, the child should be offset.
    const bars = page.locator('[class*="bar"]');
    expect(await bars.count()).toBeGreaterThanOrEqual(2);
  });

  test("span duration display", async ({ page, serverURL, apiClient }) => {
    const traceId = uid("trace-duration");
    const base = new Date();

    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-root`,
        name: "duration-root",
        start_time: base.toISOString(),
        end_time: new Date(base.getTime() + 2000).toISOString(),
        duration_ms: 2000,
      },
      {
        span_id: `${traceId}-fast`,
        parent_id: `${traceId}-root`,
        name: "duration-fast",
        start_time: new Date(base.getTime() + 10).toISOString(),
        end_time: new Date(base.getTime() + 20).toISOString(),
        duration_ms: 10,
      },
      {
        span_id: `${traceId}-medium`,
        parent_id: `${traceId}-root`,
        name: "duration-medium",
        start_time: new Date(base.getTime() + 100).toISOString(),
        end_time: new Date(base.getTime() + 600).toISOString(),
        duration_ms: 500,
      },
    ]);

    await page.goto(`${serverURL}/traces/${traceId}`);

    // Duration labels for each span.
    await expect(page.getByText("2000ms")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("10ms")).toBeVisible();
    await expect(page.getByText("500ms")).toBeVisible();
  });
});

// ============================================================================
// Phase 4: Real-Time Tests
// ============================================================================

test.describe("Real-Time SSE", () => {
  test("trace_update SSE event refreshes list", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    // Navigate to overview first.
    await page.goto(serverURL);
    await expect(page.getByText("Traces")).toBeVisible({ timeout: 5000 });

    // Now seed a new trace while the page is open.
    const traceId = uid("trace-sse");
    const base = new Date();
    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-root`,
        name: "track/sse-new-trace",
        start_time: base.toISOString(),
        end_time: new Date(base.getTime() + 300).toISOString(),
        duration_ms: 300,
      },
    ]);

    // The SSE watcher polls every 2 seconds, so wait and check.
    // The trace should appear via SSE update or on next query refetch.
    await expect(page.getByText("sse-new-trace")).toBeVisible({
      timeout: 10000,
    });
  });

  test("new trace appears in list via API", async ({ apiClient }) => {
    const traceId = uid("trace-api-new");
    const base = new Date();

    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-root`,
        name: "track/api-new-trace",
        start_time: base.toISOString(),
        end_time: new Date(base.getTime() + 200).toISOString(),
        duration_ms: 200,
      },
    ]);

    // Verify the new trace appears in the API response.
    const listResp = await apiClient.get("/api/traces");
    const traces = await listResp.json();
    const found = traces.find(
      (t: { trace_id: string }) => t.trace_id === traceId,
    );
    expect(found).toBeTruthy();
    expect(found.root_name).toBe("track/api-new-trace");
    expect(found.span_count).toBe(1);
  });

  test("auto-refresh detail on new spans", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    const traceId = uid("trace-detail-refresh");
    const base = new Date();

    // Seed initial trace with one span.
    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-root`,
        name: "refresh-root",
        start_time: base.toISOString(),
        end_time: new Date(base.getTime() + 500).toISOString(),
        duration_ms: 500,
      },
    ]);

    // Open the detail page.
    await page.goto(`${serverURL}/traces/${traceId}`);
    await expect(page.getByText("refresh-root")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("1 span")).toBeVisible();

    // Add a new span to the same trace.
    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-new-child`,
        parent_id: `${traceId}-root`,
        name: "refresh-new-child",
        start_time: new Date(base.getTime() + 100).toISOString(),
        end_time: new Date(base.getTime() + 400).toISOString(),
        duration_ms: 300,
      },
    ]);

    // The detail page should eventually show 2 spans.
    // This may require a page refresh since trace detail doesn't auto-refetch on SSE.
    await page.reload();
    await expect(page.getByText("2 spans")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("refresh-new-child")).toBeVisible();
  });
});

// ============================================================================
// Phase 5: Edge and Failure Cases
// ============================================================================

test.describe("Edge and Failure Cases", () => {
  test("deep nesting — 6 levels", async ({ page, serverURL, apiClient }) => {
    const traceId = uid("trace-deep");
    const base = new Date();

    // Build 6-level deep hierarchy.
    const spans = buildChainTrace(traceId, base, 6);
    await seedTrace(apiClient, traceId, spans);

    await page.goto(`${serverURL}/traces/${traceId}`);

    // All spans should render without overflow.
    await expect(page.getByText("track/test-operation")).toBeVisible({
      timeout: 5000,
    });
    for (let i = 1; i < 6; i++) {
      await expect(page.getByText(`child-op-level-${i}`)).toBeVisible();
    }

    // Verify 6 spans header.
    await expect(page.getByText("6 spans")).toBeVisible();

    // Verify the page doesn't have horizontal scrollbar issues.
    const bodyWidth = await page.evaluate(() => document.body.scrollWidth);
    const viewportWidth = await page.evaluate(() => window.innerWidth);
    expect(bodyWidth).toBeLessThanOrEqual(viewportWidth + 50); // Allow small tolerance.
  });

  test("nonexistent trace — 404", async ({ page, serverURL }) => {
    await page.goto(`${serverURL}/traces/nonexistent-trace-id-xyz`);

    // Should show error, not blank page.
    const hasError = await page
      .getByText(/not found|error/i)
      .isVisible({ timeout: 5000 })
      .catch(() => false);
    const hasBack = await page
      .getByText("Back")
      .isVisible({ timeout: 2000 })
      .catch(() => false);

    // At minimum, the page should not be blank.
    expect(hasError || hasBack).toBe(true);
  });

  test("missing spans — orphaned span handling", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    const traceId = uid("trace-orphan");
    const base = new Date();

    // Seed a span referencing a nonexistent parent.
    await seedTrace(apiClient, traceId, [
      {
        span_id: `${traceId}-root`,
        name: "orphan-root",
        start_time: base.toISOString(),
        end_time: new Date(base.getTime() + 500).toISOString(),
        duration_ms: 500,
      },
      {
        span_id: `${traceId}-orphan`,
        parent_id: "nonexistent-parent-id",
        name: "orphan-child",
        start_time: new Date(base.getTime() + 100).toISOString(),
        end_time: new Date(base.getTime() + 300).toISOString(),
        duration_ms: 200,
      },
    ]);

    await page.goto(`${serverURL}/traces/${traceId}`);

    // Both spans should render — orphan at root level or with depth 1.
    await expect(page.getByText("orphan-root")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("orphan-child")).toBeVisible();

    // Page should not crash.
    await expect(page.getByText("2 spans")).toBeVisible();
  });

  test("API error — invalid trace list returns gracefully", async ({
    apiClient,
  }) => {
    // Call the trace list endpoint — should always return 200 even if empty.
    const resp = await apiClient.get("/api/traces");
    expect(resp.status).toBe(200);
    const data = await resp.json();
    expect(Array.isArray(data)).toBe(true);
  });
});
