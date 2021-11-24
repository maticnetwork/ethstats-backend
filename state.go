package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"strconv"
)

type State struct {
	db *sql.DB
}

func NewState(path string) (*State, error) {

	db, err := sql.Open("postgres", path)
	if err != nil {
		return nil, err
	}

	return NewStateWithDB(db)

}

func NewStateWithDB(db *sql.DB) (*State, error) {
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

func (s *State) migrate() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	execSQLFile := func(path string) error {
		c, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		sql := string(c)

		_, err = tx.Exec(sql)
		if err != nil {
			return err
		}
		return nil
	}

	if err := execSQLFile("./migrations/01-block.sql"); err != nil {
		return err
	}
	if err := execSQLFile("./migrations/02-reorgEvent.sql"); err != nil {
		return err
	}
	if err := execSQLFile("./migrations/03-nodeInfo.sql"); err != nil {
		return err
	}
	if err := execSQLFile("./migrations/04-nodeStats.sql"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *State) writeBlockImpl(tx *sql.Tx, block *Block, nodeID *string) error {

	difficulty, err := strconv.ParseInt(block.Diff, 10, 64)
	if err != nil {
		return err
	}

	total_difficulty, err := strconv.ParseInt(block.TotalDiff, 10, 64)
	if err != nil {
		return err
	}

	var blockExist int

	q2 := fmt.Sprintf(`SELECT count(*) FROM public.blocks Where block_hash='%s'`, block.Hash)

	row := tx.QueryRow(q2)
	row.Scan(&blockExist)

	if blockExist == 0 {
		insertDynStmt := `insert into blocks("block_number", "block_hash", "parent_hash", "time_stamp", "miner", "gas_used", "gas_limit", "difficulty", "total_difficulty", "transactions_root", "transactions_count", "uncles_count", "state_root", "node_id") values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14 )`
		_, e := tx.Exec(insertDynStmt, int(block.Number), block.Hash, block.ParentHash, int(block.Timestamp), block.Miner, int(block.GasUsed), int(block.GasLimit), difficulty, total_difficulty, block.TxHash, len(block.Txs), len(block.Uncles), block.Root, nodeID)
		if e != nil {
			return e
		}
	}

	return nil
}

func (s *State) WriteBlock(block *Block, nodeID *string) error {

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	if err := s.writeBlockImpl(tx, block, nodeID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *State) WriteReorgEvents(block *Block, nodeID *string) error {

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	insertDynStmt := `insert into reorgevents("block_number", "block_hash", "node_id") values($1, $2, $3)`
	if _, err := tx.Exec(insertDynStmt, int(block.Number), block.Hash, nodeID); err != nil {
		return err
	}

	if err := s.writeBlockImpl(tx, block, nodeID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *State) WriteNodeInfo(nodeInfo *NodeInfo, nodeID *string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	var rowExist int

	q2 := fmt.Sprintf(`SELECT count(*) FROM public.nodeinfo Where node_id='%s'`, *nodeID)

	row := tx.QueryRow(q2)
	row.Scan(&rowExist)

	if rowExist == 0 {
		insertDynStmt := `insert into nodeinfo("name", "node", "port", "network", "protocol", "api", "os", "osver", "client", "history", "node_id") values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11 )`
		_, e := tx.Exec(insertDynStmt, nodeInfo.Name, nodeInfo.Node, nodeInfo.Port, nodeInfo.Network, nodeInfo.Protocol, nodeInfo.API, nodeInfo.Os, nodeInfo.OsVer, nodeInfo.Client, nodeInfo.History, *nodeID)
		if e != nil {
			return e
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *State) WriteNodeStats(nodeStats *NodeStats, nodeId *string) error {

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	var rowExist int

	q2 := fmt.Sprintf(`SELECT count(*) FROM public.nodestats Where node_id='%s'`, *nodeId)

	row := tx.QueryRow(q2)
	row.Scan(&rowExist)

	if rowExist == 0 {
		insertDynStmt := `insert into nodestats("node_id", "active", "syncing", "mining", "hashrate", "peers", "gasprice", "uptime") values($1, $2, $3, $4, $5, $6, $7, $8)`
		_, e := tx.Exec(insertDynStmt, *nodeId, nodeStats.Active, nodeStats.Syncing, nodeStats.Mining, nodeStats.Hashrate, nodeStats.Peers, nodeStats.GasPrice, nodeStats.Uptime)
		if e != nil {
			return e
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
