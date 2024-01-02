package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	endpoint string
	http     *http.Client
}

func New(endpoint string, client ...*http.Client) *Client {
	if len(client) > 0 {
		return &Client{endpoint: endpoint, http: client[0]}
	}
	return &Client{endpoint: endpoint, http: http.DefaultClient}
}

func (c Client) Call(result interface{}, query string, vars map[string]interface{}) error {
	return c.CallContext(context.Background(), result, query, vars)
}

func (c Client) CallContext(ctx context.Context, result interface{}, query string, vars map[string]interface{}) error {
	reqdata := bytes.NewBuffer(nil)
	if err := json.NewEncoder(reqdata).Encode(Request{Query: query, Vars: vars}); err != nil {
		return fmt.Errorf("graphql: encode req: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, reqdata)
	if err != nil {
		return fmt.Errorf("graphql: create req: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("graphql: do req: %w", err)
	}
	defer resp.Body.Close()

	var returns Response
	if err := json.NewDecoder(resp.Body).Decode(&returns); err != nil {
		return fmt.Errorf("graphql: decode response: %w", err)
	}

	if returns.Errors != nil {
		return fmt.Errorf("graphql: %s", returns.Errors)
	}

	if err := json.Unmarshal(returns.Data, result); err != nil {
		return fmt.Errorf("graphql: decode result: %w", err)
	}
	return nil
}
