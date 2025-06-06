package dto

import "github.com/initia-labs/core-indexer/pkg/db"

// NFTCollectionResponse represents the response structure for NFT collection
type NFTCollectionResponse struct {
	Collection CollectionInfo `json:"collection"`
}

// NFTCollectionsResponse represents the response for NFT collections list
type NFTCollectionsResponse struct {
	Collections []db.Collection    `json:"collections"`
	Pagination  PaginationResponse `json:"pagination"`
}

// CollectionInfo represents the collection information in the response
type CollectionInfo struct {
	ObjectAddr string         `json:"object_addr"`
	Collection CollectionData `json:"collection"`
}

// CollectionData represents the collection data in the response
type CollectionData struct {
	Creator     string `json:"creator"`
	Description string `json:"description"`
	Name        string `json:"name"`
	URI         string `json:"uri"`
	NFTs        NFTs   `json:"nfts"`
}

// NFTs represents the NFTs information in the response
type NFTs struct {
	Handle string `json:"handle"`
	Length string `json:"length"`
}
