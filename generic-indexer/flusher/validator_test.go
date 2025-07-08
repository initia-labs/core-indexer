package flusher_test

import (
	"testing"

	crand "github.com/cometbft/cometbft/libs/rand"
	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"

	"github.com/alleslabs/initia-mono/generic-indexer/cosmosrpc"
	"github.com/alleslabs/initia-mono/generic-indexer/db"
	"github.com/alleslabs/initia-mono/generic-indexer/flusher"
)

func setup() *flusher.Flusher {

	clients := make([]cosmosrpc.CosmosJSONRPCClient, 0)
	clients = append(clients, &flusher.MockClient{})
	hub := &cosmosrpc.Hub{Clients: clients}

	return flusher.Flusher{}.WithCustomRPCClient(hub)
}

func randomValidatorVote(blockIDFlag types.BlockIDFlag) (types.Address, types.CommitSig, string) {
	val := crand.Bytes(crypto.AddressSize)
	sig := types.CommitSig{BlockIDFlag: blockIDFlag, ValidatorAddress: val}
	consensusAddress, _ := bech32.ConvertAndEncode(flusher.Bech32PrefixCons, val)
	return val, sig, consensusAddress
}

func TestInserValidator(t *testing.T) {
	flusher := setup()

	sigs := make([]types.CommitSig, 0)

	commitFlags := []types.BlockIDFlag{
		types.BlockIDFlagCommit,
		types.BlockIDFlagAbsent,
		types.BlockIDFlagCommit,
		types.BlockIDFlagAbsent,
		types.BlockIDFlagCommit,
	}
	expectVotes := []db.BlockVote{
		db.VOTE,
		db.ABSENT,
		db.VOTE,
		db.ABSENT,
		db.VOTE,
	}
	expect := make(map[string]db.BlockVote)
	for idx := 0; idx < 5; idx++ {
		_, sig, con := randomValidatorVote(commitFlags[idx])
		sigs = append(sigs, sig)
		expect[con] = expectVotes[idx]
	}

	blockVote, err := flusher.ProcessCommitSignatureVote(sigs)
	require.NoError(t, err)

	require.Equal(t, blockVote, expect)
}
