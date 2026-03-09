import { createElement } from "react";

/**
 * Formats text with markdown code blocks and inline code.
 * Returns ReactNode array for rendering.
 */
export function formatCode(text: string, codeBlockClass: string, inlineCodeClass: string): React.ReactNode[] {
  const parts = text.split(/(```[\s\S]*?```)/g);
  return parts.map((part, i) => {
    if (part.startsWith("```") && part.endsWith("```")) {
      const inner = part.slice(3, -3);
      const newlineIdx = inner.indexOf("\n");
      const code = newlineIdx >= 0 ? inner.slice(newlineIdx + 1) : inner;
      return createElement("pre", { key: i, className: codeBlockClass },
        createElement("code", null, code),
      );
    }
    const inlineParts = part.split(/(`[^`]+`)/g);
    return createElement("span", { key: i },
      ...inlineParts.map((ip, j) =>
        ip.startsWith("`") && ip.endsWith("`")
          ? createElement("code", { key: j, className: inlineCodeClass }, ip.slice(1, -1))
          : createElement("span", { key: j }, ip),
      ),
    );
  });
}
