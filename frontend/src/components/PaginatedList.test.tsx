import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PaginatedList } from "./PaginatedList";

describe("PaginatedList", () => {
  it("renders children", () => {
    render(
      <PaginatedList remainingCount={0} hasNextPage={false} isFetchingNextPage={false} onLoadMore={() => {}}>
        <div>Item 1</div>
        <div>Item 2</div>
      </PaginatedList>,
    );
    expect(screen.getByText("Item 1")).toBeInTheDocument();
    expect(screen.getByText("Item 2")).toBeInTheDocument();
  });

  it("shows +N more button when there are remaining items", () => {
    render(
      <PaginatedList remainingCount={42} hasNextPage={true} isFetchingNextPage={false} onLoadMore={() => {}}>
        <div>Item 1</div>
      </PaginatedList>,
    );
    expect(screen.getByText("+42 more")).toBeInTheDocument();
  });

  it("shows custom label via remainingLabel", () => {
    render(
      <PaginatedList remainingCount={5} remainingLabel="completed" hasNextPage={true} isFetchingNextPage={false} onLoadMore={() => {}}>
        <div>Item 1</div>
      </PaginatedList>,
    );
    expect(screen.getByText("+5 completed")).toBeInTheDocument();
  });

  it("calls onLoadMore when +N button is clicked", async () => {
    const user = userEvent.setup();
    const onLoadMore = vi.fn();
    render(
      <PaginatedList remainingCount={10} hasNextPage={true} isFetchingNextPage={false} onLoadMore={onLoadMore}>
        <div>Item 1</div>
      </PaginatedList>,
    );
    await user.click(screen.getByText("+10 more"));
    expect(onLoadMore).toHaveBeenCalledTimes(1);
  });

  it("shows loading spinner when fetching next page", () => {
    render(
      <PaginatedList remainingCount={10} hasNextPage={true} isFetchingNextPage={true} onLoadMore={() => {}}>
        <div>Item 1</div>
      </PaginatedList>,
    );
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("does not show footer when hasNextPage is false", () => {
    render(
      <PaginatedList remainingCount={0} hasNextPage={false} isFetchingNextPage={false} onLoadMore={() => {}}>
        <div>Item 1</div>
      </PaginatedList>,
    );
    expect(screen.queryByText(/more/)).not.toBeInTheDocument();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("does not show footer when remainingCount is 0 even if hasNextPage", () => {
    render(
      <PaginatedList remainingCount={0} hasNextPage={true} isFetchingNextPage={false} onLoadMore={() => {}}>
        <div>Item 1</div>
      </PaginatedList>,
    );
    expect(screen.queryByText(/more/)).not.toBeInTheDocument();
  });
});
