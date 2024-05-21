package transaction

import (
    "math/big"
    "testing"

    "rpcproxy/model"
)

func TestRBTree_GetTransactionsWithinRange(t *testing.T) {
    // Create a new RBTree
    tree := NewRBTree()

    // Helper function to create a transaction with a specific gas price
    createTx := func(uuid string, gasPrice int64) *model.Transaction {
        return &model.Transaction{
            UUID:     uuid,
            GasPrice: big.NewInt(gasPrice),
        }
    }

    // Inserting transactions into the RBTree
    txs := []*model.Transaction{
        createTx("tx1", 100),
        createTx("tx2", 200),
        createTx("tx3", 300),
        createTx("tx4", 400),
        createTx("tx5", 500),
    }
    for _, tx := range txs {
        tree.Insert(tx)
    }

    // Defining the gas price range
    minGasPrice := big.NewInt(200)
    maxGasPrice := big.NewInt(400)

    // Getting transactions within the range
    result := tree.GetTransactionsWithinRange(minGasPrice, maxGasPrice)

    // Expected transactions
    expected := []*model.Transaction{
        createTx("tx2", 200),
        createTx("tx3", 300),
        createTx("tx4", 400),
    }

    // Verify the result
    if len(result) != len(expected) {
        t.Fatalf("Expected %d transactions, got %d", len(expected), len(result))
    }

    for i, tx := range result {
        if tx.UUID != expected[i].UUID || tx.GasPrice.Cmp(expected[i].GasPrice) != 0 {
            t.Errorf("Expected transaction %v, got %v", expected[i], tx)
        }
    }
}
