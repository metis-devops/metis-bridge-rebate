build:
	rm -rf dist && mkdir dist
	go build -o ./dist .

goabigen:
	mkdir -p internal/goabi
	abigen --abi ./abis/ERC20.json -pkg goabi --type ERC20 --out internal/goabi/ERC20.go
	abigen --abi ./abis/L1StandardBridge.json -pkg goabi --type L1StandardBridge --out internal/goabi/L1StandardBridge.go
