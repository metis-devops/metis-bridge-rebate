package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metis-devops/metis-bridge-rebate/internal/goabi"
	"github.com/metis-devops/metis-bridge-rebate/internal/repository"
	"github.com/metis-devops/metis-bridge-rebate/internal/services"
	"github.com/metis-devops/metis-bridge-rebate/internal/services/policy"
	"github.com/metis-devops/metis-bridge-rebate/internal/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func main() {
	var (
		MaxDripUSD    float64
		MetisEndpoint string
		EtherEndpoint string
		MysqlEndpoint string

		ConfirmationNumber uint64
		RangeSyncNumber    uint64
		StartFromHeight    uint64

		KeyPath    string
		OpenFaucet bool

		// for the common usage case
		DripAmount      float64
		DripHeight      uint64
		ReservedBalance float64
	)

	flag.Float64Var(&MaxDripUSD, "maxdrip", 250, "max drip usd value")
	flag.StringVar(&EtherEndpoint, "l1rpc", "https://goerli.infura.io/v3/", "l1 rpc endpoint")
	flag.StringVar(&MetisEndpoint, "l2rpc", "https://goerli.gateway.metisdevops.link", "l2 rpc endpoint")
	flag.StringVar(&MysqlEndpoint, "mysql", "root:passwd@tcp(127.0.0.1:3306)/metis?parseTime=true", "mysql endpoint")
	flag.Uint64Var(&ConfirmationNumber, "confirm", 32, "confirmation number for a new despoit")
	flag.Uint64Var(&RangeSyncNumber, "range", 50000, "range sync at once")
	//  13627429 mainnet 7501326 goerli
	flag.Uint64Var(&StartFromHeight, "start-block", 7501326, "initial from height")

	flag.Uint64Var(&DripHeight, "height", 7945105, "height to transfer a drip")
	flag.Float64Var(&DripAmount, "drip", 0.01, "metis amount to transfer")
	flag.Float64Var(&ReservedBalance, "reserved", 1, "reserved balance")

	flag.StringVar(&KeyPath, "key", "key.txt", "private key path")
	flag.BoolVar(&OpenFaucet, "faucet", false, "open faucet or not")
	flag.Parse()

	if RangeSyncNumber < 1000 {
		RangeSyncNumber = 1000
	}
	if DripAmount <= 0 {
		DripAmount = 0.01
	}

	// connect to db
	db, err := repository.Connect(MysqlEndpoint)
	if err != nil {
		logrus.Fatalf("unable to connect to mysql: %s", err)
	}
	defer db.Close()

	basectx, cancel := context.WithCancel(context.Background())
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		for range stop {
			cancel()
		}
	}()

	l1rpc, err := ethclient.Dial(EtherEndpoint)
	if err != nil {
		logrus.Fatalf("unable to connect to l1 rpc: %s", err)
	}
	defer l1rpc.Close()

	l1ChainId, err := l1rpc.ChainID(context.Background())
	if err != nil {
		logrus.Fatalf("unable to get chain id: %s", err)
	}

	if id := l1ChainId.Uint64(); id != utils.EthMainnnetChainId && id != utils.EthGoerliChainid {
		logrus.Fatalf("wrong layer1 network: %d", id)
	}

	eg, egctx := errgroup.WithContext(basectx)

	// Data syncing service
	eg.Go(func() error {
		bridgeAdddress := utils.MetisL1BridgeAddress(l1ChainId.Uint64())
		bridge, err := goabi.NewL1StandardBridge(bridgeAdddress, l1rpc)
		if err != nil {
			return fmt.Errorf("unable to create bridge instance: %s", err)
		}

		syncer := &services.DataSync{
			EtherClient:        l1rpc,
			Repositroy:         repository.NewMetis(db),
			Bridge:             bridge,
			RangeSync:          RangeSyncNumber,
			ConfirmationNumber: ConfirmationNumber,
			DripHeight:         DripHeight,
		}
		if err := syncer.Prefight(egctx, StartFromHeight); err != nil {
			return err
		}
		logrus.Info("fetching new events")
		timer := time.NewTimer(0)
		for {
			select {
			case <-egctx.Done():
				return nil
			case <-timer.C:
				syncer.Run(basectx)
				timer.Reset(time.Minute * 5)
			}
		}
	})

	// Faucet service
	eg.Go(func() error {
		if !OpenFaucet {
			return nil
		}

		// connect to l2rpc
		l2rpc, err := ethclient.Dial(MetisEndpoint)
		if err != nil {
			return fmt.Errorf("unable to connect to l2 rpc: %s", err)
		}
		defer l2rpc.Close()

		l2ChainId, err := l2rpc.ChainID(context.Background())
		if err != nil {
			return fmt.Errorf("unable to get chain id: %s", err)
		}
		if id := l2ChainId.Uint64(); id != utils.MetisAndromedaChainId && id != utils.MetisGoerliChainId {
			return fmt.Errorf("wrong layer2 network: %d", id)
		}

		prvkey, wallet, err := utils.ReadPrvkey(KeyPath)
		if err != nil {
			return fmt.Errorf("unable to read pricate key: %s", err)
		}
		logrus.Infof("Current wallet address is %s", wallet)

		faucet := &services.Faucet{
			EthClient:   l1rpc,
			MetisClient: l2rpc,
			Repositroy:  repository.NewMetis(db),
			Uniswap:     utils.NewUniswap(),
			// uniswap doesn't have goerli subgraph
			MetisL1Contract: utils.MetisL1TokenAddress(utils.EthMainnnetChainId),
			Prvkey:          prvkey,
			Account:         wallet,
			Eip155Signer:    types.NewEIP155Signer(l2ChainId),
			DefaultDrip:     utils.ToWei(DripAmount),
			MaxDripUSD:      MaxDripUSD,
			ReservedBalance: ReservedBalance,
			DripPolicies: []*policy.Drip{
				{
					Name:         "Default",
					MatchAll:     true,
					MatchToken:   nil,
					CheckIfFirst: true,
					CheckIfNoGas: true,
					MinUSDEqual:  200,
					StartTime:    time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC),
					EndTime:      time.Date(2099, time.December, 31, 0, 0, 0, 0, time.UTC),
					RebateType:   policy.DefaultRebateType,
				},
			},
		}
		if err := faucet.Initial(egctx); err != nil {
			return err
		}

		timer := time.NewTimer(0)
		for {
			select {
			case <-egctx.Done():
				return nil
			case <-timer.C:
				faucet.SendDrips(egctx)
				select {
				case <-egctx.Done():
					return nil
				case <-time.After(time.Second * 5):
					faucet.CheckDrips(egctx)
				}
				timer.Reset(time.Minute)
			}
		}
	})

	if err := eg.Wait(); err != nil {
		logrus.Fatal(err)
	}
}
