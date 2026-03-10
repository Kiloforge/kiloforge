const POSTHOG_HOST = 'https://us.i.posthog.com'

let posthogInstance: { capture: (event: string, props?: Record<string, unknown>) => void } | null = null

export function initAnalytics(): void {
  if (posthogInstance) return

  const apiKey = import.meta.env.VITE_POSTHOG_API_KEY as string | undefined

  // Skip initialization if no API key is configured.
  if (!apiKey) return

  // Skip in non-browser environments (e.g., test runners).
  if (typeof window === 'undefined') return

  // Dynamic import avoids posthog-js side effects at module load time,
  // which would fail in test environments where `document` is not defined.
  void import('posthog-js').then(({ default: posthog }) => {
    posthog.init(apiKey, {
      api_host: POSTHOG_HOST,
      autocapture: false,
      capture_pageview: false, // We handle this manually on route changes.
      persistence: 'localStorage',
    })
    posthogInstance = posthog
  })
}

export function capturePageview(path: string): void {
  if (!posthogInstance) return
  posthogInstance.capture('$pageview', { $current_url: path })
}
