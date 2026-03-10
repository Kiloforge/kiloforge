export class FetchError extends Error {
  status: number;
  body: unknown;

  constructor(status: number, body: unknown) {
    super(`Request failed with status ${status}`);
    this.name = "FetchError";
    this.status = status;
    this.body = body;
  }
}

const DEFAULT_TIMEOUT_MS = 30_000;

export async function fetcher<T>(
  url: string,
  init?: RequestInit & { timeoutMs?: number },
): Promise<T> {
  const { timeoutMs = DEFAULT_TIMEOUT_MS, ...fetchInit } = init ?? {};

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  // Respect caller's signal if provided
  if (fetchInit.signal) {
    fetchInit.signal.addEventListener("abort", () => controller.abort());
  }

  try {
    const res = await fetch(url, { ...fetchInit, signal: controller.signal });
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: `Error ${res.status}` }));
      throw new FetchError(res.status, body);
    }
    return res.json() as Promise<T>;
  } finally {
    clearTimeout(timeoutId);
  }
}
