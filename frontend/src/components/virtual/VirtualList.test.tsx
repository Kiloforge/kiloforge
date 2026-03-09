import { describe, it, expect, vi, beforeAll, afterAll } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { VirtualList } from "./VirtualList";
import type { VirtualListRef } from "./VirtualList";
import { createRef } from "react";

// Mock ResizeObserver that fires immediately with mocked dimensions
class MockResizeObserver {
  callback: ResizeObserverCallback;
  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
  }
  observe = vi.fn().mockImplementation((target: HTMLElement) => {
    // Fire callback immediately with the target's dimensions
    const height = target.hasAttribute("data-virtual-list") ? 200 : 40;
    const entry = {
      target,
      contentRect: { width: 300, height, top: 0, left: 0, bottom: height, right: 300, x: 0, y: 0, toJSON: () => {} },
      borderBoxSize: [{ blockSize: height, inlineSize: 300 }],
      contentBoxSize: [{ blockSize: height, inlineSize: 300 }],
      devicePixelContentBoxSize: [{ blockSize: height, inlineSize: 300 }],
    } as unknown as ResizeObserverEntry;
    // Schedule callback so it runs after React commit phase
    Promise.resolve().then(() => this.callback([entry], this as unknown as ResizeObserver));
  });
  unobserve = vi.fn();
  disconnect = vi.fn();
}
globalThis.ResizeObserver = MockResizeObserver as unknown as typeof ResizeObserver;

// jsdom has no layout engine. Patch HTMLElement so the virtualizer
// sees real dimensions for our virtual list container and items.
const origGetBCR = HTMLElement.prototype.getBoundingClientRect;
const origClientHeight = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "clientHeight");
const origScrollHeight = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollHeight");
const origOffsetHeight = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "offsetHeight");

beforeAll(() => {
  Object.defineProperty(HTMLElement.prototype, "clientHeight", {
    configurable: true,
    get() {
      if (this.hasAttribute?.("data-virtual-list")) return 200;
      return origClientHeight?.get?.call(this) ?? 0;
    },
  });
  Object.defineProperty(HTMLElement.prototype, "scrollHeight", {
    configurable: true,
    get() {
      if (this.hasAttribute?.("data-virtual-list")) return 10000;
      return origScrollHeight?.get?.call(this) ?? 0;
    },
  });
  Object.defineProperty(HTMLElement.prototype, "offsetHeight", {
    configurable: true,
    get() {
      if (this.getAttribute?.("data-index") !== null) return 40;
      if (this.hasAttribute?.("data-virtual-list")) return 200;
      return origOffsetHeight?.get?.call(this) ?? 0;
    },
  });
  HTMLElement.prototype.getBoundingClientRect = function () {
    if (this.hasAttribute?.("data-virtual-list")) {
      return { top: 0, left: 0, bottom: 200, right: 300, width: 300, height: 200, x: 0, y: 0, toJSON: () => {} };
    }
    if (this.getAttribute?.("data-index") !== null) {
      return { top: 0, left: 0, bottom: 40, right: 300, width: 300, height: 40, x: 0, y: 0, toJSON: () => {} };
    }
    return origGetBCR.call(this);
  };
});

afterAll(() => {
  HTMLElement.prototype.getBoundingClientRect = origGetBCR;
  if (origClientHeight) Object.defineProperty(HTMLElement.prototype, "clientHeight", origClientHeight);
  if (origScrollHeight) Object.defineProperty(HTMLElement.prototype, "scrollHeight", origScrollHeight);
  if (origOffsetHeight) Object.defineProperty(HTMLElement.prototype, "offsetHeight", origOffsetHeight);
});

function makeItems(count: number): string[] {
  return Array.from({ length: count }, (_, i) => `Item ${i}`);
}

// Helper: render and flush microtasks so ResizeObserver callbacks fire
async function renderAndFlush(jsx: React.ReactElement) {
  let result: ReturnType<typeof render>;
  await act(async () => {
    result = render(jsx);
  });
  return result!;
}

describe("VirtualList", () => {
  it("renders only visible items, not all items", async () => {
    const items = makeItems(1000);
    const renderItem = (item: string) => <div>{item}</div>;

    await renderAndFlush(
      <VirtualList
        items={items}
        estimateSize={() => 40}
        renderItem={renderItem}
        overscan={2}
        style={{ height: 200 }}
      />,
    );

    const rendered = screen.queryAllByText(/^Item \d+$/);
    expect(rendered.length).toBeGreaterThan(0);
    expect(rendered.length).toBeLessThan(50);
  });

  it("supports custom renderItem function", async () => {
    const items = ["Alpha", "Beta"];
    const renderItem = (item: string) => <div data-testid={`custom-${item}`}>{item}!</div>;

    await renderAndFlush(
      <VirtualList
        items={items}
        estimateSize={() => 40}
        renderItem={renderItem}
        style={{ height: 200 }}
      />,
    );

    expect(screen.getByTestId("custom-Alpha")).toBeInTheDocument();
    expect(screen.getByText("Alpha!")).toBeInTheDocument();
  });

  it("applies className to container", async () => {
    await renderAndFlush(
      <VirtualList
        items={["a"]}
        estimateSize={() => 40}
        renderItem={(item) => <div>{item}</div>}
        className="my-custom-class"
        style={{ height: 200 }}
      />,
    );

    expect(document.querySelector(".my-custom-class")).toBeTruthy();
  });

  it("handles empty items array", async () => {
    await renderAndFlush(
      <VirtualList
        items={[]}
        estimateSize={() => 40}
        renderItem={(item: string) => <div>{item}</div>}
        style={{ height: 200 }}
      />,
    );

    expect(document.querySelector("[data-virtual-list]")).toBeTruthy();
    expect(screen.queryByText(/Item/)).not.toBeInTheDocument();
  });

  it("exposes scrollToIndex via ref", async () => {
    const ref = createRef<VirtualListRef>();
    const items = makeItems(100);

    await renderAndFlush(
      <VirtualList
        ref={ref}
        items={items}
        estimateSize={() => 40}
        renderItem={(item) => <div>{item}</div>}
        style={{ height: 200 }}
      />,
    );

    expect(ref.current).toBeTruthy();
    expect(typeof ref.current!.scrollToIndex).toBe("function");
    act(() => { ref.current!.scrollToIndex(50); });
  });

  it("exposes scrollToBottom via ref", async () => {
    const ref = createRef<VirtualListRef>();
    const items = makeItems(100);

    await renderAndFlush(
      <VirtualList
        ref={ref}
        items={items}
        estimateSize={() => 40}
        renderItem={(item) => <div>{item}</div>}
        style={{ height: 200 }}
      />,
    );

    expect(ref.current).toBeTruthy();
    expect(typeof ref.current!.scrollToBottom).toBe("function");
    act(() => { ref.current!.scrollToBottom(); });
  });

  it("renders items with correct indices", async () => {
    const items = ["Zero", "One", "Two"];
    const renderItem = (item: string, index: number) => (
      <div data-testid={`item-${index}`}>{item}</div>
    );

    await renderAndFlush(
      <VirtualList
        items={items}
        estimateSize={() => 40}
        renderItem={renderItem}
        style={{ height: 200 }}
      />,
    );

    expect(screen.getByTestId("item-0")).toHaveTextContent("Zero");
    expect(screen.getByTestId("item-1")).toHaveTextContent("One");
    expect(screen.getByTestId("item-2")).toHaveTextContent("Two");
  });

  it("uses default overscan of 5", async () => {
    const items = makeItems(100);
    const renderItem = (item: string) => <div>{item}</div>;

    await renderAndFlush(
      <VirtualList
        items={items}
        estimateSize={() => 40}
        renderItem={renderItem}
        style={{ height: 200 }}
      />,
    );

    const rendered = screen.queryAllByText(/^Item \d+$/);
    expect(rendered.length).toBeGreaterThan(0);
    expect(rendered.length).toBeLessThan(20);
  });

  it("does not show scroll-to-bottom button when autoFollow is off", async () => {
    await renderAndFlush(
      <VirtualList
        items={makeItems(100)}
        estimateSize={() => 40}
        renderItem={(item) => <div>{item}</div>}
        style={{ height: 200 }}
      />,
    );

    expect(screen.queryByText(/Scroll to bottom/)).not.toBeInTheDocument();
  });

  it("renders the inner sizing div with correct total height", async () => {
    await renderAndFlush(
      <VirtualList
        items={makeItems(10)}
        estimateSize={() => 40}
        renderItem={(item) => <div>{item}</div>}
        style={{ height: 200 }}
      />,
    );

    const inner = document.querySelector("[data-virtual-list] > div") as HTMLElement;
    expect(inner).toBeTruthy();
    expect(inner.style.height).toBe("400px");
  });

  it("shows scroll-to-bottom button when autoFollow enabled and not at bottom", async () => {
    // Simulate user scrolled up: scrollHeight - scrollTop - clientHeight > threshold
    const origSH = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollHeight");
    Object.defineProperty(HTMLElement.prototype, "scrollHeight", {
      configurable: true,
      get() {
        if (this.hasAttribute?.("data-virtual-list")) return 5000;
        return origSH?.get?.call(this) ?? 0;
      },
    });

    await renderAndFlush(
      <VirtualList
        items={makeItems(100)}
        estimateSize={() => 40}
        renderItem={(item) => <div>{item}</div>}
        autoFollow
        style={{ height: 200 }}
      />,
    );

    // Fire a scroll event to trigger at-bottom check
    const container = document.querySelector("[data-virtual-list]") as HTMLElement;
    // scrollHeight(5000) - scrollTop(0) - clientHeight(200) = 4800 > threshold
    // So NOT at bottom => button should appear
    await act(async () => {
      container.dispatchEvent(new Event("scroll"));
    });

    expect(screen.queryByText(/Scroll to bottom/)).toBeInTheDocument();

    // Restore
    if (origSH) Object.defineProperty(HTMLElement.prototype, "scrollHeight", origSH);
  });

  it("scrollToIndex accepts align option", async () => {
    const ref = createRef<VirtualListRef>();

    await renderAndFlush(
      <VirtualList
        ref={ref}
        items={makeItems(100)}
        estimateSize={() => 40}
        renderItem={(item) => <div>{item}</div>}
        style={{ height: 200 }}
      />,
    );

    // Should accept align options without throwing
    act(() => { ref.current!.scrollToIndex(50, { align: "center" }); });
    act(() => { ref.current!.scrollToIndex(0, { align: "start" }); });
    act(() => { ref.current!.scrollToIndex(99, { align: "end" }); });
  });

  it("calls onScrollStateChange callback on scroll", async () => {
    const onScrollStateChange = vi.fn();

    await renderAndFlush(
      <VirtualList
        items={makeItems(100)}
        estimateSize={() => 40}
        renderItem={(item) => <div>{item}</div>}
        onScrollStateChange={onScrollStateChange}
        style={{ height: 200 }}
      />,
    );

    const container = document.querySelector("[data-virtual-list]") as HTMLElement;
    await act(async () => {
      container.dispatchEvent(new Event("scroll"));
    });

    expect(onScrollStateChange).toHaveBeenCalled();
  });
});
