import posthog from 'posthog-js'

const POSTHOG_API_KEY = 'phc_kiloforge_placeholder'
const POSTHOG_HOST = 'https://us.i.posthog.com'

let initialized = false

export function initAnalytics(): void {
  if (initialized) return

  // Skip initialization if key is still placeholder or empty.
  if (!POSTHOG_API_KEY || POSTHOG_API_KEY === 'phc_kiloforge_placeholder') {
    return
  }

  posthog.init(POSTHOG_API_KEY, {
    api_host: POSTHOG_HOST,
    autocapture: false,
    capture_pageview: false, // We handle this manually on route changes.
    persistence: 'localStorage',
  })
  initialized = true
}

export function capturePageview(path: string): void {
  if (!initialized) return
  posthog.capture('$pageview', { $current_url: path })
}
