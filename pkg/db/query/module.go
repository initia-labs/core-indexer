package query

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/db"
)

const (
	moduleListSelect = `name AS module_name, digest, is_verify, publisher_id AS address,
		block_info.height, block_info.timestamp AS latest_updated,
		(SELECT COUNT(*) > 1 FROM module_histories WHERE module_histories.module_id = modules.id) AS is_republished`
	moduleBlockInfoJoin = `LEFT JOIN LATERAL (
		SELECT blocks.height, blocks.timestamp FROM module_histories
		JOIN blocks ON blocks.height = module_histories.block_height
		WHERE module_histories.module_id = modules.id
		ORDER BY module_histories.block_height DESC LIMIT 1
	) AS block_info ON true`
	moduleListOrder = `(SELECT MAX(block_height) FROM module_histories WHERE module_histories.module_id = modules.id) DESC`
)

// ModuleListQuery returns a query for listing modules with pagination.
func ModuleListQuery(d *gorm.DB, limit, offset int) *gorm.DB {
	return d.Model(&db.Module{}).
		Select(moduleListSelect).
		Joins(moduleBlockInfoJoin).
		Order(moduleListOrder).
		Limit(limit).
		Offset(offset)
}

// ModuleCountQuery returns a query for counting modules.
func ModuleCountQuery(d *gorm.DB) *gorm.DB {
	return d.Model(&db.Module{})
}

// ModuleByIDQuery returns a query for a single module by vmAddress and name.
func ModuleByIDQuery(d *gorm.DB, vmAddress, name string) *gorm.DB {
	return d.Model(&db.Module{}).
		Select(moduleListSelect).
		Joins(moduleBlockInfoJoin).
		Where("modules.id = ?", fmt.Sprintf("%s::%s", vmAddress, name))
}

// ModuleHistoriesListQuery returns a query for module histories with pagination.
func ModuleHistoriesListQuery(d *gorm.DB, moduleID string, limit, offset int) *gorm.DB {
	return d.Model(&db.ModuleHistory{}).
		Select(
			"module_histories.remark",
			"module_histories.upgrade_policy",
			"block_height AS height",
			"blocks.timestamp",
		).
		Joins("LEFT JOIN blocks ON blocks.height = module_histories.block_height").
		Where("module_histories.module_id = ?", moduleID).
		Order("module_histories.block_height DESC").
		Limit(limit).
		Offset(offset)
}

// ModuleHistoriesCountQuery returns a query for counting module histories.
func ModuleHistoriesCountQuery(d *gorm.DB, moduleID string) *gorm.DB {
	return d.Model(&db.ModuleHistory{}).Where("module_histories.module_id = ?", moduleID)
}

// ModulePublishInfoQuery returns a query for module publish info (limit 2).
func ModulePublishInfoQuery(d *gorm.DB, moduleID string) *gorm.DB {
	return d.Model(&db.ModuleHistory{}).
		Select(
			"transactions.hash",
			"blocks.timestamp",
			"proposals.id AS proposal_proposal_id",
			"proposals.title AS proposal_proposal_title",
			"module_histories.block_height AS height",
		).
		Joins("LEFT JOIN transactions ON transactions.id = module_histories.tx_id").
		Joins("LEFT JOIN blocks ON blocks.height = module_histories.block_height").
		Joins("LEFT JOIN proposals ON proposals.id = module_histories.proposal_id").
		Where("module_histories.module_id = ?", moduleID).
		Limit(2).
		Order("module_histories.block_height DESC")
}

// ModuleProposalsListQuery returns a query for module proposals with pagination.
func ModuleProposalsListQuery(d *gorm.DB, moduleID string, limit, offset int) *gorm.DB {
	return d.Model(&db.ModuleProposal{}).
		Select(
			"proposals.id",
			"proposals.title",
			"proposals.status",
			"proposals.voting_end_time",
			"proposals.deposit_end_time",
			"proposals.types",
			"proposals.is_expedited",
			"proposals.is_emergency",
			"proposals.resolved_height",
			"proposals.proposer_id AS proposer",
		).
		Joins("LEFT JOIN proposals ON proposals.id = module_proposals.proposal_id").
		Where("module_proposals.module_id = ?", moduleID).
		Order("module_proposals.proposal_id DESC").
		Limit(limit).
		Offset(offset)
}

// ModuleProposalsCountQuery returns a query for counting module proposals.
func ModuleProposalsCountQuery(d *gorm.DB, moduleID string) *gorm.DB {
	return d.Model(&db.ModuleProposal{}).Where("module_proposals.module_id = ?", moduleID)
}

// ModuleTransactionsListQuery returns a query for module transactions with pagination.
func ModuleTransactionsListQuery(d *gorm.DB, moduleID string, limit, offset int) *gorm.DB {
	return d.Model(&db.ModuleTransaction{}).
		Select(
			"blocks.height",
			"blocks.timestamp",
			"transactions.sender",
			"transactions.hash",
			"transactions.success",
			"transactions.messages",
			"transactions.is_send",
			"transactions.is_ibc",
			"transactions.is_move_execute",
			"transactions.is_move_execute_event",
			"transactions.is_move_publish",
			"transactions.is_move_script",
			"transactions.is_move_upgrade",
			"transactions.is_opinit",
		).
		Joins("LEFT JOIN blocks ON blocks.height = module_transactions.block_height").
		Joins("LEFT JOIN transactions ON transactions.id = module_transactions.tx_id").
		Where("module_transactions.module_id = ?", moduleID).
		Order("module_transactions.block_height DESC").
		Limit(limit).
		Offset(offset)
}

// ModuleTransactionsCountQuery returns a query for counting module transactions.
func ModuleTransactionsCountQuery(d *gorm.DB, moduleID string) *gorm.DB {
	return d.Model(&db.ModuleTransaction{}).Where("module_transactions.module_id = ?", moduleID)
}

// ModuleStatsQuery returns a raw query for module stats (total_txs, total_histories, total_proposals).
// Call .Scan(dest) on the result to load into a struct with those column names.
func ModuleStatsQuery(d *gorm.DB, moduleID string) *gorm.DB {
	return d.Raw(`
		SELECT
			(SELECT COUNT(*) FROM module_transactions WHERE module_id = ?) AS total_txs,
			(SELECT COUNT(*) FROM module_histories WHERE module_id = ?) AS total_histories,
			(SELECT COUNT(*) FROM module_proposals WHERE module_id = ?) AS total_proposals
	`, moduleID, moduleID, moduleID)
}
