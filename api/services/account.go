package services

import (
	"fmt"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/pkg/db"
)

type AccountService interface {
	GetAccountByAccountAddress(accountAddress string) (*db.Account, error)
	GetAccountProposals(pagination dto.PaginationQuery, accountAddress string) (*dto.AccountProposalsResponse, error)
	GetAccountTxs(
		pagination dto.PaginationQuery,
		accountAddress string,
		search string,
		isSend bool,
		isIbc bool,
		isOpinit bool,
		isMovePublish bool,
		isMoveUpgrade bool,
		isMoveExecute bool,
		isMoveScript bool,
		isSigner *bool,
	) (*dto.AccounTxsResponse, error)
}

type accountService struct {
	repo repositories.AccountRepositoryI
}

func NewAccountService(repo repositories.AccountRepositoryI) AccountService {
	return &accountService{
		repo: repo,
	}
}

func (s *accountService) GetAccountByAccountAddress(accountAddress string) (*db.Account, error) {
	account, err := s.repo.GetAccountByAccountAddress(accountAddress)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *accountService) GetAccountProposals(pagination dto.PaginationQuery, accountAddress string) (*dto.AccountProposalsResponse, error) {
	proposals, total, err := s.repo.GetAccountProposals(pagination, accountAddress)
	if err != nil {
		return nil, err
	}

	response := &dto.AccountProposalsResponse{
		Proposals: make([]dto.AccountProposal, len(proposals)),
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	for idx, proposal := range proposals {
		response.Proposals[idx] = dto.AccountProposal{
			DepositEndTime: proposal.DepositEndTime,
			ID:             int64(proposal.ID),
			IsEmergency:    proposal.IsEmergency,
			IsExpedited:    proposal.IsExpedited,
			Proposer:       proposal.ProposerID,
			ResolvedHeight: int64(proposal.ResolvedHeight),
			Status:         proposal.Status,
			Title:          proposal.Title,
			Type:           proposal.Type,
			VotingEndTime:  proposal.VotingEndTime,
		}
	}

	return response, nil
}

func (s *accountService) GetAccountTxs(
	pagination dto.PaginationQuery,
	accountAddress string,
	search string,
	isSend bool,
	isIbc bool,
	isOpinit bool,
	isMovePublish bool,
	isMoveUpgrade bool,
	isMoveExecute bool,
	isMoveScript bool,
	isSigner *bool,
) (*dto.AccounTxsResponse, error) {
	txs, total, err := s.repo.GetAccountTxs(
		pagination, accountAddress, search, isSend, isIbc, isOpinit, isMovePublish, isMoveUpgrade, isMoveExecute, isMoveScript, isSigner)

	if err != nil {
		return nil, err
	}

	response := &dto.AccounTxsResponse{
		AccounTxs: make([]dto.AccountTx, len(txs)),
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	for idx, tx := range txs {
		response.AccounTxs[idx] = dto.AccountTx{
			Created:  tx.Timestamp,
			Hash:     fmt.Sprintf("%x", tx.Hash),
			Height:   tx.Height,
			IsIbc:    tx.IsIbc,
			IsSend:   tx.IsSend,
			IsSigner: tx.IsSigner,
			Messages: tx.Messages,
			Sender:   tx.Address,
			Success:  tx.Success,
		}
	}

	return response, nil
}
