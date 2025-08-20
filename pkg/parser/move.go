package parser

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/cometbft/cometbft/crypto/tmhash"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"
)

type PublishModuleData struct {
	ModuleIdPath  string                  `json:"module_id"`
	UpgradePolicy movetypes.UpgradePolicy `json:"upgrade_policy"`
}

func DecodePublishModuleData(data string) (vmapi.ModuleInfoResponse, movetypes.UpgradePolicy, error) {
	bz := []byte(data)

	var pmb PublishModuleData
	err := json.Unmarshal(bz, &pmb)
	if err != nil {
		return vmapi.ModuleInfoResponse{}, pmb.UpgradePolicy, err
	}
	s := strings.Split(pmb.ModuleIdPath, "::")
	addr, err := vmtypes.NewAccountAddress(s[0])
	if len(s) != 2 {
		return vmapi.ModuleInfoResponse{}, pmb.UpgradePolicy, errors.New("Decode Publish Module Data: module patch length miss match")
	}
	if err != nil {
		return vmapi.ModuleInfoResponse{}, pmb.UpgradePolicy, err
	}
	return vmapi.ModuleInfoResponse{
		Address: addr,
		Name:    s[1],
	}, pmb.UpgradePolicy, nil
}

func GetModuleDigest(rawBytes []byte) string {
	rawBytesBuf := make([]byte, base64.StdEncoding.EncodedLen(len(rawBytes)))
	base64.StdEncoding.Encode(rawBytesBuf, rawBytes)

	return hex.EncodeToString(tmhash.Sum(rawBytesBuf))
}

func DecodeEvent[T any](data string) (T, error) {
	var e T
	err := json.Unmarshal([]byte(data), &e)
	if err != nil {
		return e, err
	}
	return e, nil
}

func DecodeResource[T any](rawData string) (T, error) {
	var e T
	err := json.Unmarshal([]byte(rawData), &e)
	if err != nil {
		return e, err
	}
	return e, nil
}
