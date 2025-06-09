package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

// NFTService defines the interface for NFT-related operations
type NFTService interface {
	GetCollections(pagination dto.PaginationQuery, search string) (*dto.NFTCollectionsResponse, error)
	// GetCollectionsByAccountAddress(pagination dto.PaginationQuery, accountAddress string) (*dto.CollectionsByAccountAddressResponse, error)
	GetNFTByNFTAddress(collectionAddress string, nftAddress string) (*dto.NFTByAddressResponse, error)
	GetNFTsByAccountAddress(pagination dto.PaginationQuery, accountAddress string, collectionAddress string, search string) (*dto.NFTsByAddressResponse, error)
	GetNFTsByCollectionAddress(pagination dto.PaginationQuery, collectionAddress string, search string) (*dto.NFTsByAddressResponse, error)
	GetNFTMintInfo(nftAddress string) (*dto.NFTMintInfoResponse, error)
	GetNFTMutateEvents(pagination dto.PaginationQuery, nftAddress string) (*dto.NFTMutateEventsResponse, error)
	GetNFTTxs(pagination dto.PaginationQuery, nftAddress string) (*dto.NFTTxsResponse, error)
}

// nftService implements the NFTService interface
type nftService struct {
	repo repositories.NFTRepository
}

// NewNFTService creates a new instance of NFTService
func NewNFTService(repo repositories.NFTRepository) NFTService {
	return &nftService{
		repo: repo,
	}
}

// GetCollections retrieves NFT collections with pagination and search
func (s *nftService) GetCollections(pagination dto.PaginationQuery, search string) (*dto.NFTCollectionsResponse, error) {
	foundCollections, total, err := s.repo.GetCollections(pagination, search)
	if err != nil {
		return nil, err
	}

	collections := make([]dto.NFTCollectionResponse, len(foundCollections))

	for idx, collection := range foundCollections {
		collections[idx] = dto.NFTCollectionResponse{
			ObjectAddr: collection.ID,
			Collection: dto.NFTCollectionCollectionResponse{
				Creator:     collection.Creator,
				Description: collection.Description,
				Name:        collection.Name,
				URI:         collection.URI,
			},
		}
	}

	response := &dto.NFTCollectionsResponse{
		Collections: collections,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	return response, nil
}

func (s *nftService) GetNFTByNFTAddress(collectionAddress string, nftAddress string) (*dto.NFTByAddressResponse, error) {
	nft, err := s.repo.GetNFTByNFTAddress(collectionAddress, nftAddress)

	if err != nil {
		return nil, err
	}

	return &dto.NFTByAddressResponse{
		ObjectAddr:     nft.ID,
		CollectionAddr: nft.Collection,
		CollectionName: nft.CollectionName,
		OwnerAddr:      nft.Owner,
		NFT: dto.NFTByAddressNFTResponse{
			Collection: dto.NFTByAddressNFTCollectionResponse{
				Inner: nft.Collection,
			},
			Description: nft.Description,
			TokenID:     nft.TokenID,
			URI:         nft.URI,
			IsBurned:    nft.IsBurned,
		},
	}, nil
}

func (s *nftService) GetNFTsByAccountAddress(pagination dto.PaginationQuery, accountAddress string, collectionAddress string, search string) (*dto.NFTsByAddressResponse, error) {
	nfts, total, err := s.repo.GetNFTsByAccountAddress(pagination, accountAddress, collectionAddress, search)

	if err != nil {
		return nil, err
	}

	response := &dto.NFTsByAddressResponse{
		Tokens: make([]dto.NFTByAddressResponse, len(nfts)),
		Pagination: dto.PaginationResponse{
			NextKey: nil, Total: total,
		},
	}

	for idx, nft := range nfts {
		response.Tokens[idx] = dto.NFTByAddressResponse{
			ObjectAddr:     nft.ID,
			CollectionAddr: nft.Collection,
			CollectionName: nft.CollectionName,
			OwnerAddr:      nft.Owner,
			NFT: dto.NFTByAddressNFTResponse{
				Collection: dto.NFTByAddressNFTCollectionResponse{
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

func (s *nftService) GetNFTsByCollectionAddress(pagination dto.PaginationQuery, collectionAddress string, search string) (*dto.NFTsByAddressResponse, error) {
	nfts, total, err := s.repo.GetNFTsByCollectionAddress(pagination, collectionAddress, search)

	if err != nil {
		return nil, err
	}

	response := &dto.NFTsByAddressResponse{
		Tokens: make([]dto.NFTByAddressResponse, len(nfts)),
		Pagination: dto.PaginationResponse{
			NextKey: nil, Total: total,
		},
	}

	for idx, nft := range nfts {
		response.Tokens[idx] = dto.NFTByAddressResponse{
			ObjectAddr:     nft.ID,
			CollectionAddr: nft.Collection,
			CollectionName: nft.CollectionName,
			OwnerAddr:      nft.Owner,
			NFT: dto.NFTByAddressNFTResponse{
				Collection: dto.NFTByAddressNFTCollectionResponse{
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

func (s *nftService) GetNFTMintInfo(nftAddress string) (*dto.NFTMintInfoResponse, error) {
	mintInfo, err := s.repo.GetNFTMintInfo(nftAddress)
	if err != nil {
		return nil, err
	}

	return &dto.NFTMintInfoResponse{
		Height:    mintInfo.Height,
		Minter:    mintInfo.Address,
		TxHash:    mintInfo.Hash,
		Timestamp: mintInfo.Timestamp,
	}, nil
}

func (s *nftService) GetNFTMutateEvents(pagination dto.PaginationQuery, nftAddress string) (*dto.NFTMutateEventsResponse, error) {
	mutateEvents, total, err := s.repo.GetNFTMutateEvents(pagination, nftAddress)
	if err != nil {
		return nil, err
	}

	response := &dto.NFTMutateEventsResponse{
		Items: mutateEvents,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	return response, nil
}

func (s *nftService) GetNFTTxs(pagination dto.PaginationQuery, nftAddress string) (*dto.NFTTxsResponse, error) {
	txs, total, err := s.repo.GetNFTTxs(pagination, nftAddress)
	if err != nil {
		return nil, err
	}

	response := &dto.NFTTxsResponse{
		Items: make([]dto.NFTTxResponse, len(txs)),
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	for idx, tx := range txs {
		response.Items[idx] = dto.NFTTxResponse{
			IsNFTBurn:     tx.IsNFTBurn,
			IsNFTMint:     tx.IsNFTMint,
			IsNFTTransfer: tx.IsNFTTransfer,
			Timestamp:     tx.Timestamp,
			TxHash:        tx.Hash,
		}
	}

	return response, nil
}
