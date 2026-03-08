package badge

import "fmt"

// TrackBadgeMarkdown returns markdown for a track status badge image linked to the detail page.
func TrackBadgeMarkdown(trackID, baseURL string) string {
	return fmt.Sprintf("[![status](%s/api/badges/track/%s)](%s/tracks/%s)",
		baseURL, trackID, baseURL, trackID)
}

// PRBadgeMarkdown returns markdown for a PR status badge linked to the detail page.
func PRBadgeMarkdown(slug string, prNum int, baseURL string) string {
	return fmt.Sprintf("[![agents](%s/api/badges/pr/%s/%d)](%s/pr/%s/%d)",
		baseURL, slug, prNum, baseURL, slug, prNum)
}

// AgentBadgeMarkdown returns markdown for an agent status badge linked to a target URL.
func AgentBadgeMarkdown(agentID, baseURL, linkURL string) string {
	return fmt.Sprintf("[![agent](%s/api/badges/agent/%s)](%s)",
		baseURL, agentID, linkURL)
}
