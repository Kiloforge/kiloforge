import { test as base } from "@playwright/test";

/** Custom fixtures for E2E tests. */
export const test = base.extend<{
  /** Base URL of the test server (from E2E_BASE_URL or playwright config). */
  serverURL: string;
  /** Helper for direct REST API calls against the test server. */
  apiClient: {
    get: (path: string) => Promise<Response>;
    post: (path: string, body?: unknown) => Promise<Response>;
    put: (path: string, body?: unknown) => Promise<Response>;
    del: (path: string) => Promise<Response>;
  };
}>({
  serverURL: async ({ baseURL }, use) => {
    await use(baseURL ?? "http://localhost:39517");
  },

  apiClient: async ({ baseURL }, use) => {
    const base = baseURL ?? "http://localhost:39517";
    const client = {
      get: (path: string) => fetch(`${base}${path}`),
      post: (path: string, body?: unknown) =>
        fetch(`${base}${path}`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: body ? JSON.stringify(body) : undefined,
        }),
      put: (path: string, body?: unknown) =>
        fetch(`${base}${path}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: body ? JSON.stringify(body) : undefined,
        }),
      del: (path: string) => fetch(`${base}${path}`, { method: "DELETE" }),
    };
    await use(client);
  },
});

export { expect } from "@playwright/test";
