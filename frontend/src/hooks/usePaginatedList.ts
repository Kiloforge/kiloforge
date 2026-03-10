import { useInfiniteQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import type { PaginatedResponse } from "../types/api";
import { fetcher } from "../api/fetcher";

const DEFAULT_LIMIT = 50;

interface UsePaginatedListOptions {
  queryKey: readonly unknown[];
  url: string;
  params?: Record<string, string>;
  limit?: number;
  enabled?: boolean;
}

export interface UsePaginatedListResult<T> {
  items: T[];
  totalCount: number;
  shownCount: number;
  remainingCount: number;
  isLoading: boolean;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: () => Promise<unknown>;
}

export function usePaginatedList<T>(options: UsePaginatedListOptions): UsePaginatedListResult<T> {
  const { queryKey, url, params, limit = DEFAULT_LIMIT, enabled } = options;

  const query = useInfiniteQuery<PaginatedResponse<T>>({
    queryKey,
    queryFn: ({ pageParam }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        for (const [k, v] of Object.entries(params)) {
          searchParams.set(k, v);
        }
      }
      searchParams.set("limit", String(limit));
      if (pageParam) {
        searchParams.set("cursor", pageParam as string);
      }
      return fetcher<PaginatedResponse<T>>(`${url}?${searchParams.toString()}`).then(
        (res) => res ?? { items: [], total_count: 0 },
      );
    },
    getNextPageParam: (lastPage) => lastPage?.next_cursor || undefined,
    initialPageParam: "",
    enabled,
  });

  const items = useMemo(
    () => query.data?.pages.flatMap((p) => p.items) ?? [],
    [query.data],
  );

  const totalCount = query.data?.pages[0]?.total_count ?? 0;
  const shownCount = items.length;

  return {
    items,
    totalCount,
    shownCount,
    remainingCount: Math.max(0, totalCount - shownCount),
    isLoading: query.isLoading,
    hasNextPage: query.hasNextPage ?? false,
    isFetchingNextPage: query.isFetchingNextPage,
    fetchNextPage: query.fetchNextPage,
  };
}
