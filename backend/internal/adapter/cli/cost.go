package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"crelay/internal/adapter/persistence/jsonfile"
	"crelay/internal/adapter/agent"
	"crelay/internal/adapter/config"

	"github.com/spf13/cobra"
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Show token usage and cost per agent",
	RunE:  runCost,
}

var flagCostJSON bool

func init() {
	costCmd.Flags().BoolVar(&flagCostJSON, "json", false, "Output as JSON")
}

func runCost(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("load config: %w (have you run 'crelay init'?)", err)
	}

	tracker := agent.NewQuotaTracker(cfg.DataDir)
	if err := tracker.Load(); err != nil {
		return fmt.Errorf("load quota data: %w", err)
	}

	store, err := jsonfile.LoadAgentStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load agent store: %w", err)
	}

	if flagCostJSON {
		return printCostJSON(tracker, store)
	}

	return printCostTable(tracker, store, cfg)
}

type costEntry struct {
	AgentID  string  `json:"agent_id"`
	Role     string  `json:"role"`
	Ref      string  `json:"ref"`
	Tokens   int     `json:"tokens"`
	CostUSD  float64 `json:"cost_usd"`
}

func printCostJSON(tracker *agent.QuotaTracker, store *jsonfile.AgentStore) error {
	total := tracker.GetTotalUsage()
	var entries []costEntry
	for _, a := range store.AgentList {
		usage := tracker.GetAgentUsage(a.ID)
		if usage == nil {
			continue
		}
		entries = append(entries, costEntry{
			AgentID: a.ID[:8],
			Role:    a.Role,
			Ref:     a.Ref,
			Tokens:  usage.InputTokens + usage.OutputTokens,
			CostUSD: usage.TotalCostUSD,
		})
	}

	out := map[string]any{
		"total_cost_usd": total.TotalCostUSD,
		"total_tokens":   total.InputTokens + total.OutputTokens,
		"agent_count":    total.AgentCount,
		"agents":         entries,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func printCostTable(tracker *agent.QuotaTracker, store *jsonfile.AgentStore, cfg *config.Config) error {
	total := tracker.GetTotalUsage()

	if total.AgentCount == 0 {
		fmt.Println("No cost data available.")
		return nil
	}

	fmt.Printf("Total Cost:  $%.2f", total.TotalCostUSD)
	if cfg.MaxSessionCostUSD > 0 {
		fmt.Printf(" / $%.2f", cfg.MaxSessionCostUSD)
	}
	fmt.Println()
	fmt.Printf("Tokens:      %s in / %s out\n", formatTokens(total.InputTokens), formatTokens(total.OutputTokens))
	fmt.Printf("Agents:      %d\n", total.AgentCount)
	fmt.Println()

	for _, a := range store.AgentList {
		usage := tracker.GetAgentUsage(a.ID)
		if usage == nil {
			continue
		}
		fmt.Printf("  %-10s %-10s %-30s %s tokens  $%.2f\n",
			a.ID[:8], a.Role, a.Ref,
			formatTokens(usage.InputTokens+usage.OutputTokens),
			usage.TotalCostUSD)
	}

	return nil
}
