package dto

import "encoding/json"

type NftCollectionCollectionNft struct {
	Length int64 `json:"length"`
}

type NftCollectionCollection struct {
	Creator     string                      `json:"creator"`
	Description string                      `json:"description"`
	Name        string                      `json:"name"`
	URI         string                      `json:"uri"`
	Nft         *NftCollectionCollectionNft `json:"nft,omitempty"`
}

type NftCollectionResponse struct {
	ObjectAddr string                  `json:"object_addr"`
	Collection NftCollectionCollection `json:"collection"`
}

// NftCollectionsResponse represents the response for Nft collections list
type NftCollectionsResponse struct {
	Collections []NftCollectionResponse `json:"collections"`
	Pagination  PaginationResponse      `json:"pagination"`
}

type NftByAddressModel struct {
	TokenID        string `json:"token_id"`
	URI            string `json:"uri"`
	Description    string `json:"description"`
	IsBurned       bool   `json:"is_burned"`
	Owner          string `json:"owner"`
	ID             string `json:"id"`
	Collection     string `json:"collection"`
	CollectionName string `json:"collection_name"`
}

type NftByAddressNftCollection struct {
	Inner string `json:"inner"`
}

type NftByAddressNft struct {
	Collection  NftByAddressNftCollection `json:"collection"`
	Description string                    `json:"description"`
	TokenID     string                    `json:"token_id"`
	URI         string                    `json:"uri"`
	IsBurned    bool                      `json:"is_burned"`
}

type NftByAddressResponse struct {
	ObjectAddr     string          `json:"object_addr"`
	CollectionAddr string          `json:"collection_addr"`
	CollectionName string          `json:"collection_name"`
	OwnerAddr      string          `json:"owner_addr"`
	Nft            NftByAddressNft `json:"nft"`
}

type NftsByAddressResponse struct {
	Tokens     []NftByAddressResponse `json:"tokens"`
	Pagination PaginationResponse     `json:"pagination"`
}

type NftMintInfoModel struct {
	Address   string `json:"address"`
	Hash      string `json:"hash"`
	Height    int64  `json:"height"`
	Timestamp string `json:"timestamp"`
}

type NftMintInfoResponse struct {
	Height    int64  `json:"height"`
	Minter    string `json:"minter"`
	Timestamp string `json:"timestamp"`
	TxHash    string `json:"txhash"`
}

type MutateEventModel struct {
	MutatedFieldName string `json:"mutated_field_name"`
	NewValue         string `json:"new_value"`
	OldValue         string `json:"old_value"`
	Remark           json.RawMessage `json:"remark"`
	Timestamp        string `json:"timestamp"`
}

type NftMutateEventsResponse struct {
	NftMutateEvents []MutateEventModel `json:"nft_mutate_events"`
	Pagination      PaginationResponse `json:"pagination"`
}

type NftTxModel struct {
	IsNftBurn     bool   `json:"is_nft_burn"`
	IsNftMint     bool   `json:"is_nft_mint"`
	IsNftTransfer bool   `json:"is_nft_transfer"`
	Hash          string `json:"hash"`
	Height        int64  `json:"height"`
	Timestamp     string `json:"timestamp"`
}

type NftTx struct {
	IsNftBurn     bool   `json:"is_nft_burn"`
	IsNftMint     bool   `json:"is_nft_mint"`
	IsNftTransfer bool   `json:"is_nft_transfer"`
	Timestamp     string `json:"timestamp"`
	TxHash        string `json:"txhash"`
}

type NftTxsResponse struct {
	NftTxs     []NftTx            `json:"nft_txs"`
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
	IsNftBurn          bool   `json:"is_nft_burn"`
	IsNftMint          bool   `json:"is_nft_mint"`
	IsNftTransfer      bool   `json:"is_nft_transfer"`
	NftID              string `json:"nft_id"`
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
