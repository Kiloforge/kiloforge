package compose

import (
	"bytes"
	"fmt"
	"text/template"
)

// ComposeConfig holds the values used to render the docker-compose.yml template.
type ComposeConfig struct {
	OrchestratorPort int
	DataDir          string
}

const composeTemplate = `services: {}
`

// GenerateComposeFile renders the docker-compose.yml template with the given config.
func GenerateComposeFile(cfg ComposeConfig) ([]byte, error) {
	tmpl, err := template.New("compose").Parse(composeTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse compose template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return nil, fmt.Errorf("render compose template: %w", err)
	}

	return buf.Bytes(), nil
}
