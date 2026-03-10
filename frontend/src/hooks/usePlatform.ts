import { useMemo } from "react";

export interface PlatformInfo {
  isMac: boolean;
  mod: string;
  shift: string;
  modKey: "Meta" | "Control";
}

function detectMac(): boolean {
  if (typeof navigator === "undefined") return false;
  // navigator.platform is deprecated but widely supported; userAgent as fallback
  const platform = navigator.platform?.toLowerCase() ?? "";
  if (platform.includes("mac")) return true;
  return /macintosh|mac os x/i.test(navigator.userAgent);
}

export function usePlatform(): PlatformInfo {
  return useMemo(() => {
    const isMac = detectMac();
    return {
      isMac,
      mod: isMac ? "⌘" : "Ctrl",
      shift: isMac ? "⇧" : "Shift+",
      modKey: isMac ? "Meta" : "Control",
    };
  }, []);
}
