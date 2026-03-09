import { vi } from "vitest";

export class MockEventSource {
  url: string;
  readyState: number = 0; // CONNECTING

  onopen: ((event: Event) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;

  close = vi.fn(() => {
    this.readyState = 2; // CLOSED
  });

  private listeners: Record<string, ((event: MessageEvent) => void)[]> = {};

  constructor(url: string) {
    this.url = url;
    MockEventSource._instances.push(this);
  }

  addEventListener(type: string, listener: (event: MessageEvent) => void) {
    if (!this.listeners[type]) {
      this.listeners[type] = [];
    }
    this.listeners[type].push(listener);
  }

  removeEventListener(type: string, listener: (event: MessageEvent) => void) {
    if (this.listeners[type]) {
      this.listeners[type] = this.listeners[type].filter((l) => l !== listener);
    }
  }

  // Test helpers
  static _instances: MockEventSource[] = [];

  static reset() {
    MockEventSource._instances = [];
  }

  static get latest(): MockEventSource | undefined {
    return MockEventSource._instances[MockEventSource._instances.length - 1];
  }

  simulateOpen() {
    this.readyState = 1; // OPEN
    this.onopen?.(new Event("open"));
  }

  simulateEvent(type: string, data: unknown) {
    const event = new MessageEvent(type, { data: JSON.stringify(data) });
    this.listeners[type]?.forEach((l) => l(event));
  }

  simulateError() {
    this.onerror?.(new Event("error"));
  }
}
