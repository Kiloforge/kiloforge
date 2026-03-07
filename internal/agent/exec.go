package agent

import "os/exec"

// newCommand wraps exec.CommandContext for testability.
var newCommand = exec.CommandContext
