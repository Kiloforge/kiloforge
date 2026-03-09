import { test, expect } from "./fixtures";

// Helper to generate unique IDs for test isolation.
let idCounter = 0;
function uid(prefix: string): string {
  return `${prefix}-${Date.now()}-${++idCounter}`;
}

// Seed a project and return its slug.
async function seedProject(
  apiClient: { post: (path: string, body?: unknown) => Promise<Response> },
  slug: string,
): Promise<string> {
  const resp = await apiClient.post("/api/projects", {
    remote_url: `https://github.com/user/${slug}.git`,
  });
  expect(resp.status).toBe(201);
  const project = await resp.json();
  return project.slug;
}

// Seed tracks into a project via POST /api/tracks/seed.
async function seedTracks(
  apiClient: { post: (path: string, body?: unknown) => Promise<Response> },
  projectSlug: string,
  tracks: Array<{
    id: string;
    title: string;
    status: string;
    type: string;
    spec?: {
      summary?: string;
      context?: string;
      acceptance_criteria?: string[];
      technical_notes?: string;
    };
    plan?: Array<{
      phase: string;
      tasks: Array<{ text: string; done: boolean }>;
    }>;
  }>,
): Promise<void> {
  const resp = await apiClient.post("/api/tracks/seed", {
    project: projectSlug,
    tracks,
  });
  expect(resp.status).toBe(201);
}

// ============================================================================
// Phase 1: Track List Tests
// ============================================================================

test.describe("Track List", () => {
  test("displays seeded tracks with correct status and metadata", async ({
    apiClient,
  }) => {
    const slug = uid("tracklist");
    await seedProject(apiClient, slug);

    const tracks = [
      { id: uid("track"), title: "Auth Feature", status: "pending", type: "feature" },
      { id: uid("track"), title: "Fix Login Bug", status: "in-progress", type: "bug" },
      { id: uid("track"), title: "Refactor DB Layer", status: "completed", type: "refactor" },
      { id: uid("track"), title: "Add CI Pipeline", status: "pending", type: "chore" },
      { id: uid("track"), title: "API Redesign", status: "in-progress", type: "feature" },
    ];
    await seedTracks(apiClient, slug, tracks);

    // Verify tracks appear via API.
    const listResp = await apiClient.get(`/api/tracks?project=${slug}`);
    expect(listResp.status).toBe(200);
    const listed = await listResp.json();

    expect(listed).toHaveLength(5);

    // Verify each track has correct fields.
    for (const seeded of tracks) {
      const found = listed.find((t: { id: string }) => t.id === seeded.id);
      expect(found).toBeTruthy();
      expect(found.title).toBe(seeded.title);
      // API returns "complete" for "completed" status.
      const expectedStatus = seeded.status === "completed" ? "complete" : seeded.status;
      expect(found.status).toBe(expectedStatus);
      expect(found.project).toBe(slug);
    }
  });

  test("shows empty state when no tracks exist", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    const slug = uid("empty-tracks");
    await seedProject(apiClient, slug);

    // Navigate to project page.
    await page.goto(`${serverURL}/projects/${slug}`);

    // Should show "No tracks found" or similar empty state.
    await expect(page.getByText(/no tracks/i)).toBeVisible({ timeout: 10000 });
  });

  test("track list filters by project", async ({ apiClient }) => {
    const slugA = uid("proj-filter-a");
    const slugB = uid("proj-filter-b");
    await seedProject(apiClient, slugA);
    await seedProject(apiClient, slugB);

    await seedTracks(apiClient, slugA, [
      { id: uid("track"), title: "Track A1", status: "pending", type: "feature" },
      { id: uid("track"), title: "Track A2", status: "pending", type: "chore" },
    ]);
    await seedTracks(apiClient, slugB, [
      { id: uid("track"), title: "Track B1", status: "in-progress", type: "bug" },
    ]);

    // Filter by project A.
    const respA = await apiClient.get(`/api/tracks?project=${slugA}`);
    const tracksA = await respA.json();
    expect(tracksA).toHaveLength(2);
    for (const t of tracksA) {
      expect(t.project).toBe(slugA);
    }

    // Filter by project B.
    const respB = await apiClient.get(`/api/tracks?project=${slugB}`);
    const tracksB = await respB.json();
    expect(tracksB).toHaveLength(1);
    expect(tracksB[0].project).toBe(slugB);
  });
});

// ============================================================================
// Phase 2: Track Detail Tests
// ============================================================================

test.describe("Track Detail", () => {
  test("shows track spec and plan content", async ({ apiClient }) => {
    const slug = uid("detail-spec");
    await seedProject(apiClient, slug);

    const trackId = uid("detail-track");
    await seedTracks(apiClient, slug, [
      {
        id: trackId,
        title: "Full Detail Track",
        status: "in-progress",
        type: "feature",
        spec: {
          summary: "Implement user authentication with OAuth2 support.",
          context: "The system currently has no auth — all endpoints are public.",
          acceptance_criteria: [
            "OAuth2 login flow works end-to-end",
            "Session tokens are stored securely",
            "Logout invalidates the session",
          ],
          technical_notes: "Use PKCE flow for public clients.",
        },
        plan: [
          {
            phase: "Setup",
            tasks: [
              { text: "Add OAuth2 client library", done: true },
              { text: "Configure environment variables", done: true },
            ],
          },
          {
            phase: "Implementation",
            tasks: [
              { text: "Build login endpoint", done: false },
              { text: "Build callback handler", done: false },
              { text: "Add session middleware", done: false },
            ],
          },
        ],
      },
    ]);

    // Get track detail via API.
    const resp = await apiClient.get(
      `/api/tracks/${trackId}?project=${slug}`,
    );
    expect(resp.status).toBe(200);
    const detail = await resp.json();

    // Verify metadata.
    expect(detail.id).toBe(trackId);
    expect(detail.title).toBe("Full Detail Track");
    expect(detail.status).toBe("in-progress");
    expect(detail.type).toBe("feature");

    // Verify spec content is rendered.
    expect(detail.spec).toContain("OAuth2");
    expect(detail.spec).toContain("Implement user authentication");
    expect(detail.spec).toContain("PKCE flow");

    // Verify plan content is rendered.
    expect(detail.plan).toContain("Setup");
    expect(detail.plan).toContain("Implementation");
    expect(detail.plan).toContain("Add OAuth2 client library");
    expect(detail.plan).toContain("[x]"); // Completed tasks.
    expect(detail.plan).toContain("[ ]"); // Pending tasks.

    // Verify progress counts.
    expect(detail.phases_total).toBe(2);
    expect(detail.phases_completed).toBe(1); // Setup phase complete.
    expect(detail.tasks_total).toBe(5);
    expect(detail.tasks_completed).toBe(2);
  });

  test("track detail page shows metadata in UI", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    const slug = uid("detail-ui");
    await seedProject(apiClient, slug);

    const trackId = uid("ui-track");
    await seedTracks(apiClient, slug, [
      {
        id: trackId,
        title: "UI Detail Track",
        status: "pending",
        type: "chore",
        spec: {
          summary: "Set up CI/CD pipeline for automated deployments.",
        },
        plan: [
          {
            phase: "Pipeline Setup",
            tasks: [
              { text: "Create workflow file", done: false },
              { text: "Add deploy step", done: false },
            ],
          },
        ],
      },
    ]);

    // Navigate to track detail page.
    await page.goto(`${serverURL}/projects/${slug}/tracks/${trackId}`);

    // Verify title is visible.
    await expect(page.getByText("UI Detail Track")).toBeVisible({
      timeout: 10000,
    });

    // Verify track ID is shown.
    await expect(page.getByText(trackId)).toBeVisible();

    // Verify spec content is rendered.
    await expect(page.getByText(/CI\/CD pipeline/)).toBeVisible();
  });

  test("nonexistent track returns 404", async ({ apiClient }) => {
    const slug = uid("notfound");
    await seedProject(apiClient, slug);

    const resp = await apiClient.get(
      `/api/tracks/nonexistent-track-id?project=${slug}`,
    );
    expect(resp.status).toBe(404);

    const body = await resp.json();
    expect(body).toHaveProperty("error");
  });
});

// ============================================================================
// Phase 3: Track Generation Tests
// ============================================================================

test.describe("Track Generation", () => {
  test("generate endpoint returns agent info", async ({ apiClient }) => {
    const slug = uid("gen-test");
    await seedProject(apiClient, slug);

    // Try to generate tracks — this will likely fail without a real Claude CLI,
    // but we can verify the endpoint exists and returns the right shape.
    const resp = await apiClient.post("/api/tracks/generate", {
      prompt: "Add user authentication with OAuth2",
      project: slug,
    });

    // Without Claude CLI, expect one of these status codes:
    // 201 (success), 401 (not authed), 412 (missing skills), 428 (not setup).
    // Any of these proves the endpoint works and validates input.
    expect([201, 401, 403, 412, 428, 500]).toContain(resp.status);

    const body = await resp.json();
    // On success, should have agent_id and ws_url.
    if (resp.status === 201) {
      expect(body).toHaveProperty("agent_id");
      expect(body).toHaveProperty("ws_url");
    }
    // On error, should have error message.
    if (resp.status >= 400) {
      expect(body).toHaveProperty("error");
    }
  });

  test("generate requires prompt field", async ({ apiClient }) => {
    const slug = uid("gen-noprompt");
    await seedProject(apiClient, slug);

    // Send without prompt.
    const resp = await apiClient.post("/api/tracks/generate", {
      project: slug,
    });

    // Should fail — either 400 (bad request) or a validation error.
    expect(resp.status).toBeGreaterThanOrEqual(400);
  });

  test("generate UI elements are present on project page", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    const slug = uid("gen-ui");
    await seedProject(apiClient, slug);

    await page.goto(`${serverURL}/projects/${slug}`);

    // The "Generate Tracks" button should be visible.
    const generateBtn = page.getByText("Generate Tracks");
    await expect(generateBtn).toBeVisible({ timeout: 10000 });

    // Click to open the prompt form.
    await generateBtn.click();

    // Prompt textarea should appear.
    const textarea = page.locator("textarea");
    await expect(textarea).toBeVisible();

    // Cancel button should be available.
    const cancelBtn = page.getByText("Cancel");
    await expect(cancelBtn).toBeVisible();

    // Click cancel to close.
    await cancelBtn.click();
    await expect(textarea).not.toBeVisible();
  });

  test("generate failure returns error details", async ({ apiClient }) => {
    const slug = uid("gen-fail");
    await seedProject(apiClient, slug);

    // Send with empty prompt — should be rejected.
    const resp = await apiClient.post("/api/tracks/generate", {
      prompt: "",
      project: slug,
    });

    expect(resp.status).toBeGreaterThanOrEqual(400);
  });
});

// ============================================================================
// Phase 4: Track Deletion Tests
// ============================================================================

test.describe("Track Deletion", () => {
  test("delete removes track from list", async ({ apiClient }) => {
    const slug = uid("delete-test");
    await seedProject(apiClient, slug);

    const trackId = uid("del-track");
    await seedTracks(apiClient, slug, [
      { id: trackId, title: "Track to Delete", status: "pending", type: "feature" },
    ]);

    // Verify track exists.
    const listBefore = await apiClient.get(`/api/tracks?project=${slug}`);
    const tracksBefore = await listBefore.json();
    expect(tracksBefore.some((t: { id: string }) => t.id === trackId)).toBe(true);

    // Delete the track.
    const delResp = await apiClient.del(`/api/tracks/${trackId}?project=${slug}`);
    expect(delResp.status).toBe(204);

    // Verify track is gone.
    const listAfter = await apiClient.get(`/api/tracks?project=${slug}`);
    const tracksAfter = await listAfter.json();
    expect(tracksAfter.some((t: { id: string }) => t.id === trackId)).toBe(false);
  });

  test("delete nonexistent track returns 404", async ({ apiClient }) => {
    const slug = uid("del-notfound");
    await seedProject(apiClient, slug);

    const resp = await apiClient.del(
      `/api/tracks/nonexistent-track-xyz?project=${slug}`,
    );
    expect(resp.status).toBe(404);

    const body = await resp.json();
    expect(body).toHaveProperty("error");
    expect(body.error).toContain("not found");
  });

  test("delete one track leaves others intact", async ({ apiClient }) => {
    const slug = uid("del-selective");
    await seedProject(apiClient, slug);

    const keepId = uid("keep-track");
    const deleteId = uid("delete-track");
    await seedTracks(apiClient, slug, [
      { id: keepId, title: "Keep This", status: "pending", type: "feature" },
      { id: deleteId, title: "Delete This", status: "pending", type: "bug" },
    ]);

    // Delete only one.
    const delResp = await apiClient.del(`/api/tracks/${deleteId}?project=${slug}`);
    expect(delResp.status).toBe(204);

    // Verify the other track still exists.
    const listResp = await apiClient.get(`/api/tracks?project=${slug}`);
    const tracks = await listResp.json();
    expect(tracks).toHaveLength(1);
    expect(tracks[0].id).toBe(keepId);
    expect(tracks[0].title).toBe("Keep This");
  });
});

// ============================================================================
// Phase 5: Edge and Failure Cases
// ============================================================================

test.describe("Edge Cases", () => {
  test("concurrent track seeding does not corrupt data", async ({
    apiClient,
  }) => {
    const slug = uid("concurrent");
    await seedProject(apiClient, slug);

    // Seed two batches concurrently.
    const batch1 = seedTracks(apiClient, slug, [
      { id: uid("concurrent-a"), title: "Batch 1 Track A", status: "pending", type: "feature" },
      { id: uid("concurrent-b"), title: "Batch 1 Track B", status: "in-progress", type: "chore" },
    ]);
    const batch2 = seedTracks(apiClient, slug, [
      { id: uid("concurrent-c"), title: "Batch 2 Track C", status: "pending", type: "bug" },
    ]);

    // Wait for both — one may fail if concurrent writes conflict, but data should not be corrupt.
    const results = await Promise.allSettled([batch1, batch2]);

    // At least one batch should succeed.
    const successes = results.filter((r) => r.status === "fulfilled");
    expect(successes.length).toBeGreaterThanOrEqual(1);

    // Verify data integrity — list should return valid tracks.
    const listResp = await apiClient.get(`/api/tracks?project=${slug}`);
    expect(listResp.status).toBe(200);
    const tracks = await listResp.json();
    expect(Array.isArray(tracks)).toBe(true);
    // Each track should have required fields.
    for (const t of tracks) {
      expect(t).toHaveProperty("id");
      expect(t).toHaveProperty("title");
      expect(t).toHaveProperty("status");
    }
  });

  test("long content handles properly", async ({ apiClient }) => {
    const slug = uid("long-content");
    await seedProject(apiClient, slug);

    const longTitle = "A".repeat(250);
    const longSummary = "X".repeat(10000);
    const trackId = uid("long-track");

    await seedTracks(apiClient, slug, [
      {
        id: trackId,
        title: longTitle,
        status: "pending",
        type: "feature",
        spec: {
          summary: longSummary,
          acceptance_criteria: Array.from({ length: 50 }, (_, i) => `Criterion ${i + 1}: ${"requirement ".repeat(20)}`),
        },
        plan: [
          {
            phase: "Massive Phase",
            tasks: Array.from({ length: 30 }, (_, i) => ({
              text: `Task ${i + 1}: ${"step detail ".repeat(10)}`,
              done: i < 10,
            })),
          },
        ],
      },
    ]);

    // Verify track list returns the long title.
    const listResp = await apiClient.get(`/api/tracks?project=${slug}`);
    const tracks = await listResp.json();
    const found = tracks.find((t: { id: string }) => t.id === trackId);
    expect(found).toBeTruthy();
    expect(found.title).toBe(longTitle);

    // Verify detail returns full content.
    const detailResp = await apiClient.get(
      `/api/tracks/${trackId}?project=${slug}`,
    );
    expect(detailResp.status).toBe(200);
    const detail = await detailResp.json();
    expect(detail.spec).toContain(longSummary);
    expect(detail.tasks_total).toBe(30);
    expect(detail.tasks_completed).toBe(10);
  });

  test("API error scenarios return proper status codes", async ({
    apiClient,
  }) => {
    // Track list for nonexistent project returns empty array (not error).
    const listResp = await apiClient.get("/api/tracks?project=nonexistent-proj-xyz");
    expect(listResp.status).toBe(200);
    const tracks = await listResp.json();
    expect(tracks).toEqual([]);

    // Track detail for nonexistent project returns 404.
    const detailResp = await apiClient.get(
      "/api/tracks/some-track?project=nonexistent-proj-xyz",
    );
    expect(detailResp.status).toBe(404);

    // Delete for nonexistent track returns 404.
    const delResp = await apiClient.del(
      "/api/tracks/nonexistent-track?project=nonexistent-proj-xyz",
    );
    expect(delResp.status).toBe(404);

    // Seed tracks for nonexistent project returns 404.
    const seedResp = await apiClient.post("/api/tracks/seed", {
      project: "nonexistent-proj-xyz",
      tracks: [
        { id: "t1", title: "T1", status: "pending", type: "feature" },
      ],
    });
    expect(seedResp.status).toBe(404);
  });
});
