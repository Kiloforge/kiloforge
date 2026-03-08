export class FetchError extends Error {
  constructor(
    public status: number,
    public body: unknown,
  ) {
    super(`Request failed with status ${status}`);
    this.name = "FetchError";
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
