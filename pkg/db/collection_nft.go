package db

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UpsertCollection(ctx context.Context, dbTx *gorm.DB, collections []Collection) error {
	if len(collections) == 0 {
		return nil
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name",
				"description",
				"uri",
			}),
		}).
		CreateInBatches(&collections, BatchSize)
	return result.Error
}

func InsertCollectionTransactions(ctx context.Context, dbTx *gorm.DB, collectionTransactions []CollectionTransaction) error {
	if len(collectionTransactions) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).CreateInBatches(collectionTransactions, BatchSize).Error
}

func InsertNftsOnConflictDoUpdate(ctx context.Context, dbTx *gorm.DB, nftTransactions []*Nft) error {
	if len(nftTransactions) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"owner",
				"is_burned",
				"description",
				"uri",
			}),
		}).CreateInBatches(nftTransactions, BatchSize).Error
}

func UpdateBurnedNftsOnConflictDoUpdate(ctx context.Context, dbTx *gorm.DB, nftIDs []string) error {
	if len(nftIDs) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).
		Model(&Nft{}).
		Where("id IN ?", nftIDs).
		Update("is_burned", true).Error
}

func InsertNftTransactions(ctx context.Context, dbTx *gorm.DB, nftTransactions []NftTransaction) error {
	if len(nftTransactions) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).CreateInBatches(nftTransactions, BatchSize).Error
}

func InsertNftHistories(ctx context.Context, dbTx *gorm.DB, nftHistories []NftHistory) error {
	if len(nftHistories) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).CreateInBatches(nftHistories, BatchSize).Error
}

func GetNftsByIDs(ctx context.Context, dbTx *gorm.DB, ids []string) ([]*Nft, error) {
	var nfts []*Nft
	result := dbTx.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&nfts)
	if result.Error != nil {
		return nil, result.Error
	}
	return nfts, nil
}

func InsertCollectionMutationEvents(ctx context.Context, dbTx *gorm.DB, collectionMutationEvents []CollectionMutationEvent) error {
	if len(collectionMutationEvents) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).CreateInBatches(collectionMutationEvents, BatchSize).Error
}

func UpdateCollectionURI(ctx context.Context, dbTx *gorm.DB, collectionID string, uri string) error {
	return dbTx.WithContext(ctx).
		Model(&Collection{}).
		Where("id = ?", collectionID).
		Update("uri", uri).Error
}

func UpdateCollectionDescription(ctx context.Context, dbTx *gorm.DB, collectionID string, description string) error {
	return dbTx.WithContext(ctx).
		Model(&Collection{}).
		Where("id = ?", collectionID).
		Update("description", description).Error
}

func UpdateCollectionName(ctx context.Context, dbTx *gorm.DB, collectionID string, name string) error {
	return dbTx.WithContext(ctx).
		Model(&Collection{}).
		Where("id = ?", collectionID).
		Update("name", name).Error
}

func UpdateNftURI(ctx context.Context, dbTx *gorm.DB, nftID string, uri string) error {
	return dbTx.WithContext(ctx).
		Model(&Nft{}).
		Where("id = ?", nftID).
		Update("uri", uri).Error
}

func UpdateNftDescription(ctx context.Context, dbTx *gorm.DB, nftID string, description string) error {
	return dbTx.WithContext(ctx).
		Model(&Nft{}).
		Where("id = ?", nftID).
		Update("description", description).Error
}

func InsertNftMutationEvents(ctx context.Context, dbTx *gorm.DB, nftMutationEvents []NftMutationEvent) error {
	if len(nftMutationEvents) == 0 {
		return nil
	}
	return dbTx.WithContext(ctx).CreateInBatches(nftMutationEvents, BatchSize).Error
}
