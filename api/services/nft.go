package services

import (
	"fmt"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

// NftService defines the interface for Nft-related operations
type NftService interface {
	GetCollections(pagination dto.PaginationQuery, search string) (*dto.NftCollectionsResponse, error)
	GetCollectionsByAccountAddress(accountAddress string) (*dto.NftCollectionsResponse, error)
	GetCollectionsByCollectionAddress(collectionAddress string) (*dto.NftCollectionResponse, error)
	GetCollectionActivities(pagination dto.PaginationQuery, collectionAddress string, search string) (*dto.CollectionActivitiesResponse, error)
	GetCollectionCreator(collectionAddress string) (*dto.CollectionCreatorResponse, error)
	GetCollectionMutateEvents(pagination dto.PaginationQuery, collectionAddress string) (*dto.CollectionMutateEventsResponse, error)
	GetNftByNftAddress(collectionAddress string, nftAddress string) (*dto.NftByAddressResponse, error)
	GetNftsByAccountAddress(pagination dto.PaginationQuery, accountAddress string, collectionAddress string, search string) (*dto.NftsByAddressResponse, error)
	GetNftsByCollectionAddress(pagination dto.PaginationQuery, collectionAddress string, search string) (*dto.NftsByAddressResponse, error)
	GetNftMintInfo(nftAddress string) (*dto.NftMintInfoResponse, error)
	GetNftMutateEvents(pagination dto.PaginationQuery, nftAddress string) (*dto.NftMutateEventsResponse, error)
	GetNftTxs(pagination dto.PaginationQuery, nftAddress string) (*dto.NftTxsResponse, error)
}

// nftService implements the NftService interface
type nftService struct {
	repo repositories.NftRepositoryI
}

// NewNftService creates a new instance of NftService
func NewNftService(repo repositories.NftRepositoryI) NftService {
	return &nftService{
		repo: repo,
	}
}

// GetCollections retrieves Nft collections with pagination and search
func (s *nftService) GetCollections(pagination dto.PaginationQuery, search string) (*dto.NftCollectionsResponse, error) {
	foundCollections, total, err := s.repo.GetCollections(pagination, search)
	if err != nil {
		return nil, err
	}

	collections := make([]dto.NftCollectionResponse, len(foundCollections))

	for idx, collection := range foundCollections {
		collections[idx] = dto.NftCollectionResponse{
			ObjectAddr: collection.ID,
			Collection: dto.NftCollectionCollection{
				Creator:     collection.Creator,
				Description: collection.Description,
				Name:        collection.Name,
				URI:         collection.URI,
				Nft:         nil,
			},
		}
	}

	response := &dto.NftCollectionsResponse{
		Collections: collections,
		Pagination:  dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}

	return response, nil
}

func (s *nftService) GetCollectionsByAccountAddress(accountAddress string) (*dto.NftCollectionsResponse, error) {
	foundCollections, err := s.repo.GetCollectionsByAccountAddress(accountAddress)
	if err != nil {
		return nil, err
	}

	collections := make([]dto.NftCollectionResponse, len(foundCollections))

	for idx, collection := range foundCollections {
		collections[idx] = dto.NftCollectionResponse{
			ObjectAddr: collection.ID,
			Collection: dto.NftCollectionCollection{
				Creator:     collection.Creator,
				Description: collection.Description,
				Name:        collection.Name,
				URI:         collection.URI,
				Nft: &dto.NftCollectionCollectionNft{
					Length: collection.Count,
				},
			},
		}
	}

	response := &dto.NftCollectionsResponse{
		Collections: collections,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   fmt.Sprintf("%d", len(foundCollections)),
		},
	}

	return response, nil
}

func (s *nftService) GetCollectionsByCollectionAddress(collectionAddress string) (*dto.NftCollectionResponse, error) {
	collection, err := s.repo.GetCollectionsByCollectionAddress(collectionAddress)
	if err != nil {
		return nil, err
	}

	return &dto.NftCollectionResponse{
		ObjectAddr: collection.ID,
		Collection: dto.NftCollectionCollection{
			Creator:     collection.Creator,
			Description: collection.Description,
			Name:        collection.Name,
			URI:         collection.URI,
			Nft:         nil,
		},
	}, nil
}

func (s *nftService) GetCollectionActivities(pagination dto.PaginationQuery, collectionAddress string, search string) (*dto.CollectionActivitiesResponse, error) {
	activities, total, err := s.repo.GetCollectionActivities(pagination, collectionAddress, search)
	if err != nil {
		return nil, err
	}

	collectionActivities := make([]dto.CollectionActivityModel, len(activities))
	for idx, activity := range activities {
		collectionActivities[idx] = dto.CollectionActivityModel{
			Hash:               fmt.Sprintf("%x", activity.Hash),
			Timestamp:          activity.Timestamp,
			IsNftBurn:          activity.IsNftBurn,
			IsNftMint:          activity.IsNftMint,
			IsNftTransfer:      activity.IsNftTransfer,
			NftID:              activity.NftID,
			TokenID:            activity.TokenID,
			IsCollectionCreate: activity.IsCollectionCreate,
		}
	}

	return &dto.CollectionActivitiesResponse{
		CollectionActivities: collectionActivities,
		Pagination:           dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}, nil
}

func (s *nftService) GetCollectionCreator(collectionAddress string) (*dto.CollectionCreatorResponse, error) {
	creator, err := s.repo.GetCollectionCreator(collectionAddress)
	if err != nil {
		return nil, err
	}

	creator.Hash = fmt.Sprintf("%x", creator.Hash)

	return &dto.CollectionCreatorResponse{
		Creator: *creator,
	}, nil
}

func (s *nftService) GetCollectionMutateEvents(pagination dto.PaginationQuery, collectionAddress string) (*dto.CollectionMutateEventsResponse, error) {
	mutateEvents, total, err := s.repo.GetCollectionMutateEvents(pagination, collectionAddress)
	if err != nil {
		return nil, err
	}

	return &dto.CollectionMutateEventsResponse{
		CollectionMutateEvents: mutateEvents,
		Pagination:             dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}, nil

}

func (s *nftService) GetNftByNftAddress(collectionAddress string, nftAddress string) (*dto.NftByAddressResponse, error) {
	nft, err := s.repo.GetNftByNftAddress(collectionAddress, nftAddress)

	if err != nil {
		return nil, err
	}

	return &dto.NftByAddressResponse{
		ObjectAddr:     nft.ID,
		CollectionAddr: nft.Collection,
		CollectionName: nft.CollectionName,
		OwnerAddr:      nft.Owner,
		Nft: dto.NftByAddressNft{
			Collection: dto.NftByAddressNftCollection{
				Inner: nft.Collection,
			},
			Description: nft.Description,
			TokenID:     nft.TokenID,
			URI:         nft.URI,
			IsBurned:    nft.IsBurned,
		},
	}, nil
}

func (s *nftService) GetNftsByAccountAddress(pagination dto.PaginationQuery, accountAddress string, collectionAddress string, search string) (*dto.NftsByAddressResponse, error) {
	nfts, total, err := s.repo.GetNftsByAccountAddress(pagination, accountAddress, collectionAddress, search)

	if err != nil {
		return nil, err
	}

	response := &dto.NftsByAddressResponse{
		Tokens:     make([]dto.NftByAddressResponse, len(nfts)),
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}

	for idx, nft := range nfts {
		response.Tokens[idx] = dto.NftByAddressResponse{
			ObjectAddr:     nft.ID,
			CollectionAddr: nft.Collection,
			CollectionName: nft.CollectionName,
			OwnerAddr:      nft.Owner,
			Nft: dto.NftByAddressNft{
				Collection: dto.NftByAddressNftCollection{
					Inner: nft.Collection,
				},
				Description: nft.Description,
				TokenID:     nft.TokenID,
				URI:         nft.URI,
				IsBurned:    nft.IsBurned,
			},
		}
	}

	return response, nil
}

func (s *nftService) GetNftsByCollectionAddress(pagination dto.PaginationQuery, collectionAddress string, search string) (*dto.NftsByAddressResponse, error) {
	nfts, total, err := s.repo.GetNftsByCollectionAddress(pagination, collectionAddress, search)

	if err != nil {
		return nil, err
	}

	response := &dto.NftsByAddressResponse{
		Tokens:     make([]dto.NftByAddressResponse, len(nfts)),
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}

	for idx, nft := range nfts {
		response.Tokens[idx] = dto.NftByAddressResponse{
			ObjectAddr:     nft.ID,
			CollectionAddr: nft.Collection,
			CollectionName: nft.CollectionName,
			OwnerAddr:      nft.Owner,
			Nft: dto.NftByAddressNft{
				Collection: dto.NftByAddressNftCollection{
					Inner: nft.Collection,
				},
				Description: nft.Description,
				TokenID:     nft.TokenID,
				URI:         nft.URI,
				IsBurned:    nft.IsBurned,
			},
		}
	}

	return response, nil
}

func (s *nftService) GetNftMintInfo(nftAddress string) (*dto.NftMintInfoResponse, error) {
	mintInfo, err := s.repo.GetNftMintInfo(nftAddress)
	if err != nil {
		return nil, err
	}

	return &dto.NftMintInfoResponse{
		Height:    mintInfo.Height,
		Minter:    mintInfo.Address,
		TxHash:    fmt.Sprintf("%x", mintInfo.Hash),
		Timestamp: mintInfo.Timestamp,
	}, nil
}

func (s *nftService) GetNftMutateEvents(pagination dto.PaginationQuery, nftAddress string) (*dto.NftMutateEventsResponse, error) {
	mutateEvents, total, err := s.repo.GetNftMutateEvents(pagination, nftAddress)
	if err != nil {
		return nil, err
	}

	response := &dto.NftMutateEventsResponse{
		NftMutateEvents: mutateEvents,
		Pagination:      dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}

	return response, nil
}

func (s *nftService) GetNftTxs(pagination dto.PaginationQuery, nftAddress string) (*dto.NftTxsResponse, error) {
	txs, total, err := s.repo.GetNftTxs(pagination, nftAddress)
	if err != nil {
		return nil, err
	}

	response := &dto.NftTxsResponse{
		NftTxs:     make([]dto.NftTx, len(txs)),
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}

	for idx, tx := range txs {
		response.NftTxs[idx] = dto.NftTx{
			IsNftBurn:     tx.IsNftBurn,
			IsNftMint:     tx.IsNftMint,
			IsNftTransfer: tx.IsNftTransfer,
			Timestamp:     tx.Timestamp,
			TxHash:        fmt.Sprintf("%x", tx.Hash),
		}
	}

	return response, nil
}
