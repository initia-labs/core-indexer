package repositories

type TxRepository interface {
	GetTxCount() (*int64, error)
}
