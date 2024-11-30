package utils

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ericlee42/metis-bridge-rebate/internal/graphql"
)

type Uniswap struct {
	client *graphql.Client
}

func NewUniswap() *Uniswap {
	const uniswapv3_subgraph_api = "https://subgraph.satsuma-prod.com/b91d46e5018b/metis/community/uniswap-v3-mainnet/version/0.0.1/api"
	return &Uniswap{graphql.New(uniswapv3_subgraph_api)}
}

var ErrNoTokenInfo = errors.New("no token info result")

type Uniswaper interface {
	GetToken(ctx context.Context, tokenAddress string) (*GetTokenResult, error)
}

type UniswapPrice struct {
	High  float64 `json:"high,string"`
	Low   float64 `json:"low,string"`
	Open  float64 `json:"open,string"`
	Close float64 `json:"close,string"`
}

type UniswapToken struct {
	Name       string  `json:"name"`
	Symbol     string  `json:"symbol"`
	Decimals   int     `json:"decimals,string"`
	DerivedETH float64 `json:"derivedETH,string"`
}

type GetTokenResult struct {
	ValueInEther float64
	ValueInUSD   float64
	Info         UniswapToken
}

func (c Uniswap) GetToken(ctx context.Context, tokenAddress string) (*GetTokenResult, error) {
	newctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	tokenAddress = strings.ToLower(tokenAddress)
	if tokenAddress == EtherL1Address {
		tokenAddress = WETH9Adddress
	}

	var result struct {
		EthPrices []UniswapPrice `json:"ethPrice"`
		TokenInfo []UniswapToken `json:"tokens"`
	}

	vars := map[string]interface{}{
		"address":   tokenAddress,
		"startTime": time.Now().UTC().Add(-time.Hour).Unix(),
		"weth":      WETH9Adddress,
	}
	if err := c.client.CallContext(newctx, &result, uniswapQuery, vars); err != nil {
		return nil, err
	}

	// fmt.Printf("%+v", result)

	if len(result.EthPrices) == 0 {
		return nil, errors.New("no eth price result")
	}

	if len(result.TokenInfo) == 0 {
		return nil, ErrNoTokenInfo
	}

	ethPrice := result.EthPrices[0].Close
	ethValue := result.TokenInfo[0].DerivedETH
	tokenPrice := ethValue * ethPrice
	return &GetTokenResult{ethValue, tokenPrice, result.TokenInfo[0]}, nil
}
