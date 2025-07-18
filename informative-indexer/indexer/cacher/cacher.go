package cacher

import (
	"github.com/initia-labs/core-indexer/pkg/db"
)

type Cacher struct {
	valAccAddrToOperator  map[string]db.ValidatorAddress
	valConsAddrToOperator map[string]db.ValidatorAddress
}

func NewCacher() *Cacher {
	return &Cacher{
		valAccAddrToOperator:  make(map[string]db.ValidatorAddress),
		valConsAddrToOperator: make(map[string]db.ValidatorAddress),
	}
}

func (c *Cacher) SetValidatorAddresses(validatorAddresses []db.ValidatorAddress) {
	for _, validatorAddress := range validatorAddresses {
		c.valAccAddrToOperator[validatorAddress.AccountID] = validatorAddress
		c.valConsAddrToOperator[validatorAddress.ConsensusAddress] = validatorAddress
	}
}

func (c *Cacher) SetValidator(validatorAddress db.ValidatorAddress) {
	c.valAccAddrToOperator[validatorAddress.AccountID] = validatorAddress
	c.valConsAddrToOperator[validatorAddress.ConsensusAddress] = validatorAddress
}

func (c *Cacher) GetValidatorByAccAddr(accAddress string) (db.ValidatorAddress, bool) {
	validator, ok := c.valAccAddrToOperator[accAddress]
	return validator, ok
}

func (c *Cacher) GetValidatorByConsAddr(consAddress string) (db.ValidatorAddress, bool) {
	validator, ok := c.valConsAddrToOperator[consAddress]
	return validator, ok
}
