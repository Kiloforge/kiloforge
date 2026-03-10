import type PostHog from 'posthog-js'

const POSTHOG_API_KEY = 'phc_kiloforge_placeholder'
const POSTHOG_HOST = 'https://us.i.posthog.com'

let posthogInstance: PostHog | null = null

export function initAnalytics(): void {
  if (posthogInstance) return

  // Skip initialization if key is still placeholder or empty.
  if (!POSTHOG_API_KEY || POSTHOG_API_KEY === 'phc_kiloforge_placeholder') {
    return
  }

  // Skip in non-browser environments (e.g., test runners).
  if (typeof window === 'undefined') return

  // Dynamic import avoids posthog-js side effects at module load time,
  // which would fail in test environments where `document` is not defined.
  void import('posthog-js').then(({ default: posthog }) => {
    posthog.init(POSTHOG_API_KEY, {
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
