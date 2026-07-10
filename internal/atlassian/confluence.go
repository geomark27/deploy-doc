package atlassian

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Page represents a Confluence page (minimal fields we need).
type Page struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ParentID string `json:"parentId"`
	SpaceID  string `json:"spaceId"`
	Version  int
	WebURL   string
}

// SearchResult represents the Confluence search response.
type searchResponse struct {
	Results []struct {
		Content struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Links struct {
				WebUI string `json:"webui"`
			} `json:"_links"`
			Expandable struct {
				Container string `json:"container"`
			} `json:"_expandable"`
		} `json:"content"`
	} `json:"results"`
}

// pageResponse represents a single page response from Confluence API.
type pageResponse struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ParentID string `json:"parentId"`
	SpaceID  string `json:"spaceId"`
	AuthorID string `json:"authorId"`
	Version  struct {
		Number int `json:"number"`
	} `json:"version"`
	Links struct {
		WebUI string `json:"webui"`
	} `json:"_links"`
}

// deployDocSearchLimit is the max number of recent deploy docs shown when
// asking the user where to place a new document.
const deployDocSearchLimit = 10

// deployDocTitlePrefix is the common prefix of every deploy document title.
const deployDocTitlePrefix = "Documento de Despliegue"

// ResolveSpaceID returns the numeric space ID for a given space key via the v2 API.
func (c *Client) ResolveSpaceID(spaceKey string) (string, error) {
	path := fmt.Sprintf("/wiki/api/v2/spaces?keys=%s&limit=1", url.QueryEscape(spaceKey))
	body, err := c.Get(path)
	if err != nil {
		return "", fmt.Errorf("error resolviendo space %q: %w", spaceKey, err)
	}

	var result struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parseando space: %w", err)
	}
	if len(result.Results) == 0 {
		return "", fmt.Errorf("no se encontró el space %q en Confluence", spaceKey)
	}
	return result.Results[0].ID, nil
}

// FindLastDeployDoc finds the most recently created deploy documents in the given
// space using the v2 pages API (direct DB lookup, sorted by creation date). This
// avoids the search-index lag of CQL, which made freshly created docs invisible.
// Results are filtered client-side by the deploy-doc title prefix.
//
// When accountID is non-empty, only documents created by that user are returned
// (used to show "my docs" instead of the whole team's). If the user has no deploy
// docs of their own, it falls back to the team-wide list and reports that via the
// returned fellBack flag so the caller can warn the user.
func (c *Client) FindLastDeployDoc(spaceKey, accountID string) (pages []Page, fellBack bool, err error) {
	spaceID, err := c.ResolveSpaceID(spaceKey)
	if err != nil {
		return nil, false, err
	}

	// Over-fetch since the v2 API can't filter by title prefix server-side;
	// we keep only deploy docs and cap to deployDocSearchLimit afterwards.
	path := fmt.Sprintf("/wiki/api/v2/pages?space-id=%s&sort=-created-date&limit=100", url.QueryEscape(spaceID))

	body, err := c.Get(path)
	if err != nil {
		return nil, false, fmt.Errorf("error buscando documentos: %w", err)
	}

	var result struct {
		Results []pageResponse `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, false, fmt.Errorf("error parseando respuesta: %w", err)
	}

	// Collect all deploy docs, tracking which ones belong to the current user.
	all := make([]Page, 0, deployDocSearchLimit)
	mine := make([]Page, 0, deployDocSearchLimit)
	for _, r := range result.Results {
		if !strings.HasPrefix(r.Title, deployDocTitlePrefix) {
			continue
		}
		p := Page{
			ID:       r.ID,
			Title:    r.Title,
			ParentID: r.ParentID,
			SpaceID:  r.SpaceID,
			Version:  r.Version.Number,
			WebURL:   c.BaseURL + "/wiki" + r.Links.WebUI,
		}
		if len(all) < deployDocSearchLimit {
			all = append(all, p)
		}
		if accountID != "" && r.AuthorID == accountID && len(mine) < deployDocSearchLimit {
			mine = append(mine, p)
		}
	}

	// Prefer the user's own docs; fall back to the team-wide list only when the
	// user has none (e.g. a first-time user who hasn't created any yet).
	if accountID != "" && len(mine) > 0 {
		return mine, false, nil
	}
	if accountID != "" && len(mine) == 0 {
		return all, true, nil
	}
	return all, false, nil
}

// GetPage returns a page by ID including its parentId.
func (c *Client) GetPage(pageID string) (*Page, error) {
	path := fmt.Sprintf("/wiki/api/v2/pages/%s", pageID)

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo página %s: %w", pageID, err)
	}

	var resp pageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando página: %w", err)
	}

	return &Page{
		ID:       resp.ID,
		Title:    resp.Title,
		ParentID: resp.ParentID,
		SpaceID:  resp.SpaceID,
		Version:  resp.Version.Number,
		WebURL:   c.BaseURL + "/wiki" + resp.Links.WebUI,
	}, nil
}

// FindDeployDocByIssue searches for an existing deploy doc matching the given issue key.
// spaceKey restricts the search to a specific Confluence space; empty string searches all spaces.
func (c *Client) FindDeployDocByIssue(issueKey, spaceKey string) (*Page, error) {
	cqlBase := fmt.Sprintf(`title ~ "Documento de Despliegue" AND title ~ "%s"`, issueKey)
	if spaceKey != "" {
		cqlBase += fmt.Sprintf(` AND space = "%s"`, spaceKey)
	}
	path := fmt.Sprintf("/wiki/rest/api/search?cql=%s&limit=1", url.QueryEscape(cqlBase))

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error buscando documento existente: %w", err)
	}

	var result searchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, nil
	}

	r := result.Results[0]
	return &Page{
		ID:     r.Content.ID,
		Title:  r.Content.Title,
		WebURL: c.BaseURL + "/wiki" + r.Content.Links.WebUI,
	}, nil
}

// FindPageByTitle looks up a page by its EXACT title via the v2 pages API.
// Unlike CQL search, this is a direct DB lookup with no indexing delay, so it
// reliably detects a page that was just created. This is the same uniqueness
// condition Confluence enforces on creation, so it catches the "title already
// exists" 400 before we attempt to create. spaceKey restricts the search to a
// specific space; empty string searches all spaces.
func (c *Client) FindPageByTitle(title, spaceKey string) (*Page, error) {
	path := fmt.Sprintf("/wiki/api/v2/pages?title=%s&limit=1", url.QueryEscape(title))
	if spaceKey != "" {
		path += "&space-key=" + url.QueryEscape(spaceKey)
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error buscando documento existente: %w", err)
	}

	var result struct {
		Results []pageResponse `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, nil
	}

	r := result.Results[0]
	return &Page{
		ID:       r.ID,
		Title:    r.Title,
		ParentID: r.ParentID,
		SpaceID:  r.SpaceID,
		Version:  r.Version.Number,
		WebURL:   c.BaseURL + "/wiki" + r.Links.WebUI,
	}, nil
}

// CreatePage creates a new Confluence page under the given parent.
func (c *Client) CreatePage(spaceID, parentID, title string, adfBody map[string]any) (*Page, error) {
	adfStr, err := marshalADF(adfBody)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"spaceId":  spaceID,
		"parentId": parentID,
		"title":    title,
		"status":   "current",
		"body": map[string]any{
			"representation": "atlas_doc_format",
			"value":          adfStr,
		},
	}

	body, err := c.Post("/wiki/api/v2/pages", payload)
	if err != nil {
		return nil, fmt.Errorf("error creando página: %w", err)
	}

	var resp pageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta de creación: %w", err)
	}

	return &Page{
		ID:     resp.ID,
		Title:  resp.Title,
		WebURL: c.BaseURL + "/wiki" + resp.Links.WebUI,
	}, nil
}

// UpdatePage updates an existing Confluence page with new content.
func (c *Client) UpdatePage(pageID, title string, currentVersion int, adfBody map[string]any) (*Page, error) {
	adfStr, err := marshalADF(adfBody)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"id":     pageID,
		"status": "current",
		"title":  title,
		"version": map[string]any{
			"number":  currentVersion + 1,
			"message": "Actualizado via gtt",
		},
		"body": map[string]any{
			"representation": "atlas_doc_format",
			"value":          adfStr,
		},
	}

	body, err := c.Put(fmt.Sprintf("/wiki/api/v2/pages/%s", pageID), payload)
	if err != nil {
		return nil, fmt.Errorf("error actualizando página: %w", err)
	}

	var resp pageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta de actualización: %w", err)
	}

	return &Page{
		ID:     resp.ID,
		Title:  resp.Title,
		WebURL: c.BaseURL + "/wiki" + resp.Links.WebUI,
	}, nil
}

// FindQAPage searches for an existing QA consolidated page for the given module and sprint.
// Uses the v2 pages API (direct DB lookup) instead of CQL search to avoid indexing delays.
// spaceKey restricts the search to a specific space; empty string searches all spaces.
func (c *Client) FindQAPage(module string, sprint int, spaceKey string) (*Page, error) {
	title := fmt.Sprintf("Consolidado de Pruebas QA - %s - Sprint %d", module, sprint)
	path := fmt.Sprintf("/wiki/api/v2/pages?title=%s&limit=1", url.QueryEscape(title))
	if spaceKey != "" {
		path += "&space-key=" + url.QueryEscape(spaceKey)
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error buscando página QA: %w", err)
	}

	var result struct {
		Results []pageResponse `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, nil
	}

	r := result.Results[0]
	return &Page{
		ID:     r.ID,
		Title:  r.Title,
		WebURL: c.BaseURL + "/wiki" + r.Links.WebUI,
	}, nil
}

// FindQAPagesForModule returns recent QA consolidated pages for the given module.
// If module is empty, returns any recent QA consolidated pages (used for Kanban mode).
// spaceKey restricts the search to a specific space; empty string searches all spaces.
func (c *Client) FindQAPagesForModule(module, spaceKey string) ([]Page, error) {
	titleFilter := `"Consolidado de Pruebas QA"`
	if module != "" {
		titleFilter = fmt.Sprintf(`"Consolidado de Pruebas QA - %s"`, module)
	}
	cqlBase := fmt.Sprintf(`title ~ %s ORDER BY created DESC`, titleFilter)
	if spaceKey != "" {
		cqlBase = fmt.Sprintf(`title ~ %s AND space = "%s" ORDER BY created DESC`, titleFilter, spaceKey)
	}
	path := fmt.Sprintf("/wiki/rest/api/search?cql=%s&limit=5", url.QueryEscape(cqlBase))

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error buscando páginas QA de referencia: %w", err)
	}

	var result searchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	pages := make([]Page, 0, len(result.Results))
	for _, r := range result.Results {
		pages = append(pages, Page{
			ID:     r.Content.ID,
			Title:  r.Content.Title,
			WebURL: c.BaseURL + "/wiki" + r.Content.Links.WebUI,
		})
	}
	return pages, nil
}

// FindQAKanbanPage searches for an existing QA Kanban consolidated page by period label.
func (c *Client) FindQAKanbanPage(period, spaceKey string) (*Page, error) {
	title := fmt.Sprintf("Consolidado de Pruebas QA - %s", period)
	path := fmt.Sprintf("/wiki/api/v2/pages?title=%s&limit=1", url.QueryEscape(title))
	if spaceKey != "" {
		path += "&space-key=" + url.QueryEscape(spaceKey)
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error buscando página QA Kanban: %w", err)
	}

	var result struct {
		Results []pageResponse `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, nil
	}

	r := result.Results[0]
	return &Page{
		ID:     r.ID,
		Title:  r.Title,
		WebURL: c.BaseURL + "/wiki" + r.Links.WebUI,
	}, nil
}

// marshalADF serializes the ADF document to a JSON string.
func marshalADF(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("error serializando ADF: %w", err)
	}
	return string(b), nil
}
