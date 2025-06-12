package dto

type NFTCollectionCollectionNFT struct {
	Length int64 `json:"length"`
}

type NFTCollectionCollection struct {
	Creator     string                      `json:"creator"`
	Description string                      `json:"description"`
	Name        string                      `json:"name"`
	URI         string                      `json:"uri"`
	NFT         *NFTCollectionCollectionNFT `json:"nft,omitempty"`
}

type NFTCollectionResponse struct {
	ObjectAddr string                  `json:"object_addr"`
	Collection NFTCollectionCollection `json:"collection"`
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

type NFTByAddressNFTCollection struct {
	Inner string `json:"inner"`
}

type NFTByAddressNFT struct {
	Collection  NFTByAddressNFTCollection `json:"collection"`
	Description string                    `json:"description"`
	TokenID     string                    `json:"token_id"`
	URI         string                    `json:"uri"`
	IsBurned    bool                      `json:"is_burned"`
}

type NFTByAddressResponse struct {
	ObjectAddr     string          `json:"object_addr"`
	CollectionAddr string          `json:"collection_addr"`
	CollectionName string          `json:"collection_name"`
	OwnerAddr      string          `json:"owner_addr"`
	NFT            NFTByAddressNFT `json:"nft"`
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

type MutateEventModel struct {
	MutatedFieldName string `json:"mutated_field_name"`
	NewValue         string `json:"new_value"`
	OldValue         string `json:"old_value"`
	Remark           string `json:"remark"`
	Timestamp        string `json:"timestamp"`
}

type NFTMutateEventsResponse struct {
	NFTMutateEvents []MutateEventModel `json:"nft_mutate_events"`
	Pagination      PaginationResponse `json:"pagination"`
}

type NFTTxModel struct {
	IsNFTBurn     bool   `json:"is_nft_burn"`
	IsNFTMint     bool   `json:"is_nft_mint"`
	IsNFTTransfer bool   `json:"is_nft_transfer"`
	Hash          string `json:"hash"`
	Height        int64  `json:"height"`
	Timestamp     string `json:"timestamp"`
}

type NFTTx struct {
	IsNFTBurn     bool   `json:"is_nft_burn"`
	IsNFTMint     bool   `json:"is_nft_mint"`
	IsNFTTransfer bool   `json:"is_nft_transfer"`
	Timestamp     string `json:"timestamp"`
	TxHash        string `json:"txhash"`
}

type NFTTxsResponse struct {
	NFTTxs     []NFTTx            `json:"nft_txs"`
	Pagination PaginationResponse `json:"pagination"`
}

type CollectionByAccountAddressModel struct {
	Name        string `json:"name"`
	URI         string `json:"uri"`
	Description string `json:"description"`
	ID          string `json:"id"`
	Count       int64  `json:"count"`
	Creator     string `json:"creator"`
}

type CollectionActivityModel struct {
	Hash               string `json:"hash"`
	Timestamp          string `json:"timestamp"`
	IsNFTBurn          bool   `json:"is_nft_burn"`
	IsNFTMint          bool   `json:"is_nft_mint"`
	IsNFTTransfer      bool   `json:"is_nft_transfer"`
	NFTID              string `json:"nft_id"`
	TokenID            string `json:"token_id"`
	IsCollectionCreate bool   `json:"is_collection_create"`
}

type CollectionActivitiesResponse struct {
	CollectionActivities []CollectionActivityModel `json:"collection_activities"`
	Pagination           PaginationResponse        `json:"pagination"`
}

type CollectionCreatorModel struct {
	Height    int64  `json:"height"`
	Timestamp string `json:"timestamp"`
	Creator   string `json:"creator"`
	Hash      string `json:"hash"`
}

type CollectionCreatorResponse struct {
	Creator CollectionCreatorModel `json:"creator"`
}

type CollectionMutateEventsResponse struct {
	CollectionMutateEvents []MutateEventModel `json:"collection_mutate_events"`
	Pagination             PaginationResponse `json:"pagination"`
}
