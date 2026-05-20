package atlassian

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// PageContent holds a Confluence page with its full content and metadata.
type PageContent struct {
	ID          string
	Title       string
	WebURL      string
	SpaceName   string
	SpaceKey    string
	Version     int
	CreatedDate string
	UpdatedDate string
	CreatedBy   string
	UpdatedBy   string
	StorageBody string
}

type confluenceV1PageResp struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Space struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"space"`
	Version struct {
		Number int    `json:"number"`
		When   string `json:"when"`
		By     struct {
			DisplayName string `json:"displayName"`
		} `json:"by"`
	} `json:"version"`
	History struct {
		CreatedDate string `json:"createdDate"`
		CreatedBy   struct {
			DisplayName string `json:"displayName"`
		} `json:"createdBy"`
	} `json:"history"`
	Body struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
	Links struct {
		WebUI string `json:"webui"`
	} `json:"_links"`
}

// GetPageWithContent fetches a Confluence page including its full body and metadata.
func (c *Client) GetPageWithContent(pageID string) (*PageContent, error) {
	path := fmt.Sprintf("/wiki/rest/api/content/%s?expand=body.storage,version,space,history", pageID)

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo contenido de página %s: %w", pageID, err)
	}

	var resp confluenceV1PageResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando página: %w", err)
	}

	parseDate := func(s string) string {
		if s == "" {
			return ""
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t.Format("2006-01-02")
		}
		if len(s) >= 10 {
			return s[:10]
		}
		return s
	}

	return &PageContent{
		ID:          resp.ID,
		Title:       resp.Title,
		WebURL:      c.BaseURL + "/wiki" + resp.Links.WebUI,
		SpaceName:   resp.Space.Name,
		SpaceKey:    resp.Space.Key,
		Version:     resp.Version.Number,
		CreatedDate: parseDate(resp.History.CreatedDate),
		UpdatedDate: parseDate(resp.Version.When),
		CreatedBy:   resp.History.CreatedBy.DisplayName,
		UpdatedBy:   resp.Version.By.DisplayName,
		StorageBody: resp.Body.Storage.Value,
	}, nil
}

// FindPagesByText searches Confluence for pages whose content mentions the query.
// spaceKey restricts the search to a specific space; empty string searches all spaces.
func (c *Client) FindPagesByText(query, spaceKey string) ([]Page, error) {
	cql := fmt.Sprintf(`text ~ "%s"`, query)
	if spaceKey != "" {
		cql += fmt.Sprintf(` AND space = "%s"`, spaceKey)
	}
	cql += " ORDER BY lastModified DESC"

	// Confluence CQL parser expects %20 for spaces, not +
	encCQL := strings.ReplaceAll(url.QueryEscape(cql), "+", "%20")
	path := fmt.Sprintf("/wiki/rest/api/search?cql=%s&limit=10", encCQL)

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error buscando páginas: %w", err)
	}

	var result searchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parseando resultados: %w", err)
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

// BuildIssueTxt formats a PageContent as a structured plain-text document.
func BuildIssueTxt(issueKey string, page *PageContent) string {
	var sb strings.Builder

	bar80 := strings.Repeat("=", 80)
	bar40 := strings.Repeat("-", 80)

	sb.WriteString(bar80 + "\n")
	sb.WriteString(issueKey + " - " + page.Title + "\n")
	sb.WriteString(bar80 + "\n")
	sb.WriteString("Fuente: Confluence - " + page.SpaceName + "\n")
	sb.WriteString("URL: " + page.WebURL + "\n")
	if page.CreatedDate != "" {
		line := "Creado: " + page.CreatedDate
		if page.Version > 0 {
			line += fmt.Sprintf(" | Versión: %d", page.Version)
		}
		sb.WriteString(line + "\n")
	}
	if page.UpdatedBy != "" {
		sb.WriteString("Última edición: " + page.UpdatedBy + "\n")
	}
	sb.WriteString(bar40 + "\n")

	if page.StorageBody != "" {
		sb.WriteString("\n")
		sb.WriteString(StorageToText(page.StorageBody))
	}

	sb.WriteString("\n" + bar80 + "\n")
	sb.WriteString("FIN DEL DOCUMENTO\n")
	sb.WriteString(bar80 + "\n")

	return sb.String()
}
