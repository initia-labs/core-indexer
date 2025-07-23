package repositories

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

var _ AccountRepositoryI = &AccountRepository{}

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{
		db: db,
	}
}

func (r *AccountRepository) GetAccountByAccountAddress(accountAddress string) (*db.Account, error) {
	var record db.Account

	if err := r.db.Model(&db.Account{}).
		Where("address = ?", accountAddress).
		First(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("GetAccountType: failed to fetch account type")
		return nil, err
	}

	return &record, nil
}

func (r *AccountRepository) GetAccountProposals(pagination dto.PaginationQuery, accountAddress string) ([]db.Proposal, int64, error) {
	record := make([]db.Proposal, 0)
	total := int64(0)

	if err := r.db.Model(&db.Proposal{}).
		Select(`
			proposals.title,
			proposals.status,
			proposals.voting_end_time,
			proposals.deposit_end_time,
			proposals.type,
			proposals.id,
			proposals.proposer_id,
			proposals.is_expedited,
			proposals.is_emergency,
			proposals.resolved_height
		`).
		Where("proposals.proposer_id = ?", accountAddress).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "proposals.id",
			},
			Desc: pagination.Reverse,
		}).
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("GetAccountProposals: failed to fetch proposals")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := r.db.Model(&db.Proposal{}).
			Where("proposals.proposer_id = ?", accountAddress).
			Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("GetAccountProposals: failed to count proposals")
			return nil, 0, err
		}
	}

	return record, total, nil
}

func (r *AccountRepository) GetAccountTxs(
	pagination dto.PaginationQuery,
	accountAddress string,
	search string,
	isSend bool,
	isIbc bool,
	isOpinit bool,
	isMovePublish bool,
	isMoveUpgrade bool,
	isMoveExecute bool,
	isMoveScript bool,
	isSigner *bool,
) ([]dto.AccountTxModel, int64, error) {
	record := make([]dto.AccountTxModel, 0)
	total := int64(0)

	query := r.db.Model(&db.AccountTransaction{}).
		Select(`
			blocks.height,
			blocks.timestamp,
			accounts.address as sender,
			transactions.hash,
			transactions.success,
			transactions.messages,
			transactions.is_send,
			transactions.is_ibc,
			transactions.is_move_publish,
			transactions.is_move_upgrade,
			transactions.is_move_execute,
			transactions.is_move_script,
			transactions.is_opinit,
			account_transactions.is_signer
		`).
		Joins("LEFT JOIN blocks ON account_transactions.block_height = blocks.height").
		Joins("LEFT JOIN transactions ON account_transactions.transaction_id = transactions.id").
		Joins("LEFT JOIN accounts ON transactions.sender = accounts.address").
		Where("account_transactions.account_id = ?", accountAddress).
		Order((clause.OrderByColumn{
			Column: clause.Column{
				Name: "account_transactions.block_height",
			},
			Desc: pagination.Reverse,
		})).
		Limit(pagination.Limit).
		Offset(pagination.Offset)

	countQuery := r.db.Model(&db.AccountTransaction{}).
		Joins("LEFT JOIN transactions ON account_transactions.transaction_id = transactions.id").
		Where("account_transactions.account_id = ?", accountAddress)

	if isSigner != nil {
		query = query.Where("account_transactions.is_signer = ?", *isSigner)
		countQuery = countQuery.Where("account_transactions.is_signer = ?", *isSigner)
	}

	if isSend {
		query = query.Where("transactions.is_send = ?", isSend)
		countQuery = countQuery.Where("transactions.is_send = ?", isSend)
	}

	if isIbc {
		query = query.Where("transactions.is_ibc = ?", isIbc)
		countQuery = countQuery.Where("transactions.is_ibc = ?", isIbc)
	}

	if isOpinit {
		query = query.Where("transactions.is_opinit = ?", isOpinit)
		countQuery = countQuery.Where("transactions.is_opinit = ?", isOpinit)
	}

	if isMovePublish {
		query = query.Where("transactions.is_move_publish = ?", isMovePublish)
		countQuery = countQuery.Where("transactions.is_move_publish = ?", isMovePublish)
	}

	if isMoveUpgrade {
		query = query.Where("transactions.is_move_upgrade = ?", isMoveUpgrade)
		countQuery = countQuery.Where("transactions.is_move_upgrade = ?", isMoveUpgrade)
	}

	if isMoveExecute {
		query = query.Where("transactions.is_move_execute = ?", isMoveExecute)
		countQuery = countQuery.Where("transactions.is_move_execute = ?", isMoveExecute)
	}

	if isMoveScript {
		query = query.Where("transactions.is_move_script = ?", isMoveScript)
		countQuery = countQuery.Where("transactions.is_move_script = ?", isMoveScript)
	}

	search = strings.TrimSpace(search)

	if search != "" {
		if utils.IsTxHash(search) {
			query = query.Where("transactions.hash = ?", "\\x"+search)
			countQuery = countQuery.Where("transactions.hash = ?", "\\x"+search)
		} else {
			return nil, 0, nil
		}
	}

	if err := query.Find(&record).Error; err != nil {
		logger.Get().Error().Err(err).Msg("GetAccountTxs: failed to fetch account transactions")
		return nil, 0, err
	}

	if pagination.CountTotal {
		if err := countQuery.Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("GetAccountTxs: failed to count account transactions")
			return nil, 0, err
		}
	}

	return record, total, nil
}
