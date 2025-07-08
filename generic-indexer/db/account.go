package db

type Account struct {
	Address string
}

type AccountTransaction struct {
	TxId        string
	Accounts    []string
	BlockHeight int64
	Signer      string
}
