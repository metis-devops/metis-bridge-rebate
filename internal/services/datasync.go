package services

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ericlee42/metis-bridge-rebate/internal/goabi"
	"github.com/ericlee42/metis-bridge-rebate/internal/repository"
	"github.com/ericlee42/metis-bridge-rebate/internal/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/islishude/bigint"
	"github.com/sirupsen/logrus"
)

type DataSync struct {
	EtherClient        *ethclient.Client
	Bridge             *goabi.L1StandardBridge
	Repositroy         repository.Metis
	ConfirmationNumber uint64
	RangeSync          uint64
	DripHeight         uint64

	height uint64
}

func (s *DataSync) Prefight(basectx context.Context, startFrom uint64) (err error) {
	newctx, cancel := context.WithTimeout(basectx, time.Second*5)
	defer cancel()
	s.height, err = s.Repositroy.InitHeight(newctx, startFrom)
	return
}

func (s *DataSync) Run(basectx context.Context) {
	if err := s.tryToSync(basectx); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		logrus.Errorf("sync fail: %s", err)
	}
}

func (s *DataSync) tryToSync(basectx context.Context) error {
	latestBlock, err := func() (uint64, error) {
		newctx, cancle := context.WithTimeout(basectx, time.Second*10)
		defer cancle()
		block, err := s.EtherClient.BlockNumber(newctx)
		if err != nil {
			return 0, err
		}
		if block > s.ConfirmationNumber {
			return block - s.ConfirmationNumber, nil
		}
		return block, nil
	}()
	if err != nil {
		return err
	}

	for targetHeight := latestBlock; s.height < targetHeight; {
		startHeight, endHeight := s.height, s.height+s.RangeSync
		if endHeight > targetHeight {
			endHeight = targetHeight
		}
		if err := s.syncWithRange(basectx, startHeight, endHeight); err != nil {
			return err
		}
		s.height = endHeight + 1
	}
	return nil
}

func (s *DataSync) syncWithRange(basectx context.Context, startHeight, endHeight uint64) error {
	logrus.Infof("Syncing from %d to %d", startHeight, endHeight)

	newctx, cancel := context.WithTimeout(basectx, time.Minute*10)
	defer cancel()

	formatERC20DepositEvent := func(event *goabi.L1StandardBridgeERC20DepositInitiated) (*repository.Deposit, error) {
		var status = repository.DepositStatusUnprocessed
		l2token := strings.ToLower(event.L2Token.Hex())
		if l2token == utils.MetisL2Address {
			status = repository.DepositStatusIgnore
		}

		if s.DripHeight > event.Raw.BlockNumber {
			status = repository.DepositStatusIgnore
			logrus.Infof("Tx %s is not ready to have a drip[DripHeight %v > TxHeight %v]", event.Raw.TxHash, s.DripHeight, event.Raw.BlockNumber)
		}

		return &repository.Deposit{
			Height:  event.Raw.BlockNumber,
			Txid:    event.Raw.TxHash.Hex(),
			L1Token: strings.ToLower(event.L1Token.Hex()),
			L2Token: l2token,
			From:    strings.ToLower(event.From.Hex()),
			To:      strings.ToLower(event.To.Hex()),
			Amount:  bigint.FromBigInt(event.Amount),
			Status:  status,
		}, nil
	}

	formatEthDespoitEvent := func(event *goabi.L1StandardBridgeETHDepositInitiated) (*repository.Deposit, error) {
		var status = repository.DepositStatusUnprocessed
		if s.DripHeight > event.Raw.BlockNumber {
			status = repository.DepositStatusIgnore
			logrus.Infof("Tx %s is not ready to have a drip[DripHeight %v > TxHeight %v]", event.Raw.TxHash, s.DripHeight, event.Raw.BlockNumber)
		}

		return &repository.Deposit{
			Height:  event.Raw.BlockNumber,
			Txid:    event.Raw.TxHash.Hex(),
			L1Token: strings.ToLower(utils.EtherL1Address),
			L2Token: strings.ToLower(utils.EtherL2Address),
			From:    strings.ToLower(event.From.Hex()),
			To:      strings.ToLower(event.To.Hex()),
			Amount:  bigint.FromBigInt(event.Amount),
			Status:  status,
		}, nil
	}

	header, err := s.EtherClient.HeaderByNumber(newctx, big.NewInt(int64(endHeight)))
	if err != nil {
		return fmt.Errorf("syncWithRange: get tail header: %w", err)
	}
	var tail = &repository.Height{Number: endHeight, Blockhash: header.Hash().String()}

	filterOption := &bind.FilterOpts{Context: newctx, Start: startHeight, End: &endHeight}
	erc20Iter, err := s.Bridge.FilterERC20DepositInitiated(filterOption, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("syncWithRange: filter erc20 deposit event: %w", err)
	}
	defer erc20Iter.Close()

	var deposits []*repository.Deposit
	for erc20Iter.Next() {
		dpt, err := formatERC20DepositEvent(erc20Iter.Event)
		if err != nil {
			return err
		}
		deposits = append(deposits, dpt)
	}
	if err := erc20Iter.Error(); err != nil {
		return fmt.Errorf("syncWithRange: filter erc20 deposit event: %w", err)
	}

	etherInter, err := s.Bridge.FilterETHDepositInitiated(filterOption, nil, nil)
	if err != nil {
		return fmt.Errorf("syncWithRange: filter ether deposit event: %w", err)
	}
	defer etherInter.Close()
	for etherInter.Next() {
		dpt, err := formatEthDespoitEvent(etherInter.Event)
		if err != nil {
			return err
		}
		deposits = append(deposits, dpt)
	}
	if err := etherInter.Error(); err != nil {
		return fmt.Errorf("syncWithRange: filter ether deposit event: %w", err)
	}

	if err := s.Repositroy.SaveSyncedData(newctx, deposits, tail); err != nil {
		return fmt.Errorf("syncWithRange: %w", err)
	}

	logrus.Infof("Done: NewDeposits %d BlockTime %s", len(deposits), time.Unix(int64(header.Time), 0))
	return nil
}
