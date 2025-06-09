package repositories

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

// moduleRepository implements ModuleRepository
type moduleRepository struct {
	db *gorm.DB
}

func NewModuleRepository(db *gorm.DB) ModuleRepository {
	return &moduleRepository{
		db: db,
	}
}

// GetModules retrieves modules with pagination
func (r *moduleRepository) GetModules(pagination dto.PaginationQuery) ([]dto.ModuleResponse, int64, error) {
	var modules []dto.ModuleResponse
	var total int64

	query := r.db.Model(&db.Module{})

	// TODO: Consider optimizing this query
	if err := query.
		Select(
			"name AS module_name",
			"digest",
			"is_verify",
			"publisher_id AS address",
			"block_info.height",
			"block_info.timestamp AS latest_updated",
			"(SELECT COUNT(*) > 1 FROM module_histories WHERE module_histories.module_id = modules.id) AS is_republished",
		).
		Joins(
			"LEFT JOIN LATERAL (SELECT blocks.height, blocks.timestamp FROM module_histories JOIN blocks ON blocks.height = module_histories.block_height WHERE module_histories.module_id = modules.id ORDER BY module_histories.block_height DESC LIMIT 1) AS block_info ON true",
		).
		Order("(SELECT MAX(block_height) FROM module_histories WHERE module_histories.module_id = modules.id) DESC").
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&modules).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query modules")
		return nil, 0, err
	}

	// Get total count if requested
	if pagination.CountTotal {
		if err := query.Count(&total).Error; err != nil {
			logger.Get().Error().Err(err).Msg("Failed to count modules")
			return nil, 0, err
		}
	}

	return modules, total, nil
}

func (r *moduleRepository) GetModuleById(vmAddress string, name string) (*dto.ModuleResponse, error) {
	var module dto.ModuleResponse

	query := r.db.Model(&db.Module{})

	// TODO: Consider optimizing this query
	if err := query.
		Select(
			"name AS module_name",
			"digest",
			"is_verify",
			"publisher_id AS address",
			"block_info.height",
			"block_info.timestamp AS latest_updated",
			"(SELECT COUNT(*) > 1 FROM module_histories WHERE module_histories.module_id = modules.id) AS is_republished",
		).
		Joins(
			"LEFT JOIN LATERAL (SELECT blocks.height, blocks.timestamp FROM module_histories JOIN blocks ON blocks.height = module_histories.block_height WHERE module_histories.module_id = modules.id ORDER BY module_histories.block_height DESC LIMIT 1) AS block_info ON true",
		).
		Where("modules.id = ?", vmAddress+"::"+name).
		First(&module).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NewNotFound(fmt.Sprintf("module %s::%s not found", vmAddress, name))
		}
		logger.Get().Error().Err(err).Msg("Failed to query module")
		return nil, err
	}

	return &module, nil
}

func (r *moduleRepository) GetModuleHistories(pagination dto.PaginationQuery, vmAddress string, name string) ([]dto.ModuleHistory, int64, error) {
	return nil, 0, nil

	// query := `
	// 	SELECT
	// 		module_histories.block_height AS height,
	// 		blocks.timestamp AS latest_updated
	// 	FROM module_histories
	// 	JOIN blocks ON blocks.height = module_histories.block_height
	// 	WHERE module_histories.module_id = $1
	// 	ORDER BY module_histories.block_height %s
	// 	LIMIT $2 OFFSET $3
	// `

	// // Set order direction based on reverse flag
	// orderDirection := "ASC"
	// if pagination.Reverse {
	// 	orderDirection = "DESC"
	// }
	// query = fmt.Sprintf(query, orderDirection)

	// // Build the count query
	// countQuery := `
	// 	SELECT COUNT(*)
	// 	FROM module_histories
	// 	WHERE module_histories.module_id = $1
	// `

	// id := fmt.Sprintf("%s::%s", vmAddress, name)

	// // Execute queries
	// rows, err := r.db.Query(query, id, pagination.Limit, pagination.Offset)
	// if err != nil {
	// 	logger.Get().Error().Err(err).Msg("Failed to query module histories")
	// 	return nil, 0, err
	// }
	// defer rows.Close()

	// // Get total count if requested
	// var total int64
	// if pagination.CountTotal {
	// 	err = r.db.QueryRow(countQuery, id).Scan(&total)
	// 	if err != nil {
	// 		logger.Get().Error().Err(err).Msg("Failed to get module histories count")
	// 		return nil, 0, err
	// 	}
	// }

	// // Scan results
	// var histories []dto.ModuleHistory
	// for rows.Next() {
	// 	var history dto.ModuleHistory
	// 	if err := rows.Scan(&history.Height, &history.LatestUpdated); err != nil {
	// 		logger.Get().Error().Err(err).Msg("Failed to scan module history")
	// 		return nil, 0, err
	// 	}
	// 	histories = append(histories, history)
	// }

	// // Check for errors from iterating over rows
	// if err := rows.Err(); err != nil {
	// 	logger.Get().Error().Err(err).Msg("Error iterating module histories")
	// 	return nil, 0, err
	// }

	// return histories, total, nil
}
