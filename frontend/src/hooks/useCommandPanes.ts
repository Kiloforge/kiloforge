import { useState, useCallback, useRef, useEffect } from "react";

const STORAGE_KEY = "kf-command-panes";

export type SplitDirection = "horizontal" | "vertical";

export interface PaneLeaf {
  kind: "leaf";
  id: string;
  agentId: string | null;
}

export interface PaneSplit {
  kind: "split";
  id: string;
  direction: SplitDirection;
  children: [PaneNode, PaneNode];
  /** Ratio of first child (0-1). Default 0.5. */
  ratio: number;
}

export type PaneNode = PaneLeaf | PaneSplit;

let nextId = 1;
function genId(): string {
  return `pane-${nextId++}`;
}

function makeLeaf(agentId: string | null = null): PaneLeaf {
  return { kind: "leaf", id: genId(), agentId };
}

function loadState(): { root: PaneNode; activePaneId: string } | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    if (parsed?.root && parsed?.activePaneId) {
      // Reset nextId to avoid collisions
      const maxId = findMaxId(parsed.root);
      nextId = maxId + 1;
      return parsed;
    }
  } catch {
    // ignore corrupt data
  }
  return null;
}

function findMaxId(node: PaneNode): number {
  const num = parseInt(node.id.replace("pane-", ""), 10) || 0;
  if (node.kind === "leaf") return num;
  return Math.max(num, findMaxId(node.children[0]), findMaxId(node.children[1]));
}

function saveState(root: PaneNode, activePaneId: string) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ root, activePaneId }));
  } catch {
    // ignore quota errors
  }
}

function findNode(root: PaneNode, id: string): PaneNode | null {
  if (root.id === id) return root;
  if (root.kind === "split") {
    return findNode(root.children[0], id) || findNode(root.children[1], id);
  }
  return null;
}

function collectLeaves(node: PaneNode): PaneLeaf[] {
  if (node.kind === "leaf") return [node];
  return [...collectLeaves(node.children[0]), ...collectLeaves(node.children[1])];
}

function replaceNode(root: PaneNode, id: string, replacement: PaneNode): PaneNode {
  if (root.id === id) return replacement;
  if (root.kind === "split") {
    return {
      ...root,
      children: [
        replaceNode(root.children[0], id, replacement),
        replaceNode(root.children[1], id, replacement),
      ] as [PaneNode, PaneNode],
    };
  }
  return root;
}

/** Remove a leaf by id, collapsing its parent split to the sibling. */
function removeLeaf(root: PaneNode, id: string): PaneNode | null {
  if (root.kind === "leaf") {
    return root.id === id ? null : root;
  }
  // If one child is the target, return the other
  if (root.children[0].id === id) return root.children[1];
  if (root.children[1].id === id) return root.children[0];
  // Recurse
  const left = removeLeaf(root.children[0], id);
  if (left !== root.children[0]) {
    if (left === null) return root.children[1];
    return { ...root, children: [left, root.children[1]] as [PaneNode, PaneNode] };
  }
  const right = removeLeaf(root.children[1], id);
  if (right !== root.children[1]) {
    if (right === null) return root.children[0];
    return { ...root, children: [root.children[0], right] as [PaneNode, PaneNode] };
  }
  return root;
}

export interface UseCommandPanesResult {
  root: PaneNode;
  activePaneId: string;
  setActivePaneId: (id: string) => void;
  splitPane: (id: string, direction: SplitDirection) => void;
  closePane: (id: string) => boolean; // returns true if last pane was closed
  setAgentId: (paneId: string, agentId: string | null) => void;
  focusNext: () => void;
  focusPrev: () => void;
  leafCount: number;
  reset: () => void;
}

export function useCommandPanes(): UseCommandPanesResult {
  const loaded = useRef(false);
  const [state, setState] = useState<{ root: PaneNode; activePaneId: string }>(() => {
    const saved = loadState();
    if (saved) {
      loaded.current = true;
      return saved;
    }
    const leaf = makeLeaf();
    return { root: leaf, activePaneId: leaf.id };
  });

  // Persist on change
  useEffect(() => {
    saveState(state.root, state.activePaneId);
  }, [state]);

  const splitPane = useCallback((id: string, direction: SplitDirection) => {
    setState((prev) => {
      const target = findNode(prev.root, id);
      if (!target || target.kind !== "leaf") return prev;
      const newLeaf = makeLeaf();
      const split: PaneSplit = {
        kind: "split",
        id: genId(),
        direction,
        children: [target, newLeaf],
        ratio: 0.5,
      };
      const newRoot = replaceNode(prev.root, id, split);
      return { root: newRoot, activePaneId: newLeaf.id };
    });
  }, []);

  const closePane = useCallback((id: string): boolean => {
    let lastClosed = false;
    setState((prev) => {
      const leaves = collectLeaves(prev.root);
      if (leaves.length <= 1) {
        lastClosed = true;
        return prev;
      }
      const result = removeLeaf(prev.root, id);
      if (!result) {
        lastClosed = true;
        return prev;
      }
      // If active pane was closed, pick the first remaining leaf
      let newActive = prev.activePaneId;
      if (prev.activePaneId === id) {
        const remaining = collectLeaves(result);
        newActive = remaining[0]?.id ?? prev.activePaneId;
      }
      return { root: result, activePaneId: newActive };
    });
    return lastClosed;
  }, []);

  const setAgentId = useCallback((paneId: string, agentId: string | null) => {
    setState((prev) => {
      const target = findNode(prev.root, paneId);
      if (!target || target.kind !== "leaf") return prev;
      const updated: PaneLeaf = { ...target, agentId };
      return { ...prev, root: replaceNode(prev.root, paneId, updated) };
    });
  }, []);

  const setActivePaneId = useCallback((id: string) => {
    setState((prev) => ({ ...prev, activePaneId: id }));
  }, []);

  const focusNext = useCallback(() => {
    setState((prev) => {
      const leaves = collectLeaves(prev.root);
      const idx = leaves.findIndex((l) => l.id === prev.activePaneId);
      const next = (idx + 1) % leaves.length;
      return { ...prev, activePaneId: leaves[next].id };
    });
  }, []);

  const focusPrev = useCallback(() => {
    setState((prev) => {
      const leaves = collectLeaves(prev.root);
      const idx = leaves.findIndex((l) => l.id === prev.activePaneId);
      const next = (idx - 1 + leaves.length) % leaves.length;
      return { ...prev, activePaneId: leaves[next].id };
    });
  }, []);

  const reset = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY);
    const leaf = makeLeaf();
    setState({ root: leaf, activePaneId: leaf.id });
  }, []);

  const leafCount = collectLeaves(state.root).length;

  return {
    root: state.root,
    activePaneId: state.activePaneId,
    setActivePaneId,
    splitPane,
    closePane,
    setAgentId,
    focusNext,
    focusPrev,
    leafCount,
    reset,
  };
}
