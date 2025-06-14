package types

const (
	ModulePublishedEventKey    = "0x1::code::ModulePublishedEvent"
	CreateCollectionEventKey   = "0x1::collection::CreateCollectionEvent"
	CollectionMutationEventKey = "0x1::collection::MutationEvent"
	CollectionMintEventKey     = "0x1::collection::MintEvent"
	CollectionBurnEventKey     = "0x1::collection::BurnEvent"
	ObjectTransferEventKey     = "0x1::object::TransferEvent"
	NftMutationEventKey        = "0x1::nft::MutationEvent"
)

type CreateCollectionEvent struct {
	Collection string `json:"collection"`
	Creator    string `json:"creator"`
	Name       string `json:"name"`
}

type CollectionMutationEvent struct {
	Collection       string `json:"collection"`
	MutatedFieldName string `json:"mutated_field_name"`
	OldValue         string `json:"old_value"`
	NewValue         string `json:"new_value"`
}

type CollectionMintEvent struct {
	Collection string `json:"collection"`
	Nft        string `json:"nft"`
	TokenID    string `json:"token_id"`
}

type CollectionBurnEvent struct {
	Collection string `json:"collection"`
	Nft        string `json:"nft"`
	TokenID    string `json:"token_id"`
}

type NftMutationEvent struct {
	Nft              string `json:"nft"`
	MutatedFieldName string `json:"mutated_field_name"`
	OldValue         string `json:"old_value"`
	NewValue         string `json:"new_value"`
}

type ObjectTransferEvent struct {
	Object string `json:"object"`
	From   string `json:"from"`
	To     string `json:"to"`
}
