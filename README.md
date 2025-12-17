# Metis Rebate Service For Bridge Users

0.01 Metis Token will be airdropped to first time users!

# Start the service

```console
$ docker compose up -d
```

Before that you need to configure correct parameters for faucet service in the docker-compose.yml :

```
Usage of metis-bridge-rebate:
  -confirm uint
        confirmation number for a new despoit (default 32)
  -drip float
        metis amount to transfer (default 0.01)
  -faucet
        open faucet or not
  -height uint
        height to transfer a drip (default 7945105)
  -key string
        private key path (default "key.txt")
  -l1rpc string
        l1 rpc endpoint (default "https://goerli.infura.io/v3/")
  -l2rpc string
        l2 rpc endpoint (default "https://goerli.gateway.metisdevops.link")
  -maxdrip float
        max drip usd value (default 250)
  -mysql string
        mysql endpoint (default "root:passwd@tcp(127.0.0.1:3306)/metis?parseTime=true")
  -range uint
        range sync at once (default 50000)
  -reserved float
        reserved balance (default 1)
  -start-block uint
        initial from height (default 7501326)
  -uniswap-v3-apikey string
        the uniswap v3 graphql api key
  -uniswap-v3-graphql string
        the uniswap v3 graphql endpoint (default "https://gateway.thegraph.com/api/subgraphs/id/5zvR82QoaXYFyDEKLZ9t6v9adgnptxYpKpSbxtgVENFV")
```
