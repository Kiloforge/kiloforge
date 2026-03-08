package compose

import (
	"bytes"
	"fmt"
	"text/template"
)

// ComposeConfig holds the values used to render the docker-compose.yml template.
type ComposeConfig struct {
	GiteaPort int
	OrchestratorPort int
	DataDir   string
}

const composeTemplate = `services:
  gitea:
    image: gitea/gitea:latest
    container_name: conductor-gitea
    restart: unless-stopped
    ports:
      - "{{ .GiteaPort }}:3000"
      - "2222:22"
    volumes:
      - gitea-data:/data
    environment:
      - GITEA__security__INSTALL_LOCK=true
      - GITEA__server__ROOT_URL=http://localhost:{{ .OrchestratorPort }}/
      - GITEA__server__HTTP_PORT=3000
      - GITEA__database__DB_TYPE=sqlite3
      - GITEA__service__DISABLE_REGISTRATION=true
      - GITEA__webhook__ALLOWED_HOST_LIST=*
    extra_hosts:
      - "host.docker.internal:host-gateway"
    healthcheck:
      test: ["CMD", "curl", "-sf", "http://localhost:3000/api/v1/version"]
      interval: 5s
      timeout: 3s
      retries: 12
      start_period: 10s

volumes:
  gitea-data:
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
