package utils

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	EtherL1Address = "0x0000000000000000000000000000000000000000"
	WETH9Adddress  = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	EtherL2Address = "0x420000000000000000000000000000000000000a"
	MetisL2Address = "0xdeaddeaddeaddeaddeaddeaddeaddeaddead0000"
)

func IsStableL1Token(u string) bool {
	const (
		USDTAddress = "0xdac17f958d2ee523a2206206994597c13d831ec7"
		USDCAddress = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
		DaiAddress  = "0x6b175474e89094c44da98b954eedeac495271d0f"
		BUSDAddress = "0x4fabb145d64652a948d72533023f6e7a623c7c53"
	)
	switch strings.ToLower(u) {
	case USDTAddress, USDCAddress, DaiAddress, BUSDAddress:
		return true
	default:
		return false
	}
}

func MetisL1BridgeAddress(chainId uint64) common.Address {
	switch chainId {
	case EthMainnnetChainId:
		// https://etherscan.io/address/0x3980c9ed79d2c191a89e02fa3529c60ed6e9c04b
		return common.HexToAddress("0x3980c9ed79d2c191A89E02Fa3529C60eD6e9c04b")
	case EthGoerliChainid:
		// https://goerli.etherscan.io/address/0xcf7257a86a5dbba34babcd2680f209eb9a05b2d2
		return common.HexToAddress("0xCF7257A86A5dBba34bAbcd2680f209eb9a05b2d2")
	default:
		panic(fmt.Sprintf("invalid Ethereum chain id %v", chainId))
	}
}

func MetisL1TokenAddress(chainId uint64) string {
	switch chainId {
	case EthMainnnetChainId:
		return "0x9e32b13ce7f2e80a01932b42553652e053d6ed8e"
	case EthGoerliChainid:
		return "0x114f836434a9aa9ca584491e7965b16565bf5d7b"
	default:
		panic(fmt.Sprintf("invalid Ethereum chain id %v", chainId))
	}
}

func ReadPrvkey(keyPath string) (*ecdsa.PrivateKey, common.Address, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, common.Address{}, err
	}

	rawkey := strings.TrimSpace(string(data))
	prvkey, err := crypto.HexToECDSA(strings.TrimPrefix(rawkey, "0x"))
	if err != nil {
		return nil, common.Address{}, err
	}
	address := crypto.PubkeyToAddress(*prvkey.Public().(*ecdsa.PublicKey))
	return prvkey, address, nil
}

const uniswapQuery = `
query tokenHourDatas($startTime: Int!, $address: Bytes!, $weth: Bytes!) {
	ethPrice: tokenHourDatas(
	  first: 1
	  skip: 0
	  where: {token: $weth, periodStartUnix_gt: $startTime}
	  orderBy: periodStartUnix
	  orderDirection: asc
	) {
	  high
	  low
	  open
	  close
	}
	tokens(where: {id: $address}) {
	  name
	  symbol
	  decimals
	  derivedETH
	}
  }
`
