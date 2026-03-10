import { Component } from "react";
import type { ErrorInfo, ReactNode } from "react";
import styles from "./TerminalBubbles.module.css";

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: string;
}

/**
 * Error boundary that catches render errors in individual terminal messages.
 * Displays a compact inline fallback instead of crashing the entire terminal.
 */
export class MessageErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: "" };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error: error.message || "Render error" };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.warn("[MessageErrorBoundary] Render error in terminal message:", error, info.componentStack);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className={`${styles.message} ${styles.errorMessage}`}>
          <span className={styles.messageIcon}>err</span>
          <div className={styles.messageContent}>
            Failed to render message: {this.state.error}
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
