package sdkconfig

import (
	"github.com/cosmos/cosmos-sdk/types"
	initiaapp "github.com/initia-labs/initia/app"
)

// ConfigureSDK sets up the SDK configuration for Initia chain
func ConfigureSDK() {
	sdkConfig := types.GetConfig()
	sdkConfig.SetCoinType(initiaapp.CoinType)

	accountPubKeyPrefix := initiaapp.AccountAddressPrefix + "pub"
	validatorAddressPrefix := initiaapp.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := initiaapp.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := initiaapp.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := initiaapp.AccountAddressPrefix + "valconspub"

	sdkConfig.SetBech32PrefixForAccount(initiaapp.AccountAddressPrefix, accountPubKeyPrefix)
	sdkConfig.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	sdkConfig.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	sdkConfig.SetAddressVerifier(initiaapp.VerifyAddressLen())
	sdkConfig.Seal()
}
