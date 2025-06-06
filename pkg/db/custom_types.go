package db

type AccountType string

const (
	BaseAccount              AccountType = "BaseAccount"
	InterchainAccount        AccountType = "InterchainAccount"
	ModuleAccount            AccountType = "ModuleAccount"
	ContinuousVestingAccount AccountType = "ContinuousVestingAccount"
	DelayedVestingAccount    AccountType = "DelayedVestingAccount"
	ClawbackVestingAccount   AccountType = "ClawbackVestingAccount"
	ContractAccount          AccountType = "ContractAccount"
)

type UpgradePolicy string

const (
	Arbitrary  UpgradePolicy = "Arbitrary"
	Compatible UpgradePolicy = "Compatible"
	Immutable  UpgradePolicy = "Immutable"
)

type CommitSignatureType string

const (
	Propose CommitSignatureType = "PROPOSE"
	Vote    CommitSignatureType = "VOTE"
	Absent  CommitSignatureType = "ABSENT"
)

type SlashType string

const (
	Jailed   SlashType = "Jailed"
	Slashed  SlashType = "Slashed"
	Unjailed SlashType = "Unjailed"
)

type FinalizeBlockEventsMode string

const (
	BeginBlock FinalizeBlockEventsMode = "BeginBlock"
	EndBlock   FinalizeBlockEventsMode = "EndBlock"
)

type ProposalStatus string

const (
	ProposalStatusNil           ProposalStatus = "Nil"
	ProposalStatusDepositPeriod ProposalStatus = "DepositPeriod"
	ProposalStatusVotingPeriod  ProposalStatus = "VotingPeriod"
	ProposalStatusPassed        ProposalStatus = "Passed"
	ProposalStatusRejected      ProposalStatus = "Rejected"
	ProposalStatusFailed        ProposalStatus = "Failed"
	ProposalStatusInactive      ProposalStatus = "Inactive"
	ProposalStatusCancelled     ProposalStatus = "Cancelled"
)
