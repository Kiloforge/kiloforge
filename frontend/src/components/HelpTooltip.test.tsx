import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { HelpTooltip } from "./HelpTooltip";

describe("HelpTooltip", () => {
  it("renders trigger with question mark icon", () => {
    render(<HelpTooltip term="Tracks" definition="Units of work." />);
    expect(screen.getByRole("button", { name: /what is tracks/i })).toBeInTheDocument();
  });

  it("shows definition on hover", () => {
    render(<HelpTooltip term="Tracks" definition="Units of work." />);
    fireEvent.mouseEnter(screen.getByRole("button", { name: /what is tracks/i }));
    expect(screen.getByText("Units of work.")).toBeInTheDocument();
  });

  it("hides definition on mouse leave", () => {
    render(<HelpTooltip term="Tracks" definition="Units of work." />);
    const trigger = screen.getByRole("button", { name: /what is tracks/i });
    fireEvent.mouseEnter(trigger);
    expect(screen.getByText("Units of work.")).toBeInTheDocument();
    fireEvent.mouseLeave(trigger);
    expect(screen.queryByText("Units of work.")).not.toBeInTheDocument();
  });

  it("toggles definition on click", () => {
    render(<HelpTooltip term="Tracks" definition="Units of work." />);
    const trigger = screen.getByRole("button", { name: /what is tracks/i });
    fireEvent.click(trigger);
    expect(screen.getByText("Units of work.")).toBeInTheDocument();
    fireEvent.click(trigger);
    expect(screen.queryByText("Units of work.")).not.toBeInTheDocument();
  });

  it("displays term as tooltip title", () => {
    render(<HelpTooltip term="Agents" definition="AI workers." />);
    fireEvent.mouseEnter(screen.getByRole("button", { name: /what is agents/i }));
    expect(screen.getByText("Agents")).toBeInTheDocument();
  });
});
