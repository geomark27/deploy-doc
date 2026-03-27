package atlassian

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the base HTTP client for Atlassian APIs.
type Client struct {
	BaseURL    string
	AuthHeader string
	HTTP       *http.Client
}

// NewClient creates a new Atlassian API client.
func NewClient(baseURL, email, token string) *Client {
	auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		AuthHeader: "Basic " + auth,
		HTTP:       &http.Client{Timeout: 15 * time.Second},
	}
}

// Get performs a GET request to the given path.
func (c *Client) Get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.AuthHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, apiError(resp.StatusCode, body)
	}

	return body, nil
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, payload any) ([]byte, error) {
	return c.doJSON("POST", path, payload)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(path string, payload any) ([]byte, error) {
	return c.doJSON("PUT", path, payload)
}

// doJSON executes a request with a JSON body and returns the response body.
func (c *Client) doJSON(method, path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, c.BaseURL+path, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.AuthHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, apiError(resp.StatusCode, body)
	}

	return body, nil
}

// apiError returns a human-readable error based on the HTTP status code.
func apiError(statusCode int, body []byte) error {
	switch statusCode {
	case 401:
		return fmt.Errorf("credenciales inválidas (401). Verifica tu token ejecutando: deploy-doc init")
	case 403:
		return fmt.Errorf("sin permisos (403). Tu token no tiene acceso a este recurso")
	case 404:
		return fmt.Errorf("recurso no encontrado (404). Verifica que el issue o página existan")
	case 409:
		return fmt.Errorf("conflicto (409): versión desactualizada o recurso duplicado")
	default:
		return fmt.Errorf("error de API (%d): %s", statusCode, string(body))
	}
}
