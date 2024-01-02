package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"
)

func (t Metis) InitHeight(ctx context.Context, defaultHeight uint64) (uint64, error) {
	const query = "SELECT `number` FROM `height`;"

	var hegiht uint64
	if err := t.db.QueryRowxContext(ctx, query).Scan(&hegiht); err != nil {
		if err == sql.ErrNoRows {
			const init = "INSERT INTO `height` (`number`,`blockhash`) VALUES (?,?);"
			if _, err := t.db.ExecContext(ctx, init, defaultHeight, ""); err != nil {
				return 0, fmt.Errorf("InitHeight: init height %w", err)
			}
			return defaultHeight, nil
		}
		return 0, fmt.Errorf("InitHeight: get height %w", err)
	}
	return hegiht + 1, nil
}

func (m Metis) SaveSyncedData(ctx context.Context, deposits []*Deposit, tail *Height) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SaveSyncedData: begin tx %w", err)
	}

	defer func() {
		if err == nil {
			return
		}
		if rollbackError := tx.Rollback(); rollbackError != nil {
			logrus.Errorf("SaveSyncedData: rollback: %s", rollbackError)
		}
	}()

	const insertDepositQuery = "INSERT INTO `deposits` (`height`,`txid`,`l1token`,`l2token`,`from`,`to`,`amount`,`status`) VALUES (?,?,?,?,?,?,?,?);"
	for _, item := range deposits {
		args := []interface{}{item.Height, item.Txid, item.L1Token, item.L2Token, item.From, item.To, item.Amount, item.Status}
		if _, err := tx.ExecContext(ctx, insertDepositQuery, args...); err != nil {
			return fmt.Errorf("SaveSyncedData: insert deposit data: %w", err)
		}
	}

	const updateHeightQuery = "UPDATE `height` SET `number`=?,`blockhash`=?;"
	if _, err := tx.ExecContext(ctx, updateHeightQuery, tail.Number, tail.Blockhash); err != nil {
		return fmt.Errorf("SaveSyncedData: update height data: %w", err)
	}
	return tx.Commit()
}
