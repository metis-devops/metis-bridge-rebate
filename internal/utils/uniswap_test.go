package utils

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestUniswap_GetTokenPrice(t *testing.T) {
	const uniswapv3_subgraph_api = "https://thegraph.com/explorer/api/playground/QmTZ8ejXJxRo7vDBS4uwqBeGoxLSWbhaA7oXa1RvxunLy7"

	resp, err := http.Get("https://raw.githubusercontent.com/MetisProtocol/metis-bridge-resources/master/l1-token-list.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	c := NewUniswap(uniswapv3_subgraph_api, "")

	type Info struct {
		/*
					"nativeNetwork": "ethereum",
			        "nativeContractAddress": "0x0000000000000000000000000000000000000000",
			        "denomination": 18,
			        "chainlinkFeedAddress": "0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419",
			        "logo": "https://raw.githubusercontent.com/MetisProtocol/metis-bridge-resources/master/tokens/ETH/logo.png",
			        "coingeckoId": "ethereum"
		*/
		Contract string `json:"nativeContractAddress"`
	}

	var res map[string]Info
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	for coinName, info := range res {
		info, err := c.GetToken(ctx, info.Contract)
		if err != nil {
			t.Errorf("CoinName %s Err %s", coinName, err)
			continue
		}
		t.Logf("Coin: %s USD: %f, ETH %f", coinName, info.ValueInUSD, info.ValueInEther)
	}
}
