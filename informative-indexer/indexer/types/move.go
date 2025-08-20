package types

const (
	ModulePublishedEventKey    = "0x1::code::ModulePublishedEvent"
	CollectionCreateEventKey   = "0x1::collection::CreateEvent"
	CollectionMutationEventKey = "0x1::collection::MutationEvent"
	CollectionMintEventKey     = "0x1::collection::MintEvent"
	CollectionBurnEventKey     = "0x1::collection::BurnEvent"
	NftCreateEventKey          = "0x1::nft::CreateEvent"
	NftMutationEventKey        = "0x1::nft::MutationEvent"
	ObjectTransferEventKey     = "0x1::object::TransferEvent"
	ObjectCreateEventKey       = "0x1::object::CreateEvent"
)

type CollectCreateEvent struct {
	Collection  string `json:"collection"`
	Creator     string `json:"creator"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URI         string `json:"uri"`
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

type NftCreateEvent struct {
	Collection  string `json:"collection"`
	TokenID     string `json:"token_id"`
	Description string `json:"description"`
	URI         string `json:"uri"`
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

type ObjectCreateEvent struct {
	Object  string `json:"object"`
	Owner   string `json:"owner"`
	Version string `json:"version"`
}
