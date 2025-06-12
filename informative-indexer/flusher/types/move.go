package types

const (
	ModulePublishedEventKey    = "0x1::code::ModulePublishedEvent"
	CreateCollectionEventKey   = "0x1::collection::CreateCollectionEvent"
	CollectionMutationEventKey = "0x1::collection::MutationEvent"
	CollectionMintEventKey     = "0x1::collection::MintEvent"
	ObjectTransferEventKey     = "0x1::object::TransferEvent"
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

type ObjectTransferEvent struct {
	Object string `json:"object"`
	From   string `json:"from"`
	To     string `json:"to"`
}
