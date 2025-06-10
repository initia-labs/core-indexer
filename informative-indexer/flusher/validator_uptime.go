package flusher

import (
	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"

	"github.com/initia-labs/core-indexer/pkg/db"
)

func Bech32ValConsPub(val bytes.HexBytes) (string, error) {
	bech32PrefixCons := sdk.GetConfig().GetBech32ConsensusAddrPrefix()
	consensusAddress, err := bech32.ConvertAndEncode(bech32PrefixCons, val)
	if err != nil {
		return "", err
	}
	return consensusAddress, err
}

// ProcessCommitSignatureVote checks all validator votes from the last commit signature
// to determine which validators voted as "absent" or "proposed" a block. It returns
// a mapping of validator consensus addresses to their respective votes.
func (f *Flusher) ProcessCommitSignatureVote(sigs []types.CommitSig) (map[string]db.CommitSignatureType, error) {
	commitSigs := make(map[string]db.CommitSignatureType)
	for _, commitSig := range sigs {
		if commitSig.ValidatorAddress.String() == "" {
			continue
		}
		consensusAddress, err := Bech32ValConsPub(commitSig.ValidatorAddress)
		if err != nil {
			return nil, err
		}

		switch commitSig.BlockIDFlag {
		case types.BlockIDFlagAbsent:
			commitSigs[consensusAddress] = db.Absent
		case types.BlockIDFlagCommit:
			commitSigs[consensusAddress] = db.Vote
		}
	}
	return commitSigs, nil
}
