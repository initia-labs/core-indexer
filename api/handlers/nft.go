package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/initia-labs/core-indexer/pkg/parser"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/services"
)

// NftHandler handles Nft-related HTTP requests
type NftHandler struct {
	service services.NftService
}

// NewNftHandler creates a new instance of NftHandler
func NewNftHandler(service services.NftService) *NftHandler {
	return &NftHandler{
		service: service,
	}
}

// GetCollections godoc
// @Summary Get Nft collections
// @Description Retrieve a list of Nft collections with optional search and pagination
// @Tags Nft
// @Accept json
// @Produce json
// @Param search query string false "Search term for filtering Nfts"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total Nfts" default(false)
// @Success 200 {object} dto.NftCollectionsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/collections [get]
func (h *NftHandler) GetCollections(c *fiber.Ctx) error {
	// Parse pagination parameters manually
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	// Get search parameter
	search := c.Query("search")

	// Get collections from service
	response, err := h.service.GetCollections(c.UserContext(), *pagination, search)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetCollectionsByAccountAddress godoc
// @Summary Get Nft collections by account address
// @Description Retrieve a list of Nft collections owned by a specific account address
// @Tags Nft
// @Accept json
// @Produce json
// @Param accountAddress path string true "Account address of the Nft owner"
// @Success 200 {object} dto.NftCollectionsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/collections/by_account/{accountAddress} [get]
func (h *NftHandler) GetCollectionsByAccountAddress(c *fiber.Ctx) error {
	accountAddress, err := parser.AccAddressFromString(c.Params("accountAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetCollectionsByAccountAddress(c.UserContext(), parser.BytesToHexWithPrefix(accountAddress))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetCollectionsByCollectionAddress godoc
// @Summary Get Nft collection by collection address
// @Description Retrieve a specific Nft collection by its collection address
// @Tags Nft
// @Accept json
// @Produce json
// @Param collectionAddress path string true "Collection address of the Nft"
// @Success 200 {object} dto.NftCollectionResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/collections/{collectionAddress} [get]
func (h *NftHandler) GetCollectionsByCollectionAddress(c *fiber.Ctx) error {
	collectionAddress, err := parser.AccAddressFromString(c.Params("collectionAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetCollectionsByCollectionAddress(c.UserContext(), parser.BytesToHexWithPrefix(collectionAddress))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetCollectionActivities godoc
// @Summary Get Nft collection activities
// @Description Retrieve activities related to a specific Nft collection with optional search and pagination
// @Tags Nft
// @Accept json
// @Produce json
// @Param collectionAddress path string true "Collection address of the Nft"
// @Param search query string false "Search term for filtering activities"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.reverse query boolean false "Reverse order for pagination" default(false)
// @Param pagination.count_total query boolean false "Count total number of transactions" default(false)
// @Success 200 {object} dto.CollectionActivitiesResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/collections/{collectionAddress}/activities [get]
func (h *NftHandler) GetCollectionActivities(c *fiber.Ctx) error {
	collectionAddress, err := parser.AccAddressFromString(c.Params("collectionAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	search := c.Query("search")

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetCollectionActivities(c.UserContext(), *pagination, parser.BytesToHexWithPrefix(collectionAddress), search)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetCollectionCreator godoc
// @Summary Get Nft collection creator
// @Description Retrieve the creator of a specific Nft collection by its address
// @Tags Nft
// @Accept json
// @Produce json
// @Param collectionAddress path string true "Collection address of the Nft"
// @Success 200 {object} dto.CollectionCreatorResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/collections/{collectionAddress}/creator [get]
func (h *NftHandler) GetCollectionCreator(c *fiber.Ctx) error {
	collectionAddress, err := parser.AccAddressFromString(c.Params("collectionAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetCollectionCreator(c.UserContext(), parser.BytesToHexWithPrefix(collectionAddress))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetCollectionMutateEvents godoc
// @Summary Get Nft collection mutate events
// @Description Retrieve mutate events for a specific Nft collection by its address with pagination
// @Tags Nft
// @Accept json
// @Produce json
// @Param collectionAddress path string true "Collection address of the Nft"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total events" default(false)
// @Success 200 {object} dto.CollectionMutateEventsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/collections/{collectionAddress}/mutate_events [get]
func (h *NftHandler) GetCollectionMutateEvents(c *fiber.Ctx) error {
	collectionAddress, err := parser.AccAddressFromString(c.Params("collectionAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetCollectionMutateEvents(c.UserContext(), *pagination, parser.BytesToHexWithPrefix(collectionAddress))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetNftByNftAddress godoc
// @Summary Get Nft by collection address and Nft address
// @Description Retrieve a specific Nft by its collection address and Nft address
// @Tags Nft
// @Accept json
// @Produce json
// @Param collectionAddress path string true "Collection address of the Nft"
// @Param nftAddress path string true "Nft address"
// @Success 200 {object} dto.NftByAddressResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/tokens/by_collection/{collectionAddress}/{nftAddress} [get]
func (h *NftHandler) GetNftByNftAddress(c *fiber.Ctx) error {
	collectionAddress, err := parser.AccAddressFromString(c.Params("collectionAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}
	nftAddress, err := parser.AccAddressFromString(c.Params("nftAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetNftByNftAddress(c.UserContext(), parser.BytesToHexWithPrefix(collectionAddress), parser.BytesToHexWithPrefix(nftAddress))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetNftsByCollectionAddress godoc
// @Summary Get Nfts by collection address
// @Description Retrieve a list of Nfts by their collection address with optional search and pagination
// @Tags Nft
// @Accept json
// @Produce json
// @Param collectionAddress path string true "Collection address of the Nfts"
// @Param search query string false "Search term for filtering Nfts"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total Nfts" default(false)
// @Success 200 {object} dto.NftsByAddressResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/tokens/by_collection/{collectionAddress} [get]
func (h *NftHandler) GetNftsByCollectionAddress(c *fiber.Ctx) error {
	collectionAddress, err := parser.AccAddressFromString(c.Params("collectionAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	search := c.Query("search")

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetNftsByCollectionAddress(c.UserContext(), *pagination, parser.BytesToHexWithPrefix(collectionAddress), search)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetNftsByAccountAddress godoc
// @Summary Get Nfts by account address
// @Description Retrieve a list of Nfts owned by a specific account address with optional search and collection filtering
// @Tags Nft
// @Accept json
// @Produce json
// @Param accountAddress path string true "Account address of the Nfts owner"
// @Param search query string false "Search term for filtering Nfts"
// @Param collectionAddress query string false "Collection address to filter Nfts"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total Nfts" default(false)
// @Success 200 {object} dto.NftsByAddressResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/tokens/by_account/{accountAddress} [get]
func (h *NftHandler) GetNftsByAccountAddress(c *fiber.Ctx) error {
	accountAddress, err := parser.AccAddressFromString(c.Params("accountAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	search := c.Query("search")
	collectionAddressStr := c.Query("collectionAddress")
	if collectionAddressStr != "" {
		if collectionAddress, err := parser.AccAddressFromString(collectionAddressStr); err != nil {
			return apperror.HandleErrorResponse(c, err)
		} else {
			collectionAddressStr = parser.BytesToHexWithPrefix(collectionAddress)
		}
	}

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetNftsByAccountAddress(c.UserContext(), *pagination, parser.BytesToHexWithPrefix(accountAddress), collectionAddressStr, search)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetNftMintInfo godoc
// @Summary Get Nft mint information
// @Description Retrieve mint information for a specific Nft by its address
// @Tags Nft
// @Accept json
// @Produce json
// @Param nftAddress path string true "Nft address"
// @Success 200 {object} dto.NftMintInfoResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/token/{nftAddress}/mint_info [get]
func (h *NftHandler) GetNftMintInfo(c *fiber.Ctx) error {
	nftAddress, err := parser.AccAddressFromString(c.Params("nftAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetNftMintInfo(c.UserContext(), parser.BytesToHexWithPrefix(nftAddress))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetNftMutateEvents godoc
// @Summary Get Nft mutate events
// @Description Retrieve mutate events for a specific Nft by its address with pagination
// @Tags Nft
// @Accept json
// @Produce json
// @Param nftAddress path string true "Nft address"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total Nfts" default(false)
// @Success 200 {object} dto.NftMutateEventsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/token/{nftAddress}/mutate_events [get]
func (h *NftHandler) GetNftMutateEvents(c *fiber.Ctx) error {
	nftAddress, err := parser.AccAddressFromString(c.Params("nftAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetNftMutateEvents(c.UserContext(), *pagination, parser.BytesToHexWithPrefix(nftAddress))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetNftTxs godoc
// @Summary Get Nft transactions
// @Description Retrieve transactions related to a specific Nft by its address with pagination
// @Tags Nft
// @Accept json
// @Produce json
// @Param nftAddress path string true "Nft address"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total Nfts" default(false)
// @Param pagination.reverse query boolean false "Whether to reverse the order of transactions" default(true)
// @Success 200 {object} dto.NftTxsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/nft/v1/token/{nftAddress}/txs [get]
func (h *NftHandler) GetNftTxs(c *fiber.Ctx) error {
	nftAddress, err := parser.AccAddressFromString(c.Params("nftAddress"))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetNftTxs(c.UserContext(), *pagination, parser.BytesToHexWithPrefix(nftAddress))
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}
