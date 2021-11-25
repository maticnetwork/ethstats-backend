package main

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
)

func setupPostgresql(t *testing.T) (*sql.DB, func()) {
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
	var db *sql.DB
	if err := pool.Retry(func() error {
		db, err = sql.Open("postgres", endpoint)
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

func TestState_Blocks(t *testing.T) {

	db, closeFn := setupPostgresql(t)
	defer closeFn()

	s, err := NewStateWithDB(db)
	assert.NoError(t, err)

	fmt.Println(s)

	txs := [1]TxStats{{Hash: "0x0"}}
	uncles := []Block{}
	testBlock := &Block{
		Number:     99999,
		Hash:       "0x024915c6bfecb55a46e3a4061b606b39db86aa511223b5f92ec9bdf54e568e88",
		ParentHash: "0x23e64005d9f365b7d090b577889f3a92be4474c88858a39c79a79db91a9e21b3",
		Timestamp:  2637578243,
		Miner:      "0x9fb29aac15b9a4b7f17c3385939b007540f4d791",
		GasUsed:    0,
		GasLimit:   0,
		Diff:       "1",
		TotalDiff:  "99999",
		TxHash:     "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b423",
		Txs:        txs[:],
		Uncles:     uncles,
		Root:       "0x86a0906f755bfda86527e49a598fc6592235ee4bcf8592c49b8e5c59e46c0655",
	}

	if err := s.WriteBlock(testBlock); err != nil {
		t.Fatal(err)
	}
}
