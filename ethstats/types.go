package ethstats

import (
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"
)

type TxStats struct {
	Hash string `json:"hash" db:"txn_hash"`
}

// blockStats is the information to report about individual blocks.
type Block struct {
	Number     int       `json:"number" db:"number"`
	Hash       string    `json:"hash" db:"hash"`
	ParentHash string    `json:"parentHash" db:"parent_hash"`
	Timestamp  int       `json:"timestamp" db:"timestamp"`
	Miner      string    `json:"miner" db:"miner"`
	GasUsed    uint64    `json:"gasUsed" db:"gas_used"`
	GasLimit   uint64    `json:"gasLimit" db:"gas_limit"`
	Diff       *argBig   `json:"difficulty" db:"difficulty"`
	TotalDiff  *argBig   `json:"totalDifficulty" db:"total_difficulty"`
	Txs        []TxStats `json:"transactions"`
	TxHash     string    `json:"transactionsRoot" db:"transactions_root"`
	Root       string    `json:"stateRoot" db:"state_root"`
	Uncles     []Block   `json:"uncles"`
}

// nodeInfo is the collection of meta information about a node that is displayed
// on the monitoring page.
type NodeInfo struct {
	Name      string    `json:"name" db:"node_id"`
	Node      string    `json:"node" db:"node"`
	Port      int       `json:"port" db:"port"`
	Network   string    `json:"net" db:"network"`
	Protocol  string    `json:"protocol" db:"protocol"`
	API       string    `json:"api" db:"api"`
	Os        string    `json:"os" db:"os"`
	OsVer     string    `json:"os_v" db:"osver"`
	Client    string    `json:"client" db:"client"`
	History   bool      `json:"canUpdateHistory" db:"history"`
	CreatedAt time.Time `db:"created_at"`
}

// nodeStats is the information to report about the local node.
type NodeStats struct {
	Active   bool `json:"active" db:"active"`
	Syncing  bool `json:"syncing" db:"syncing"`
	Mining   bool `json:"mining" db:"mining"`
	Hashrate int  `json:"hashrate" db:"hashrate"`
	Peers    int  `json:"peers" db:"peers"`
	GasPrice int  `json:"gasPrice" db:"gasprice"`
	Uptime   int  `json:"uptime" db:"uptime"`
}

type HeadEvent struct {
	Added   []BlockStub `json:"added"`
	Removed []BlockStub `json:"removed"`
	Type    string      `json:"type" db:"typ"`
}

type BlockStub struct {
	ParentHash string `json:"parent_hash" db:"parent_hash"`
	Hash       string `json:"hash" db:"block_hash"`
	Number     int    `json:"number" db:"block_number"`
}

type argBig big.Int

func argBigPtr(b *big.Int) *argBig {
	v := argBig(*b)
	return &v
}

func (a *argBig) Value() (driver.Value, error) {
	return (*big.Int)(a).String(), nil
}

func (a *argBig) Scan(value interface{}) error {
	var i sql.NullString
	if err := i.Scan(value); err != nil {
		return err
	}
	if _, ok := (*big.Int)(a).SetString(i.String, 10); ok {
		return nil
	}
	return fmt.Errorf("cannot convert to big.Int (%s)", reflect.TypeOf(value))
}

func (a *argBig) UnmarshalText(input []byte) error {
	buf, err := decodeToHex(input)
	if err != nil {
		return err
	}
	b := new(big.Int)
	b.SetBytes(buf)
	*a = argBig(*b)
	return nil
}

func decodeToHex(b []byte) ([]byte, error) {
	str := string(b)
	str = strings.TrimPrefix(str, "0x")
	if len(str)%2 != 0 {
		str = "0" + str
	}
	return hex.DecodeString(str)
}
