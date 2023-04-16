package main

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
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

	ethAPIBackend *eth.EthAPIBackend

	tracersApi *tracers.API

	traceCallConfig *tracers.TraceCallConfig

	blockNumber rpc.BlockNumberOrHash
}

func mainHook(ctx *cli.Context, stack *node.Node, backend ethapi.Backend) {
	log.Info("main hook is running!!!")

	ethApiBackend, ok := backend.(*eth.EthAPIBackend)
	if !ok {
		log.Crit("backend is not EthAPIBackend instance")
		return
	}
	bn := rpc.LatestBlockNumber

	es := &EventSystem{
		txsCh:         make(chan core.NewTxsEvent, txChanSize),
		ethAPIBackend: ethApiBackend,
		blockNumber:   rpc.BlockNumberOrHash{BlockNumber: &bn},
	}

	tracer := "callTracer"

	es.traceCallConfig = &tracers.TraceCallConfig{
		TraceConfig: tracers.TraceConfig{
			Tracer: &tracer,
		},
	}

	es.tracersApi = tracers.NewAPI(es.ethAPIBackend)

	es.txsSub = backend.SubscribeNewTxsEvent(es.txsCh)
	// Make sure none of the subscriptions are empty
	if es.txsSub == nil {
		log.Crit("main hook subscribe for event system failed")
		return
	}
	log.Info("main hook subscribe to new txs event!!!")
	es.eventLoop()
}

func (es *EventSystem) handleTxsEvent(tx *types.Transaction) {
	log.Info("main hook received new tx", "txHash", tx.Hash())

	context := context.Background()
	log.Info("00000000000000000")
	signer := types.MakeSigner(es.ethAPIBackend.ChainConfig(), es.ethAPIBackend.CurrentBlock().Number())
	from, err := types.Sender(signer, tx)
	if err != nil {
		log.Crit("main hook sign error", "error", err.Error())
		return
	}
	log.Info("11111111111111111")
	gas := hexutil.Uint64(tx.Gas())
	nonce := hexutil.Uint64(tx.Nonce())
	input := hexutil.Bytes(tx.Data())
	access := tx.AccessList()
	log.Info("22222222222222222222")
	// apis := ethapi.GetAPIs(backend)
	txArg := ethapi.TransactionArgs{
		From:                 &from,
		To:                   tx.To(),
		Gas:                  &gas,
		GasPrice:             (*hexutil.Big)(tx.GasPrice()),
		MaxFeePerGas:         (*hexutil.Big)(tx.GasFeeCap()),
		MaxPriorityFeePerGas: (*hexutil.Big)(tx.GasTipCap()),
		Value:                (*hexutil.Big)(tx.Value()),
		Nonce:                &nonce,
		Input:                &input,
		AccessList:           &access,
		ChainID:              (*hexutil.Big)(tx.ChainId()),
	}
	log.Info("3333333333333333333")
	result, err := es.tracersApi.TraceCall(context, txArg, es.blockNumber, es.traceCallConfig)
	log.Info("444444444444444444")
	if err != nil {
		log.Crit("main hook traceCall has error", "error", err.Error())
		return
	}
	log.Info("5555555555555555")
	var have *logger.ExecutionResult
	if err := json.Unmarshal(result.(json.RawMessage), &have); err != nil {
		log.Crit("main hook failed to unmarshal result %v", "error", err)
		return
	}
	log.Info("exec completed")
	log.Info("main hook", "value", have)
}

func (es *EventSystem) eventLoop() {
	defer func() {
		es.txsSub.Unsubscribe()
	}()

	for {
		select {
		case txEvent := <-es.txsCh:
			go es.handleTxsEvent(txEvent.Txs[0])
			// for _, tx := range txEvent.Txs {
			// 	go es.handleTxsEvent(tx)
			// }
		case subError := <-es.txsSub.Err():
			log.Crit("main hook tx subscribe error", "error", subError.Error())
		}
	}
}
