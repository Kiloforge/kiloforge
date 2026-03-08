package cli

import (
	"context"
	"fmt"
	"strings"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/compose"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/gitea"
	"kiloforge/internal/adapter/persistence/jsonfile"
	"kiloforge/internal/adapter/pidfile"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show relay status, quota usage, and agent costs",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("load config: %w (have you run 'crelay init'?)", err)
	}

	// Check relay daemon.
	pidMgr := pidfile.New(cfg.DataDir)
	relayRunning, relayPID, _ := pidMgr.IsRunning()

	// Check Gitea via API.
	giteaStatus := "stopped"
	giteaVersion := ""
	client := gitea.NewClientWithToken(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.APIToken)
	if v, err := client.CheckVersion(ctx); err == nil {
		giteaStatus = "running"
		giteaVersion = v
	}

	// Try compose ps for container details.
	composeInfo := ""
	if runner, err := compose.Detect(); err == nil {
		if ps, err := runner.Ps(ctx, cfg.DataDir); err == nil {
			composeInfo = strings.TrimSpace(ps)
		}
	}

	fmt.Println("Conductor Relay Status")
	fmt.Println("======================")

	if relayRunning {
		fmt.Printf("Relay:       running (PID %d) on :%d\n", relayPID, cfg.RelayPort)
	} else {
		fmt.Println("Relay:       stopped")
	}

	if giteaVersion != "" {
		fmt.Printf("Gitea:       %s (v%s) — %s\n", giteaStatus, giteaVersion, cfg.GiteaURL())
	} else {
		fmt.Printf("Gitea:       %s\n", giteaStatus)
	}
	fmt.Printf("Data:        %s\n", cfg.DataDir)
	fmt.Printf("Compose:     %s\n", cfg.ComposeFile)
	fmt.Printf("Server:      http://localhost:%d\n", cfg.RelayPort)
	if cfg.IsDashboardEnabled() {
		fmt.Printf("Dashboard:   http://localhost:%d/-/\n", cfg.RelayPort)
	} else {
		fmt.Println("Dashboard:   disabled")
	}

	// Load quota tracker data.
	tracker := agent.NewQuotaTracker(cfg.DataDir)
	if err := tracker.Load(); err == nil {
		printQuotaStatus(tracker, cfg)
	}

	// Load agent store for per-agent breakdown.
	if store, err := jsonfile.LoadAgentStore(cfg.DataDir); err == nil {
		printAgentCosts(tracker, store)
	}

	if composeInfo != "" {
		fmt.Println()
		fmt.Println(composeInfo)
	}

	return nil
}

func printQuotaStatus(tracker *agent.QuotaTracker, cfg *config.Config) {
	total := tracker.GetTotalUsage()

	quotaLabel := "OK"
	if tracker.IsRateLimited() {
		quotaLabel = fmt.Sprintf("LIMITED (retry after %s)", tracker.RetryAfter().Round(1e9))
	}

	fmt.Println()
	fmt.Printf("Quota:       %s\n", quotaLabel)
	fmt.Printf("Cost:        $%.2f", total.TotalCostUSD)
	if cfg.MaxSessionCostUSD > 0 {
		fmt.Printf(" / $%.2f", cfg.MaxSessionCostUSD)
	}
	fmt.Println()
	fmt.Printf("Tokens:      %s in / %s out (%d agents)\n",
		formatTokens(total.InputTokens), formatTokens(total.OutputTokens), total.AgentCount)
}

func printAgentCosts(tracker *agent.QuotaTracker, store *jsonfile.AgentStore) {
	agents := store.AgentList
	if len(agents) == 0 {
		return
	}

	// Only show agents that have usage data.
	type agentRow struct {
		id     string
		ref    string
		tokens int
		cost   float64
	}
	var rows []agentRow
	for _, a := range agents {
		usage := tracker.GetAgentUsage(a.ID)
		if usage == nil {
			continue
		}
		rows = append(rows, agentRow{
			id:     a.ID[:8],
			ref:    a.Ref,
			tokens: usage.InputTokens + usage.OutputTokens,
			cost:   usage.TotalCostUSD,
		})
	}

	if len(rows) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Agent Usage:")
	for _, r := range rows {
		fmt.Printf("  %-10s %-30s tokens: %-10s cost: $%.2f\n",
			r.id, r.ref, formatTokens(r.tokens), r.cost)
	}
}

// formatTokens formats an integer with comma separators.
func formatTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteByte(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}
