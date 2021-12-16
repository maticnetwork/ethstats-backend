package main

type TxStats struct {
	Hash string `json:"hash"`
}

// nodeInfo is the collection of meta information about a node that is displayed
// on the monitoring page.
type NodeInfo struct {
	Name     string `json:"name"`
	Node     string `json:"node"`
	Port     int    `json:"port"`
	Network  string `json:"net"`
	Protocol string `json:"protocol"`
	API      string `json:"api"`
	Os       string `json:"os"`
	OsVer    string `json:"os_v"`
	Client   string `json:"client"`
	History  bool   `json:"canUpdateHistory"`
}

// nodeStats is the information to report about the local node.
type NodeStats struct {
	Active   bool `json:"active"`
	Syncing  bool `json:"syncing"`
	Mining   bool `json:"mining"`
	Hashrate int  `json:"hashrate"`
	Peers    int  `json:"peers"`
	GasPrice int  `json:"gasPrice"`
	Uptime   int  `json:"uptime"`
}

// blockStats is the information to report about individual blocks.
type BlockDB struct {
	Number     int    `json:"number"`
	Hash       string `json:"hash"`
	ParentHash string `json:"parentHash"`
	Timestamp  int    `json:"timestamp"`
	Miner      string `json:"miner"`
	GasUsed    uint64 `json:"gasUsed"`
	GasLimit   uint64 `json:"gasLimit"`
	Diff       string `json:"difficulty"`
	TotalDiff  string `json:"totalDifficulty"`
	Txs        int    `json:"transactions"`
	TxHash     string `json:"transactionsRoot"`
	Root       string `json:"stateRoot"`
	Uncles     int    `json:"uncles"`
}
