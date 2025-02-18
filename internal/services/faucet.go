package services

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metis-devops/metis-bridge-rebate/internal/goabi"
	"github.com/metis-devops/metis-bridge-rebate/internal/repository"
	"github.com/metis-devops/metis-bridge-rebate/internal/services/policy"
	"github.com/metis-devops/metis-bridge-rebate/internal/utils"
	"github.com/sirupsen/logrus"
)

type Faucet struct {
	EthClient       *ethclient.Client
	MetisClient     *ethclient.Client
	Repositroy      repository.Metis
	Uniswap         utils.Uniswaper
	MetisL1Contract string

	Prvkey       *ecdsa.PrivateKey
	Account      common.Address
	Eip155Signer types.Signer
	nonce        uint64

	DefaultDrip     *big.Int
	MaxDripUSD      float64
	ReservedBalance float64
	DripPolicies    []*policy.Drip
}

func (s *Faucet) Initial(basectx context.Context) (err error) {
	var hasDefaultDripPolicy bool
	for _, p := range s.DripPolicies {
		if p.MatchAll {
			hasDefaultDripPolicy = true
			break
		}
	}

	if !hasDefaultDripPolicy {
		return errors.New("no default drip policy")
	}

	if s.DefaultDrip == nil || s.DefaultDrip.Sign() < 1 {
		s.DefaultDrip = big.NewInt(1e16)
	}

	newctx, cancel := context.WithTimeout(basectx, time.Second)
	defer cancel()
	s.nonce, err = s.MetisClient.NonceAt(newctx, s.Account, nil)
	return
}

func (s *Faucet) SendDrips(basectx context.Context) {
	newctx, cancel := context.WithTimeout(basectx, time.Minute*5)
	defer cancel()
	if err := s.checkBalance(basectx); err != nil {
		logrus.Errorf("check balance: %s", err)
		return
	}
	tokens, err := utils.GetBridgeTokens(newctx)
	if err != nil {
		logrus.Errorf("Get supported tokens: %s", err)
		return
	}
	if err := s.tryToSendDrip(newctx, tokens); err != nil {
		logrus.Errorf("failed to transfer drips: %s", err)
	}
}

func (s *Faucet) tryToSendDrip(ctx context.Context, bridgeTokens map[string]string) error {
	recset := make(map[string]bool)
	for item := range s.Repositroy.GetDepositTxStream(ctx, repository.DepositStatusUnprocessed) {
		if item.Error != nil {
			return item.Error
		}

		logrus.Infof("Try to send drip: Txid %s Receiver %s", item.Data.Txid, item.Data.To)
		var policy *policy.Drip
		for _, p := range s.DripPolicies {
			if p.Match(item.Data.CreatedAt, item.Data.L1Token) {
				policy = p
			}
		}

		var shouldTransfer = true
		err := s.shouldTransfer(ctx, policy, item.Data, recset, bridgeTokens)
		if err != nil {
			if v, ok := err.(ErrorNoNeedToTransfer); ok {
				logrus.Infof("Don't need to give a drip: %s", v.msg)
				shouldTransfer = false
			} else {
				return err
			}
		}

		var drip *repository.Drip
		var tx *types.Transaction
		if shouldTransfer {
			dripAmount, err := s.calMetisDrip(ctx, policy, item.Data.Txid)
			if err != nil {
				return err
			}

			tx, err = s.makeDripTx(ctx, item.Data.To, dripAmount)
			if err != nil {
				return err
			}
			rawtx, err := tx.MarshalBinary()
			if err != nil {
				return err
			}

			drip = &repository.Drip{
				Pid:    item.Data.Id,
				Txid:   tx.Hash().String(),
				From:   s.Account.Hex(),
				To:     item.Data.To,
				Amount: utils.ToEther(dripAmount),
				Rawtx:  rawtx,
			}
			recset[item.Data.To] = true
		}
		if err := s.Repositroy.NewDrip(ctx, item.Data, drip); err != nil {
			return err
		}
		if tx != nil && drip != nil {
			s.nonce += 1
			logrus.Infof("Drip: send %f Metis to %s [ Tx %s ]", drip.Amount, drip.To, drip.Txid)
			if err := s.MetisClient.SendTransaction(ctx, tx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Faucet) shouldTransfer(basectx context.Context, pc *policy.Drip, item *repository.Deposit, recset map[string]bool, bridgeTokens map[string]string) (err error) {
	if pc == nil {
		return ErrorNoNeedToTransfer{msg: "No policy found"}
	}

	if _, support := bridgeTokens[item.L2Token]; !support {
		return ErrorNoNeedToTransfer{msg: fmt.Sprintf("%s token is not supported", item.L2Token)}
	}

	if recset[item.To] {
		return ErrorNoNeedToTransfer{msg: "has transfered in current loop"}
	}

	newctx, cancel := context.WithTimeout(basectx, time.Second*10)
	defer cancel()

	if pc.MinUSDEqual > 0 {
		var rate float64 = 1
		if !utils.IsStableL1Token(item.L1Token) {
			tokenInfo, err := s.Uniswap.GetToken(newctx, item.L1Token)
			if err != nil {
				if err == utils.ErrNoTokenInfo {
					return ErrorNoNeedToTransfer{msg: err.Error()}
				}
				return err
			}
			rate = tokenInfo.ValueInUSD
		}

		var decimal uint8 = 18
		if item.L1Token != utils.EtherL1Address {
			l1toten, err := goabi.NewERC20(common.HexToAddress(item.L1Token), s.EthClient)
			if err != nil {
				return err
			}
			decimal, err = l1toten.Decimals(&bind.CallOpts{Context: newctx})
			if err != nil {
				return err
			}
		}

		if amount := item.Amount.Readable(int64(decimal)); rate*amount < pc.MinUSDEqual {
			return ErrorNoNeedToTransfer{msg: fmt.Sprintf("Amount %f < Min %f USD", rate*amount, pc.MinUSDEqual)}
		}
	}

	if pc.CheckIfFirst {
		first, err := s.Repositroy.HasGotDrip(newctx, item.To)
		if err != nil {
			return err
		}
		if !first {
			return ErrorNoNeedToTransfer{msg: "transfered before"}
		}

		// should be a fresh address
		nonce, err := s.MetisClient.NonceAt(newctx, common.HexToAddress(item.To), nil)
		if err != nil {
			return err
		}
		if nonce > 0 {
			return ErrorNoNeedToTransfer{msg: "nonce > 0"}
		}
	}

	if pc.CheckIfNoGas {
		// should not have Metis balance
		balance, err := s.MetisClient.BalanceAt(newctx, common.HexToAddress(item.To), nil)
		if err != nil {
			return err
		}
		if balance.Sign() > 0 {
			return ErrorNoNeedToTransfer{msg: "metis balance > 0"}
		}
	}

	// should be an EOA
	code, err := s.MetisClient.CodeAt(newctx, common.HexToAddress(item.To), nil)
	if err != nil {
		return err
	}
	if len(code) > 0 {
		return ErrorNoNeedToTransfer{msg: "not EOA"}
	}

	return nil
}

func (s *Faucet) makeDripTx(basectx context.Context, toAddr string, amount *big.Int) (*types.Transaction, error) {
	newctx, cancel := context.WithTimeout(basectx, time.Second*5)
	defer cancel()

	gasPrice, err := s.MetisClient.SuggestGasPrice(newctx)
	if err != nil {
		return nil, err
	}

	receiver := common.HexToAddress(toAddr)
	gas, err := s.MetisClient.EstimateGas(newctx,
		ethereum.CallMsg{From: s.Account, To: &receiver, Value: amount})
	if err != nil {
		return nil, err
	}

	rawtx := &types.LegacyTx{
		Nonce:    s.nonce,
		GasPrice: gasPrice,
		Gas:      gas,
		To:       &receiver,
		Value:    amount,
	}
	return types.SignNewTx(s.Prvkey, s.Eip155Signer, rawtx)
}

func (s *Faucet) CheckDrips(basectx context.Context) {
	newctx, cancel := context.WithTimeout(basectx, time.Minute*5)
	defer cancel()
	if err := s.tryToCheckDrip(newctx); err != nil {
		logrus.Errorf("failed to check drips: %s", err)
	}
}

func (s *Faucet) tryToCheckDrip(ctx context.Context) error {
	// check if current balance is less than reserved balance
	balance, err := s.MetisClient.BalanceAt(ctx, s.Account, nil)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	if m := utils.ToEther(balance); m < s.ReservedBalance {
		return fmt.Errorf("current balance %f is less than min reserved %f", m, s.ReservedBalance)
	}

	for item := range s.Repositroy.GetPendingDripsStream(ctx) {
		if item.Error != nil {
			return item.Error
		}
		done, err := s.getTxStatus(ctx, item.Data)
		if err != nil {
			return err
		}
		if !done {
			var tx = new(types.Transaction)
			_ = tx.UnmarshalBinary(item.Data.Rawtx)
			_ = s.MetisClient.SendTransaction(ctx, tx)
			continue
		}
		logrus.Infof("Updating deposit %d status [ Tx %s ]", item.Data.Id, item.Data.Txid)
		if err := s.Repositroy.UpdateDripStatus(ctx, item.Data.Id, repository.DepositStatusDone); err != nil {
			return err
		}
	}
	return nil
}

func (s *Faucet) getTxStatus(ctx context.Context, tx *repository.PendingDrip) (bool, error) {
	newctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	_, err := s.MetisClient.TransactionReceipt(newctx, common.HexToHash(tx.Txid))
	if err != nil {
		if err == ethereum.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Faucet) calMetisDrip(ctx context.Context, pc *policy.Drip, txHash string) (*big.Int, error) {
	if pc.RebateType == policy.DefaultRebateType {
		return s.DefaultDrip, nil
	}

	if pc.RebateType == policy.GasFeeRebateType {
		txid := common.HexToHash(txHash)
		tx, _, err := s.EthClient.TransactionByHash(ctx, txid)
		if err != nil {
			return nil, err
		}

		receipt, err := s.EthClient.TransactionReceipt(ctx, txid)
		if err != nil {
			return nil, err
		}

		gasCost := utils.ToEther(new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(receipt.GasUsed)))
		tokenInfo, err := s.Uniswap.GetToken(ctx, s.MetisL1Contract)
		if err != nil {
			return nil, err
		}

		var amount = gasCost / tokenInfo.ValueInEther
		if amount*tokenInfo.ValueInUSD > s.MaxDripUSD {
			amount = s.MaxDripUSD / tokenInfo.ValueInUSD
		}

		return utils.ToWei(amount), nil
	}

	return nil, fmt.Errorf("not supported rebate type: %v", pc.RebateType)
}

func (s *Faucet) checkBalance(ctx context.Context) error {
	newctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	balance, err := s.MetisClient.BalanceAt(newctx, s.Account, nil)
	if err != nil {
		return err
	}

	if balance.Cmp(big.NewInt(5e17)) < 0 {
		return fmt.Errorf("insufficient Balance: %f < 0.5 Metis", utils.ToEther(balance))
	}
	return nil
}
