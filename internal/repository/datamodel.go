package repository

import (
	"time"

	"github.com/islishude/bigint"
)

type DepositStatus uint8

const (
	DepositStatusUnprocessed DepositStatus = iota
	DepositStatusProcessing
	DepositStatusDone
	DepositStatusIgnore
)

type Deposit struct {
	Id        uint64        `db:"id"`
	Txid      string        `db:"txid"`
	Height    uint64        `db:"height"`
	L1Token   string        `db:"l1token"`
	L2Token   string        `db:"l2token"`
	From      string        `db:"from"`
	To        string        `db:"to"`
	Amount    bigint.Int    `db:"amount"`
	Status    DepositStatus `db:"status"`
	CreatedAt time.Time     `db:"ctime"`
	UpdatedAt time.Time     `db:"mtime"`
}

type Height struct {
	Number    uint64 `db:"number"`
	Blockhash string `db:"blockhash"`
}

type Drip struct {
	Pid       uint64    `db:"pid"`
	Txid      string    `db:"txid"`
	From      string    `db:"from"`
	To        string    `db:"to"`
	Amount    float64   `db:"amount"`
	Rawtx     []byte    `db:"rawtx"`
	CreatedAt time.Time `db:"ctime"`
}
