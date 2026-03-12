import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SlashCommandAutocomplete } from "./SlashCommandAutocomplete";

const COMMANDS = [
  { slashCommand: "/kf-interactive", label: "Interactive", description: "General-purpose assistant" },
  { slashCommand: "/kf-architect", label: "Architect", description: "Generate implementation tracks" },
  { slashCommand: "/kf-advisor-product", label: "Product Advisor", description: "Product design guidance" },
  { slashCommand: "/kf-advisor-reliability", label: "Reliability Advisor", description: "Testing and CI audits" },
];

describe("SlashCommandAutocomplete", () => {
  // Task 1: renders command list when input starts with '/'
  it("renders all commands when input is just '/'", () => {
    render(
      <SlashCommandAutocomplete
        input="/"
        commands={COMMANDS}
        selectedIndex={0}
        onSelect={vi.fn()}
      />,
    );
    for (const cmd of COMMANDS) {
      expect(screen.getByText(cmd.slashCommand)).toBeInTheDocument();
      expect(screen.getByText(cmd.description)).toBeInTheDocument();
    }
  });

  it("does not render when input does not start with '/'", () => {
    const { container } = render(
      <SlashCommandAutocomplete
        input="hello"
        commands={COMMANDS}
        selectedIndex={0}
        onSelect={vi.fn()}
      />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("does not render when input is empty", () => {
    const { container } = render(
      <SlashCommandAutocomplete
        input=""
        commands={COMMANDS}
        selectedIndex={0}
        onSelect={vi.fn()}
      />,
    );
    expect(container.firstChild).toBeNull();
  });

  // Task 2: filtering narrows commands
  it("filters commands as user types more characters", () => {
    render(
      <SlashCommandAutocomplete
        input="/kf-a"
        commands={COMMANDS}
        selectedIndex={0}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByText("/kf-architect")).toBeInTheDocument();
    expect(screen.getByText("/kf-advisor-product")).toBeInTheDocument();
    expect(screen.getByText("/kf-advisor-reliability")).toBeInTheDocument();
    expect(screen.queryByText("/kf-interactive")).not.toBeInTheDocument();
  });

  it("filters to single match", () => {
    render(
      <SlashCommandAutocomplete
        input="/kf-arch"
        commands={COMMANDS}
        selectedIndex={0}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByText("/kf-architect")).toBeInTheDocument();
    expect(screen.queryByText("/kf-interactive")).not.toBeInTheDocument();
    expect(screen.queryByText("/kf-advisor-product")).not.toBeInTheDocument();
  });

  it("renders nothing when no commands match", () => {
    const { container } = render(
      <SlashCommandAutocomplete
        input="/zzz"
        commands={COMMANDS}
        selectedIndex={0}
        onSelect={vi.fn()}
      />,
    );
    expect(container.firstChild).toBeNull();
  });

  // Task 3: keyboard navigation and selection
  it("highlights the item at selectedIndex", () => {
    render(
      <SlashCommandAutocomplete
        input="/"
        commands={COMMANDS}
        selectedIndex={1}
        onSelect={vi.fn()}
      />,
    );
    const items = screen.getAllByRole("option");
    expect(items[0]).not.toHaveAttribute("aria-selected", "true");
    expect(items[1]).toHaveAttribute("aria-selected", "true");
    expect(items[2]).not.toHaveAttribute("aria-selected", "true");
  });

  it("calls onSelect with the command when item is clicked", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <SlashCommandAutocomplete
        input="/"
        commands={COMMANDS}
        selectedIndex={0}
        onSelect={onSelect}
      />,
    );
    await user.click(screen.getByText("/kf-architect"));
    expect(onSelect).toHaveBeenCalledWith("/kf-architect");
  });

  it("renders with listbox role for accessibility", () => {
    render(
      <SlashCommandAutocomplete
        input="/"
        commands={COMMANDS}
        selectedIndex={0}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByRole("listbox")).toBeInTheDocument();
  });

  it("wraps selectedIndex when it exceeds filtered list length", () => {
    render(
      <SlashCommandAutocomplete
        input="/kf-arch"
        commands={COMMANDS}
        selectedIndex={5}
        onSelect={vi.fn()}
      />,
    );
    // Only 1 match, selectedIndex 5 should wrap to 0
    const items = screen.getAllByRole("option");
    expect(items).toHaveLength(1);
    expect(items[0]).toHaveAttribute("aria-selected", "true");
  });
});
