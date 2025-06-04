package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"database/sql/driver"
)

const (
	TableNameAccountTransaction         = "account_transactions"
	TableNameAccount                    = "accounts"
	TableNameBlock                      = "blocks"
	TableNameCollectionMutationEvent    = "collection_mutation_events"
	TableNameCollectionProposal         = "collection_proposals"
	TableNameCollectionTransaction      = "collection_transactions"
	TableNameCollection                 = "collections"
	TableNameFinalizeBlockEvent         = "finalize_block_events"
	TableNameLcdTxResult                = "lcd_tx_results"
	TableNameModuleHistory              = "module_histories"
	TableNameModuleProposal             = "module_proposals"
	TableNameModuleTransaction          = "module_transactions"
	TableNameModule                     = "modules"
	TableNameMoveEvent                  = "move_events"
	TableNameNftHistory                 = "nft_histories"
	TableNameNftMutationEvent           = "nft_mutation_events"
	TableNameNftProposal                = "nft_proposals"
	TableNameNftTransaction             = "nft_transactions"
	TableNameNft                        = "nfts"
	TableNameOpinitTransaction          = "opinit_transactions"
	TableNameProposalDeposit            = "proposal_deposits"
	TableNameProposalVote               = "proposal_votes"
	TableNameProposalVotesLegacy        = "proposal_votes_legacy"
	TableNameProposal                   = "proposals"
	TableNameSchemaMigration            = "schema_migrations"
	TableNameTracking                   = "tracking"
	TableNameTransactionEvent           = "transaction_events"
	TableNameTransaction                = "transactions"
	TableNameValidatorBondedTokenChange = "validator_bonded_token_changes"
	TableNameValidatorCommitSignature   = "validator_commit_signatures"
	TableNameValidatorHistoricalPower   = "validator_historical_powers"
	TableNameValidatorSlashEvent        = "validator_slash_events"
	TableNameValidatorVoteCount         = "validator_vote_counts"
	TableNameValidator                  = "validators"
	TableNameVMAddress                  = "vm_addresses"
)

// AccountTransaction mapped from table <account_transactions>
type AccountTransaction struct {
	IsSigner      bool   `gorm:"column:is_signer;not null" json:"is_signer"`
	BlockHeight   int64  `gorm:"column:block_height;not null;index:ix_account_transactions_account_id_block_height_desc,priority:2,sort:desc" json:"block_height"`
	TransactionID string `gorm:"column:transaction_id;primaryKey;type:character varying" json:"transaction_id"`
	AccountID     string `gorm:"column:account_id;primaryKey;type:character varying;index:ix_account_transactions_account_id_block_height_desc,priority:1" json:"account_id"`

	// Foreign key relationships
	Account     Account     `gorm:"foreignKey:AccountID;references:Address" json:"-"`
	Block       Block       `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	Transaction Transaction `gorm:"foreignKey:TransactionID;references:ID" json:"-"`
}

// TableName AccountTransaction's table name
func (*AccountTransaction) TableName() string {
	return TableNameAccountTransaction
}

// Account mapped from table <accounts>
type Account struct {
	Address     string `gorm:"column:address;primaryKey;type:character varying" json:"address"`
	Type        string `gorm:"column:type;type:accounttype" json:"type"`
	Name        string `gorm:"column:name;type:character varying" json:"name"`
	VMAddressID string `gorm:"column:vm_address_id;type:character varying" json:"vm_address_id"`

	// Foreign key relationship
	VMAddress VMAddress `gorm:"foreignKey:VMAddressID;references:VMAddress" json:"-"`
}

// TableName Account's table name
func (*Account) TableName() string {
	return TableNameAccount
}

// Block mapped from table <blocks>
type Block struct {
	Height    int32     `gorm:"column:height;primaryKey;autoIncrement:true" json:"height"`
	Timestamp time.Time `gorm:"column:timestamp;not null;type:timestamp;index:ix_blocks_timestamp" json:"timestamp"`
	Proposer  string    `gorm:"column:proposer;type:character varying" json:"proposer"`
	Hash      []byte    `gorm:"column:hash;not null" json:"hash"`

	// Foreign key relationship
	ProposerValidator Validator `gorm:"foreignKey:Proposer;references:OperatorAddress" json:"-"`
}

// TableName Block's table name
func (*Block) TableName() string {
	return TableNameBlock
}

// CollectionMutationEvent mapped from table <collection_mutation_events>
type CollectionMutationEvent struct {
	MutatedFieldName string `gorm:"column:mutated_field_name;not null;type:character varying" json:"mutated_field_name"`
	OldValue         string `gorm:"column:old_value;not null;type:character varying" json:"old_value"`
	NewValue         string `gorm:"column:new_value;not null;type:character varying" json:"new_value"`
	BlockHeight      int32  `gorm:"column:block_height;not null;index:ix_collection_mutation_events_block_height" json:"block_height"`
	Remark           JSON   `gorm:"column:remark;type:json;not null" json:"remark"`
	ProposalID       int32  `gorm:"column:proposal_id" json:"proposal_id"`
	TxID             string `gorm:"column:tx_id;type:character varying" json:"tx_id"`
	CollectionID     string `gorm:"column:collection_id;type:character varying" json:"collection_id"`

	// Foreign key relationships
	Block       Block       `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	Collection  Collection  `gorm:"foreignKey:CollectionID;references:ID" json:"-"`
	Proposal    Proposal    `gorm:"foreignKey:ProposalID;references:ID" json:"-"`
	Transaction Transaction `gorm:"foreignKey:TxID;references:ID" json:"-"`
}

// TableName CollectionMutationEvent's table name
func (*CollectionMutationEvent) TableName() string {
	return TableNameCollectionMutationEvent
}

// CollectionProposal mapped from table <collection_proposals>
type CollectionProposal struct {
	ProposalID   int32  `gorm:"column:proposal_id;not null" json:"proposal_id"`
	CollectionID string `gorm:"column:collection_id" json:"collection_id"`
	NftID        string `gorm:"column:nft_id" json:"nft_id"`

	// Foreign key relationships
	Collection Collection `gorm:"foreignKey:CollectionID;references:ID" json:"-"`
	Nft        Nft        `gorm:"foreignKey:NftID;references:ID" json:"-"`
	Proposal   Proposal   `gorm:"foreignKey:ProposalID;references:ID" json:"-"`
}

// TableName CollectionProposal's table name
func (*CollectionProposal) TableName() string {
	return TableNameCollectionProposal
}

// CollectionTransaction mapped from table <collection_transactions>
type CollectionTransaction struct {
	IsNftTransfer      bool   `gorm:"column:is_nft_transfer;not null" json:"is_nft_transfer"`
	IsNftMint          bool   `gorm:"column:is_nft_mint;not null" json:"is_nft_mint"`
	IsNftBurn          bool   `gorm:"column:is_nft_burn;not null" json:"is_nft_burn"`
	IsCollectionCreate bool   `gorm:"column:is_collection_create;not null" json:"is_collection_create"`
	BlockHeight        int32  `gorm:"column:block_height;not null;index:ix_collection_transactions_block_height;index:ix_collection_transactions_collection_id_block_height,priority:2" json:"block_height"`
	TxID               string `gorm:"column:tx_id;type:character varying;index:ix_collection_transactions_tx_id" json:"tx_id"`
	CollectionID       string `gorm:"column:collection_id;type:character varying;index:ix_collection_transactions_collection_id;index:ix_collection_transactions_collection_id_block_height,priority:1" json:"collection_id"`
	NftID              string `gorm:"column:nft_id;type:character varying;index:ix_collection_transactions_nft_id" json:"nft_id"`

	// Foreign key relationships
	Block       Block       `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	Collection  Collection  `gorm:"foreignKey:CollectionID;references:ID" json:"-"`
	Nft         Nft         `gorm:"foreignKey:NftID;references:ID" json:"-"`
	Transaction Transaction `gorm:"foreignKey:TxID;references:ID" json:"-"`
}

// TableName CollectionTransaction's table name
func (*CollectionTransaction) TableName() string {
	return TableNameCollectionTransaction
}

// Collection mapped from table <collections>
type Collection struct {
	BlockHeight int32  `gorm:"column:block_height;not null;index:ix_collections_block_height" json:"block_height"`
	URI         string `gorm:"column:uri;not null;type:character varying" json:"uri"`
	Description string `gorm:"column:description;not null;type:character varying" json:"description"`
	Name        string `gorm:"column:name;not null;type:character varying" json:"name"`
	ID          string `gorm:"column:id;primaryKey;type:text" json:"id"`
	Creator     string `gorm:"column:creator;type:text;index:ix_collections_creator" json:"creator"`

	// Foreign key relationships
	Block          Block     `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	CollectionAddr VMAddress `gorm:"foreignKey:ID;references:VMAddress" json:"-"`
	CreatorAddr    VMAddress `gorm:"foreignKey:Creator;references:VMAddress" json:"-"`
}

// TableName Collection's table name
func (*Collection) TableName() string {
	return TableNameCollection
}

// FinalizeBlockEvent mapped from table <finalize_block_events>
type FinalizeBlockEvent struct {
	BlockHeight int64  `gorm:"column:block_height;primaryKey;index:ix_finalize_block_events_event_key_block_height_desc,priority:2,sort:desc" json:"block_height"`
	EventKey    string `gorm:"column:event_key;not null;type:character varying;index:ix_finalize_block_events_event_key_block_height_desc,priority:1" json:"event_key"`
	EventValue  string `gorm:"column:event_value;not null;type:character varying" json:"event_value"`
	EventIndex  int    `gorm:"column:event_index;primaryKey;type:integer" json:"event_index"`
	Mode        string `gorm:"column:mode;not null;type:finalize_block_events_mode" json:"mode"`
}

// TableName FinalizeBlockEvent's table name
func (*FinalizeBlockEvent) TableName() string {
	return TableNameFinalizeBlockEvent
}

// LcdTxResult mapped from table <lcd_tx_results>
type LcdTxResult struct {
	BlockHeight   int32  `gorm:"column:block_height;not null;index:ix_lcd_tx_results_block_height" json:"block_height"`
	Result        JSON   `gorm:"column:result;not null;type:json" json:"result"`
	TransactionID string `gorm:"column:transaction_id;type:character varying" json:"transaction_id"`

	// Foreign key relationships
	Block       Block       `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	Transaction Transaction `gorm:"foreignKey:TransactionID;references:ID" json:"-"`
}

// TableName LcdTxResult's table name
func (*LcdTxResult) TableName() string {
	return TableNameLcdTxResult
}

// ModuleHistory mapped from table <module_histories>
type ModuleHistory struct {
	UpgradePolicy string `gorm:"column:upgrade_policy;not null;type:upgradepolicy" json:"upgrade_policy"`
	BlockHeight   int32  `gorm:"column:block_height;not null;index:ix_module_histories_block_height" json:"block_height"`
	Remark        JSON   `gorm:"column:remark;not null;type:json" json:"remark"`
	ProposalID    int32  `gorm:"column:proposal_id" json:"proposal_id"`
	TxID          string `gorm:"column:tx_id;type:character varying" json:"tx_id"`
	ModuleID      string `gorm:"column:module_id;type:character varying;index:ix_module_histories_module_id" json:"module_id"`

	// Foreign key relationships
	Block       Block       `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	Module      Module      `gorm:"foreignKey:ModuleID;references:ID" json:"-"`
	Proposal    Proposal    `gorm:"foreignKey:ProposalID;references:ID" json:"-"`
	Transaction Transaction `gorm:"foreignKey:TxID;references:ID" json:"-"`
}

// TableName ModuleHistory's table name
func (*ModuleHistory) TableName() string {
	return TableNameModuleHistory
}

// ModuleProposal mapped from table <module_proposals>
type ModuleProposal struct {
	ProposalID int32  `gorm:"column:proposal_id;primaryKey" json:"proposal_id"`
	ModuleID   string `gorm:"column:module_id;primaryKey;type:character varying" json:"module_id"`

	// Foreign key relationships
	Module   Module   `gorm:"foreignKey:ModuleID;references:ID" json:"-"`
	Proposal Proposal `gorm:"foreignKey:ProposalID;references:ID" json:"-"`
}

// TableName ModuleProposal's table name
func (*ModuleProposal) TableName() string {
	return TableNameModuleProposal
}

// ModuleTransaction mapped from table <module_transactions>
type ModuleTransaction struct {
	IsEntry     bool   `gorm:"column:is_entry;not null" json:"is_entry"`
	BlockHeight int32  `gorm:"column:block_height;not null;index:ix_module_transactions_block_height;index:ix_module_transactions_module_id_block_height_desc,priority:2,sort:desc" json:"block_height"`
	TxID        string `gorm:"column:tx_id;type:character varying" json:"tx_id"`
	ModuleID    string `gorm:"column:module_id;type:character varying;index:ix_module_transactions_module_id_block_height_desc,priority:1" json:"module_id"`

	// Foreign key relationships
	Block       Block       `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	Module      Module      `gorm:"foreignKey:ModuleID;references:ID" json:"-"`
	Transaction Transaction `gorm:"foreignKey:TxID;references:ID" json:"-"`
}

// TableName ModuleTransaction's table name
func (*ModuleTransaction) TableName() string {
	return TableNameModuleTransaction
}

// Module mapped from table <modules>
type Module struct {
	Name                string `gorm:"column:name;not null;type:varchar" json:"name"`
	UpgradePolicy       string `gorm:"column:upgrade_policy;not null;type:upgradepolicy" json:"upgrade_policy"`
	ModuleEntryExecuted int32  `gorm:"column:module_entry_executed;not null" json:"module_entry_executed"`
	IsVerify            bool   `gorm:"column:is_verify;not null" json:"is_verify"`
	PublishTxID         string `gorm:"column:publish_tx_id;type:character varying" json:"publish_tx_id"`
	PublisherID         string `gorm:"column:publisher_id;type:character varying" json:"publisher_id"`
	ID                  string `gorm:"column:id;primaryKey;type:character varying" json:"id"`
	Digest              string `gorm:"column:digest;type:character varying" json:"digest"`

	// Foreign key relationships
	PublishTx Transaction `gorm:"foreignKey:PublishTxID;references:ID" json:"-"`
	Publisher VMAddress   `gorm:"foreignKey:PublisherID;references:VMAddress" json:"-"`
}

// TableName Module's table name
func (*Module) TableName() string {
	return TableNameModule
}

// JSONB custom type for PostgreSQL JSONB fields
type JSONB json.RawMessage

// Scan scans value into JSONB, implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	result := json.RawMessage{}
	err := json.Unmarshal(bytes, &result)
	*j = JSONB(result)
	return err
}

// Value return jsonb value, implement driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// MoveEvent mapped from table <move_events>
type MoveEvent struct {
	TypeTag         string `gorm:"column:type_tag;not null;type:character varying;index:ix_move_events_type_tag_block_height_desc,priority:1" json:"type_tag"`
	Data            JSONB  `gorm:"column:data;not null;type:jsonb" json:"data"`
	BlockHeight     int64  `gorm:"column:block_height;primaryKey;index:ix_move_events_type_tag_block_height_desc,priority:2,sort:desc" json:"block_height"`
	TransactionHash string `gorm:"column:transaction_hash;primaryKey;type:character varying" json:"transaction_hash"`
	EventIndex      int    `gorm:"column:event_index;primaryKey;type:integer" json:"event_index"`
}

// TableName MoveEvent's table name
func (*MoveEvent) TableName() string {
	return TableNameMoveEvent
}

// NftHistory mapped from table <nft_histories>
type NftHistory struct {
	BlockHeight int32  `gorm:"column:block_height;not null;index:ix_nft_histories_block_height" json:"block_height"`
	Remark      JSON   `gorm:"column:remark;type:json;not null" json:"remark"`
	ProposalID  int32  `gorm:"column:proposal_id" json:"proposal_id"`
	TxID        string `gorm:"column:tx_id;type:character varying" json:"tx_id"`
	From        string `gorm:"column:from;type:character varying" json:"from"`
	To          string `gorm:"column:to;type:character varying" json:"to"`
	NftID       string `gorm:"column:nft_id;type:character varying" json:"nft_id"`

	// Foreign key relationships
	Block       Block       `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	FromAddr    VMAddress   `gorm:"foreignKey:From;references:VMAddress" json:"-"`
	Nft         Nft         `gorm:"foreignKey:NftID;references:ID" json:"-"`
	Proposal    Proposal    `gorm:"foreignKey:ProposalID;references:ID" json:"-"`
	ToAddr      VMAddress   `gorm:"foreignKey:To;references:VMAddress" json:"-"`
	Transaction Transaction `gorm:"foreignKey:TxID;references:ID" json:"-"`
}

// TableName NftHistory's table name
func (*NftHistory) TableName() string {
	return TableNameNftHistory
}

// NftMutationEvent mapped from table <nft_mutation_events>
type NftMutationEvent struct {
	MutatedFieldName string `gorm:"column:mutated_field_name;not null;type:character varying" json:"mutated_field_name"`
	OldValue         string `gorm:"column:old_value;not null;type:character varying" json:"old_value"`
	NewValue         string `gorm:"column:new_value;not null;type:character varying" json:"new_value"`
	BlockHeight      int32  `gorm:"column:block_height;not null;index:ix_nft_mutation_events_block_height" json:"block_height"`
	Remark           JSON   `gorm:"column:remark;type:json;not null" json:"remark"`
	ProposalID       int32  `gorm:"column:proposal_id" json:"proposal_id"`
	TxID             string `gorm:"column:tx_id;type:character varying" json:"tx_id"`
	NftID            string `gorm:"column:nft_id;type:character varying" json:"nft_id"`

	// Foreign key relationships
	Block       Block       `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	Nft         Nft         `gorm:"foreignKey:NftID;references:ID" json:"-"`
	Proposal    Proposal    `gorm:"foreignKey:ProposalID;references:ID" json:"-"`
	Transaction Transaction `gorm:"foreignKey:TxID;references:ID" json:"-"`
}

// TableName NftMutationEvent's table name
func (*NftMutationEvent) TableName() string {
	return TableNameNftMutationEvent
}

// NftProposal mapped from table <nft_proposals>
type NftProposal struct {
	NftID      string `gorm:"column:nft_id;type:character varying" json:"nft_id"`
	ProposalID int32  `gorm:"column:proposal_id;not null" json:"proposal_id"`

	// Foreign key relationships
	Nft      Nft      `gorm:"foreignKey:NftID;references:ID" json:"-"`
	Proposal Proposal `gorm:"foreignKey:ProposalID;references:ID" json:"-"`
}

// TableName NftProposal's table name
func (*NftProposal) TableName() string {
	return TableNameNftProposal
}

// NftTransaction mapped from table <nft_transactions>
type NftTransaction struct {
	IsNftTransfer bool   `gorm:"column:is_nft_transfer;not null" json:"is_nft_transfer"`
	IsNftMint     bool   `gorm:"column:is_nft_mint;not null" json:"is_nft_mint"`
	IsNftBurn     bool   `gorm:"column:is_nft_burn;not null" json:"is_nft_burn"`
	BlockHeight   int32  `gorm:"column:block_height;not null;index:ix_nft_transactions_block_height;index:ix_nft_transactions_nft_id_block_height_desc,priority:2,sort:desc" json:"block_height"`
	TxID          string `gorm:"column:tx_id;type:character varying" json:"tx_id"`
	NftID         string `gorm:"column:nft_id;type:character varying;index:ix_nft_transactions_nft_id_block_height_desc,priority:1" json:"nft_id"`

	// Foreign key relationships
	Block       Block       `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	Nft         Nft         `gorm:"foreignKey:NftID;references:ID" json:"-"`
	Transaction Transaction `gorm:"foreignKey:TxID;references:ID" json:"-"`
}

// TableName NftTransaction's table name
func (*NftTransaction) TableName() string {
	return TableNameNftTransaction
}

// Nft mapped from table <nfts>
type Nft struct {
	URI         string `gorm:"column:uri;not null;type:character varying" json:"uri"`
	Description string `gorm:"column:description;not null;type:character varying" json:"description"`
	TokenID     string `gorm:"column:token_id;not null;type:character varying" json:"token_id"`
	Remark      JSON   `gorm:"column:remark;type:json;not null" json:"remark"`
	ProposalID  int32  `gorm:"column:proposal_id" json:"proposal_id"`
	TxID        string `gorm:"column:tx_id;type:character varying;index:ix_nfts_tx_id" json:"tx_id"`
	Owner       string `gorm:"column:owner;type:character varying;index:ix_nfts_owner" json:"owner"`
	ID          string `gorm:"column:id;primaryKey;type:character varying" json:"id"`
	Collection  string `gorm:"column:collection;type:character varying;index:ix_nfts_collection" json:"collection"`
	IsBurned    bool   `gorm:"column:is_burned;not null" json:"is_burned"`

	// Foreign key relationships
	CollectionRef Collection  `gorm:"foreignKey:Collection;references:ID" json:"-"`
	NftAddr       VMAddress   `gorm:"foreignKey:ID;references:VMAddress" json:"-"`
	OwnerAddr     VMAddress   `gorm:"foreignKey:Owner;references:VMAddress" json:"-"`
	Proposal      Proposal    `gorm:"foreignKey:ProposalID;references:ID" json:"-"`
	TxRef         Transaction `gorm:"foreignKey:TxID;references:ID" json:"-"`
}

// TableName Nft's table name
func (*Nft) TableName() string {
	return TableNameNft
}

// OpinitTransaction mapped from table <opinit_transactions>
type OpinitTransaction struct {
	BridgeID    int32  `gorm:"column:bridge_id;not null" json:"bridge_id"`
	BlockHeight int32  `gorm:"column:block_height;not null;index:ix_opinit_transactions_block_height" json:"block_height"`
	TxID        string `gorm:"column:tx_id;type:character varying" json:"tx_id"`

	// Foreign key relationships
	Block       Block       `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	Transaction Transaction `gorm:"foreignKey:TxID;references:ID" json:"-"`
}

// TableName OpinitTransaction's table name
func (*OpinitTransaction) TableName() string {
	return TableNameOpinitTransaction
}

// ProposalDeposit mapped from table <proposal_deposits>
type ProposalDeposit struct {
	ProposalID    int32  `gorm:"column:proposal_id;not null" json:"proposal_id"`
	Amount        JSON   `gorm:"column:amount;not null;type:json" json:"amount"`
	TransactionID string `gorm:"column:transaction_id;type:character varying" json:"transaction_id"`
	Depositor     string `gorm:"column:depositor;type:character varying" json:"depositor"`

	// Foreign key relationships
	DepositorAccount Account     `gorm:"foreignKey:Depositor;references:Address" json:"-"`
	Proposal         Proposal    `gorm:"foreignKey:ProposalID;references:ID" json:"-"`
	Transaction      Transaction `gorm:"foreignKey:TransactionID;references:ID" json:"-"`
}

// TableName ProposalDeposit's table name
func (*ProposalDeposit) TableName() string {
	return TableNameProposalDeposit
}

// ProposalVote mapped from table <proposal_votes>
type ProposalVote struct {
	ProposalID       int32   `gorm:"column:proposal_id;not null" json:"proposal_id"`
	IsVoteWeighted   bool    `gorm:"column:is_vote_weighted;not null" json:"is_vote_weighted"`
	IsValidator      bool    `gorm:"column:is_validator;not null" json:"is_validator"`
	ValidatorAddress string  `gorm:"column:validator_address;type:character varying" json:"validator_address"`
	Yes              float64 `gorm:"column:yes;not null" json:"yes"`
	No               float64 `gorm:"column:no;not null" json:"no"`
	Abstain          float64 `gorm:"column:abstain;not null" json:"abstain"`
	NoWithVeto       float64 `gorm:"column:no_with_veto;not null" json:"no_with_veto"`
	TransactionID    string  `gorm:"column:transaction_id;type:character varying" json:"transaction_id"`
	Voter            string  `gorm:"column:voter;type:character varying" json:"voter"`

	// Foreign key relationships
	Proposal     Proposal    `gorm:"foreignKey:ProposalID;references:ID" json:"-"`
	Transaction  Transaction `gorm:"foreignKey:TransactionID;references:ID" json:"-"`
	Validator    Validator   `gorm:"foreignKey:ValidatorAddress;references:OperatorAddress" json:"-"`
	VoterAccount Account     `gorm:"foreignKey:Voter;references:Address" json:"-"`
}

// TableName ProposalVote's table name
func (*ProposalVote) TableName() string {
	return TableNameProposalVote
}

// ProposalVotesLegacy mapped from table <proposal_votes_legacy>
type ProposalVotesLegacy struct {
	ProposalID       int32   `gorm:"column:proposal_id;not null" json:"proposal_id"`
	IsVoteWeighted   bool    `gorm:"column:is_vote_weighted;not null" json:"is_vote_weighted"`
	IsValidator      bool    `gorm:"column:is_validator;not null" json:"is_validator"`
	ValidatorAddress string  `gorm:"column:validator_address;type:character varying" json:"validator_address"`
	Yes              float64 `gorm:"column:yes;not null" json:"yes"`
	No               float64 `gorm:"column:no;not null" json:"no"`
	Abstain          float64 `gorm:"column:abstain;not null" json:"abstain"`
	NoWithVeto       float64 `gorm:"column:no_with_veto;not null" json:"no_with_veto"`
	TransactionID    string  `gorm:"column:transaction_id;type:character varying" json:"transaction_id"`
	Voter            string  `gorm:"column:voter;type:character varying" json:"voter"`

	// Foreign key relationships
	Proposal     Proposal    `gorm:"foreignKey:ProposalID;references:ID" json:"-"`
	Transaction  Transaction `gorm:"foreignKey:TransactionID;references:ID" json:"-"`
	Validator    Validator   `gorm:"foreignKey:ValidatorAddress;references:OperatorAddress" json:"-"`
	VoterAccount Account     `gorm:"foreignKey:Voter;references:Address" json:"-"`
}

// TableName ProposalVotesLegacy's table name
func (*ProposalVotesLegacy) TableName() string {
	return TableNameProposalVotesLegacy
}

// JSON custom type for JSON fields
type JSON json.RawMessage

// Scan scans value into JSON, implements sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	result := json.RawMessage{}
	err := json.Unmarshal(bytes, &result)
	*j = JSON(result)
	return err
}

// Value return json value, implement driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// Proposal mapped from table <proposals>
type Proposal struct {
	ID                     int32     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	Type                   string    `gorm:"column:type;not null;type:character varying" json:"type"`
	Title                  string    `gorm:"column:title;not null;type:character varying" json:"title"`
	Description            string    `gorm:"column:description;not null;type:character varying" json:"description"`
	ProposalRoute          string    `gorm:"column:proposal_route;not null;type:character varying" json:"proposal_route"`
	Status                 string    `gorm:"column:status;not null;type:proposalstatus" json:"status"`
	SubmitTime             time.Time `gorm:"column:submit_time;not null;type:timestamp" json:"submit_time"`
	DepositEndTime         time.Time `gorm:"column:deposit_end_time;not null;type:timestamp" json:"deposit_end_time"`
	VotingTime             time.Time `gorm:"column:voting_time;type:timestamp" json:"voting_time"`
	VotingEndTime          time.Time `gorm:"column:voting_end_time;type:timestamp" json:"voting_end_time"`
	Content                JSON      `gorm:"column:content;type:json" json:"content"`
	TotalDeposit           JSON      `gorm:"column:total_deposit;not null;type:json" json:"total_deposit"`
	Yes                    int64     `gorm:"column:yes;not null" json:"yes"`
	No                     int64     `gorm:"column:no;not null" json:"no"`
	Abstain                int64     `gorm:"column:abstain;not null" json:"abstain"`
	NoWithVeto             int64     `gorm:"column:no_with_veto;not null" json:"no_with_veto"`
	IsExpedited            bool      `gorm:"column:is_expedited;not null" json:"is_expedited"`
	Version                string    `gorm:"column:version;not null;type:character varying" json:"version"`
	ResolvedHeight         int32     `gorm:"column:resolved_height;index:ix_proposals_resolved_height" json:"resolved_height"`
	Types                  JSON      `gorm:"column:types;not null;type:json" json:"types"`
	Messages               JSON      `gorm:"column:messages;not null;type:json" json:"messages"`
	CreatedHeight          int32     `gorm:"column:created_height" json:"created_height"`
	Metadata               string    `gorm:"column:metadata;not null;type:character varying" json:"metadata"`
	FailedReason           string    `gorm:"column:failed_reason;not null;type:character varying;default:''" json:"failed_reason"`
	ResolvedVotingPower    int64     `gorm:"column:resolved_voting_power" json:"resolved_voting_power"`
	CreatedTx              string    `gorm:"column:created_tx;type:character varying" json:"created_tx"`
	ProposerID             string    `gorm:"column:proposer_id;type:character varying" json:"proposer_id"`
	IsEmergency            bool      `gorm:"column:is_emergency;not null;default:false" json:"is_emergency"`
	EmergencyStartTime     time.Time `gorm:"column:emergency_start_time;type:timestamp" json:"emergency_start_time"`
	EmergencyNextTallyTime time.Time `gorm:"column:emergency_next_tally_time;type:timestamp" json:"emergency_next_tally_time"`

	// Foreign key relationships
	CreatedBlock  Block       `gorm:"foreignKey:CreatedHeight;references:Height" json:"-"`
	ResolvedBlock Block       `gorm:"foreignKey:ResolvedHeight;references:Height" json:"-"`
	CreatedTxRef  Transaction `gorm:"foreignKey:CreatedTx;references:ID" json:"-"`
	Proposer      Account     `gorm:"foreignKey:ProposerID;references:Address" json:"-"`
}

// TableName Proposal's table name
func (*Proposal) TableName() string {
	return TableNameProposal
}

// SchemaMigration mapped from table <schema_migrations>
type SchemaMigration struct {
	Version int64 `gorm:"column:version;primaryKey" json:"version"`
	Dirty   bool  `gorm:"column:dirty;not null" json:"dirty"`
}

// TableName SchemaMigration's table name
func (*SchemaMigration) TableName() string {
	return TableNameSchemaMigration
}

// Tracking mapped from table <tracking>
type Tracking struct {
	ChainID                      string `gorm:"column:chain_id;primaryKey;type:character varying" json:"chain_id"`
	Topic                        string `gorm:"column:topic;not null;type:character varying" json:"topic"`
	KafkaOffset                  int32  `gorm:"column:kafka_offset;not null" json:"kafka_offset"`
	ReplayTopic                  string `gorm:"column:replay_topic;not null;type:character varying" json:"replay_topic"`
	ReplayOffset                 int32  `gorm:"column:replay_offset;not null" json:"replay_offset"`
	LatestInformativeBlockHeight int32  `gorm:"column:latest_informative_block_height" json:"latest_informative_block_height"`
	TxCount                      int64  `gorm:"column:tx_count;not null;default:0" json:"tx_count"`
}

// TableName Tracking's table name
func (*Tracking) TableName() string {
	return TableNameTracking
}

// TransactionEvent mapped from table <transaction_events>
type TransactionEvent struct {
	BlockHeight     int64  `gorm:"column:block_height;primaryKey;index:ix_transaction_events_block_height_desc,sort:desc;index:ix_transactions_events_event_key_block_height_desc,sort:desc,priority:2" json:"block_height"`
	TransactionHash string `gorm:"column:transaction_hash;primaryKey;type:character varying" json:"transaction_hash"`
	EventIndex      int    `gorm:"column:event_index;primaryKey;type:integer" json:"event_index"`
	EventKey        string `gorm:"column:event_key;not null;type:character varying;index:ix_transactions_events_event_key_block_height_desc,priority:1" json:"event_key"`
	EventValue      string `gorm:"column:event_value;not null;type:character varying" json:"event_value"`
}

// TableName TransactionEvent's table name
func (*TransactionEvent) TableName() string {
	return TableNameTransactionEvent
}

// Transaction mapped from table <transactions>
type Transaction struct {
	Hash               []byte          `gorm:"column:hash;not null" json:"hash"`
	BlockHeight        int64           `gorm:"column:block_height;not null;type:bigint;index:ix_transactions_block_height;index:ix_transactions_block_height_block_index,priority:1,sort:desc" json:"block_height"`
	GasUsed            int64           `gorm:"column:gas_used;not null;type:integer" json:"gas_used"`
	GasLimit           int64           `gorm:"column:gas_limit;not null;type:integer" json:"gas_limit"`
	GasFee             string          `gorm:"column:gas_fee;not null;type:character varying" json:"gas_fee"`
	ErrMsg             string          `gorm:"column:err_msg;type:character varying" json:"err_msg"`
	Success            bool            `gorm:"column:success;not null" json:"success"`
	Memo               string          `gorm:"column:memo;not null;type:character varying" json:"memo"`
	Messages           json.RawMessage `gorm:"column:messages;not null;type:json" json:"messages"`
	IsIbc              bool            `gorm:"column:is_ibc;not null" json:"is_ibc"`
	IsSend             bool            `gorm:"column:is_send;not null" json:"is_send"`
	IsMovePublish      bool            `gorm:"column:is_move_publish;not null" json:"is_move_publish"`
	IsMoveExecuteEvent bool            `gorm:"column:is_move_execute_event;not null" json:"is_move_execute_event"`
	IsMoveExecute      bool            `gorm:"column:is_move_execute;not null" json:"is_move_execute"`
	IsMoveUpgrade      bool            `gorm:"column:is_move_upgrade;not null" json:"is_move_upgrade"`
	IsMoveScript       bool            `gorm:"column:is_move_script;not null" json:"is_move_script"`
	IsNftTransfer      bool            `gorm:"column:is_nft_transfer;not null" json:"is_nft_transfer"`
	IsNftMint          bool            `gorm:"column:is_nft_mint;not null" json:"is_nft_mint"`
	IsNftBurn          bool            `gorm:"column:is_nft_burn;not null" json:"is_nft_burn"`
	IsCollectionCreate bool            `gorm:"column:is_collection_create;not null" json:"is_collection_create"`
	IsOpinit           bool            `gorm:"column:is_opinit;not null" json:"is_opinit"`
	IsInstantiate      bool            `gorm:"column:is_instantiate;not null" json:"is_instantiate"`
	IsMigrate          bool            `gorm:"column:is_migrate;not null" json:"is_migrate"`
	IsUpdateAdmin      bool            `gorm:"column:is_update_admin;not null" json:"is_update_admin"`
	IsClearAdmin       bool            `gorm:"column:is_clear_admin;not null" json:"is_clear_admin"`
	IsStoreCode        bool            `gorm:"column:is_store_code;not null" json:"is_store_code"`
	ID                 string          `gorm:"column:id;primaryKey;type:character varying" json:"id"`
	Sender             string          `gorm:"column:sender;type:character varying" json:"sender"`
	BlockIndex         int             `gorm:"column:block_index;not null;type:integer;index:ix_transactions_block_height_block_index,priority:2,sort:desc;default:0" json:"block_index"`

	// Foreign key relationships
	Block         Block   `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	SenderAccount Account `gorm:"foreignKey:Sender;references:Address" json:"-"`
}

// TableName Transaction's table name
func (*Transaction) TableName() string {
	return TableNameTransaction
}

// ValidatorBondedTokenChange mapped from table <validator_bonded_token_changes>
type ValidatorBondedTokenChange struct {
	BlockHeight      int64  `gorm:"column:block_height;type:bigint;not null;index:ix_validator_bonded_token_changes_validator_address_block_height,priority:2,sort:desc" json:"block_height"`
	ValidatorAddress string `gorm:"column:validator_address;not null;type:character varying;index:ix_validator_bonded_token_changes_validator_address_block_height,priority:1" json:"validator_address"`
	TransactionID    string `gorm:"column:transaction_id;type:character varying;index:ix_validator_bonded_token_changes_transaction_id" json:"transaction_id"`
	Tokens           JSON   `gorm:"column:tokens;not null;type:json" json:"tokens"`

	// Foreign key relationships
	Block       Block       `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	Transaction Transaction `gorm:"foreignKey:TransactionID;references:ID" json:"-"`
	Validator   Validator   `gorm:"foreignKey:ValidatorAddress;references:OperatorAddress" json:"-"`
}

// TableName ValidatorBondedTokenChange's table name
func (*ValidatorBondedTokenChange) TableName() string {
	return TableNameValidatorBondedTokenChange
}

// ValidatorCommitSignature mapped from table <validator_commit_signatures>
type ValidatorCommitSignature struct {
	ValidatorAddress string `gorm:"column:validator_address;primaryKey;type:character varying" json:"validator_address"`
	BlockHeight      int64  `gorm:"column:block_height;bigint;primaryKey" json:"block_height"`
	Vote             string `gorm:"column:vote;not null;type:commit_signature_type" json:"vote"`

	// Foreign key relationship
	Validator Validator `gorm:"foreignKey:ValidatorAddress;references:OperatorAddress" json:"-"`
}

// TableName ValidatorCommitSignature's table name
func (*ValidatorCommitSignature) TableName() string {
	return TableNameValidatorCommitSignature
}

// ValidatorHistoricalPower mapped from table <validator_historical_powers>
type ValidatorHistoricalPower struct {
	ValidatorAddress     string    `gorm:"column:validator_address;type:character varying;index:unique_validator_historicail_power,unique" json:"validator_address"`
	Tokens               JSON      `gorm:"column:tokens;type:json;not null" json:"tokens"`
	VotingPower          int64     `gorm:"column:voting_power;not null" json:"voting_power"`
	HourRoundedTimestamp time.Time `gorm:"column:hour_rounded_timestamp;type:timestamp;not null;index:unique_validator_historicail_power,unique" json:"hour_rounded_timestamp"`
	Timestamp            time.Time `gorm:"column:timestamp;type:timestamp;not null" json:"timestamp"`

	// Foreign key relationship
	Validator Validator `gorm:"foreignKey:ValidatorAddress;references:OperatorAddress" json:"-"`
}

// TableName ValidatorHistoricalPower's table name
func (*ValidatorHistoricalPower) TableName() string {
	return TableNameValidatorHistoricalPower
}

// ValidatorSlashEvent mapped from table <validator_slash_events>
type ValidatorSlashEvent struct {
	ValidatorAddress string `gorm:"column:validator_address;type:character varying;not null" json:"validator_address"`
	BlockHeight      int64  `gorm:"column:block_height;not null;type:bigint" json:"block_height"`
	Type             string `gorm:"column:type;type:slashtype" json:"type"`

	// Foreign key relationships
	Block     Block     `gorm:"foreignKey:BlockHeight;references:Height" json:"-"`
	Validator Validator `gorm:"foreignKey:ValidatorAddress;references:OperatorAddress" json:"-"`
}

// TableName ValidatorSlashEvent's table name
func (*ValidatorSlashEvent) TableName() string {
	return TableNameValidatorSlashEvent
}

// ValidatorVoteCount mapped from table <validator_vote_counts>
type ValidatorVoteCount struct {
	ValidatorAddress string `gorm:"column:validator_address;primaryKey;type:character varying" json:"validator_address"`
	Last100          int32  `gorm:"column:last_100;not null" json:"last_100"`

	// Foreign key relationship
	Validator Validator `gorm:"foreignKey:ValidatorAddress;references:OperatorAddress" json:"-"`
}

// TableName ValidatorVoteCount's table name
func (*ValidatorVoteCount) TableName() string {
	return TableNameValidatorVoteCount
}

// Validator mapped from table <validators>
type Validator struct {
	OperatorAddress     string          `gorm:"column:operator_address;primaryKey;type:character varying" json:"operator_address"`
	ConsensusAddress    string          `gorm:"column:consensus_address;not null;type:character varying" json:"consensus_address"`
	VotingPowers        json.RawMessage `gorm:"column:voting_powers;not null;type:json" json:"voting_powers"`
	VotingPower         int64           `gorm:"column:voting_power;not null" json:"voting_power"`
	Moniker             string          `gorm:"column:moniker;not null;type:character varying" json:"moniker"`
	Identity            string          `gorm:"column:identity;not null;type:character varying" json:"identity"`
	Website             string          `gorm:"column:website;not null;type:character varying" json:"website"`
	Details             string          `gorm:"column:details;not null;type:character varying" json:"details"`
	CommissionRate      string          `gorm:"column:commission_rate;not null;type:character varying" json:"commission_rate"`
	CommissionMaxRate   string          `gorm:"column:commission_max_rate;not null;type:character varying" json:"commission_max_rate"`
	CommissionMaxChange string          `gorm:"column:commission_max_change;not null;type:character varying" json:"commission_max_change"`
	Jailed              bool            `gorm:"column:jailed;not null" json:"jailed"`
	IsActive            bool            `gorm:"column:is_active" json:"is_active"`
	ConsensusPubkey     string          `gorm:"column:consensus_pubkey;type:character varying" json:"consensus_pubkey"`
	AccountID           string          `gorm:"column:account_id;type:character varying" json:"account_id"`

	// Foreign key relationship
	Account Account `gorm:"foreignKey:AccountID;references:Address" json:"-"`
}

// TableName Validator's table name
func (*Validator) TableName() string {
	return TableNameValidator
}

// VMAddress mapped from table <vm_addresses>
type VMAddress struct {
	VMAddress string `gorm:"column:vm_address;uniqueIndex:vm_addresses_vm_address_key;primaryKey;not null;type:character varying" json:"vm_address"`
}

// TableName VMAddress's table name
func (*VMAddress) TableName() string {
	return TableNameVMAddress
}
