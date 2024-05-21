package transaction

import (
	"math/big"
	"rpcproxy/model"

	"github.com/emirpasic/gods/trees/redblacktree"
)

type RBTree struct {
	tree *redblacktree.Tree
}

func NewRBTree() *RBTree {
	return &RBTree{
		tree: redblacktree.NewWithStringComparator(),
	}
}

func (t *RBTree) Insert(tx *model.Transaction) {
	t.tree.Put(tx.UUID, tx)
}

func (t *RBTree) Remove(uuid string) *model.Transaction {
	value, found := t.tree.Get(uuid)
	if found {
		t.tree.Remove(uuid)
		return value.(*model.Transaction)
	}
	return nil
}

func (t *RBTree) GetTransactionsWithinRange(minGasPrice, maxGasPrice *big.Int) []*model.Transaction {
	var transactions []*model.Transaction

	// Find the leftmost node that is within or greater than minGasPrice
	iterator := t.tree.Iterator()
	for iterator.Next() {
		tx := iterator.Value().(*model.Transaction)
		if tx.GasPrice.Cmp(minGasPrice) >= 0 {
			transactions = append(transactions, tx)
			break
		}
	}

	// Collect all transactions within the range from the starting node
	for iterator.Next() {
		tx := iterator.Value().(*model.Transaction)
		if tx.GasPrice.Cmp(maxGasPrice) > 0 {
			break
		}
		if tx.GasPrice.Cmp(minGasPrice) >= 0 && tx.GasPrice.Cmp(maxGasPrice) <= 0 {
			transactions = append(transactions, tx)
		}
	}

	return transactions
}
