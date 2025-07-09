package validator

type ValidatorTokenChange struct {
	ValidatorAddr string
	Denom         string
	Amount        int64
	TxHash        string
}

type Processor struct {
	stakeChanges map[string]int64
	validators   map[string]bool
}
