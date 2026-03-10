import { useEffect, useCallback } from "react";

interface ShortcutActions {
  tileAll: () => void;
  cycleFocusNext: () => void;
  cycleFocusPrev: () => void;
  toggleMinimize: () => void;
  toggleMaximize: () => void;
  closeFocused: () => void;
  snapLeft: () => void;
  snapRight: () => void;
  showHelp: () => void;
  toggleFullScreen?: () => void;
}

function isInputTarget(target: EventTarget | null): boolean {
  if (!target || !(target instanceof HTMLElement)) return false;
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || target.isContentEditable;
}

export function useKeyboardShortcuts(actions: ShortcutActions) {
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey;
      if (!mod) return;

      // Cmd/Ctrl+? (with or without shift, since ? requires shift)
      if (e.key === "?" || (e.shiftKey && e.key === "/")) {
        e.preventDefault();
        actions.showHelp();
        return;
      }

      if (!e.shiftKey) return;

      // Window-level shortcuts fire even when input is focused
      const windowShortcuts: Record<string, () => void> = {
        w: actions.closeFocused,
        "]": actions.cycleFocusNext,
        "[": actions.cycleFocusPrev,
      };

      const key = e.key.toLowerCase();

      if (windowShortcuts[key]) {
        e.preventDefault();
        windowShortcuts[key]();
        return;
      }

      // Skip remaining shortcuts when an input/textarea is focused
      if (isInputTarget(e.target)) return;

      // Full-screen command mode takes priority over maximize
      if (key === "f" && actions.toggleFullScreen) {
        e.preventDefault();
        actions.toggleFullScreen();
        return;
      }

      const shortcuts: Record<string, () => void> = {
        t: actions.tileAll,
        m: actions.toggleMinimize,
        f: actions.toggleMaximize,
        arrowleft: actions.snapLeft,
        arrowright: actions.snapRight,
      };

      if (shortcuts[key]) {
        e.preventDefault();
        shortcuts[key]();
      }
    },
    [actions],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}

export interface ShortcutEntry {
  keys: string;
  description: string;
}

export function getShortcutList(isMac: boolean): ShortcutEntry[] {
  const mod = isMac ? "⌘" : "Ctrl+";
  const shift = isMac ? "⇧" : "Shift+";
  return [
    { keys: `${mod}${shift}T`, description: "Tile all windows" },
    { keys: `${mod}${shift}]`, description: "Focus next window" },
    { keys: `${mod}${shift}[`, description: "Focus previous window" },
    { keys: `${mod}${shift}M`, description: "Minimize / restore window" },
    { keys: `${mod}${shift}F`, description: "Toggle full-screen command mode" },
    { keys: `${mod}${shift}W`, description: "Close focused window" },
    { keys: `${mod}${shift}←`, description: "Snap window to left half" },
    { keys: `${mod}${shift}→`, description: "Snap window to right half" },
    { keys: `${mod}?`, description: "Show keyboard shortcuts" },
  ];
}

/** @deprecated Use getShortcutList(isMac) instead */
export const SHORTCUT_LIST: ShortcutEntry[] = getShortcutList(true);
