import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { GaugeCard } from "./GaugeCard";

describe("GaugeCard", () => {
  it("renders label and formatted value", () => {
    render(<GaugeCard label="Budget" value={50} max={100} unit="$" />);
    expect(screen.getByText("Budget")).toBeInTheDocument();
    expect(screen.getByText("$50")).toBeInTheDocument();
  });

  it("renders SVG arc at 0%", () => {
    const { container } = render(<GaugeCard label="Cost" value={0} max={100} />);
    const arc = container.querySelector("[data-testid='gauge-fill']");
    expect(arc).toBeInTheDocument();
    // At 0%, stroke-dashoffset should equal the arc length (no fill)
    expect(arc).toHaveAttribute("stroke-dashoffset");
  });

  it("renders SVG arc at 50%", () => {
    const { container } = render(<GaugeCard label="Cost" value={50} max={100} />);
    const arc = container.querySelector("[data-testid='gauge-fill']");
    expect(arc).toBeInTheDocument();
    const offset = Number(arc!.getAttribute("stroke-dashoffset"));
    const dasharray = Number(arc!.getAttribute("stroke-dasharray")!.split(",")[0]);
    // 50% fill means offset is ~half the arc length
    expect(offset).toBeCloseTo(dasharray * 0.5, 0);
  });

  it("renders SVG arc at 100%", () => {
    const { container } = render(<GaugeCard label="Cost" value={100} max={100} />);
    const arc = container.querySelector("[data-testid='gauge-fill']");
    expect(arc).toBeInTheDocument();
    const offset = Number(arc!.getAttribute("stroke-dashoffset"));
    expect(offset).toBeCloseTo(0, 0);
  });

  it("applies green color class below 60%", () => {
    const { container } = render(<GaugeCard label="Budget" value={30} max={100} />);
    const arc = container.querySelector("[data-testid='gauge-fill']");
    expect(arc?.getAttribute("class")).toMatch(/green/);
  });

  it("applies yellow color class between 60-85%", () => {
    const { container } = render(<GaugeCard label="Budget" value={70} max={100} />);
    const arc = container.querySelector("[data-testid='gauge-fill']");
    expect(arc?.getAttribute("class")).toMatch(/yellow/);
  });

  it("applies red color class above 85%", () => {
    const { container } = render(<GaugeCard label="Budget" value={90} max={100} />);
    const arc = container.querySelector("[data-testid='gauge-fill']");
    expect(arc?.getAttribute("class")).toMatch(/red/);
  });

  it("renders flat display when max is 0 or undefined", () => {
    const { container } = render(<GaugeCard label="Agents" value={5} />);
    // No SVG arc when max is not provided
    const svg = container.querySelector("svg");
    expect(svg).toBeNull();
    expect(screen.getByText("5")).toBeInTheDocument();
    expect(screen.getByText("Agents")).toBeInTheDocument();
  });

  it("renders subtitle when provided", () => {
    render(<GaugeCard label="Budget" value={50} max={100} subtitle="of $100" />);
    expect(screen.getByText("of $100")).toBeInTheDocument();
  });
});
