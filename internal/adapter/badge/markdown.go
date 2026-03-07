package badge

import "fmt"

// TrackBadgeMarkdown returns markdown for a track status badge image linked to the detail page.
func TrackBadgeMarkdown(trackID, relayURL string) string {
	return fmt.Sprintf("[![status](%s/api/badges/track/%s)](%s/tracks/%s)",
		relayURL, trackID, relayURL, trackID)
}

// PRBadgeMarkdown returns markdown for a PR status badge linked to the detail page.
func PRBadgeMarkdown(slug string, prNum int, relayURL string) string {
	return fmt.Sprintf("[![agents](%s/api/badges/pr/%s/%d)](%s/pr/%s/%d)",
		relayURL, slug, prNum, relayURL, slug, prNum)
}

// AgentBadgeMarkdown returns markdown for an agent status badge linked to a target URL.
func AgentBadgeMarkdown(agentID, relayURL, linkURL string) string {
	return fmt.Sprintf("[![agent](%s/api/badges/agent/%s)](%s)",
		relayURL, agentID, linkURL)
}
