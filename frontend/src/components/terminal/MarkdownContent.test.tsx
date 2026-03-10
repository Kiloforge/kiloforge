import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { MarkdownContent } from "./MarkdownContent";

describe("MarkdownContent", () => {
  it("renders empty string without error", () => {
    const { container } = render(<MarkdownContent text="" />);
    expect(container.querySelector(".markdown")).toBeTruthy();
  });

  it("renders plain text", () => {
    render(<MarkdownContent text="Hello world" />);
    expect(screen.getByText("Hello world")).toBeTruthy();
  });

  it("renders inline code", () => {
    render(<MarkdownContent text="Use `foo()` here" />);
    expect(screen.getByText("foo()")).toBeTruthy();
    expect(screen.getByText("foo()").tagName).toBe("CODE");
  });

  it("renders fenced code block with copy button", () => {
    render(<MarkdownContent text={'```\nconsole.log("hi")\n```'} />);
    expect(screen.getByText('console.log("hi")')).toBeTruthy();
    expect(screen.getByText("Copy")).toBeTruthy();
  });

  it("renders links with target=_blank", () => {
    render(<MarkdownContent text="[link](https://example.com)" />);
    const link = screen.getByText("link");
    expect(link.tagName).toBe("A");
    expect(link.getAttribute("target")).toBe("_blank");
  });

  it("coerces non-string text prop to string", () => {
    // Simulate a runtime type mismatch (e.g., server sends number)
    const { container } = render(<MarkdownContent text={42 as unknown as string} />);
    expect(container.textContent).toContain("42");
  });

  it("handles null/undefined text gracefully", () => {
    const { container } = render(<MarkdownContent text={null as unknown as string} />);
    expect(container.querySelector(".markdown")).toBeTruthy();
  });
});
