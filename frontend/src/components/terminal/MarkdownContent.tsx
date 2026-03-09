import { useState, useCallback } from "react";
import ReactMarkdown from "react-markdown";
import type { Components } from "react-markdown";
import styles from "./MarkdownContent.module.css";

interface Props {
  text: string;
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
    // Extract code text for copy button
    let codeText = "";
    if (
      children &&
      typeof children === "object" &&
      "props" in (children as React.ReactElement)
    ) {
      const el = children as React.ReactElement<{ children?: React.ReactNode }>;
      codeText = typeof el.props.children === "string" ? el.props.children : "";
    }
    return (
      <div className={styles.codeBlockWrapper}>
        <CopyButton code={codeText} />
        <pre className={styles.codeBlock}>{children}</pre>
      </div>
    );
  },
  code({ children, className }) {
    // If className starts with "language-", it's a fenced code block (rendered inside <pre>)
    if (className) {
      return <code className={styles.codeBlockCode}>{children}</code>;
    }
    // Inline code
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
  return (
    <div className={styles.markdown}>
      <ReactMarkdown components={components}>{text}</ReactMarkdown>
    </div>
  );
}
