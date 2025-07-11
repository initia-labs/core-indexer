package proposal

import (
	"fmt"
	"strconv"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/initia-labs/core-indexer/informative-indexer/flusher/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

// handleEvent routes events to appropriate handlers based on event type
func (p *Processor) handleEvent(event abci.Event) error {
	switch event.Type {
	case cosmosgovtypes.EventTypeSubmitProposal:
		return p.handleSubmitProposalEvent(event)
	case cosmosgovtypes.EventTypeProposalDeposit:
		return p.handleProposalDepositEvent(event)
	case cosmosgovtypes.EventTypeCancelProposal:
		return p.handleCancelProposalEvent(event)
	case cosmosgovtypes.EventTypeProposalVote:
		return p.handleProposalVoteEvent(event)
	default:
		return nil
	}
}

// handleEvent routes events to appropriate handlers based on event type
func (p *Processor) handleSubmitProposalEvent(event abci.Event) error {
	if value, found := utils.FindAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID); found {
		proposalID, err := parser.ParseInt32(value)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}
		p.newProposals[proposalID] = p.txProcessor.txID
	}
	return nil
}

func (p *Processor) handleProposalDepositEvent(event abci.Event) error {
	if value, found := utils.FindAttribute(event.Attributes, cosmosgovtypes.AttributeKeyVotingPeriodStart); found {
		proposalID, err := parser.ParseInt32(value)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}
		p.proposalStatusChanges[proposalID] = db.ProposalStatusVotingPeriod
	} else {
		value, found := utils.FindAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID)
		if !found {
			return fmt.Errorf("failed to filter proposal id")
		}

		proposalID, err := parser.ParseInt32(value)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}

		depositor, found := utils.FindAttribute(event.Attributes, cosmosgovtypes.AttributeKeyDepositor)
		if !found {
			return fmt.Errorf("failed to filter depositor")
		}

		coin, found := utils.FindAttribute(event.Attributes, sdk.AttributeKeyAmount)
		if !found {
			return fmt.Errorf("failed to filter amount")
		}

		amount, denom, err := parser.ParseCoinAmount(coin)
		if err != nil {
			return fmt.Errorf("failed to parse amount: %w", err)
		}

		p.proposalDeposits = append(p.proposalDeposits, db.ProposalDeposit{
			Depositor:     depositor,
			Amount:        db.JSON(fmt.Sprintf(`[{"amount": "%d", "denom": "%s"}]`, amount, denom)),
			ProposalID:    proposalID,
			TransactionID: p.txProcessor.txID,
		})
	}
	return nil
}

func (p *Processor) handleCancelProposalEvent(event abci.Event) error {
	if value, found := utils.FindAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID); found {
		proposalID, err := parser.ParseInt32(value)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}
		p.proposalStatusChanges[proposalID] = db.ProposalStatusCancelled
	}
	return nil
}

func (p *Processor) handleProposalVoteEvent(event abci.Event) error {
	value, found := utils.FindAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID)
	if !found {
		return fmt.Errorf("failed to filter proposal id")
	}

	proposalID, err := parser.ParseInt32(value)
	if err != nil {
		return fmt.Errorf("failed to parse proposal id: %w", err)
	}

	voter, found := utils.FindAttribute(event.Attributes, cosmosgovtypes.AttributeKeyVoter)
	if !found {
		return fmt.Errorf("failed to filter voter")
	}

	value, found = utils.FindAttribute(event.Attributes, cosmosgovtypes.AttributeKeyOption)
	if !found {
		return fmt.Errorf("failed to filter option")
	}

	options, err := parser.DecodeEvent[govv1types.WeightedVoteOptions](value)
	if err != nil {
		return fmt.Errorf("failed to decode option: %w", err)
	}

	votes := map[govv1types.VoteOption]string{
		govv1types.OptionYes:        "0",
		govv1types.OptionAbstain:    "0",
		govv1types.OptionNo:         "0",
		govv1types.OptionNoWithVeto: "0",
	}
	for _, option := range options {
		votes[option.GetOption()] = option.GetWeight()
	}

	yesVote, _ := strconv.ParseFloat(votes[govv1types.OptionYes], 64)
	noVote, _ := strconv.ParseFloat(votes[govv1types.OptionNo], 64)
	abstainVote, _ := strconv.ParseFloat(votes[govv1types.OptionAbstain], 64)
	noWithVetoVote, _ := strconv.ParseFloat(votes[govv1types.OptionNoWithVeto], 64)

	p.proposalVotes = append(p.proposalVotes, db.ProposalVote{
		Voter:          voter,
		ProposalID:     proposalID,
		TransactionID:  p.txProcessor.txID,
		IsVoteWeighted: len(options) > 1,
		IsValidator:    false,
		Yes:            yesVote,
		No:             noVote,
		Abstain:        abstainVote,
		NoWithVeto:     noWithVetoVote,
	})
	return nil
}
