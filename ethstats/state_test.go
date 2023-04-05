package ethstats

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
)

func setupPostgresql(t *testing.T) (*sqlx.DB, func()) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.Run("postgres", "latest", []string{"POSTGRES_HOST_AUTH_METHOD=trust"})
	if err != nil {
		t.Fatalf("Could not start resource: %s", err)
	}

	endpoint := fmt.Sprintf("postgres://postgres@localhost:%s/postgres?sslmode=disable", resource.GetPort("5432/tcp"))

	// wait for the db to be running
	var db *sqlx.DB
	if err := pool.Retry(func() error {
		db, err = sqlx.Open("postgres", endpoint)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		if err := pool.Purge(resource); err != nil {
			t.Fatalf("Could not purge resource: %s", err)
		}
	}
	return db, cleanup
}

var (
	one    = big.NewInt(1)
	config = &Config{ShouldSaveBlockTxs: true}
)

func TestState_WriteBlock(t *testing.T) {
	db, closeFn := setupPostgresql(t)
	defer closeFn()

	s, err := NewStateWithDB(db)
	assert.NoError(t, err)

	hash := "0x1234"

	block := &Block{
		Number:    99999,
		Hash:      hash,
		Timestamp: time.Now().Nanosecond(),
		Txs:       []TxStats{{Hash: "0x0"}},
		Diff:      argBigPtr(one),
	}

	assert.NoError(t, s.WriteBlock(config, block))

	block2, err := s.GetBlock(hash)
	assert.NoError(t, err)
	assert.Len(t, block2.Txs, 1)
}

func TestState_DeleteOlderData(t *testing.T) {
	db, closeFn := setupPostgresql(t)
	defer closeFn()

	s, err := NewStateWithDB(db)
	assert.NoError(t, err)

	//Block A
	hashA := "0x1234"
	blockA := &Block{
		Number:    99999,
		Hash:      hashA,
		Timestamp: time.Now().Nanosecond(),
		Txs:       []TxStats{{Hash: "0x0"}},
		Diff:      argBigPtr(one),
	}

	//Block B to be inserted after 2 seconds
	hashB := "0x1235"
	blockB := &Block{
		Number:    99998,
		Hash:      hashB,
		Timestamp: time.Now().Nanosecond(),
		Txs:       []TxStats{{Hash: "0x1"}, {Hash: "0x2"}},
		Diff:      argBigPtr(one),
	}

	//Writing Block A
	assert.NoError(t, s.WriteBlock(config, blockA))

	//Sleeping for 2 seconds
	time.Sleep(2 * time.Second)

	//Writing Block B
	assert.NoError(t, s.WriteBlock(config, blockB))

	//Checking Presence of Block A before Deletion
	block2A, err := s.GetBlock(hashA)
	assert.NoError(t, err)
	assert.Len(t, block2A.Txs, 1)

	//Checking Presence of Block B before Deletion
	block2B, err := s.GetBlock(hashB)
	assert.NoError(t, err)
	assert.Len(t, block2B.Txs, 2)

	//Deleting data older than 2 seconds
	assert.NoError(t, s.DeleteOlderData(2))

	//This Block Should be Deleted and return nil
	block2A, err = s.GetBlock(hashA)
	assert.NoError(t, err)
	assert.Nil(t, block2A)

	//This Block should exist and have two transactions
	block2B, err = s.GetBlock(hashB)
	assert.NoError(t, err)
	assert.Len(t, block2B.Txs, 2)
}

func TestState_NodeInfo(t *testing.T) {
	db, closeFn := setupPostgresql(t)
	defer closeFn()

	s, err := NewStateWithDB(db)
	assert.NoError(t, err)

	info := &NodeInfo{
		Name: "a",
		Node: "b",
	}
	assert.NoError(t, s.WriteNodeInfo(info))

	info2, err := s.GetNodeInfo("a")
	assert.NoError(t, err)

	info2.CreatedAt = time.Time{}
	assert.Equal(t, info, info2)

	// get stats should be available but empty
	stats, err := s.GetNodeStats("a")
	assert.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestState_NodeStats(t *testing.T) {
	db, closeFn := setupPostgresql(t)
	defer closeFn()

	s, err := NewStateWithDB(db)
	assert.NoError(t, err)

	info := &NodeInfo{
		Name: "b",
	}
	assert.NoError(t, s.WriteNodeInfo(info))

	stats := &NodeStats{
		Peers: 100,
	}
	assert.NoError(t, s.WriteNodeStats("b", stats))

	stats2, err := s.GetNodeStats("b")
	assert.NoError(t, err)
	assert.Equal(t, stats, stats2)
}

func TestState_HeadEvent(t *testing.T) {
	db, closeFn := setupPostgresql(t)
	defer closeFn()

	s, err := NewStateWithDB(db)
	assert.NoError(t, err)

	info := &NodeInfo{Name: "b"}
	assert.NoError(t, s.WriteNodeInfo(info))

	evnt := &HeadEvent{
		Added: []BlockStub{
			{Hash: "0x1", Number: 1},
		},
		Removed: []BlockStub{
			{Hash: "0x1", Number: 1},
		},
		Type: "fork",
	}

	eventID, err := s.WriteHeadEvent("b", evnt)
	assert.NoError(t, err)

	evnt2, err := s.GetHeadEvent(eventID)
	assert.NoError(t, err)
	assert.Equal(t, evnt, evnt2)
}
