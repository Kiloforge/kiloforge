/** Extracts a human-readable label and optional detail from tool input. */
export interface ToolSummary {
  label: string;
  detail?: string;
}

/**
 * Returns a human-readable summary for a tool call based on tool name
 * and its input parameters. Unknown tools fall back to JSON display.
 */
export function formatToolSummary(
  toolName: string | undefined,
  toolInput: Record<string, unknown> | undefined,
): ToolSummary {
  if (!toolName || !toolInput) {
    return { label: toolName ?? "unknown" };
  }

  const str = (key: string): string | undefined => {
    const v = toolInput[key];
    return typeof v === "string" ? v : undefined;
  };

  switch (toolName) {
    case "Bash": {
      const desc = str("description");
      const cmd = str("command");
      return {
        label: desc || cmd || "Bash",
        detail: cmd ? cmd : undefined,
      };
    }
    case "Read": {
      const fp = str("file_path");
      return { label: fp || "Read", detail: fp ? jsonDetail(toolInput, ["file_path"]) : undefined };
    }
    case "Write": {
      const fp = str("file_path");
      return { label: fp || "Write", detail: fp ? jsonDetail(toolInput, ["file_path"]) : undefined };
    }
    case "Edit": {
      const fp = str("file_path");
      return { label: fp || "Edit", detail: fp ? jsonDetail(toolInput, ["file_path"]) : undefined };
    }
    case "Glob": {
      const pattern = str("pattern");
      const path = str("path");
      const suffix = path ? ` in ${path}` : "";
      return { label: pattern ? `${pattern}${suffix}` : "Glob" };
    }
    case "Grep": {
      const pattern = str("pattern");
      const path = str("path");
      const suffix = path ? ` in ${path}` : "";
      return { label: pattern ? `/${pattern}/${suffix}` : "Grep" };
    }
    case "Agent": {
      const desc = str("description");
      return { label: desc || "Agent", detail: str("prompt") };
    }
    case "WebFetch": {
      const url = str("url");
      return { label: url || "WebFetch" };
    }
    case "WebSearch": {
      const query = str("query");
      return { label: query || "WebSearch" };
    }
    case "NotebookEdit": {
      const nb = str("notebook_path");
      return { label: nb || "NotebookEdit" };
    }
    default:
      return {
        label: toolName,
        detail: Object.keys(toolInput).length > 0
          ? JSON.stringify(toolInput, null, 2)
          : undefined,
      };
  }
}

/** JSON-stringify toolInput excluding already-shown keys. */
function jsonDetail(
  input: Record<string, unknown>,
  exclude: string[],
): string | undefined {
  const rest: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(input)) {
    if (!exclude.includes(k)) rest[k] = v;
  }
  return Object.keys(rest).length > 0
    ? JSON.stringify(rest, null, 2)
    : undefined;
}
