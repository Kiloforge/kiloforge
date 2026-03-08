import { createContext, useContext, useCallback, useState, useRef, type ReactNode } from "react";

export type ToastVariant = "error" | "warning" | "success";

export interface Toast {
  id: string;
  variant: ToastVariant;
  message: string;
  detail?: string;
}

interface ToastContextValue {
  toasts: Toast[];
  addToast: (variant: ToastVariant, message: string, detail?: string) => void;
  removeToast: (id: string) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error("useToast must be used within ToastProvider");
  return ctx;
}

/** Safe version that returns null outside of provider */
export function useToastSafe(): ToastContextValue | null {
  return useContext(ToastContext);
}

const AUTO_DISMISS_MS = 5000;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const nextId = useRef(0);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const addToast = useCallback(
    (variant: ToastVariant, message: string, detail?: string) => {
      const id = String(++nextId.current);
      setToasts((prev) => [...prev, { id, variant, message, detail }]);
      setTimeout(() => removeToast(id), AUTO_DISMISS_MS);
    },
    [removeToast],
  );

  return (
    <ToastContext.Provider value={{ toasts, addToast, removeToast }}>
      {children}
    </ToastContext.Provider>
  );
}
