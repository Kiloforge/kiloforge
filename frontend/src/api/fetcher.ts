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

export async function fetcher<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: `Error ${res.status}` }));
    throw new FetchError(res.status, body);
  }
  return res.json() as Promise<T>;
}
