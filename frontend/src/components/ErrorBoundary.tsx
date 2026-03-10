import { Component } from "react";
import type { ReactNode, ErrorInfo } from "react";
import styles from "./ErrorBoundary.module.css";

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.warn("[ErrorBoundary] Caught render error:", error, info.componentStack);
  }

  handleReload = () => {
    window.location.reload();
  };

  handleDismiss = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className={styles.container}>
          <div className={styles.card}>
            <h2 className={styles.title}>Something went wrong</h2>
            <p className={styles.message}>
              An unexpected error occurred. You can try reloading the page or dismissing this error.
            </p>
            {this.state.error && (
              <pre className={styles.detail}>{this.state.error.message}</pre>
            )}
            <div className={styles.actions}>
              <button className={styles.reloadBtn} onClick={this.handleReload}>
                Reload Page
              </button>
              <button className={styles.dismissBtn} onClick={this.handleDismiss}>
                Dismiss
              </button>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
