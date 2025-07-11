package flusher

import (
	"fmt"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/initia/app/params"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type BankEventProcessor struct {
	txID string
}

func newBankEventProcessor(txID string) *BankEventProcessor {
	return &BankEventProcessor{
		txID: txID,
	}
}

func (f *Flusher) processBankEvents(txResult *mq.TxResult, height int64, txData *db.Transaction) error {
	processor := newBankEventProcessor(db.GetTxID(txResult.Hash, height))

	processor.processSDKMessages(txResult, f.encodingConfig, txData)
	return nil
}

func (p *BankEventProcessor) processSDKMessages(txResult *mq.TxResult, encodingConfig *params.EncodingConfig, txData *db.Transaction) error {
	sdkTx, err := encodingConfig.TxConfig.TxDecoder()(txResult.Tx)
	if err != nil {
		return fmt.Errorf("failed to decode SDK transaction: %w", err)
	}

	for _, msg := range sdkTx.GetMsgs() {
		switch msg.(type) {
		case *banktypes.MsgSend, *banktypes.MsgMultiSend:
			txData.IsSend = true
		}
	}

	return nil
}
