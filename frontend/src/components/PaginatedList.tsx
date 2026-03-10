import type { ReactNode } from "react";
import { InlineSpinner } from "./InlineSpinner";
import styles from "./PaginatedList.module.css";

interface PaginatedListProps {
  children: ReactNode;
  remainingCount: number;
  remainingLabel?: string;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  onLoadMore: () => void;
}

export function PaginatedList({
  children,
  remainingCount,
  remainingLabel = "more",
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
}: PaginatedListProps) {
  const showFooter = hasNextPage && remainingCount > 0;

  return (
    <div>
      {children}
      {isFetchingNextPage && (
        <div className={styles.footer}>
          <InlineSpinner />
        </div>
      )}
      {showFooter && !isFetchingNextPage && (
        <div className={styles.footer}>
          <button className={styles.loadMore} onClick={onLoadMore}>
            +{remainingCount} {remainingLabel}
          </button>
        </div>
      )}
    </div>
  );
}
