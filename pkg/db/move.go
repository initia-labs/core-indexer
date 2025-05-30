package db

type Module struct {
	Address             string `json:"address"`
	Name                string `json:"name"`
	ModuleEntryExecuted int64  `json:"module_entry_executed"`
	PublishTxId         string `json:"publish_tx_id"`
	PublisherId         string `json:"publisher_id"`
	Id                  string `json:"id"`
	Digest              string `json:"digest"`

	// TODO: this field is not used need to revisit
	IsVerify bool `json:"is_verify"`
	// TODO: enum revisit type
	UpgradePolicy string `json:"upgrade_policy"`
}
