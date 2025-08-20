package db

import (
	"fmt"

	movetypes "github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
)

func GetUpgradePolicy(policy movetypes.UpgradePolicy) string {
	switch policy {
	case movetypes.UpgradePolicy_UNSPECIFIED:
		return string(Arbitrary)
	case movetypes.UpgradePolicy_COMPATIBLE:
		return string(Compatible)
	case movetypes.UpgradePolicy_IMMUTABLE:
		return string(Immutable)
	}
	panic("invalid upgrade policy")
}

func GetModuleID(module vmapi.ModuleInfoResponse) string {
	return fmt.Sprintf("%s::%s", module.Address.String(), module.Name)
}
