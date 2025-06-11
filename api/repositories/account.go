package repositories

import (
	"strings"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
	"gorm.io/gorm"
)

type accountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &accountRepository{
		db: db,
	}
}

func (r *accountRepository) GetAccountByAccountAddress(accountAddress string) (*db.Account, error) {
	var record db.Account

	err := r.db.Model(&db.Account{}).
		Where("vm_address_id = ?", accountAddress).
		First(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("GetAccountType: failed to fetch account type")
		return nil, err
	}

	return &record, nil
}

func (r *accountRepository) GetAccountProposals(pagination dto.PaginationQuery, accountAddress string) ([]db.Proposal, int64, error) {
	var record []db.Proposal
	var count int64

	orderDirection := "asc"
	if pagination.Reverse {
		orderDirection = "desc"
	}

	err := r.db.Model(&db.Proposal{}).
		Select(`
			title, 
			status,
			voting_end_time,
			deposit_end_time,
			types,
			id,
			is_expedited,
			is_emergency,
			resolved_height
		`).
		Joins("LEFT JOIN accounts ON accounts.address = proposals.proposer_id").
		Where("accounts.vm_address_id = ?", accountAddress).
		Order("id " + orderDirection).
		Offset(int(pagination.Offset)).
		Limit(int(pagination.Limit)).
		Find(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("GetAccountProposals: failed to fetch proposals")
		return nil, 0, err
	}

	if pagination.CountTotal {
		err = r.db.Model(&db.Proposal{}).
			Joins("LEFT JOIN accounts ON accounts.address = proposals.proposer_id").
			Where("accounts.vm_address_id = ?", accountAddress).
			Count(&count).Error

		if err != nil {
			logger.Get().Error().Err(err).Msg("GetAccountProposals: failed to count proposals")
			return nil, 0, err
		}
	}

	return record, count, nil
}

func (r *accountRepository) GetAccountTxs(
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
	var record []dto.AccountTxModel
	var count int64

	orderDirection := "asc"
	if pagination.Reverse {
		orderDirection = "desc"
	}

	query := r.db.Model(&db.AccountTransaction{}).
		Select(`
			blocks.height,
			blocks.timestamp,
			account_transactions.account_id as address,
			encode(transactions.hash::bytea, 'hex') as hash,
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
		Joins("LEFT JOIN accounts ON accounts.address = account_transactions.account_id").
		Joins("LEFT JOIN blocks ON account_transactions.block_height = blocks.height").
		Joins("LEFT JOIN transactions ON account_transactions.transaction_id = transactions.id").
		Where("accounts.vm_address_id = ?", accountAddress).
		Order("account_transactions.block_height " + orderDirection).
		Offset(int(pagination.Offset)).
		Limit(int(pagination.Limit))

	countQuery := r.db.Model(&db.AccountTransaction{}).
		Joins("LEFT JOIN accounts ON accounts.address = account_transactions.account_id").
		Joins("LEFT JOIN transactions ON account_transactions.transaction_id = transactions.id").
		Where("accounts.vm_address_id = ?", accountAddress)

	if isSigner != nil {
		query = query.Where("account_transactions.is_signer = ?", *isSigner)
		countQuery = countQuery.Where("account_transactions.is_signer = ?", *isSigner)
	}

	search = strings.TrimSpace(search)

	if search != "" {
		if utils.IsTxHash(search) {
			query = query.Where("transactions.hash = ?", "\\x"+search)
			countQuery = countQuery.Where("transactions.hash = ?", "\\x"+search)
		}
	}

	query = query.
		Where("transactions.is_send = ?", isSend).
		Where("transactions.is_ibc = ?", isIbc).
		Where("transactions.is_opinit = ?", isOpinit).
		Where("transactions.is_move_publish = ?", isMovePublish).
		Where("transactions.is_move_upgrade = ?", isMoveUpgrade).
		Where("transactions.is_move_execute = ?", isMoveExecute).
		Where("transactions.is_move_script = ?", isMoveScript)

	err := query.Find(&record).Error

	if err != nil {
		logger.Get().Error().Err(err).Msg("GetAccountTxs: failed to fetch account transactions")
		return nil, 0, err
	}

	if pagination.CountTotal {
		err = countQuery.Count(&count).Error
		if err != nil {
			logger.Get().Error().Err(err).Msg("GetAccountTxs: failed to count account transactions")
			return nil, 0, err
		}
	}

	return record, count, nil
}

func (r *accountRepository) GetAccountTxsStats(accountAddress string) {}
