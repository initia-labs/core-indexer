package cacher

import (
	"context"

	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/rs/zerolog"
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

func (c *Cacher) InitCacher(ctx context.Context, dbClient *gorm.DB, logger *zerolog.Logger) error {
	validatorAddresses, err := db.QueryValidatorAddresses(ctx, dbClient)
	if err != nil {
		return err
	}

	for _, validatorAddress := range validatorAddresses {
		c.valAccAddrToOperator[validatorAddress.AccountID] = validatorAddress
		c.valConsAddrToOperator[validatorAddress.ConsensusAddress] = validatorAddress
	}
	logger.Info().Msgf("Total validators loaded to cache = %d", len(validatorAddresses))

	return nil
}

func (c *Cacher) SetValidator(validator db.Validator) {
	validatorAddress := db.ValidatorAddress{
		ConsensusAddress: validator.ConsensusAddress,
		AccountID:        validator.AccountID,
		OperatorAddress:  validator.OperatorAddress,
	}

	c.valAccAddrToOperator[validator.AccountID] = validatorAddress
	c.valConsAddrToOperator[validator.ConsensusAddress] = validatorAddress
}

func (c *Cacher) GetValidatorByAccAddr(accAddress string) (db.ValidatorAddress, bool) {
	validator, ok := c.valAccAddrToOperator[accAddress]
	return validator, ok
}

func (c *Cacher) GetValidatorByConsAddr(consAddress string) (db.ValidatorAddress, bool) {
	validator, ok := c.valConsAddrToOperator[consAddress]
	return validator, ok
}
