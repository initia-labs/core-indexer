package cosmosrpc

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/rs/zerolog"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	matcher = mock.MatchedBy(func(ctx interface{}) bool {
		_, ok := ctx.(context.Context)
		return ok
	})
)

func TestRebalance(t *testing.T) {
	ctx := context.Background()
	clients := make([]CosmosJSONRPCClient, 0)

	for idx := 0; idx < 100; idx++ {
		m := MockClient{url: fmt.Sprintf("%d", idx)}
		status := coretypes.ResultStatus{}
		height, _ := rand.Int(rand.Reader, big.NewInt(10000000))
		status.SyncInfo.LatestBlockHeight = height.Int64()
		m.On("Status", matcher).Return(&status, nil)
		clients = append(clients, &m)
	}

	hub := Hub{Clients: clients}

	err := hub.Rebalance(ctx)
	require.NoError(t, err)

	actives := hub.GetActiveClients()
	for idx := 0; idx < len(actives)-1; idx++ {
		require.True(t, actives[idx].Height > actives[idx+1].Height)
	}

	clients = make([]CosmosJSONRPCClient, 0)
	for idx := 0; idx < 100; idx++ {
		m := MockClient{url: fmt.Sprintf("%d", idx)}
		status := coretypes.ResultStatus{}
		m.On("Status", matcher).Return(&status, fmt.Errorf("ERROR!!"))
		clients = append(clients, &m)
	}
	logger := zerolog.Nop()
	hub = Hub{Clients: clients, logger: &logger}
	err = hub.Rebalance(ctx)
	require.Error(t, err)
}

func TestCannotGetBlockDataFromAllRPC(t *testing.T) {
	ctx := context.Background()
	currentHeight := int64(100)

	badClient := MockClient{url: "ERROR CLIENT"}
	badClient.On("Block", matcher).Return(&coretypes.ResultBlock{}, fmt.Errorf("ERROR!!"))

	staleClient := MockClient{url: "STALE CLIENT"}
	staleClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(1, nil, nil, nil)}, nil)
	logger := zerolog.Nop()
	hub := Hub{
		activeClients: []ActiveClient{ActiveClient{Client: &badClient},
			ActiveClient{Client: &staleClient}},
		logger: &logger,
	}

	_, err := hub.Block(ctx, &currentHeight)
	require.EqualError(t, err, "RPC: Stale block data")
}
func TestBlockSuccess(t *testing.T) {
	ctx := context.Background()
	currentHeight := int64(100)
	AClient := MockClient{url: "A CLIENT"}
	AClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(100, nil, nil, nil)}, nil)
	BClient := MockClient{url: "B CLIENT"}
	BClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(100, nil, nil, nil)}, nil)
	CClient := MockClient{url: "C CLIENT"}
	CClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(100, nil, nil, nil)}, nil)
	DClient := MockClient{url: "D CLIENT"}
	DClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(100, nil, nil, nil)}, nil)
	logger := zerolog.Nop()

	hub := Hub{
		activeClients: []ActiveClient{
			ActiveClient{Client: &AClient},
			ActiveClient{Client: &BClient},
			ActiveClient{Client: &CClient},
			ActiveClient{Client: &DClient},
		},
		logger: &logger,
	}

	res, err := hub.Block(ctx, &currentHeight)
	require.NoError(t, err)
	require.Equal(t, res, &coretypes.ResultBlock{Block: tmtypes.MakeBlock(100, nil, nil, nil)})

}
func TestBlockWithStaleRPCData(t *testing.T) {
	ctx := context.Background()
	currentHeight := int64(100)

	AClient := MockClient{url: "A CLIENT"}
	AClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(99, nil, nil, nil)}, nil)
	BClient := MockClient{url: "B CLIENT"}
	BClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(99, nil, nil, nil)}, nil)
	CClient := MockClient{url: "C CLIENT"}
	CClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(99, nil, nil, nil)}, nil)
	DClient := MockClient{url: "D CLIENT"}
	DClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(100, nil, nil, nil)}, nil)
	logger := zerolog.Nop()
	hub := Hub{
		activeClients: []ActiveClient{
			ActiveClient{Client: &AClient},
			ActiveClient{Client: &BClient},
			ActiveClient{Client: &CClient},
			ActiveClient{Client: &DClient},
		},
		logger: &logger,
	}

	res, err := hub.Block(ctx, &currentHeight)
	require.NoError(t, err)
	require.Equal(t, res, &coretypes.ResultBlock{Block: tmtypes.MakeBlock(100, nil, nil, nil)})
}

func TestBlockWithAllStaleRPCData(t *testing.T) {
	ctx := context.Background()
	currentHeight := int64(101)

	AClient := MockClient{url: "A CLIENT"}
	AClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(99, nil, nil, nil)}, nil)
	BClient := MockClient{url: "B CLIENT"}
	BClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(99, nil, nil, nil)}, nil)
	CClient := MockClient{url: "C CLIENT"}
	CClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(99, nil, nil, nil)}, nil)
	DClient := MockClient{url: "D CLIENT"}
	DClient.On("Block", matcher).Return(&coretypes.ResultBlock{Block: tmtypes.MakeBlock(100, nil, nil, nil)}, nil)
	logger := zerolog.Nop()

	hub := Hub{
		activeClients: []ActiveClient{
			ActiveClient{Client: &AClient},
			ActiveClient{Client: &BClient},
			ActiveClient{Client: &CClient},
			ActiveClient{Client: &DClient},
		},
		logger: &logger,
	}

	_, err := hub.Block(ctx, &currentHeight)
	require.Error(t, err)
}

func TestCannotGetBlockResultsDataFromAllRPC(t *testing.T) {
	ctx := context.Background()
	currentHeight := int64(100)

	badClient := MockClient{url: "ERROR CLIENT"}
	badClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{}, fmt.Errorf("ERROR!!"))

	staleClient := MockClient{url: "STALE CLIENT"}
	staleClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 1}, nil)
	logger := zerolog.Nop()
	hub := Hub{
		activeClients: []ActiveClient{
			ActiveClient{Client: &badClient},
			ActiveClient{Client: &staleClient},
		},
		logger: &logger,
	}

	_, err := hub.BlockResults(ctx, &currentHeight)
	require.EqualError(t, err, "RPC: Stale block results data")
}
func TestBlockResultsSuccess(t *testing.T) {
	ctx := context.Background()
	currentHeight := int64(100)

	AClient := MockClient{url: "A CLIENT"}
	AClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 100}, nil)
	BClient := MockClient{url: "B CLIENT"}
	BClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 100}, nil)
	CClient := MockClient{url: "C CLIENT"}
	CClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 100}, nil)
	DClient := MockClient{url: "D CLIENT"}
	DClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 100}, nil)

	logger := zerolog.Nop()
	hub := Hub{
		activeClients: []ActiveClient{
			ActiveClient{Client: &AClient},
			ActiveClient{Client: &BClient},
			ActiveClient{Client: &CClient},
			ActiveClient{Client: &DClient},
		},
		logger: &logger,
	}

	res, err := hub.BlockResults(ctx, &currentHeight)
	require.NoError(t, err)
	require.Equal(t, res, &coretypes.ResultBlockResults{Height: 100})
}

func TestBlockResultsWithStaleRPCData(t *testing.T) {
	ctx := context.Background()
	currentHeight := int64(100)

	AClient := MockClient{url: "A CLIENT"}
	AClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 99}, nil)
	BClient := MockClient{url: "B CLIENT"}
	BClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 99}, nil)
	CClient := MockClient{url: "C CLIENT"}
	CClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 99}, nil)
	DClient := MockClient{url: "D CLIENT"}
	DClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 100}, nil)
	logger := zerolog.Nop()
	hub := Hub{
		activeClients: []ActiveClient{
			ActiveClient{Client: &AClient},
			ActiveClient{Client: &BClient},
			ActiveClient{Client: &CClient},
			ActiveClient{Client: &DClient},
		},
		logger: &logger,
	}

	res, err := hub.BlockResults(ctx, &currentHeight)
	require.NoError(t, err)
	require.Equal(t, res, &coretypes.ResultBlockResults{Height: 100})
}

func TestBlockResultsWithAllStaleRPCData(t *testing.T) {
	ctx := context.Background()
	currentHeight := int64(101)

	AClient := MockClient{url: "A CLIENT"}
	AClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 99}, nil)
	BClient := MockClient{url: "B CLIENT"}
	BClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 99}, nil)
	CClient := MockClient{url: "C CLIENT"}
	CClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 99}, nil)
	DClient := MockClient{url: "D CLIENT"}
	DClient.On("BlockResults", matcher).Return(&coretypes.ResultBlockResults{Height: 100}, nil)

	logger := zerolog.Nop()

	hub := Hub{
		activeClients: []ActiveClient{
			ActiveClient{Client: &AClient},
			ActiveClient{Client: &BClient},
			ActiveClient{Client: &CClient},
			ActiveClient{Client: &DClient},
		},
		logger: &logger,
	}

	_, err := hub.BlockResults(ctx, &currentHeight)
	require.Error(t, err)
}
