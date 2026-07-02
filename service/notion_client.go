package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NotionClient is a minimal HTTP client for the Notion API.
type NotionClient struct {
	Token      string
	HTTPClient *http.Client
}

// NotionPage represents a simplified Notion page object.
type NotionPage struct {
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
}

// NewNotionClient creates a new Notion API client.
func NewNotionClient(token string) *NotionClient {
	return &NotionClient{
		Token: token,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// QueryDatabaseAll paginates through a Notion database and returns all pages.
func (c *NotionClient) QueryDatabaseAll(databaseID string) ([]NotionPage, error) {
	var allPages []NotionPage
	var nextCursor string
	for {
		pages, cursor, err := c.queryDatabasePage(databaseID, nextCursor)
		if err != nil {
			return nil, err
		}
		allPages = append(allPages, pages...)
		if cursor == "" {
			break
		}
		nextCursor = cursor
	}
	return allPages, nil
}

func (c *NotionClient) queryDatabasePage(databaseID, startCursor string) ([]NotionPage, string, error) {
	url := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", databaseID)
	bodyMap := map[string]interface{}{}
	if startCursor != "" {
		bodyMap["start_cursor"] = startCursor
	}
	bodyMap["page_size"] = 100

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("notion API returned %d: %s", resp.StatusCode, string(respBytes))
	}

	var result struct {
		Results    []NotionPage `json:"results"`
		NextCursor string       `json:"next_cursor"`
		HasMore    bool         `json:"has_more"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, "", err
	}
	return result.Results, result.NextCursor, nil
}
