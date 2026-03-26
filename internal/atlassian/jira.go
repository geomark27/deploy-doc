package atlassian

import (
	"encoding/json"
	"fmt"
)

// Issue represents a Jira issue (minimal fields we need).
type Issue struct {
	Key     string
	Summary string
	URL     string
}

// jiraIssueResponse represents the Jira API response for an issue.
type jiraIssueResponse struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
	} `json:"fields"`
}

// GetIssue returns a Jira issue by key (e.g. APP-1999).
func (c *Client) GetIssue(issueKey string) (*Issue, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s?fields=summary", issueKey)

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo issue %s: %w", issueKey, err)
	}

	var resp jiraIssueResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando issue: %w", err)
	}

	return &Issue{
		Key:     resp.Key,
		Summary: resp.Fields.Summary,
		URL:     fmt.Sprintf("%s/browse/%s", c.BaseURL, resp.Key),
	}, nil
}
