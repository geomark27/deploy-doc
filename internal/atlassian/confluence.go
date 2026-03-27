package atlassian

import (
	"encoding/json"
	"fmt"
	"net/url"
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
	Version  struct {
		Number int `json:"number"`
	} `json:"version"`
	Links struct {
		WebUI string `json:"webui"`
	} `json:"_links"`
}

// FindLastDeployDoc finds the last deploy document created by the authenticated user
// searching by title pattern "Documento de Despliegue".
func (c *Client) FindLastDeployDoc() ([]Page, error) {
	cql := url.QueryEscape(`title ~ "Documento de Despliegue" AND creator = currentUser() ORDER BY created DESC`)
	path := fmt.Sprintf("/wiki/rest/api/search?cql=%s&limit=5", cql)

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error buscando documentos: %w", err)
	}

	var result searchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	var pages []Page
	for _, r := range result.Results {
		pages = append(pages, Page{
			ID:     r.Content.ID,
			Title:  r.Content.Title,
			WebURL: c.BaseURL + "/wiki" + r.Content.Links.WebUI,
		})
	}

	return pages, nil
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
func (c *Client) FindDeployDocByIssue(issueKey string) (*Page, error) {
	cql := url.QueryEscape(fmt.Sprintf(`title ~ "Documento de Despliegue" AND title ~ "%s" AND creator = currentUser()`, issueKey))
	path := fmt.Sprintf("/wiki/rest/api/search?cql=%s&limit=1", cql)

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
			"message": "Actualizado via deploy-doc",
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

// marshalADF serializes the ADF document to a JSON string.
func marshalADF(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("error serializando ADF: %w", err)
	}
	return string(b), nil
}
