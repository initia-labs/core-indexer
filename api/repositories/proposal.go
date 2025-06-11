package repositories

import (
	"github.com/initia-labs/core-indexer/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/initia-labs/core-indexer/pkg/db"
)

var _ ProposalRepositoryI = &ProposalRepository{}

// ProposalRepository implements ProposalRepositoryI
type ProposalRepository struct {
	db *gorm.DB
}

func NewProposalRepository(db *gorm.DB) *ProposalRepository {
	return &ProposalRepository{
		db: db,
	}
}

func (r *ProposalRepository) GetProposals() ([]db.Proposal, error) {
	var proposals []db.Proposal

	if err := r.db.Model(&db.Proposal{}).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "id",
			},
			Desc: true,
		}).
		Find(&proposals).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query all proposals")
		return nil, err
	}

	return proposals, nil
}

func (r *ProposalRepository) GetProposalVotesByValidator(operatorAddr string) ([]db.ProposalVote, error) {
	var votes []db.ProposalVote

	if err := r.db.Model(&db.ProposalVote{}).
		Preload("Proposal").
		Preload("Validator").
		Preload("Transaction").
		Preload("Transaction.Block").
		Where("validator_address = ?", operatorAddr).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "transaction_id",
			},
			Desc: true,
		}).Find(&votes).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query proposal votes for %s", operatorAddr)
		return nil, err
	}

	return votes, nil
}
