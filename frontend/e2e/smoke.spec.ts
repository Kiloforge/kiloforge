import { test, expect } from "./fixtures";

test("smoke: playwright runs", async ({ page, serverURL }) => {
  // This test verifies the Playwright setup works.
  // It navigates to the server and checks the page loads.
  const resp = await page.goto(serverURL);
  expect(resp).not.toBeNull();
  expect(resp!.status()).toBeLessThan(500);
});

test("smoke: health endpoint via apiClient", async ({ apiClient }) => {
  const resp = await apiClient.get("/health");
  expect(resp.ok).toBe(true);
  const body = await resp.json();
  expect(body.status).toBe("ok");
});
