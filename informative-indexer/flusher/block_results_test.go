package flusher_test

import (
	// "encoding/json"
	// "fmt"
	// "io"
	// "net/http"
	// "strconv"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	// "github.com/initia-labs/core-indexer/pkg/mq"
)

type MockResponse struct {
	Result MockResult `json:"result"`
}

type MockResult struct {
	TxsResults          []*abci.ExecTxResult `json:"txs_results"`
	FinalizeBlockEvents []abci.Event         `json:"finalize_block_events"`
}

// func getBlockResultsByHeight(rpcEndpoint, height string) *mq.BlockResultMsg {
// 	url := fmt.Sprintf("%s/block_results?height=%s", rpcEndpoint, height)

// 	resp, err := http.Get(url)
// 	if err != nil {
// 		panic(fmt.Errorf("failed to send request to rpc endpoint: %w", err))
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		panic(fmt.Errorf("failed to read response body: %w", err))
// 	}

// 	// Unmarshal the JSON response
// 	var result MockResponse
// 	if err = json.Unmarshal(body, &result); err != nil {
// 		panic(fmt.Errorf("failed to unmarshal block results: %w", err))
// 	}

// 	txs := make([]mq.TxResult, 0)
// 	for idx, tx := range result.Result.TxsResults {
// 		txs = append(txs, mq.TxResult{
// 			Hash:          fmt.Sprintf("mockHash{%d}", idx),
// 			ExecTxResults: tx,
// 		})
// 	}

// 	lh, err := strconv.Atoi(height)
// 	if err != nil {
// 		panic(err)
// 	}

// 	mockResult := &mq.BlockResultMsg{
// 		Height:              int64(lh),
// 		Txs:                 txs,
// 		FinalizeBlockEvents: result.Result.FinalizeBlockEvents,
// 	}

// 	return mockResult
// }

func TestProcessBlockResults(t *testing.T) {
	//blockResultsMsg := getBlockResultsByHeight("https://rpc.initiation-2.initia.xyz", "2542435")
	//fmt.Println(blockResultsMsg.Height)
	//
	//flusher, err := NewFlusher(&Config{})
	//if err != nil {
	//	panic(fmt.Errorf("failed to new flusher: %w", err))
	//}
	//
	//err = flusher.processBlockResults(context.Background(), blockResultsMsg)
	//if err != nil {
	//	panic(fmt.Errorf("failed to process block_results: %w", err))
	//}
}
