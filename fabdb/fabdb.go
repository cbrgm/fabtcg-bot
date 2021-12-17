package fabdb

import (
	"context"
	"fmt"
	"strings"
)

type FabDBClient struct {
	client *Client
}

func NewFabDBClient() *FabDBClient {
	return &FabDBClient{
		client: NewClient(),
	}
}

func (c *FabDBClient) ListCards(ctx context.Context, query string) ([]Card, error) {
	resp, err := c.client.get(ctx, fmt.Sprintf("/cards?per_page=30&keywords=%s&page=1&use-case=browse", query))
	if err != nil {
		return nil, err
	}

	var result FaBDBSearchResponse
	if err := c.client.decodeJSON(resp, &result); err != nil {
		return []Card{}, err
	}

	if len(result.Data) <= 0 {
		return []Card{}, fmt.Errorf("JSON response does not have any card fields")
	}

	return result.Data, nil
}

func (c *FabDBClient) GetCard(ctx context.Context, identifier string) (Card, error) {
	resp, err := c.client.get(ctx, fmt.Sprintf("/cards/%s", strings.ToLower(identifier)))
	if err != nil {
		return Card{}, err
	}

	var result Card
	if err := c.client.decodeJSON(resp, &result); err != nil {
		return Card{}, err
	}

	return result, nil
}
