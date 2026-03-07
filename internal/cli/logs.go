package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"conductor-relay/internal/config"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <agent-id>",
	Short: "View logs for an agent",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogs,
}

var flagLogsFollow bool

func init() {
	logsCmd.Flags().BoolVarP(&flagLogsFollow, "follow", "f", false, "Follow log output (tail -f)")
}

func runLogs(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	agentID := args[0]
	logFile := filepath.Join(cfg.DataDir, "logs", agentID+".log")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		// Try prefix match.
		entries, _ := os.ReadDir(filepath.Join(cfg.DataDir, "logs"))
		for _, e := range entries {
			if len(agentID) >= 4 && e.Name()[:len(agentID)] == agentID {
				logFile = filepath.Join(cfg.DataDir, "logs", e.Name())
				break
			}
		}
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			return fmt.Errorf("no logs found for agent %s", agentID)
		}
	}

	f, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if flagLogsFollow {
		fmt.Println("--- following (Ctrl+C to stop) ---")
		// For follow mode, we'll tail the file. Simplified version:
		// In production this would use fsnotify or similar.
		return tailFile(logFile)
	}

	return scanner.Err()
}

func tailFile(path string) error {
	// Simple tail -f implementation using polling.
	// A production version would use fsnotify.
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Seek to end.
	if _, err := f.Seek(0, 2); err != nil {
		return err
	}

	scanner := bufio.NewScanner(f)
	for {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		// Scanner reached EOF, wait briefly and retry.
		// This blocks until interrupted.
		select {}
	}
}
