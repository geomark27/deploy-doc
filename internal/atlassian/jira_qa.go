package atlassian

import (
	"encoding/json"
	"fmt"
	"strings"
)

// QAIssue holds the evaluation data for one task in the QA consolidated report.
type QAIssue struct {
	Key             string
	URL             string
	HasCodingErrors bool   // Contador de novedades (customfield_10498) > 0
	HasDevReturns   bool   // Conteo de reprocesos (customfield_10134) > 0
	HasDeployDoc    bool   // has a remote link titled "Documento de Despliegue..."
	PRMerged        bool   // PR state = MERGED (customfield_10000)
	Observations    string // text after "::" in comments matching "Novedad::"
}

type jiraSearchResponse struct {
	Issues []struct {
		Key    string         `json:"key"`
		Fields map[string]any `json:"fields"`
	} `json:"issues"`
}

// GetQATasksForReview returns sprint tasks in Testing (10002) or En Revisión (10003) for the given module.
// Status IDs are used because the JQL name for "Pruebas" is "Testing" — filtering by display name fails.
func (c *Client) GetQATasksForReview(sprintName, module string) ([]QAIssue, error) {
	jql := fmt.Sprintf(
		`project = APP AND sprint = "%s" AND status in (10002, 10003) AND component = "%s" ORDER BY key ASC`,
		sprintName, module,
	)
	return c.searchQAIssues(jql)
}

// GetQATasksAsAssignee returns sprint tasks where the current user is the assignee.
func (c *Client) GetQATasksAsAssignee(sprintName string) ([]QAIssue, error) {
	jql := fmt.Sprintf(
		`project = APP AND sprint = "%s" AND assignee = currentUser() ORDER BY key ASC`,
		sprintName,
	)
	return c.searchQAIssues(jql)
}

func (c *Client) searchQAIssues(jql string) ([]QAIssue, error) {
	payload := map[string]any{
		"jql":        jql,
		"fields":     []string{"key", "summary", "customfield_10498", "customfield_10134", "customfield_10000"},
		"maxResults": 100,
	}

	body, err := c.Post("/rest/api/3/search/jql", payload)
	if err != nil {
		return nil, fmt.Errorf("error buscando issues: %w", err)
	}

	var resp jiraSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	issues := make([]QAIssue, 0, len(resp.Issues))
	for _, raw := range resp.Issues {
		qi := QAIssue{
			Key: raw.Key,
			URL: fmt.Sprintf("%s/browse/%s", c.BaseURL, raw.Key),
		}
		if v, ok := raw.Fields["customfield_10498"]; ok && v != nil {
			if n, ok := v.(float64); ok && n > 0 {
				qi.HasCodingErrors = true
			}
		}
		if v, ok := raw.Fields["customfield_10134"]; ok && v != nil {
			if n, ok := v.(float64); ok && n > 0 {
				qi.HasDevReturns = true
			}
		}
		if v, ok := raw.Fields["customfield_10000"]; ok && v != nil {
			if s, ok := v.(string); ok {
				qi.PRMerged = strings.Contains(s, "state=MERGED")
			}
		}
		issues = append(issues, qi)
	}
	return issues, nil
}

// GetNovedadComment returns the observation text from comments matching "Novedad::texto".
// If multiple such comments exist they are joined with "; ".
// Returns empty string (no error) when none are found.
func (c *Client) GetNovedadComment(issueKey string) (string, error) {
	body, err := c.Get(fmt.Sprintf("/rest/api/3/issue/%s/comment?orderBy=-created&maxResults=50", issueKey))
	if err != nil {
		return "", nil
	}

	var resp struct {
		Comments []struct {
			Body map[string]any `json:"body"`
		} `json:"comments"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", nil
	}

	var parts []string
	for _, comment := range resp.Comments {
		text := adfExtractText(comment.Body)
		if idx := strings.Index(strings.ToLower(text), "novedad::"); idx != -1 {
			observation := strings.TrimSpace(text[idx+len("novedad::"):])
			if observation != "" {
				parts = append(parts, observation)
			}
		}
	}
	return strings.Join(parts, "; "), nil
}

// adfExtractText recursively collects all plain text nodes from an ADF document.
func adfExtractText(node map[string]any) string {
	if node["type"] == "text" {
		if t, ok := node["text"].(string); ok {
			return t
		}
	}
	var sb strings.Builder
	if content, ok := node["content"].([]any); ok {
		for _, child := range content {
			if m, ok := child.(map[string]any); ok {
				sb.WriteString(adfExtractText(m))
			}
		}
	}
	return sb.String()
}

// HasDeployDocLink returns true if the issue has a remote link titled "Documento de Despliegue...".
func (c *Client) HasDeployDocLink(issueKey string) (bool, error) {
	body, err := c.Get(fmt.Sprintf("/rest/api/3/issue/%s/remotelink", issueKey))
	if err != nil {
		return false, nil
	}

	var links []struct {
		Object struct {
			Title string `json:"title"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &links); err != nil {
		return false, nil
	}

	for _, l := range links {
		if strings.HasPrefix(l.Object.Title, "Documento de Despliegue") {
			return true, nil
		}
	}
	return false, nil
}
