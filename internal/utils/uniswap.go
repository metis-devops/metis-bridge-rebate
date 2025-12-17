package utils

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/metis-devops/metis-bridge-rebate/internal/graphql"
)

type Uniswap struct {
	client   *graphql.Client
	cache    map[string]*GetTokenResult
	duration time.Duration // timeout for GetTokenResult cache
}

func NewUniswap(endpoint, apiKey string, duration time.Duration) *Uniswap {
	return &Uniswap{
		client:   graphql.New(endpoint, apiKey),
		cache:    make(map[string]*GetTokenResult),
		duration: duration,
	}
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
	timestamp    time.Time // cache timestamp
}

func (c Uniswap) GetToken(ctx context.Context, tokenAddress string) (*GetTokenResult, error) {
	newctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	tokenAddress = strings.ToLower(tokenAddress)
	if tokenAddress == EtherL1Address {
		tokenAddress = WETH9Adddress
	}

	// check cache first
	if res, ok := c.cache[tokenAddress]; ok {
		if time.Since(res.timestamp) < c.duration {
			return res, nil
		}
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

	if len(result.EthPrices) == 0 {
		return nil, ErrNoTokenInfo
	}

	if len(result.TokenInfo) == 0 {
		return nil, ErrNoTokenInfo
	}

	ethPrice := result.EthPrices[0].Close
	ethValue := result.TokenInfo[0].DerivedETH
	tokenPrice := ethValue * ethPrice
	res := &GetTokenResult{
		ValueInEther: ethValue,
		ValueInUSD:   tokenPrice,
		Info:         result.TokenInfo[0],
		timestamp:    time.Now(),
	}
	c.cache[tokenAddress] = res
	return res, nil
}
