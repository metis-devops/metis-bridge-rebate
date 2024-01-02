package repository

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

type DepositTxStream struct {
	Data  *Deposit
	Error error
}

func (m Metis) GetDepositTxStream(ctx context.Context, status DepositStatus) <-chan DepositTxStream {
	var stream = make(chan DepositTxStream, 5)
	const query = "SELECT * FROM `deposits` WHERE `status`=? LIMIT 100;"

	go func() {
		defer close(stream)

		rows, err := m.db.QueryxContext(ctx, query, status)
		if err != nil {
			select {
			case <-ctx.Done():
			case stream <- DepositTxStream{Error: err}:
			}
			return
		}
		defer rows.Close()

		for rows.Next() {
			var tx Deposit
			if err := rows.StructScan(&tx); err != nil {
				select {
				case <-ctx.Done():
				case stream <- DepositTxStream{Error: err}:
				}
				return
			}
			select {
			case <-ctx.Done():
			case stream <- DepositTxStream{Data: &tx}:
			}
		}
	}()

	return stream
}

func (m Metis) HasGotDrip(ctx context.Context, address string) (bool, error) {
	const query = "SELECT COUNT(*) FROM `drips` WHERE `to`=?;"
	var count int
	if err := m.db.QueryRowContext(ctx, query, address).Scan(&count); err != nil {
		return false, fmt.Errorf("HasGotDrip: %w", err)
	}
	return count == 0, nil
}

func (m Metis) NewDrip(ctx context.Context, deposit *Deposit, drip *Drip) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SaveSyncingData: begin tx %w", err)
	}

	defer func() {
		if err == nil {
			return
		}
		if rollbackError := tx.Rollback(); rollbackError != nil {
			logrus.Errorf("SaveSyncingData: rollback: %s", rollbackError)
		}
	}()

	var status = DepositStatusIgnore
	if drip != nil {
		const insertDripQuery = "INSERT INTO `drips` (`pid`,`txid`,`from`,`to`,`amount`,`rawtx`) VALUES (?,?,?,?,?,?);"
		if drip.Pid != deposit.Id {
			return fmt.Errorf("NewDrip: drip id is not same with deposit id")
		}
		args := []interface{}{drip.Pid, drip.Txid, drip.From, drip.To, drip.Amount, drip.Rawtx}
		if _, err := tx.ExecContext(ctx, insertDripQuery, args...); err != nil {
			return fmt.Errorf("NewDrip: save drip: %w", err)
		}
		status = DepositStatusProcessing
	}

	const updateDepositStatusQuery = "UPDATE `deposits` SET `status`=? WHERE id=?;"
	if _, err := tx.ExecContext(ctx, updateDepositStatusQuery, status, deposit.Id); err != nil {
		return fmt.Errorf("NewDrip: update deposit tx status: %w", err)
	}

	return tx.Commit()
}

type PendingDrip struct {
	Id    uint64 `db:"id"`
	Txid  string `db:"txid"`
	Rawtx []byte `db:"rawtx"`
}

type PendingDripStream struct {
	Data  *PendingDrip
	Error error
}

func (m Metis) GetPendingDripsStream(ctx context.Context) <-chan PendingDripStream {
	var stream = make(chan PendingDripStream, 5)
	const query = "SELECT A.id as id,B.txid as txid,B.rawtx as rawtx  FROM `deposits` as A INNER JOIN `drips` as B ON A.id=B.pid WHERE `status`=? LIMIT 20;"

	go func() {
		defer close(stream)

		rows, err := m.db.QueryxContext(ctx, query, DepositStatusProcessing)
		if err != nil {
			select {
			case <-ctx.Done():
			case stream <- PendingDripStream{Error: err}:
			}
			return
		}
		defer rows.Close()

		for rows.Next() {
			var tx PendingDrip
			if err := rows.StructScan(&tx); err != nil {
				select {
				case <-ctx.Done():
				case stream <- PendingDripStream{Error: err}:
				}
				return
			}
			select {
			case <-ctx.Done():
			case stream <- PendingDripStream{Data: &tx}:
			}
		}
	}()

	return stream
}

func (m Metis) UpdateDripStatus(ctx context.Context, id uint64, status DepositStatus) error {
	const query = "UPDATE `deposits` SET `status`=? WHERE `id`=?;"
	res, err := m.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("UpdateDripStatus: %w", err)
	}
	if count, _ := res.RowsAffected(); count != 1 {
		return fmt.Errorf("UpdateDripStatus: affected row length should be 1")
	}
	return nil
}
