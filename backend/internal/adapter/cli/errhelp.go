package cli

import (
	"fmt"

	"kiloforge/internal/adapter/prereq"
)

// notInitializedError returns an error message for when Kiloforge has never
// been initialized (no config/data directory exists).
func notInitializedError() string {
	return `Kiloforge is not initialized.

  'kf up' performs first-time setup: creates the data directory,
  starts the Cortex, and saves your configuration.

  Run 'kf up' to get started.`
}

// giteaNotRunningError returns an error message for when Kiloforge is
// initialized but the Gitea server is not currently running.
func giteaNotRunningError() string {
	return `Gitea is not running.

  The Kiloforge data directory exists, but the Gitea server isn't
  responding. It may have been stopped or failed to start.

  Run 'kf up' to start the Cortex.`
}

// configLoadError returns an error message for when config resolution fails,
// wrapping the underlying error with guidance.
func configLoadError(err error) string {
	return fmt.Sprintf(`Could not load configuration: %v

  This usually means Kiloforge hasn't been initialized yet.
  Run 'kf up' to perform first-time setup.`, err)
}

// emptyState returns a user-friendly message for commands that have no data
// to display. If hint is non-empty, it's shown as a next-step suggestion.
func emptyState(resource, hint string) string {
	msg := fmt.Sprintf("No %s.", resource)
	if hint != "" {
		msg += "\n\n  " + hint
	}
	return msg
}

// prereqErrorContext runs prerequisite checks and returns a formatted message
// if any tools are missing. Returns empty string if all prerequisites pass.
func prereqErrorContext() string {
	errs := prereq.Check()
	if len(errs) == 0 {
		return ""
	}
	return "\n\n" + prereq.FormatErrors(errs)
}
