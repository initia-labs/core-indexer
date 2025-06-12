package dto

import "github.com/initia-labs/core-indexer/pkg/db"

// NFTCollectionsResponse represents the response for NFT collections list
type NFTCollectionsResponse struct {
	Collections []db.Collection    `json:"collections"`
	Pagination  PaginationResponse `json:"pagination"`
}

type NFTByAddressModel struct {
	TokenID        string `json:"token_id"`
	URI            string `json:"uri"`
	Description    string `json:"description"`
	IsBurned       bool   `json:"is_burned"`
	Owner          string `json:"owner"`
	ID             string `json:"id"`
	Collection     string `json:"collection"`
	CollectionName string `json:"collection_name"`
}

type NFTByAddressNFTCollectionResponse struct {
	Inner string `json:"inner"`
}

type NFTByAddressNFTResponse struct {
	Collection  NFTByAddressNFTCollectionResponse `json:"collection"`
	Description string                            `json:"description"`
	TokenID     string                            `json:"token_id"`
	URI         string                            `json:"uri"`
	IsBurned    bool                              `json:"is_burned"`
}

type NFTByAddressResponse struct {
	ObjectAddr     string                  `json:"object_addr"`
	CollectionAddr string                  `json:"collection_addr"`
	CollectionName string                  `json:"collection_name"`
	OwnerAddr      string                  `json:"owner_addr"`
	NFT            NFTByAddressNFTResponse `json:"nft"`
}

type NFTsByAddressResponse struct {
	Tokens     []NFTByAddressResponse `json:"tokens"`
	Pagination PaginationResponse     `json:"pagination"`
}
