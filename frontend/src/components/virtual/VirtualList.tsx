import {
  useRef,
  useCallback,
  useImperativeHandle,
  forwardRef,
  useEffect,
  useState,
  type ReactNode,
  type CSSProperties,
} from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import styles from "./VirtualList.module.css";

export interface VirtualListProps<T> {
  items: T[];
  estimateSize: (index: number) => number;
  renderItem: (item: T, index: number) => ReactNode;
  overscan?: number;
  autoFollow?: boolean;
  className?: string;
  style?: CSSProperties;
  onScrollStateChange?: (atBottom: boolean) => void;
}

export interface VirtualListRef {
  scrollToBottom: () => void;
  scrollToIndex: (index: number, opts?: { align?: "start" | "center" | "end" }) => void;
}

const AT_BOTTOM_THRESHOLD = 50; // px from bottom to consider "at bottom"

function VirtualListInner<T>(
  {
    items,
    estimateSize,
    renderItem,
    overscan = 5,
    autoFollow = false,
    className,
    style,
    onScrollStateChange,
  }: VirtualListProps<T>,
  ref: React.ForwardedRef<VirtualListRef>,
) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [atBottom, setAtBottom] = useState(true);
  const [newCount, setNewCount] = useState(0);
  const prevItemCountRef = useRef(items.length);

  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => scrollRef.current,
    estimateSize,
    overscan,
    measureElement: (el) => el.getBoundingClientRect().height,
  });

  const checkAtBottom = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return false;
    const isAtBottom =
      el.scrollHeight - el.scrollTop - el.clientHeight < AT_BOTTOM_THRESHOLD;
    return isAtBottom;
  }, []);

  const handleScroll = useCallback(() => {
    const isAtBottom = checkAtBottom();
    setAtBottom(isAtBottom);
    if (isAtBottom) setNewCount(0);
    onScrollStateChange?.(isAtBottom);
  }, [checkAtBottom, onScrollStateChange]);

  // Track new items when not at bottom
  useEffect(() => {
    const count = items.length;
    const prev = prevItemCountRef.current;
    if (count > prev && !atBottom) {
      setNewCount((n) => n + (count - prev));
    }
    prevItemCountRef.current = count;
  }, [items.length, atBottom]);

  // Auto-follow: scroll to bottom on new items
  useEffect(() => {
    if (autoFollow && atBottom && items.length > 0) {
      virtualizer.scrollToIndex(items.length - 1, { align: "end" });
    }
  }, [autoFollow, atBottom, items.length, virtualizer]);

  const scrollToBottom = useCallback(() => {
    if (items.length > 0) {
      virtualizer.scrollToIndex(items.length - 1, { align: "end" });
    }
    setAtBottom(true);
    setNewCount(0);
  }, [items.length, virtualizer]);

  const scrollToIndex = useCallback(
    (index: number, opts?: { align?: "start" | "center" | "end" }) => {
      virtualizer.scrollToIndex(index, { align: opts?.align ?? "start" });
    },
    [virtualizer],
  );

  useImperativeHandle(ref, () => ({ scrollToBottom, scrollToIndex }), [
    scrollToBottom,
    scrollToIndex,
  ]);

  const virtualItems = virtualizer.getVirtualItems();

  return (
    <div
      ref={scrollRef}
      data-virtual-list=""
      className={`${styles.container} ${className ?? ""}`}
      style={style}
      onScroll={handleScroll}
    >
      <div
        className={styles.inner}
        style={{ height: virtualizer.getTotalSize() }}
      >
        {virtualItems.map((vItem) => (
          <div
            key={vItem.key}
            ref={virtualizer.measureElement}
            data-index={vItem.index}
            className={styles.item}
            style={{ transform: `translateY(${vItem.start}px)` }}
          >
            {renderItem(items[vItem.index], vItem.index)}
          </div>
        ))}
      </div>
      {autoFollow && !atBottom && (
        <div className={styles.scrollToBottom}>
          <button
            className={styles.scrollToBottomBtn}
            onClick={scrollToBottom}
            type="button"
          >
            ↓ Scroll to bottom
            {newCount > 0 && (
              <span className={styles.newBadge}>{newCount}</span>
            )}
          </button>
        </div>
      )}
    </div>
  );
}

export const VirtualList = forwardRef(VirtualListInner) as <T>(
  props: VirtualListProps<T> & { ref?: React.Ref<VirtualListRef> },
) => ReactNode;
