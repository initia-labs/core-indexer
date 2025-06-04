package dto

// NFTCollection represents an NFT collection in the response
type NFTCollection struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URI         string `json:"uri"`
	Description string `json:"description"`
	Creator     string `json:"creator"`
}

// NFTCollectionsResponse represents the response for NFT collections list
type NFTCollectionsResponse struct {
	Collections []NFTCollection    `json:"collections"`
	Pagination  PaginationResponse `json:"pagination"`
}

type NFTByAddress struct {
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
