package db

import (
	movetypes "github.com/initia-labs/initia/x/move/types"
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
