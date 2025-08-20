package apperror

// Common error messages
const (
	ErrMsgBadRequest      = "Invalid request"
	ErrMsgUnauthorized    = "Unauthorized"
	ErrMsgNotFound        = "Resource not found"
	ErrMsgInternal        = "Internal server error"
	ErrMsgProposalId      = "Proposal id is not a valid int32 integer"
	ErrMsgBlocks          = "Blocks argument is not a valid integer"
	ErrMsgBlocksZero      = "Blocks must be > 0"
	ErrMsgBlocksRequired  = "Blocks parameter is required"
	ErrMsgModuleNotFound  = "Module not found"
	ErrMsgDuplicateStatus = "Duplicate status in query"
	ErrMsgTxNotFound      = "Transaction not found for hash %s"
	ErrMsgNoValidTxFiles  = "No valid transaction files found for hash %s"
	ErrMsgInvalidLimit    = "Limit must be between 1 and 1000"
	ErrMsgLimitInteger    = "Limit must be in integer format"
	ErrMsgOffsetInteger   = "Offset must be in integer format"
	ErrMsgReverse         = "Reverse must be a boolean"
	ErrMsgCountTotal      = "CountTotal must be a boolean"
)
