package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

// GenesisProcessor handles the processing of genesis state
type GenesisProcessor struct {
	indexer      *Indexer
	batchInsert  *statetracker.DBBatchInsert
	logger       *zerolog.Logger
	genesisState map[string]json.RawMessage
}

// NewGenesisProcessor creates a new genesis processor
func NewGenesisProcessor(indexer *Indexer, logger *zerolog.Logger) *GenesisProcessor {
	return &GenesisProcessor{
		indexer:     indexer,
		batchInsert: statetracker.NewDBBatchInsert(indexer.cacher, logger),
		logger:      logger,
	}
}

// StartFromGenesis initializes the database from genesis state
func (f *Indexer) StartFromGenesis(ctx context.Context, logger *zerolog.Logger) error {
	processor := NewGenesisProcessor(f, logger)
	return processor.Process(ctx)
}

// Process orchestrates the entire genesis processing workflow
func (gp *GenesisProcessor) Process(ctx context.Context) error {
	// Step 1: Fetch and parse genesis data
	if err := gp.fetchGenesisData(ctx); err != nil {
		return fmt.Errorf("failed to fetch genesis data: %w", err)
	}

	// Step 2: Process accounts from auth module
	if err := gp.processAuthAccounts(); err != nil {
		return fmt.Errorf("failed to process auth accounts: %w", err)
	}

	// Step 3: Add system accounts
	gp.addSystemAccounts()

	// Step 4: Process validators from genutil module
	if err := gp.processValidators(); err != nil {
		return fmt.Errorf("failed to process validators: %w", err)
	}

	// Step 5: Process Move modules
	if err := gp.processMoveModules(); err != nil {
		return fmt.Errorf("failed to process Move modules: %w", err)
	}

	// Step 6: Persist everything to database
	return gp.persistToDatabase(ctx)
}

// fetchGenesisData retrieves and parses the genesis state
func (gp *GenesisProcessor) fetchGenesisData(ctx context.Context) error {
	genesis, err := gp.indexer.rpcClient.Genesis(ctx)
	if err != nil {
		gp.logger.Error().Msgf("Error getting genesis: %v", err)
		return err
	}

	appState, err := genesis.Genesis.AppState.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal app state: %w", err)
	}

	if err := json.Unmarshal(appState, &gp.genesisState); err != nil {
		return fmt.Errorf("failed to unmarshal genesis state: %w", err)
	}

	return nil
}

// processAuthAccounts processes accounts from the auth module
func (gp *GenesisProcessor) processAuthAccounts() error {
	var authGenesis authtypes.GenesisState
	if gp.genesisState[authtypes.ModuleName] == nil {
		return nil // No auth accounts to process
	}

	gp.indexer.encodingConfig.Codec.MustUnmarshalJSON(gp.genesisState[authtypes.ModuleName], &authGenesis)

	for _, account := range authGenesis.Accounts {
		if err := gp.addAccountFromAuth(account); err != nil {
			return err
		}
	}

	return nil
}

// addAccountFromAuth adds a single account from auth genesis
func (gp *GenesisProcessor) addAccountFromAuth(account *codectypes.Any) error {
	acc, ok := account.GetCachedValue().(sdk.AccountI)
	if !ok {
		return fmt.Errorf("expected account interface, got %T", account.GetCachedValue())
	}

	accAddr := acc.GetAddress()
	vmAddr, _ := vmtypes.NewAccountAddressFromBytes(accAddr)

	gp.batchInsert.AddAccounts(db.Account{
		Address:   sdk.AccAddress(accAddr).String(),
		VMAddress: db.VMAddress{VMAddress: vmAddr.String()},
		Type:      string(db.BaseAccount),
	})

	return nil
}

// addSystemAccounts adds predefined system accounts
func (gp *GenesisProcessor) addSystemAccounts() {
	systemAccounts := []struct {
		address   string
		vmAddress string
	}{
		{
			address:   movetypes.StdAddr.String(),
			vmAddress: vmtypes.StdAddress.String(),
		},
		{
			address:   "init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d",
			vmAddress: vmtypes.StdAddress.String(),
		},
	}

	for _, acc := range systemAccounts {
		gp.batchInsert.AddAccounts(db.Account{
			Address:   acc.address,
			VMAddress: db.VMAddress{VMAddress: acc.vmAddress},
			Type:      string(db.BaseAccount),
		})
	}
}

// processValidators processes validators from the genutil module
func (gp *GenesisProcessor) processValidators() error {
	var genutilState genutiltypes.GenesisState
	if gp.genesisState[genutiltypes.ModuleName] == nil {
		return nil // No validators to process
	}

	gp.indexer.encodingConfig.Codec.MustUnmarshalJSON(gp.genesisState[genutiltypes.ModuleName], &genutilState)

	for _, genTx := range genutilState.GenTxs {
		if err := gp.processGenesisTransaction(genTx); err != nil {
			return err
		}
	}

	return nil
}

// processGenesisTransaction processes a single genesis transaction
func (gp *GenesisProcessor) processGenesisTransaction(genTx json.RawMessage) error {
	tx, err := gp.indexer.encodingConfig.TxConfig.TxJSONDecoder()(genTx)
	if err != nil {
		return fmt.Errorf("failed to decode genesis transaction: %w", err)
	}

	for _, msg := range tx.GetMsgs() {
		if err := gp.processGenesisMessage(msg); err != nil {
			return err
		}
	}

	return nil
}

// processGenesisMessage processes a single genesis message
func (gp *GenesisProcessor) processGenesisMessage(msg sdk.Msg) error {
	if createValidatorMsg, ok := msg.(*mstakingtypes.MsgCreateValidator); ok {
		return gp.addValidatorFromGenesis(createValidatorMsg)
	}
	return nil
}

// addValidatorFromGenesis adds a validator from genesis
func (gp *GenesisProcessor) addValidatorFromGenesis(msg *mstakingtypes.MsgCreateValidator) error {
	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}

	accAddr := sdk.AccAddress(valAddr)
	vmAddr, err := vmtypes.NewAccountAddressFromBytes(accAddr)
	if err != nil {
		return fmt.Errorf("failed to create VM address: %w", err)
	}

	// Add validator account
	gp.batchInsert.AddAccounts(db.Account{
		Address:   accAddr.String(),
		VMAddress: db.VMAddress{VMAddress: vmAddr.String()},
		Type:      string(db.BaseAccount),
	})

	// Add validator
	validator, err := db.NewGenesisValidator(accAddr.String(), msg)
	if err != nil {
		return fmt.Errorf("failed to create genesis validator: %w", err)
	}
	gp.batchInsert.AddValidators(validator)

	return nil
}

// processMoveModules processes Move modules from genesis
func (gp *GenesisProcessor) processMoveModules() error {
	var moveGenesis movetypes.GenesisState
	if gp.genesisState[movetypes.ModuleName] == nil {
		return nil // No Move modules to process
	}

	gp.indexer.encodingConfig.Codec.MustUnmarshalJSON(gp.genesisState[movetypes.ModuleName], &moveGenesis)

	for _, stdlib := range moveGenesis.GetStdlibs() {
		if err := gp.addMoveModule(stdlib); err != nil {
			return err
		}
	}

	return nil
}

// addMoveModule adds a single Move module from genesis
func (gp *GenesisProcessor) addMoveModule(moduleBytes []byte) error {
	abi, err := decodeModule(moduleBytes)
	if err != nil {
		return fmt.Errorf("failed to decode module: %w", err)
	}

	moduleAddress, moduleName, err := gp.parseModuleABI(abi)
	if err != nil {
		return fmt.Errorf("failed to parse module ABI: %w", err)
	}

	// Add module
	gp.batchInsert.AddModule(db.Module{
		Name:                moduleName,
		ModuleEntryExecuted: 0,
		IsVerify:            false,
		PublishTxID:         nil,
		PublisherID:         moduleAddress,
		ID:                  fmt.Sprintf("%s::%s", moduleAddress, moduleName),
		Digest:              parser.GetModuleDigest(moduleBytes),
		UpgradePolicy:       db.GetUpgradePolicy(movetypes.UpgradePolicy_COMPATIBLE),
	})

	// Add module history
	gp.batchInsert.ModulePublishedEvents = append(gp.batchInsert.ModulePublishedEvents, db.ModuleHistory{
		UpgradePolicy: db.GetUpgradePolicy(movetypes.UpgradePolicy_COMPATIBLE),
		TxID:          nil,
		BlockHeight:   0,
		ModuleID:      fmt.Sprintf("%s::%s", moduleAddress, moduleName),
		Remark:        db.JSON("{}"),
	})

	return nil
}

// parseModuleABI parses the module ABI to extract name and address
func (gp *GenesisProcessor) parseModuleABI(abi string) (moduleAddress string, moduleName string, err error) {
	var abiJSON map[string]any
	if err := json.Unmarshal([]byte(abi), &abiJSON); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal ABI JSON: %w", err)
	}

	moduleName, ok := abiJSON["name"].(string)
	if !ok {
		return "", "", fmt.Errorf("module name is not a string")
	}

	moduleAddress, ok = abiJSON["address"].(string)
	if !ok {
		return "", "", fmt.Errorf("module address is not a string")
	}

	return moduleAddress, moduleName, nil
}

// persistToDatabase persists all processed data to the database
func (gp *GenesisProcessor) persistToDatabase(ctx context.Context) error {
	return gp.indexer.dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Initialize tracking
		if err := db.InitTracking(ctx, dbTx); err != nil {
			gp.logger.Error().Msgf("Error initializing tracking: %v", err)
			return err
		}

		// Insert genesis block
		if err := db.InsertGenesisBlock(ctx, dbTx, gp.getGenesisTime()); err != nil {
			gp.logger.Error().Msgf("Error inserting genesis block: %v", err)
			return err
		}

		// Flush all batch data
		if err := gp.batchInsert.Flush(ctx, dbTx, 0); err != nil {
			gp.logger.Error().Msgf("Error flushing batch insert: %v", err)
			return err
		}

		return nil
	})
}

// getGenesisTime extracts genesis time from the genesis state
func (gp *GenesisProcessor) getGenesisTime() time.Time {
	genesis, err := gp.indexer.rpcClient.Genesis(context.Background())
	if err != nil {
		gp.logger.Error().Msgf("Error getting genesis: %v", err)
		return time.Now()
	}
	return genesis.Genesis.GenesisTime
}

// decodeModule decodes the module using the byte code and returns the ABI as a string
func decodeModule(moduleBytes []byte) (string, error) {
	abi, err := vmapi.DecodeModuleBytes(moduleBytes)
	if err != nil {
		return "", err
	}
	return string(abi), nil
}
