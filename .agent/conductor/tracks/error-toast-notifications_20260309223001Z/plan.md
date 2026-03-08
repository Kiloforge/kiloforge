# Implementation Plan: Error Toast Notifications in Dashboard

**Track ID:** error-toast-notifications_20260309223001Z

## Phase 1: Toast Infrastructure

- [ ] Task 1.1: Choose approach — lightweight library (react-hot-toast/sonner) or custom component
- [ ] Task 1.2: Create `ToastProvider` and `useToast` hook
- [ ] Task 1.3: Create `Toast` component with error/warning/success variants, auto-dismiss, manual dismiss
- [ ] Task 1.4: Wrap app with `ToastProvider` in `main.tsx` or `App.tsx`

## Phase 2: Global Error Integration

- [ ] Task 2.1: Add global `onError` to `QueryClient` mutation defaults — auto-shows toast for all mutation failures
- [ ] Task 2.2: Format error messages — extract HTTP status and body from fetch errors
- [ ] Task 2.3: Add SSE disconnect warning toast in `useSSE` hook

## Phase 3: Verification

- [ ] Task 3.1: `npm run build` succeeds
- [ ] Task 3.2: Rebuild dist and commit
- [ ] Task 3.3: Manual test — trigger an API error, verify toast appears and auto-dismisses
