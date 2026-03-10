import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider, MutationCache } from '@tanstack/react-query'
import { ToastProvider, useToast } from './components/toast/ToastProvider'
import { ErrorBoundary } from './components/ErrorBoundary'
import { formatMutationError, setToastRef } from './api/errorToast'
import './index.css'
import App from './App'

const mutationCache = new MutationCache({
  onError: (error) => {
    formatMutationError(error)
  },
})

const queryClient = new QueryClient({
  mutationCache,
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ToastProvider>
      <QueryErrorBridge />
      <ErrorBoundary>
        <QueryClientProvider client={queryClient}>
          <BrowserRouter basename="/">
            <App />
          </BrowserRouter>
        </QueryClientProvider>
      </ErrorBoundary>
    </ToastProvider>
  </StrictMode>,
)

/** Bridges ToastProvider context to the module-level error handler */
function QueryErrorBridge() {
  const toast = useToast()
  setToastRef(toast)
  return null
}
