package utils

import (
	"context"
	"testing"
)

func TestGetBridgeTokens(t *testing.T) {
	got, err := GetBridgeTokens(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", got)
}
