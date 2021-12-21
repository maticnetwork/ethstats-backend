package main

import (
	"crypto/rand"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"math/big"

	"github.com/jmoiron/sqlx"
	"github.com/oklog/ulid"
)

//go:embed migrations/*.sql
var migrations embed.FS

type State struct {
	db *sqlx.DB
}

func NewState(path string) (*State, error) {
	db, err := sqlx.Open("postgres", path)
	if err != nil {
		return nil, err
	}
	return NewStateWithDB(db)
}

func NewStateWithDB(db *sqlx.DB) (*State, error) {
	err := db.Ping()
	if err != nil {
		return nil, err
	}
	s := &State{
		db: db,
	}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *State) Close() {
	s.db.Close()
}

func (s *State) migrate() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	sqlMigrations, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return err
	}
	for _, sqlExec := range sqlMigrations {
		sqlTableQuery, err := fs.ReadFile(migrations, "migrations/"+sqlExec.Name())
		if err != nil {
			return err
		}
		if _, err = tx.Exec(string(sqlTableQuery)); err != nil {
			return fmt.Errorf("failed to migrate sql %s: %v", sqlExec.Name(), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *State) GetBlock(hash string) (*Block, error) {
	block := Block{}

	query := "SELECT number, hash, parent_hash, timestamp, miner, gas_used, gas_limit, difficulty, total_difficulty, transactions_root, state_root FROM blocks WHERE hash=$1"
	if err := s.db.Get(&block, query, hash); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	txns := []TxStats{}
	if err := s.db.Select(&txns, "SELECT txn_hash FROM block_transactions WHERE block_hash=$1", hash); err != nil {
		return nil, err
	}
	block.Txs = txns

	return &block, nil
}

func (s *State) WriteBlock(b *Block) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var count uint64

	// do not include a block twice
	row := tx.QueryRow(`SELECT count(*) FROM blocks WHERE hash=$1`, b.Hash)
	if err := row.Scan(&count); err != nil {
		return err
	}
	if count == 1 {
		return nil
	}

	// add default values for 'difficulty' and 'total_difficulty' which are pointers
	if b.Diff == nil {
		b.Diff = argBigPtr(big.NewInt(0))
	}
	if b.TotalDiff == nil {
		b.TotalDiff = argBigPtr(big.NewInt(0))
	}

	query := `INSERT INTO blocks
		("number", "hash", "parent_hash", "timestamp", "miner", "gas_used", "gas_limit", "difficulty", "total_difficulty", "transactions_root", "transactions_count", "uncles_count", "state_root") 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT DO NOTHING`

	// write the block
	_, err = tx.Exec(query, int(b.Number), b.Hash, b.ParentHash, int(b.Timestamp), b.Miner, int(b.GasUsed), int(b.GasLimit), b.Diff, b.TotalDiff, b.TxHash, len(b.Txs), len(b.Uncles), b.Root)
	if err != nil {
		return err
	}

	// add the transactions for each block
	for _, txn := range b.Txs {
		if _, err := tx.Exec(`INSERT INTO block_transactions (block_hash, txn_hash) VALUES ($1, $2)`, b.Hash, txn.Hash); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *State) GetNodeInfo(nodeID string) (*NodeInfo, error) {
	info := NodeInfo{}
	if err := s.db.Get(&info, "SELECT * FROM nodeinfo WHERE node_id=$1", nodeID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &info, nil
}

func (s *State) WriteNodeInfo(nodeInfo *NodeInfo) error {
	nodeID := nodeInfo.Name
	if nodeID == "" {
		return fmt.Errorf("node id is empty")
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	var count int
	row := tx.QueryRow(fmt.Sprintf(`SELECT count(*) FROM nodeinfo Where node_id='%s'`, nodeID))
	if err := row.Scan(&count); err != nil {
		return err
	}
	if count == 1 {
		return nil
	}

	query := `INSERT INTO nodeinfo("node_id", "node", "port", "network", "protocol", "api", "os", "osver", "client", "history") 
		values(:node_id, :node, :port, :network, :protocol, :api, :os, :osver, :client, :history)`

	if _, err := tx.NamedExec(query, nodeInfo); err != nil {
		return err
	}

	// write the initial node stats row with empty values so we can update it later more efficiently
	query = `INSERT INTO nodestats("node_id") VALUES ($1)`

	if _, err := tx.Exec(query, nodeID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *State) GetNodeStats(nodeID string) (*NodeStats, error) {
	stats := NodeStats{}
	if err := s.db.Get(&stats, "SELECT active, syncing, mining, hashrate, peers, gasprice, uptime FROM nodestats WHERE node_id=$1", nodeID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &stats, nil
}

func (s *State) WriteNodeStats(nodeId string, stats *NodeStats) error {
	query := `UPDATE nodestats SET active = $1, syncing = $2, mining = $3, hashrate = $4, peers = $5, gasprice = $6, uptime = $7
	WHERE node_id=$8;`

	if _, err := s.db.Exec(query, stats.Active, stats.Syncing, stats.Mining, stats.Hashrate, stats.Peers, stats.GasPrice, stats.Uptime, nodeId); err != nil {
		return err
	}
	return nil
}

func (s *State) WriteReorgEvent(nodeID string, evnt *ReorgEvent) error {
	// we use an ulid to identify each reorg event
	ulid, err := newUlid()
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	writeElem := func(stub BlockStub, typ string) error {
		query := `INSERT INTO reorgentry("reorg_id", "block_number", "block_hash", "parent_hash", "typ") VALUES ($1, $2, $3, $4, $5)`
		if _, err := tx.Exec(query, ulid, stub.Number, stub.Hash, stub.ParentHash, typ); err != nil {
			return err
		}
		return nil
	}

	// write the reorg event
	if _, err := tx.Exec(`INSERT INTO reorgevents("node_id", "reorg_id") values ($1, $2)`, nodeID, ulid); err != nil {
		return err
	}

	// write the reorg elems
	for _, add := range evnt.Added {
		if err := writeElem(add, "add"); err != nil {
			return err
		}
	}
	for _, del := range evnt.Removed {
		if err := writeElem(del, "del"); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func newUlid() (string, error) {
	id, err := ulid.New(ulid.Now(), rand.Reader)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
