package utils

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func GetBridgeTokens(ctx context.Context) (map[string]string, error) {
	const endpoint = "https://raw.githubusercontent.com/MetisProtocol/metis-bridge-resources/master/metis-l2-token-list.json"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	var raw map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	res := make(map[string]string, len(raw))
	for key, value := range raw {
		res[strings.ToLower(value)] = key
	}
	return res, nil
}
