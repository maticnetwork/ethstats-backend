package main

import (
	"database/sql"
	"fmt"
	"strconv"
)

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

	return s, nil
}

func (s *State) WriteBlock(tx *sql.Tx, block *Block) error {

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
		insertDynStmt := `insert into blocks("block_number", "block_hash", "parent_hash", "time_stamp", "miner", "gas_used", "gas_limit", "difficulty", "total_difficulty", "transactions_root", "transactions_count", "uncles_count", "state_root") values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13 )`
		_, e := tx.Exec(insertDynStmt, int(block.Number), block.Hash, block.ParentHash, int(block.Timestamp), block.Miner, int(block.GasUsed), int(block.GasLimit), difficulty, total_difficulty, block.TxHash, len(block.Txs), len(block.Uncles), block.Root)
		if e != nil {
			return e
		}
	}

	return nil
}

func (s *State) WriteReorgEvents(tx *sql.Tx, block *Block) error {
	insertDynStmt := `insert into reorgevents("block_number", "block_hash", "node_info") values($1, $2, $3)`
	if _, err := tx.Exec(insertDynStmt, int(block.Number), block.Hash, "Node Name"); err != nil {
		return err
	}
	s.WriteBlock(tx, block)
	return nil
}

func (s *State) WriteNodeInfo(tx *sql.Tx, nodeInfo *NodeInfo) error {

	var rowExist int

	q2 := fmt.Sprintf(`SELECT count(*) FROM public.nodeinfo Where name='%s'`, nodeInfo.Name)

	row := tx.QueryRow(q2)
	row.Scan(&rowExist)

	if rowExist == 0 {
		insertDynStmt := `insert into nodeinfo("name", "node", "port", "network", "protocol", "api", "os", "osver", "client", "history") values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10 )`
		_, e := tx.Exec(insertDynStmt, nodeInfo.Name, nodeInfo.Node, nodeInfo.Port, nodeInfo.Network, nodeInfo.Protocol, nodeInfo.API, nodeInfo.Os, nodeInfo.OsVer, nodeInfo.Client, nodeInfo.History)
		if e != nil {
			return e
		}
	}

	return nil
}

func (s *State) WriteNodeStats(tx *sql.Tx, nodeStats *NodeStats, node_name string) error {

	var rowExist int

	q2 := fmt.Sprintf(`SELECT count(*) FROM public.nodestats Where name='%s'`, node_name)

	row := tx.QueryRow(q2)
	row.Scan(&rowExist)

	if rowExist == 0 {
		insertDynStmt := `insert into nodestats("name", "active", "syncing", "mining", "hashrate", "peers", "gasprice", "uptime") values($1, $2, $3, $4, $5, $6, $7, $8)`
		_, e := tx.Exec(insertDynStmt, node_name, nodeStats.Active, nodeStats.Syncing, nodeStats.Mining, nodeStats.Hashrate, nodeStats.Peers, nodeStats.GasPrice, nodeStats.Uptime)
		if e != nil {
			return e
		}
	}
	return nil
}
