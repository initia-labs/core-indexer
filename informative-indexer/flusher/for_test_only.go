package flusher

//func (f *Flusher) ForTestOnlyFillDbValidators(ctx context.Context, blockResults *mq.BlockResultMsg, proposer *mstakingtypes.Validator) error {
//	// use for test only
//	dbTx := f.dbClient.WithContext(ctx)
//	vals := make([]db.Validator, 0)
//	accs := make([]db.Account, 0)
//	vmAddrs := make([]db.VMAddress, 0)
//	for _, val := range f.validatorMap {
//		valAcc, err := sdk.ValAddressFromBech32(val.OperatorAddress)
//		if err != nil {
//			return fmt.Errorf("failed to convert validator address: %w", err)
//		}
//
//		accAddr := sdk.AccAddress(valAcc)
//		vmAddr, _ := vmtypes.NewAccountAddressFromBytes(accAddr)
//
//		if err := val.UnpackInterfaces(f.encodingConfig.InterfaceRegistry); err != nil {
//			return fmt.Errorf("failed to unpack validator info: %w", err)
//		}
//
//		consAddr, err := val.GetConsAddr()
//		if err != nil {
//			return errors.Join(types.ErrorNonRetryable, err)
//		}
//		vals = append(vals, db.NewValidator(val, accAddr.String(), consAddr))
//		accs = append(accs, db.Account{
//			Address:     accAddr.String(),
//			VMAddressID: vmAddr.String(),
//			Type:        string(db.BaseAccount),
//		})
//		vmAddrs = append(vmAddrs, db.VMAddress{
//			VMAddress: vmAddr.String(),
//		})
//	}
//
//	err := db.InsertVMAddressesIgnoreConflict(ctx, dbTx, vmAddrs)
//	if err != nil {
//		logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting VM addresses: %v", err)
//		return err
//	}
//
//	err = db.InsertAccountIgnoreConflict(ctx, dbTx, accs)
//	if err != nil {
//		logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting accounts: %v", err)
//		return err
//	}
//
//	err = db.InsertValidatorsIgnoreConflict(ctx, dbTx, vals)
//	if err != nil {
//		logger.Error().Int64("height", blockResults.Height).Msgf("Error inserting validators: %v", err)
//		return err
//	}
//
//	return nil
//}
