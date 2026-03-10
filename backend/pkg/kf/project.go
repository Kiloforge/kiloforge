package kf

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// QuickLink is a navigation link parsed from quick-links.md.
type QuickLink struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

// StyleGuide represents a code style guide file.
type StyleGuide struct {
	Name    string `json:"name"`    // Filename without extension (e.g., "go")
	Content string `json:"content"` // Full markdown content
}

// ProjectInfo holds all project-level metadata from .agent/kf/.
type ProjectInfo struct {
	Product           string       `json:"product"`                      // product.md content
	ProductGuidelines string       `json:"product_guidelines,omitempty"` // product-guidelines.md content
	TechStack         string       `json:"tech_stack"`                   // tech-stack.md content
	Workflow          string       `json:"workflow,omitempty"`           // workflow.md content
	QuickLinks        []QuickLink  `json:"quick_links"`                  // Parsed quick-links.md
	StyleGuides       []StyleGuide `json:"style_guides,omitempty"`       // code_styleguides/*.md
}

// TrackSummary holds aggregate statistics about track state.
type TrackSummary struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	InProgress int `json:"in_progress"`
	Completed  int `json:"completed"`
	Archived   int `json:"archived"`
}

// readFileContent reads a file and returns its content, or empty string if not found.
func readFileContent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

var quickLinkRe = regexp.MustCompile(`^-\s+\[([^\]]+)\]\(([^)]+)\)`)

// ParseQuickLinks parses quick-links.md content into structured links.
func ParseQuickLinks(content string) []QuickLink {
	var links []QuickLink
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if matches := quickLinkRe.FindStringSubmatch(line); matches != nil {
			links = append(links, QuickLink{
				Label: matches[1],
				Path:  matches[2],
			})
		}
	}
	return links
}

// ReadQuickLinksFile reads and parses quick-links.md from a file path.
func ReadQuickLinksFile(path string) ([]QuickLink, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return ParseQuickLinks(string(data)), nil
}

// ReadStyleGuides reads all .md files from the code_styleguides directory.
func ReadStyleGuides(dir string) ([]StyleGuide, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var guides []StyleGuide
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		guides = append(guides, StyleGuide{
			Name:    strings.TrimSuffix(e.Name(), ".md"),
			Content: string(data),
		})
	}
	return guides, nil
}

// --- Client methods ---

func (c *Client) quickLinksFile() string { return filepath.Join(c.KFDir, "quick-links.md") }
func (c *Client) styleGuidesDir() string { return filepath.Join(c.KFDir, "code_styleguides") }
func (c *Client) productFile() string    { return filepath.Join(c.KFDir, "product.md") }
func (c *Client) guidelinesFile() string { return filepath.Join(c.KFDir, "product-guidelines.md") }
func (c *Client) techStackFile() string  { return filepath.Join(c.KFDir, "tech-stack.md") }
func (c *Client) workflowFile() string   { return filepath.Join(c.KFDir, "workflow.md") }

// GetProjectInfo reads all project metadata files and returns a unified view.
func (c *Client) GetProjectInfo() (*ProjectInfo, error) {
	info := &ProjectInfo{
		Product:           readFileContent(c.productFile()),
		ProductGuidelines: readFileContent(c.guidelinesFile()),
		TechStack:         readFileContent(c.techStackFile()),
		Workflow:          readFileContent(c.workflowFile()),
	}

	if info.Product == "" {
		return nil, fmt.Errorf("product.md not found at %s", c.productFile())
	}

	links, err := ReadQuickLinksFile(c.quickLinksFile())
	if err != nil {
		return nil, err
	}
	info.QuickLinks = links

	guides, err := ReadStyleGuides(c.styleGuidesDir())
	if err != nil {
		return nil, err
	}
	info.StyleGuides = guides

	return info, nil
}

// GetProduct reads the product definition (product.md).
func (c *Client) GetProduct() (string, error) {
	data, err := os.ReadFile(c.productFile())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetProductGuidelines reads the product guidelines (product-guidelines.md).
func (c *Client) GetProductGuidelines() (string, error) {
	data, err := os.ReadFile(c.guidelinesFile())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// GetTechStack reads the tech stack reference (tech-stack.md).
func (c *Client) GetTechStack() (string, error) {
	data, err := os.ReadFile(c.techStackFile())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetWorkflow reads the workflow reference (workflow.md).
func (c *Client) GetWorkflow() (string, error) {
	data, err := os.ReadFile(c.workflowFile())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// GetQuickLinks reads the quick links navigation file.
func (c *Client) GetQuickLinks() ([]QuickLink, error) {
	return ReadQuickLinksFile(c.quickLinksFile())
}

// GetStyleGuides reads all code style guide files.
func (c *Client) GetStyleGuides() ([]StyleGuide, error) {
	return ReadStyleGuides(c.styleGuidesDir())
}

// GetStyleGuide reads a specific style guide by name (e.g., "go").
func (c *Client) GetStyleGuide(name string) (string, error) {
	path := filepath.Join(c.styleGuidesDir(), name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetTrackSummary computes aggregate statistics about all tracks.
func (c *Client) GetTrackSummary() (*TrackSummary, error) {
	entries, err := c.ListTracks()
	if err != nil {
		return nil, err
	}
	s := &TrackSummary{Total: len(entries)}
	for _, e := range entries {
		switch e.Status {
		case StatusPending:
			s.Pending++
		case StatusInProgress:
			s.InProgress++
		case StatusCompleted:
			s.Completed++
		case StatusArchived:
			s.Archived++
		}
	}
	return s, nil
}
