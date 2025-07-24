package indexer

import (
	"context"
	"encoding/json"
	"fmt"

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

// decodeModule decodes the module using the byte code and returns the ABI as a Base64-encoded string.
func decodeModule(moduleBytes []byte) (string, error) {
	abi, err := vmapi.DecodeModuleBytes(moduleBytes)
	if err != nil {
		return "", err
	}

	return string(abi), nil
}

func (f *Indexer) StartFromGenesis(ctx context.Context, logger *zerolog.Logger) error {
	genesis, err := f.rpcClient.Genesis(ctx)
	if err != nil {
		logger.Error().Msgf("Error getting genesis: %v", err)
		return err
	}

	appState, err := genesis.Genesis.AppState.MarshalJSON()
	if err != nil {
		return err
	}

	var genesisState map[string]json.RawMessage
	if err := json.Unmarshal(appState, &genesisState); err != nil {
		return err
	}

	var authGenesis authtypes.GenesisState
	if genesisState[authtypes.ModuleName] != nil {
		f.encodingConfig.Codec.MustUnmarshalJSON(genesisState[authtypes.ModuleName], &authGenesis)
	}

	dbBatchInsert := statetracker.NewDBBatchInsert(f.cacher, logger)
	for _, account := range authGenesis.Accounts {
		a, ok := account.GetCachedValue().(sdk.AccountI)
		if !ok {
			panic("expected account")
		}

		accAddr := a.GetAddress()
		vmAddr, _ := vmtypes.NewAccountAddressFromBytes(accAddr)
		dbBatchInsert.AddAccounts(db.Account{
			Address:   sdk.AccAddress(accAddr).String(),
			VMAddress: db.VMAddress{VMAddress: vmAddr.String()},
			Type:      string(db.BaseAccount),
		})
	}

	dbBatchInsert.AddAccounts(db.Account{
		Address:   movetypes.StdAddr.String(),
		VMAddress: db.VMAddress{VMAddress: vmtypes.StdAddress.String()},
		Type:      string(db.BaseAccount),
	})
	dbBatchInsert.AddAccounts(db.Account{
		Address:   "init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d",
		VMAddress: db.VMAddress{VMAddress: vmtypes.StdAddress.String()},
		Type:      string(db.BaseAccount),
	})

	var genutilState genutiltypes.GenesisState
	f.encodingConfig.Codec.MustUnmarshalJSON(genesisState[genutiltypes.ModuleName], &genutilState)
	for _, genTx := range genutilState.GenTxs {
		tx, err := f.encodingConfig.TxConfig.TxJSONDecoder()(genTx)
		if err != nil {
			panic(err)
		}

		for _, msg := range tx.GetMsgs() {
			if msg, ok := msg.(*mstakingtypes.MsgCreateValidator); ok {
				valAddr, _ := sdk.ValAddressFromBech32(msg.ValidatorAddress)
				accAddr := sdk.AccAddress(valAddr)

				vmAddr, _ := vmtypes.NewAccountAddressFromBytes(accAddr)
				dbBatchInsert.AddAccounts(db.Account{
					Address:   accAddr.String(),
					VMAddress: db.VMAddress{VMAddress: vmAddr.String()},
					Type:      string(db.BaseAccount),
				})

				validator, err := db.NewGenesisValidator(accAddr.String(), msg)
				if err != nil {
					return err
				}
				dbBatchInsert.AddValidators(validator)
			}
		}

	}

	var moveGenesis movetypes.GenesisState
	f.encodingConfig.Codec.MustUnmarshalJSON(genesisState[movetypes.ModuleName], &moveGenesis)

	for _, stdlib := range moveGenesis.GetStdlibs() {
		abi, err := decodeModule(stdlib)
		if err != nil {
			return err
		}

		var abiJson map[string]any
		if err := json.Unmarshal([]byte(abi), &abiJson); err != nil {
			return err
		}

		name, ok := abiJson["name"].(string)
		if !ok {
			return fmt.Errorf("name is not a string")
		}
		address, ok := abiJson["address"].(string)
		if !ok {
			return fmt.Errorf("address is not a string")
		}

		dbBatchInsert.AddModule(db.Module{
			Name:                name,
			ModuleEntryExecuted: 0,
			IsVerify:            false,
			PublishTxID:         nil,
			PublisherID:         address,
			ID:                  fmt.Sprintf("%s::%s", address, name),
			Digest:              parser.GetModuleDigest(stdlib),
			UpgradePolicy:       db.GetUpgradePolicy(movetypes.UpgradePolicy_COMPATIBLE),
		})

		dbBatchInsert.ModulePublishedEvents = append(dbBatchInsert.ModulePublishedEvents, db.ModuleHistory{
			UpgradePolicy: db.GetUpgradePolicy(movetypes.UpgradePolicy_COMPATIBLE),
			TxID:          nil,
			BlockHeight:   0,
			ModuleID:      fmt.Sprintf("%s::%s", address, name),
			Remark:        db.JSON("{}"),
		})
	}

	if err := f.dbClient.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		err := db.InitTracking(ctx, dbTx)
		if err != nil {
			logger.Error().Msgf("Error initializing tracking: %v", err)
			return err
		}
		err = db.InsertGenesisBlock(ctx, dbTx, genesis.Genesis.GenesisTime)
		if err != nil {
			logger.Error().Msgf("Error inserting genesis block: %v", err)
			return err
		}

		err = dbBatchInsert.Flush(ctx, dbTx, 0)
		if err != nil {
			logger.Error().Msgf("Error getting genesis flushing batch insert: %v", err)
			return err
		}

		return nil
	}); err != nil {
		logger.Error().Msgf("Error flushing batch insert: %v", err)
		return err
	}
	return nil
}
