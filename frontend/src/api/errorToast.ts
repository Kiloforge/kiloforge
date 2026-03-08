import { FetchError } from "./fetcher";
import type { ToastVariant } from "../components/toast/ToastProvider";

type AddToastFn = (variant: ToastVariant, message: string, detail?: string) => void;

let _addToast: AddToastFn | null = null;

/** Called once from QueryErrorBridge to connect toast context */
export function setToastRef(ref: { addToast: AddToastFn }) {
  _addToast = ref.addToast;
}

/** Called from MutationCache onError — formats and shows a toast */
export function formatMutationError(error: unknown) {
  if (!_addToast) return;

  if (error instanceof FetchError) {
    const body = error.body as Record<string, unknown> | undefined;
    const detail = body?.error
      ? String(body.error)
      : body?.message
        ? String(body.message)
        : undefined;
    _addToast("error", `Request failed (${error.status})`, detail);
  } else if (error instanceof Error) {
    _addToast("error", error.message);
  } else {
    _addToast("error", "An unexpected error occurred");
  }
}

/** Manual toast for non-query errors (SSE, WebSocket, etc.) */
export function showToast(variant: ToastVariant, message: string, detail?: string) {
  _addToast?.(variant, message, detail);
}
