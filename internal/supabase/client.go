// Package supabase es un cliente mínimo para la REST API de Supabase
// (PostgREST), usado server-side con la service_role key. No usa un driver
// de Postgres: todo pasa por HTTP contra PostgREST.
package supabase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL    string
	serviceKey string
	http       *http.Client
}

func NewClient(baseURL, serviceKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		http:       &http.Client{},
	}
}

// Get hace un GET a /rest/v1/<table> con los query params dados y decodifica
// el arreglo de resultados en out (debe ser un puntero a slice).
func (c *Client) Get(table string, query url.Values, out any) error {
	u := fmt.Sprintf("%s/rest/v1/%s?%s", c.baseURL, table, query.Encode())
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("supabase GET %s: %d %s", table, resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, out)
}

// Patch actualiza filas de <table> que cumplan query, con los campos de fields.
func (c *Client) Patch(table string, query url.Values, fields map[string]any) error {
	payload, err := json.Marshal(fields)
	if err != nil {
		return err
	}

	u := fmt.Sprintf("%s/rest/v1/%s?%s", c.baseURL, table, query.Encode())
	req, err := http.NewRequest(http.MethodPatch, u, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase PATCH %s: %d %s", table, resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("Accept", "application/json")
}
