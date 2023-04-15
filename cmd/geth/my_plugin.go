package main

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"gopkg.in/urfave/cli.v1"
)

const (
	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096
)

type EventSystem struct {
	txsCh chan core.NewTxsEvent // Channel to receive new transactions event

	// Subscriptions
	txsSub event.Subscription // Subscription for new transaction event
}

func mainHook(ctx *cli.Context, stack *node.Node, backend ethapi.Backend) {
	log.Info("main hook is running!!!")
	es := &EventSystem{
		txsCh: make(chan core.NewTxsEvent, txChanSize),
	}
	es.txsSub = backend.SubscribeNewTxsEvent(es.txsCh)
	// Make sure none of the subscriptions are empty
	if es.txsSub == nil {
		log.Crit("main hook subscribe for event system failed")
	}
	log.Info("main hook subscribe to new txs event!!!")
	es.eventLoop()
}

func handleTxsEvent(tx *types.Transaction) {
	log.Info("main hook received new tx", "txHash", tx.Hash())
}

func (es *EventSystem) eventLoop() {
	defer func() {
		es.txsSub.Unsubscribe()
	}()

	for {
		select {
		case txEvent := <-es.txsCh:
			for _, tx := range txEvent.Txs {
				go handleTxsEvent(tx)
			}
		}
	}
}
