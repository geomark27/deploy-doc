package atlassian

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Identifiers specific to one Jira configuration. They are NOT portable: the
// project key, the workflow status ids and the custom field ids all belong to
// the instance this command was written for. They are grouped and named here
// so they are findable and changeable in one place instead of being spread
// across the JQL strings below.
//
// They stay in code deliberately: the qa command is single-tenant by design
// (one project, one QA user), and custom field ids cannot be expressed
// portably. See docs/security/patrones-seguros.md (P-008).
const (
	qaJiraProject = "APP"

	// Finalizada (10001), Testing (10002), En Revisión (10003),
	// Pase a Producción (10004).
	qaStatusIDs = "10001, 10002, 10003, 10004"

	fieldNovedades  = "customfield_10498" // Contador de novedades
	fieldReprocesos = "customfield_10134" // Conteo de reprocesos
	fieldDevStatus  = "customfield_10000" // Development — estado del PR
)

// QAIssue holds the evaluation data for one task in the QA consolidated report.
type QAIssue struct {
	Key             string
	URL             string
	Summary         string
	HasCodingErrors bool   // Contador de novedades (fieldNovedades) > 0
	HasDevReturns   bool   // Conteo de reprocesos (fieldReprocesos) > 0
	HasDeployDoc    bool   // has a remote link titled "Documento de Despliegue..."
	PRMerged        bool   // PR state = MERGED (fieldDevStatus)
	Observations    string // text after "::" in comments matching "Novedad::"
	ReviewTaskKey   string // QA review task key (e.g. APP-2160)
	ReviewTaskURL   string // QA review task URL

	// DeployDocUnknown is set when the remote-link lookup failed, so the
	// report can say "unverified" instead of claiming the document is
	// missing. This table is published as QA evidence: a network or
	// permission error must never be rendered as a failed check.
	DeployDocUnknown bool
}

type jiraSearchResponse struct {
	Issues []struct {
		Key    string         `json:"key"`
		Fields map[string]any `json:"fields"`
	} `json:"issues"`
}

// GetQATasksForReview returns sprint dev tasks that have reached or passed through QA for the given module.
// Excludes QA review tasks (summary starts with "Revisión de Tarea").
// Includes Testing (10002), En Revisión (10003), Finalizada (10001) and Pase a Producción (10004).
func (c *Client) GetQATasksForReview(sprintName, module string) ([]QAIssue, error) {
	jql := fmt.Sprintf(
		`project = %s AND sprint = %s AND status in (%s) AND component = %s ORDER BY key ASC`,
		qaJiraProject, quoteLiteral(sprintName), qaStatusIDs, quoteLiteral(module),
	)
	all, err := c.searchQAIssues(jql)
	if err != nil {
		return nil, err
	}
	dev := make([]QAIssue, 0, len(all))
	for _, t := range all {
		lower := strings.ToLower(t.Summary)
		if !strings.HasPrefix(lower, "revisión de tarea") &&
			!strings.HasPrefix(lower, "revision de tarea") &&
			!strings.HasPrefix(lower, "qa") {
			dev = append(dev, t)
		}
	}
	return dev, nil
}

// GetQATasksAsAssignee returns sprint tasks assigned to the given email (or currentUser() if empty).
// Resolves the email to an accountId first since Jira Cloud JQL requires accountId, not email.
func (c *Client) GetQATasksAsAssignee(sprintName, qaEmail string) ([]QAIssue, error) {
	assignee := "currentUser()"
	if qaEmail != "" {
		accountID, err := c.resolveAccountID(qaEmail)
		if err == nil && accountID != "" {
			assignee = quoteLiteral(accountID)
		}
	}
	jql := fmt.Sprintf(
		`project = %s AND sprint = %s AND assignee = %s ORDER BY key ASC`,
		qaJiraProject, quoteLiteral(sprintName), assignee,
	)
	return c.searchQAIssues(jql)
}

// BuildReviewMap builds a map of devTaskKey → QAIssue from a list of QA review tasks.
// QA review tasks follow the pattern "Revisión de Tarea - APP-XXXX".
func BuildReviewMap(qaTasks []QAIssue) map[string]QAIssue {
	m := make(map[string]QAIssue, len(qaTasks))
	for _, t := range qaTasks {
		if key := parseDevTaskKey(t.Summary); key != "" {
			m[key] = t
		}
	}
	return m
}

// issueKeyRe matches a Jira issue key: a project key followed by a number.
var issueKeyRe = regexp.MustCompile(`[A-Z][A-Z0-9]*-\d+`)

// parseDevTaskKey extracts the dev task key from a QA review task summary,
// e.g. "Revisión de Tarea - APP-1257" → "APP-1257".
//
// The first key in the summary wins: the descriptive prefix never contains one,
// so the first match is the task under review even when a trailing note
// mentions another key. Splitting on "-" instead — as this used to — broke on
// any summary with a suffix ("… - APP-1257 - ajuste" yielded "1257 - ajuste")
// and on an em dash.
func parseDevTaskKey(summary string) string {
	return issueKeyRe.FindString(strings.ToUpper(summary))
}

// resolveAccountID looks up the Jira accountId for a given email address.
func (c *Client) resolveAccountID(email string) (string, error) {
	body, err := c.Get(fmt.Sprintf("/rest/api/3/user/search?query=%s", strings.ReplaceAll(email, "@", "%40")))
	if err != nil {
		return "", err
	}
	var users []struct {
		AccountID string `json:"accountId"`
	}
	if err := json.Unmarshal(body, &users); err != nil || len(users) == 0 {
		return "", fmt.Errorf("usuario no encontrado: %s", email)
	}
	return users[0].AccountID, nil
}

func (c *Client) searchQAIssues(jql string) ([]QAIssue, error) {
	payload := map[string]any{
		"jql":        jql,
		"fields":     []string{"key", "summary", fieldNovedades, fieldReprocesos, fieldDevStatus},
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
		if v, ok := raw.Fields["summary"]; ok && v != nil {
			if s, ok := v.(string); ok {
				qi.Summary = s
			}
		}
		if v, ok := raw.Fields[fieldNovedades]; ok && v != nil {
			if n, ok := v.(float64); ok && n > 0 {
				qi.HasCodingErrors = true
			}
		}
		if v, ok := raw.Fields[fieldReprocesos]; ok && v != nil {
			if n, ok := v.(float64); ok && n > 0 {
				qi.HasDevReturns = true
			}
		}
		if v, ok := raw.Fields[fieldDevStatus]; ok && v != nil {
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
// Returns an empty string and a nil error when the issue simply has none; an
// error means the comments could not be read and the result says nothing about
// whether observations exist.
func (c *Client) GetNovedadComment(issueKey string) (string, error) {
	body, err := c.Get(fmt.Sprintf("/rest/api/3/issue/%s/comment?orderBy=-created&maxResults=50", issueKey))
	if err != nil {
		return "", fmt.Errorf("error leyendo comentarios de %s: %w", issueKey, err)
	}

	var resp struct {
		Comments []struct {
			Body map[string]any `json:"body"`
		} `json:"comments"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("error parseando comentarios de %s: %w", issueKey, err)
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

// businessDaysAgo returns the date N business days before from, skipping weekends.
func businessDaysAgo(n int, from time.Time) time.Time {
	d := from
	for count := 0; count < n; {
		d = d.AddDate(0, 0, -1)
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			count++
		}
	}
	return d
}

// KanbanWindow returns the start date (10 business days ago) and a display label like "08/05 al 22/05/2026".
func KanbanWindow() (since time.Time, period string) {
	now := time.Now()
	since = businessDaysAgo(10, now)
	period = fmt.Sprintf("%s al %s", since.Format("02/01"), now.Format("02/01/2006"))
	return
}

// GetQATasksForReviewKanban returns all Kanban dev tasks (any component) that
// transitioned to Testing status on or after sinceDate (format YYYY-MM-DD).
func (c *Client) GetQATasksForReviewKanban(sinceDate string) ([]QAIssue, error) {
	jql := fmt.Sprintf(
		`project = APP AND status in (10001, 10002, 10003, 10004) AND status CHANGED TO "Testing" AFTER "%s" ORDER BY key ASC`,
		sinceDate,
	)
	all, err := c.searchQAIssues(jql)
	if err != nil {
		return nil, err
	}
	dev := make([]QAIssue, 0, len(all))
	for _, t := range all {
		lower := strings.ToLower(t.Summary)
		if !strings.HasPrefix(lower, "revisión de tarea") &&
			!strings.HasPrefix(lower, "revision de tarea") &&
			!strings.HasPrefix(lower, "qa") {
			dev = append(dev, t)
		}
	}
	return dev, nil
}

// GetQATasksAsAssigneeKanban returns tasks assigned to the QA user updated on or after sinceDate.
func (c *Client) GetQATasksAsAssigneeKanban(sinceDate, qaEmail string) ([]QAIssue, error) {
	assignee := "currentUser()"
	if qaEmail != "" {
		accountID, err := c.resolveAccountID(qaEmail)
		if err == nil && accountID != "" {
			assignee = fmt.Sprintf("%q", accountID)
		}
	}
	jql := fmt.Sprintf(
		`project = APP AND assignee = %s AND updated >= "%s" ORDER BY key ASC`,
		assignee, sinceDate,
	)
	return c.searchQAIssues(jql)
}

// HasDeployDocLink returns true if the issue has a remote link titled
// "Documento de Despliegue...".
//
// A false with a non-nil error means "could not check", not "no document":
// the caller must distinguish them, because the answer lands in a QA report
// published as evidence. Returning false on error — as this used to — marked
// compliant tasks as missing their document whenever the API hiccuped.
func (c *Client) HasDeployDocLink(issueKey string) (bool, error) {
	body, err := c.Get(fmt.Sprintf("/rest/api/3/issue/%s/remotelink", issueKey))
	if err != nil {
		return false, fmt.Errorf("error leyendo enlaces remotos de %s: %w", issueKey, err)
	}

	var links []struct {
		Object struct {
			Title string `json:"title"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &links); err != nil {
		return false, fmt.Errorf("error parseando enlaces remotos de %s: %w", issueKey, err)
	}

	for _, l := range links {
		if strings.HasPrefix(l.Object.Title, "Documento de Despliegue") {
			return true, nil
		}
	}
	return false, nil
}
