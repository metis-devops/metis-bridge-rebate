package graphql

import "encoding/json"

type Response struct {
	Data   json.RawMessage `json:"data"`
	Errors json.RawMessage `json:"errors,omitempty"`
}

type Request struct {
	Query string                 `json:"query"`
	Vars  map[string]interface{} `json:"variables,omitempty"`
}
