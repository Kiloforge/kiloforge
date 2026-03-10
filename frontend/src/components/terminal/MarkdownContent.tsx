import { Children, isValidElement, useState, useCallback } from "react";
import ReactMarkdown from "react-markdown";
import type { Components } from "react-markdown";
import styles from "./MarkdownContent.module.css";

interface Props {
  text: string;
}

/** Extract plain text from React children for the copy button. */
function extractText(node: React.ReactNode): string {
  if (typeof node === "string") return node;
  if (typeof node === "number") return String(node);
  if (!node) return "";
  if (isValidElement(node)) {
    return extractText((node.props as { children?: React.ReactNode }).children);
  }
  if (Array.isArray(node)) {
    return Children.toArray(node).map(extractText).join("");
  }
  return "";
}

function CopyButton({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  }, [code]);

  return (
    <button className={styles.copyBtn} onClick={handleCopy}>
      {copied ? "Copied!" : "Copy"}
    </button>
  );
}

const components: Components = {
  pre({ children }) {
    const codeText = extractText(children);
    return (
      <div className={styles.codeBlockWrapper}>
        <CopyButton code={codeText} />
        <pre className={styles.codeBlock}>{children}</pre>
      </div>
    );
  },
  code({ children, className }) {
    if (className) {
      return <code className={styles.codeBlockCode}>{children}</code>;
    }
    return <code className={styles.inlineCode}>{children}</code>;
  },
  a({ href, children }) {
    return (
      <a href={href} target="_blank" rel="noopener noreferrer" className={styles.link}>
        {children}
      </a>
    );
  },
};

export function MarkdownContent({ text }: Props) {
  const safeText = typeof text === "string" ? text : String(text ?? "");
  return (
    <div className={styles.markdown}>
      <ReactMarkdown components={components}>{safeText}</ReactMarkdown>
    </div>
  );
}
