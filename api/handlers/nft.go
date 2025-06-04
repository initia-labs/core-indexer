package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/services"
)

// NFTHandler handles NFT-related HTTP requests
type NFTHandler struct {
	service services.NFTService
}

// NewNFTHandler creates a new instance of NFTHandler
func NewNFTHandler(service services.NFTService) *NFTHandler {
	return &NFTHandler{
		service: service,
	}
}

// GetCollections godoc
// @Summary Get NFT collections
// @Description Retrieve a list of NFT collections with optional search and pagination
// @Tags NFT
// @Accept json
// @Produce json
// @Param search query string false "Search term for filtering collections"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Success 200 {object} dto.NFTCollectionsResponse
// @Router /indexer/nft/v1/collections [get]
func (h *NFTHandler) GetCollections(c *fiber.Ctx) error {
	// Parse pagination parameters manually
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	// Get search parameter
	search := c.Query("search")

	// Get collections from service
	response, err := h.service.GetCollections(*pagination, search)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetNFTByNFTAddress godoc
// @Summary Get NFT by collection address and NFT address
// @Description Retrieve a specific NFT by its collection address and NFT address
// @Tags NFT
// @Accept json
// @Produce json
// @Param collectionAddress path string true "Collection address of the NFT"
// @Param nftAddress path string true "NFT address"
// @Success 200 {object} dto.NFTByAddressResponse
// @Router /indexer/nft/v1/tokens/by_collection/{collectionAddress}/{nftAddress} [get]
func (h *NFTHandler) GetNFTByNFTAddress(c *fiber.Ctx) error {
	collectionAddress := c.Params("collectionAddress")
	nftAddress := c.Params("nftAddress")

	response, err := h.service.GetNFTByNFTAddress(collectionAddress, nftAddress)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetNFTsByCollectionAddress godoc
// @Summary Get NFTs by collection address
// @Description Retrieve a list of NFTs by their collection address with optional search and pagination
// @Tags NFT
// @Accept json
// @Produce json
// @Param collectionAddress path string true "Collection address of the NFTs"
// @Param search query string false "Search term for filtering NFTs"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total NFTs" default(false)
// @Success 200 {object} dto.NFTsByAddressResponse
// @Router /indexer/nft/v1/tokens/by_collection/{collectionAddress} [get]
func (h *NFTHandler) GetNFTsByCollectionAddress(c *fiber.Ctx) error {
	collectionAddress := c.Params("collectionAddress")

	search := c.Query("search")

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	response, err := h.service.GetNFTsByCollectionAddress(*pagination, collectionAddress, &search)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetNFTsByAccountAddress godoc
// @Summary Get NFTs by account address
// @Description Retrieve a list of NFTs owned by a specific account address with optional search and collection filtering
// @Tags NFT
// @Accept json
// @Produce json
// @Param accountAddress path string true "Account address of the NFTs owner"
// @Param search query string false "Search term for filtering NFTs"
// @Param collectionAddress query string false "Collection address to filter NFTs"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total NFTs" default(false)
// @Success 200 {object} dto.NFTsByAddressResponse
// @Router /indexer/nft/v1/tokens/by_account/{accountAddress} [get]
func (h *NFTHandler) GetNFTsByAccountAddress(c *fiber.Ctx) error {
	accountAddress := c.Params("accountAddress")

	search := c.Query("search")
	collectionAddress := c.Query("collectionAddress")

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	response, err := h.service.GetNFTsByAccountAddress(*pagination, accountAddress, &collectionAddress, &search)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetNFTMintInfo godoc
// @Summary Get NFT mint information
// @Description Retrieve mint information for a specific NFT by its address
// @Tags NFT
// @Accept json
// @Produce json
// @Param nftAddress path string true "NFT address"
// @Success 200 {object} dto.NFTMintInfoResponse
// @Router /indexer/nft/v1/token/{nftAddress}/mint-info [get]
func (h *NFTHandler) GetNFTMintInfo(c *fiber.Ctx) error {
	nftAddress := c.Params("nftAddress")

	response, err := h.service.GetNFTMintInfo(nftAddress)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetNFTMutateEvents godoc
// @Summary Get NFT mutate events
// @Description Retrieve mutate events for a specific NFT by its address with pagination
// @Tags NFT
// @Accept json
// @Produce json
// @Param nftAddress path string true "NFT address"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total NFTs" default(false)
// @Success 200 {object} dto.NFTMutateEventsResponse
// @Router /indexer/nft/v1/token/{nftAddress}/mutate-events [get]
func (h *NFTHandler) GetNFTMutateEvents(c *fiber.Ctx) error {
	nftAddress := c.Params("nftAddress")

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	response, err := h.service.GetNFTMutateEvents(*pagination, nftAddress)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetNFTTxs godoc
// @Summary Get NFT transactions
// @Description Retrieve transactions related to a specific NFT by its address with pagination
// @Tags NFT
// @Accept json
// @Produce json
// @Param nftAddress path string true "NFT address"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Whether to count total NFTs" default(false)
// @Param pagination.reverse query boolean false "Whether to reverse the order of transactions" default(true)
// @Success 200 {object} dto.NFTTxsResponse
// @Router /indexer/nft/v1/token/{nftAddress}/txs [get]
func (h *NFTHandler) GetNFTTxs(c *fiber.Ctx) error {
	nftAddress := c.Params("nftAddress")

	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	response, err := h.service.GetNFTTxs(*pagination, nftAddress)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}
