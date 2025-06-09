package dto

type NFTCollectionCollectionResponse struct {
	Creator     string `json:"creator"`
	Description string `json:"description"`
	Name        string `json:"name"`
	URI         string `json:"uri"`
}

type NFTCollectionResponse struct {
	ObjectAddr string                          `json:"object_addr"`
	Collection NFTCollectionCollectionResponse `json:"collection"`
}

// NFTCollectionsResponse represents the response for NFT collections list
type NFTCollectionsResponse struct {
	Collections []NFTCollectionResponse `json:"collections"`
	Pagination  PaginationResponse      `json:"pagination"`
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

type NFTMintInfoModel struct {
	Address   string `json:"address"`
	Hash      string `json:"hash"`
	Height    int64  `json:"height"`
	Timestamp string `json:"timestamp"`
}

type NFTMintInfoResponse struct {
	Height    int64  `json:"height"`
	Minter    string `json:"minter"`
	Timestamp string `json:"timestamp"`
	TxHash    string `json:"txhash"`
}

type NFTMutateEventResponse struct {
	MutatedFieldName string `json:"mutated_field_name"`
	NewValue         string `json:"new_value"`
	OldValue         string `json:"old_value"`
	Remark           string `json:"remark"`
	Timestamp        string `json:"timestamp"`
}

type NFTMutateEventsResponse struct {
	Items      []NFTMutateEventResponse `json:"items"`
	Pagination PaginationResponse       `json:"pagination"`
}

type NFTTx struct {
	IsNFTBurn     bool   `json:"is_nft_burn"`
	IsNFTMint     bool   `json:"is_nft_mint"`
	IsNFTTransfer bool   `json:"is_nft_transfer"`
	Hash          string `json:"hash"`
	Height        int64  `json:"height"`
	Timestamp     string `json:"timestamp"`
}

type NFTTxResponse struct {
	IsNFTBurn     bool   `json:"is_nft_burn"`
	IsNFTMint     bool   `json:"is_nft_mint"`
	IsNFTTransfer bool   `json:"is_nft_transfer"`
	Timestamp     string `json:"timestamp"`
	TxHash        string `json:"txhash"`
}

type NFTTxsResponse struct {
	Items      []NFTTxResponse    `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

type CollectionByAccountAddressModel struct {
	Name  string `json:"name"`
	URI   string `json:"uri"`
	ID    string `json:"id"`
	Count int64  `json:"count"`
}

type CollectionByAccountAddressResponse struct {
	CollectionAddress string `json:"collection_address"`
}
