package bank

import (
	"fmt"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/initia-labs/initia/app/params"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/cacher"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

func (p *Processor) InitProcessor(height int64, cacher *cacher.Cacher) {
	p.Height = height
	p.Cacher = cacher
	p.txProcessor = nil
}

func (p *Processor) Name() string {
	return "bank"
}

func (p *Processor) NewTxProcessor(txData *db.Transaction) {
	p.txProcessor = &TxProcessor{
		txData: txData,
	}
}

func (p *Processor) ProcessSDKMessages(tx *mq.TxResult, encodingConfig *params.EncodingConfig) error {
	sdkTx, err := encodingConfig.TxConfig.TxDecoder()(tx.Tx)
	if err != nil {
		return fmt.Errorf("failed to decode SDK transaction: %w", err)
	}

	for _, msg := range sdkTx.GetMsgs() {
		switch msg.(type) {
		case *banktypes.MsgSend, *banktypes.MsgMultiSend:
			p.txProcessor.txData.IsSend = true
		}
	}

	return nil
}
