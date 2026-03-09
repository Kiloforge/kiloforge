import { test, expect } from "./fixtures";

// Helper for unique scope names to isolate tests.
let counter = 0;
function uid(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}`;
}

// Helper for DELETE with JSON body (apiClient.del doesn't support body).
async function deleteWithBody(
  baseURL: string,
  path: string,
  body: unknown,
): Promise<Response> {
  return fetch(`${baseURL}${path}`, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

// Cleanup helper: release a lock if it exists (best-effort).
async function releaseLock(
  baseURL: string,
  scope: string,
  holder: string,
): Promise<void> {
  await deleteWithBody(baseURL, `/api/locks/${scope}`, {
    holder,
  }).catch(() => {});
}

interface LockInfo {
  scope: string;
  holder: string;
  acquired_at: string;
  expires_at: string;
  ttl_remaining_seconds: number;
}

// ============================================================================
// Phase 1: Acquire Tests
// ============================================================================

test.describe("Lock Acquire", () => {
  test("acquire happy path", async ({ apiClient, serverURL }) => {
    const scope = uid("acquire-happy");
    const holder = "test-worker-1";

    try {
      // Acquire lock.
      const resp = await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder,
        ttl_seconds: 30,
      });
      expect(resp.status).toBe(200);

      const lock: LockInfo = await resp.json();
      expect(lock.scope).toBe(scope);
      expect(lock.holder).toBe(holder);
      expect(lock.expires_at).toBeTruthy();
      expect(lock.ttl_remaining_seconds).toBeGreaterThan(0);

      // Verify lock appears in list.
      const listResp = await apiClient.get("/api/locks");
      expect(listResp.status).toBe(200);
      const locks: LockInfo[] = await listResp.json();
      const found = locks.find((l) => l.scope === scope);
      expect(found).toBeTruthy();
      expect(found!.holder).toBe(holder);
    } finally {
      await releaseLock(serverURL, scope, holder);
    }
  });

  test("acquire with custom TTL", async ({ apiClient, serverURL }) => {
    const scope = uid("acquire-ttl");
    const holder = "test-worker-ttl";

    try {
      const resp = await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder,
        ttl_seconds: 30,
      });
      expect(resp.status).toBe(200);

      const lock: LockInfo = await resp.json();
      // TTL remaining should be close to 30s (within a few seconds tolerance).
      expect(lock.ttl_remaining_seconds).toBeGreaterThan(25);
      expect(lock.ttl_remaining_seconds).toBeLessThanOrEqual(30);

      // Verify correct expiry in list.
      const listResp = await apiClient.get("/api/locks");
      const locks: LockInfo[] = await listResp.json();
      const found = locks.find((l) => l.scope === scope);
      expect(found).toBeTruthy();
      expect(found!.ttl_remaining_seconds).toBeGreaterThan(20);
    } finally {
      await releaseLock(serverURL, scope, holder);
    }
  });

  test("acquire returns lock info", async ({ apiClient, serverURL }) => {
    const scope = uid("acquire-info");
    const holder = "test-worker-info";

    try {
      const resp = await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder,
        ttl_seconds: 60,
      });
      expect(resp.status).toBe(200);

      const lock: LockInfo = await resp.json();

      // Verify all required fields.
      expect(lock).toHaveProperty("scope");
      expect(lock).toHaveProperty("holder");
      expect(lock).toHaveProperty("acquired_at");
      expect(lock).toHaveProperty("expires_at");
      expect(lock).toHaveProperty("ttl_remaining_seconds");

      // Verify types.
      expect(typeof lock.scope).toBe("string");
      expect(typeof lock.holder).toBe("string");
      expect(typeof lock.ttl_remaining_seconds).toBe("number");

      // Verify expires_at is in the future.
      const expiresAt = new Date(lock.expires_at).getTime();
      expect(expiresAt).toBeGreaterThan(Date.now());
    } finally {
      await releaseLock(serverURL, scope, holder);
    }
  });
});

// ============================================================================
// Phase 2: Heartbeat Tests
// ============================================================================

test.describe("Lock Heartbeat", () => {
  test("heartbeat extends TTL", async ({ apiClient, serverURL }) => {
    const scope = uid("hb-extend");
    const holder = "test-worker-hb";

    try {
      // Acquire with short TTL.
      const acqResp = await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder,
        ttl_seconds: 5,
      });
      expect(acqResp.status).toBe(200);
      const original: LockInfo = await acqResp.json();
      const originalExpiry = new Date(original.expires_at).getTime();

      // Wait a bit then heartbeat with longer TTL.
      await new Promise((r) => setTimeout(r, 2000));

      const hbResp = await apiClient.post(`/api/locks/${scope}/heartbeat`, {
        holder,
        ttl_seconds: 30,
      });
      expect(hbResp.status).toBe(200);
      const updated: LockInfo = await hbResp.json();
      const newExpiry = new Date(updated.expires_at).getTime();

      // New expiry should be later than original.
      expect(newExpiry).toBeGreaterThan(originalExpiry);
    } finally {
      await releaseLock(serverURL, scope, holder);
    }
  });

  test("heartbeat response", async ({ apiClient, serverURL }) => {
    const scope = uid("hb-response");
    const holder = "test-worker-hb-resp";

    try {
      await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder,
        ttl_seconds: 30,
      });

      const hbResp = await apiClient.post(`/api/locks/${scope}/heartbeat`, {
        holder,
        ttl_seconds: 60,
      });
      expect(hbResp.status).toBe(200);

      const lock: LockInfo = await hbResp.json();
      expect(lock.holder).toBe(holder);
      expect(lock.scope).toBe(scope);
      expect(lock.expires_at).toBeTruthy();
      expect(lock.ttl_remaining_seconds).toBeGreaterThan(50);
    } finally {
      await releaseLock(serverURL, scope, holder);
    }
  });

  test("heartbeat timing — multiple heartbeats", async ({
    apiClient,
    serverURL,
  }) => {
    const scope = uid("hb-timing");
    const holder = "test-worker-hb-timing";

    try {
      await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder,
        ttl_seconds: 10,
      });

      let prevExpiry = 0;
      for (let i = 0; i < 3; i++) {
        await new Promise((r) => setTimeout(r, 500));

        const hbResp = await apiClient.post(`/api/locks/${scope}/heartbeat`, {
          holder,
          ttl_seconds: 10,
        });
        expect(hbResp.status).toBe(200);
        const lock: LockInfo = await hbResp.json();
        const expiryMs = new Date(lock.expires_at).getTime();

        if (prevExpiry > 0) {
          // Each heartbeat should extend from now, not from original.
          expect(expiryMs).toBeGreaterThan(prevExpiry);
        }
        prevExpiry = expiryMs;
      }
    } finally {
      await releaseLock(serverURL, scope, holder);
    }
  });
});

// ============================================================================
// Phase 3: Release Tests
// ============================================================================

test.describe("Lock Release", () => {
  test("release happy path", async ({ apiClient, serverURL }) => {
    const scope = uid("release-happy");
    const holder = "test-worker-release";

    // Acquire.
    const acqResp = await apiClient.post(`/api/locks/${scope}/acquire`, {
      holder,
      ttl_seconds: 60,
    });
    expect(acqResp.status).toBe(200);

    // Release.
    const relResp = await deleteWithBody(serverURL, `/api/locks/${scope}`, {
      holder,
    });
    expect(relResp.status).toBe(200);
    const body = await relResp.json();
    expect(body.released).toBe(true);

    // Verify gone from list.
    const listResp = await apiClient.get("/api/locks");
    const locks: LockInfo[] = await listResp.json();
    const found = locks.find((l) => l.scope === scope);
    expect(found).toBeUndefined();
  });

  test("release clears only target from list", async ({
    apiClient,
    serverURL,
  }) => {
    const scope1 = uid("release-multi-1");
    const scope2 = uid("release-multi-2");
    const holder = "test-worker-multi";

    try {
      // Acquire two locks.
      await apiClient.post(`/api/locks/${scope1}/acquire`, {
        holder,
        ttl_seconds: 60,
      });
      await apiClient.post(`/api/locks/${scope2}/acquire`, {
        holder,
        ttl_seconds: 60,
      });

      // Release only the first.
      const relResp = await deleteWithBody(
        serverURL,
        `/api/locks/${scope1}`,
        { holder },
      );
      expect(relResp.status).toBe(200);

      // Verify only scope1 is gone, scope2 remains.
      const listResp = await apiClient.get("/api/locks");
      const locks: LockInfo[] = await listResp.json();
      expect(locks.find((l) => l.scope === scope1)).toBeUndefined();
      expect(locks.find((l) => l.scope === scope2)).toBeTruthy();
    } finally {
      await releaseLock(serverURL, scope2, holder);
    }
  });

  test("release event — lock disappears from API", async ({
    apiClient,
    serverURL,
  }) => {
    const scope = uid("release-event");
    const holder = "test-worker-release-ev";

    // Acquire.
    await apiClient.post(`/api/locks/${scope}/acquire`, {
      holder,
      ttl_seconds: 60,
    });

    // Confirm it exists.
    let listResp = await apiClient.get("/api/locks");
    let locks: LockInfo[] = await listResp.json();
    expect(locks.find((l) => l.scope === scope)).toBeTruthy();

    // Release.
    await deleteWithBody(serverURL, `/api/locks/${scope}`, { holder });

    // Confirm gone.
    listResp = await apiClient.get("/api/locks");
    locks = await listResp.json();
    expect(locks.find((l) => l.scope === scope)).toBeUndefined();
  });
});

// ============================================================================
// Phase 4: Conflict Tests
// ============================================================================

test.describe("Lock Conflict", () => {
  test("double acquire conflict", async ({ apiClient, serverURL }) => {
    const scope = uid("conflict-double");
    const holder1 = "worker-A";
    const holder2 = "worker-B";

    try {
      // First acquire succeeds.
      const resp1 = await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder: holder1,
        ttl_seconds: 60,
      });
      expect(resp1.status).toBe(200);

      // Second acquire (different holder, non-blocking) returns 409.
      const resp2 = await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder: holder2,
        ttl_seconds: 60,
        timeout_seconds: 0,
      });
      expect(resp2.status).toBe(409);

      const conflict = await resp2.json();
      expect(conflict.error).toBeTruthy();
      expect(conflict.current_holder).toBe(holder1);
    } finally {
      await releaseLock(serverURL, scope, holder1);
    }
  });

  test("lock by different holder — heartbeat/release rejected", async ({
    apiClient,
    serverURL,
  }) => {
    const scope = uid("conflict-holder");
    const holderA = "worker-A";
    const holderB = "worker-B";

    try {
      // Acquire as holder-A.
      await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder: holderA,
        ttl_seconds: 60,
      });

      // Heartbeat as holder-B — should be rejected.
      const hbResp = await apiClient.post(`/api/locks/${scope}/heartbeat`, {
        holder: holderB,
        ttl_seconds: 60,
      });
      expect(hbResp.status).toBe(404);

      // Release as holder-B — should be rejected.
      const relResp = await deleteWithBody(
        serverURL,
        `/api/locks/${scope}`,
        { holder: holderB },
      );
      // Backend returns 404 for wrong holder (maps ErrNotHolder to 404).
      expect([403, 404]).toContain(relResp.status);

      // Verify lock still exists with holder-A.
      const listResp = await apiClient.get("/api/locks");
      const locks: LockInfo[] = await listResp.json();
      const found = locks.find((l) => l.scope === scope);
      expect(found).toBeTruthy();
      expect(found!.holder).toBe(holderA);
    } finally {
      await releaseLock(serverURL, scope, holderA);
    }
  });

  test("wait-for-release scenario — TTL expiry", async ({
    apiClient,
    serverURL,
  }) => {
    const scope = uid("conflict-wait");
    const holder1 = "worker-first";
    const holder2 = "worker-second";

    try {
      // Acquire with short TTL (2s).
      await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder: holder1,
        ttl_seconds: 2,
      });

      // Poll until second acquire succeeds (lock should expire after ~2s).
      let acquired = false;
      const deadline = Date.now() + 10000;
      while (Date.now() < deadline) {
        const resp = await apiClient.post(`/api/locks/${scope}/acquire`, {
          holder: holder2,
          ttl_seconds: 30,
          timeout_seconds: 0,
        });
        if (resp.status === 200) {
          acquired = true;
          break;
        }
        await new Promise((r) => setTimeout(r, 500));
      }

      expect(acquired).toBe(true);
    } finally {
      await releaseLock(serverURL, scope, holder2);
    }
  });
});

// ============================================================================
// Phase 5: TTL and Edge Cases
// ============================================================================

test.describe("TTL and Edge Cases", () => {
  test("TTL auto-expiry", async ({ apiClient }) => {
    const scope = uid("ttl-expiry");
    const holder = "test-worker-ttl-exp";

    // Acquire with 2s TTL — do NOT heartbeat.
    const acqResp = await apiClient.post(`/api/locks/${scope}/acquire`, {
      holder,
      ttl_seconds: 2,
    });
    expect(acqResp.status).toBe(200);

    // Poll until lock disappears.
    let expired = false;
    const deadline = Date.now() + 10000;
    while (Date.now() < deadline) {
      const listResp = await apiClient.get("/api/locks");
      const locks: LockInfo[] = await listResp.json();
      if (!locks.find((l) => l.scope === scope)) {
        expired = true;
        break;
      }
      await new Promise((r) => setTimeout(r, 500));
    }

    expect(expired).toBe(true);
  });

  test("heartbeat after release — 404", async ({ apiClient, serverURL }) => {
    const scope = uid("hb-after-release");
    const holder = "test-worker-hb-rel";

    // Acquire and release.
    await apiClient.post(`/api/locks/${scope}/acquire`, {
      holder,
      ttl_seconds: 60,
    });
    await deleteWithBody(serverURL, `/api/locks/${scope}`, { holder });

    // Heartbeat on released lock should return 404.
    const hbResp = await apiClient.post(`/api/locks/${scope}/heartbeat`, {
      holder,
      ttl_seconds: 60,
    });
    expect(hbResp.status).toBe(404);

    // Verify no side effects — lock list should not contain this scope.
    const listResp = await apiClient.get("/api/locks");
    const locks: LockInfo[] = await listResp.json();
    expect(locks.find((l) => l.scope === scope)).toBeUndefined();
  });

  test("invalid TTL — zero and negative", async ({ apiClient, serverURL }) => {
    const scope = uid("ttl-invalid");
    const holder = "test-worker-invalid-ttl";

    // ttl_seconds: 0 — server defaults to 60.
    const resp0 = await apiClient.post(`/api/locks/${scope}/acquire`, {
      holder,
      ttl_seconds: 0,
    });
    // Server falls back to default TTL of 60s, so this still succeeds.
    expect(resp0.status).toBe(200);
    await releaseLock(serverURL, scope, holder);

    // ttl_seconds: -1 — server should treat as default.
    const scope2 = uid("ttl-invalid-neg");
    const respNeg = await apiClient.post(`/api/locks/${scope2}/acquire`, {
      holder,
      ttl_seconds: -1,
    });
    // Negative TTL is treated as default (60s) by the handler.
    expect(respNeg.status).toBe(200);
    await releaseLock(serverURL, scope2, holder);
  });

  test("wrong holder release — rejected", async ({
    apiClient,
    serverURL,
  }) => {
    const scope = uid("wrong-holder");
    const holderA = "worker-owner";
    const holderB = "worker-intruder";

    try {
      // Acquire as holder-A.
      await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder: holderA,
        ttl_seconds: 60,
      });

      // Attempt release as holder-B.
      const relResp = await deleteWithBody(
        serverURL,
        `/api/locks/${scope}`,
        { holder: holderB },
      );
      // Should be rejected (404 for wrong holder in current impl).
      expect([403, 404]).toContain(relResp.status);

      // Lock should still exist with holder-A.
      const listResp = await apiClient.get("/api/locks");
      const locks: LockInfo[] = await listResp.json();
      const found = locks.find((l) => l.scope === scope);
      expect(found).toBeTruthy();
      expect(found!.holder).toBe(holderA);
    } finally {
      await releaseLock(serverURL, scope, holderA);
    }
  });
});

// ============================================================================
// Phase 6: UI Verification (API-focused since lock UI not yet implemented)
// ============================================================================

test.describe("Lock UI Verification", () => {
  test("lock list via API matches acquire state", async ({
    apiClient,
    serverURL,
  }) => {
    const scope = uid("ui-list");
    const holder = "ui-test-worker";

    try {
      // Acquire a lock.
      const acqResp = await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder,
        ttl_seconds: 60,
      });
      expect(acqResp.status).toBe(200);

      // Verify lock list shows correct info.
      const listResp = await apiClient.get("/api/locks");
      expect(listResp.status).toBe(200);
      const locks: LockInfo[] = await listResp.json();
      const found = locks.find((l) => l.scope === scope);
      expect(found).toBeTruthy();
      expect(found!.holder).toBe(holder);
      expect(found!.scope).toBe(scope);
      expect(found!.ttl_remaining_seconds).toBeGreaterThan(0);
    } finally {
      await releaseLock(serverURL, scope, holder);
    }
  });

  test("real-time lock lifecycle via API", async ({
    apiClient,
    serverURL,
  }) => {
    const scope = uid("ui-realtime");
    const holder = "ui-test-realtime";

    // Acquire.
    const acqResp = await apiClient.post(`/api/locks/${scope}/acquire`, {
      holder,
      ttl_seconds: 60,
    });
    expect(acqResp.status).toBe(200);

    // Verify exists.
    let listResp = await apiClient.get("/api/locks");
    let locks: LockInfo[] = await listResp.json();
    expect(locks.find((l) => l.scope === scope)).toBeTruthy();

    // Release.
    const relResp = await deleteWithBody(serverURL, `/api/locks/${scope}`, {
      holder,
    });
    expect(relResp.status).toBe(200);

    // Verify gone.
    listResp = await apiClient.get("/api/locks");
    locks = await listResp.json();
    expect(locks.find((l) => l.scope === scope)).toBeUndefined();
  });

  test("conflict state reflected in API", async ({
    apiClient,
    serverURL,
  }) => {
    const scope = uid("ui-conflict");
    const holder1 = "ui-worker-1";
    const holder2 = "ui-worker-2";

    try {
      // Acquire as holder1.
      await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder: holder1,
        ttl_seconds: 60,
      });

      // Attempt as holder2 — conflict.
      const conflictResp = await apiClient.post(
        `/api/locks/${scope}/acquire`,
        {
          holder: holder2,
          ttl_seconds: 60,
          timeout_seconds: 0,
        },
      );
      expect(conflictResp.status).toBe(409);

      // List should show lock held by holder1.
      const listResp = await apiClient.get("/api/locks");
      const locks: LockInfo[] = await listResp.json();
      const found = locks.find((l) => l.scope === scope);
      expect(found).toBeTruthy();
      expect(found!.holder).toBe(holder1);
    } finally {
      await releaseLock(serverURL, scope, holder1);
    }
  });

  test("dashboard loads with active locks", async ({
    page,
    serverURL,
    apiClient,
  }) => {
    const scope = uid("ui-dashboard");
    const holder = "ui-test-dashboard";

    try {
      // Acquire a lock.
      await apiClient.post(`/api/locks/${scope}/acquire`, {
        holder,
        ttl_seconds: 60,
      });

      // Navigate to dashboard — should load without errors.
      await page.goto(serverURL);
      // Dashboard should render without crash even with active locks.
      await expect(page.getByText("Projects")).toBeVisible({ timeout: 5000 });
    } finally {
      await releaseLock(serverURL, scope, holder);
    }
  });
});
