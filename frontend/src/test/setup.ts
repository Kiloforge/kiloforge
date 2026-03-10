import "@testing-library/jest-dom/vitest";

// Polyfill localStorage for jsdom (some Node.js versions lack full support)
if (typeof globalThis.localStorage === "undefined" || typeof globalThis.localStorage.getItem !== "function") {
  const store: Record<string, string> = {};
  Object.defineProperty(globalThis, "localStorage", {
    value: {
      getItem: (key: string) => store[key] ?? null,
      setItem: (key: string, value: string) => { store[key] = value; },
      removeItem: (key: string) => { delete store[key]; },
      clear: () => { for (const k of Object.keys(store)) delete store[k]; },
      get length() { return Object.keys(store).length; },
      key: () => null,
    } as Storage,
    writable: true,
    configurable: true,
  });
}

// Polyfill ResizeObserver for jsdom (used by RelationshipOverlay)
if (typeof globalThis.ResizeObserver === "undefined") {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof globalThis.ResizeObserver;
}
