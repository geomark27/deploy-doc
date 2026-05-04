package atlassian

import (
	"encoding/json"
	"fmt"
	"strings"
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

// JiraUser represents the authenticated user as returned by /rest/api/3/myself.
type JiraUser struct {
	AccountID    string `json:"accountId"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
}

// Whoami returns the user authenticated by the current credentials.
// Useful to detect typos in the configured email — Atlassian Basic Auth
// silently returns empty results when the email/token combo is invalid,
// so we verify against /rest/api/3/myself early.
func (c *Client) Whoami() (*JiraUser, error) {
	body, err := c.Get("/rest/api/3/myself")
	if err != nil {
		return nil, fmt.Errorf("no se pudo verificar credenciales: %w", err)
	}
	var u JiraUser
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("respuesta inválida de /myself: %w", err)
	}
	return &u, nil
}

// VerifyCredentialsMatch calls Whoami and warns to stderr-style output if the
// authenticated email doesn't match the configured one. Returns the user and
// a non-nil error only when the call itself fails (auth/network).
// A mismatch is reported via the warn callback so the caller can decide format.
func (c *Client) VerifyCredentialsMatch(configuredEmail string, warn func(string)) (*JiraUser, error) {
	u, err := c.Whoami()
	if err != nil {
		return nil, err
	}
	if configuredEmail != "" && !strings.EqualFold(u.EmailAddress, configuredEmail) {
		if warn != nil {
			warn(fmt.Sprintf(
				"el email del config (%s) no coincide con la cuenta autenticada (%s). "+
					"Búsquedas pueden devolver resultados vacíos. Revisa ~/.config/gtt/config.yaml o ejecuta: gtt init",
				configuredEmail, u.EmailAddress,
			))
		}
	}
	return u, nil
}

// CreateJiraRemoteLink creates (or updates) a remote link on a Jira issue pointing to a Confluence page.
// Using globalId makes the call idempotent — re-running generate updates the link instead of duplicating it.
func (c *Client) CreateJiraRemoteLink(issueKey, pageID, pageURL, pageTitle string) error {
	payload := map[string]any{
		"globalId":     "confluence-page:" + pageID,
		"relationship": "mentioned in",
		"object": map[string]any{
			"url":   pageURL,
			"title": pageTitle,
		},
	}
	_, err := c.Post(fmt.Sprintf("/rest/api/3/issue/%s/remotelink", issueKey), payload)
	return err
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
