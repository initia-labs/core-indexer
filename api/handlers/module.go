package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/services"
)

// ModuleHandler handles module-related HTTP requests
type ModuleHandler struct {
	service services.ModuleService
}

func NewModuleHandler(service services.ModuleService) *ModuleHandler {
	return &ModuleHandler{
		service: service,
	}
}

// GetModules godoc
// @Summary Get modules
// @Description Retrieve a list of modules with pagination
// @Tags Module
// @Accept json
// @Produce json
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Count total" default(false)
// @Success 200 {object} dto.ModulesResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/module/v1/modules [get]
func (h *ModuleHandler) GetModules(c *fiber.Ctx) error {
	// Parse pagination parameters manually
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	// Get modules from service
	response, err := h.service.GetModules(*pagination)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetModuleById godoc
// @Summary Get module by id
// @Description Retrieve a module by id
// @Tags Module
// @Accept json
// @Produce json
// @Param vmAddress path string true "VM address"
// @Param name path string true "Module name"
// @Success 200 {object} dto.ModuleResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/module/v1/modules/{vmAddress}/{name} [get]
func (h *ModuleHandler) GetModuleById(c *fiber.Ctx) error {
	vmAddress := strings.ToLower(c.Params("vmAddress"))
	name := c.Params("name")

	response, err := h.service.GetModuleById(vmAddress, name)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetModuleHistories godoc
// @Summary Get module histories
// @Description Retrieve a list of module histories with pagination
// @Tags Module
// @Accept json
// @Produce json
// @Param vmAddress path string true "VM address"
// @Param name path string true "Module name"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Count total" default(false)
// @Success 200 {object} dto.ModuleHistoriesResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/module/v1/modules/{vmAddress}/{name}/histories [get]
func (h *ModuleHandler) GetModuleHistories(c *fiber.Ctx) error {
	vmAddress := strings.ToLower(c.Params("vmAddress"))
	name := c.Params("name")

	// Parse pagination parameters manually
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	// Get module histories from service
	response, err := h.service.GetModuleHistories(*pagination, vmAddress, name)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetModulePublishInfo godoc
// @Summary Get module publish info
// @Description Retrieve a module publish info
// @Tags Module
// @Accept json
// @Produce json
// @Param vmAddress path string true "VM address"
// @Param name path string true "Module name"
// @Success 200 {object} dto.ModulePublishInfoResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/module/v1/modules/{vmAddress}/{name}/publish_info [get]
func (h *ModuleHandler) GetModulePublishInfo(c *fiber.Ctx) error {
	vmAddress := strings.ToLower(c.Params("vmAddress"))
	name := c.Params("name")

	response, err := h.service.GetModulePublishInfo(vmAddress, name)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetModuleProposals godoc
// @Summary Get module proposal
// @Description Retrieve a module proposal
// @Tags Module
// @Accept json
// @Produce json
// @Param vmAddress path string true "VM address"
// @Param name path string true "Module name"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Count total" default(false)
// @Success 200 {object} dto.ModuleProposalsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/module/v1/modules/{vmAddress}/{name}/proposals [get]
func (h *ModuleHandler) GetModuleProposals(c *fiber.Ctx) error {
	vmAddress := strings.ToLower(c.Params("vmAddress"))
	name := c.Params("name")

	// Parse pagination parameters manually
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetModuleProposals(*pagination, vmAddress, name)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetModuleTransactions godoc
// @Summary Get module transaction
// @Description Retrieve a module transaction
// @Tags Module
// @Accept json
// @Produce json
// @Param vmAddress path string true "VM address"
// @Param name path string true "Module name"
// @Param pagination.offset query integer false "Offset for pagination" default(0)
// @Param pagination.limit query integer false "Limit for pagination" default(10)
// @Param pagination.count_total query boolean false "Count total" default(false)
// @Success 200 {object} dto.ModuleTxsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/module/v1/modules/{vmAddress}/{name}/transactions [get]
func (h *ModuleHandler) GetModuleTransactions(c *fiber.Ctx) error {
	vmAddress := strings.ToLower(c.Params("vmAddress"))
	name := c.Params("name")

	// Parse pagination parameters manually
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	response, err := h.service.GetModuleTransactions(*pagination, vmAddress, name)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}

// GetModuleStats godoc
// @Summary Get module stats by module id
// @Description Retrieve a module stats
// @Tags Module
// @Accept json
// @Produce json
// @Param vmAddress path string true "VM address"
// @Param name path string true "Module name"
// @Success 200 {object} dto.ModuleStatsResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/module/v1/modules/{vmAddress}/{name}/stats [get]
func (h *ModuleHandler) GetModuleStats(c *fiber.Ctx) error {
	vmAddress := strings.ToLower(c.Params("vmAddress"))
	name := c.Params("name")

	response, err := h.service.GetModuleStats(vmAddress, name)
	if err != nil {
		return apperror.HandleErrorResponse(c, err)
	}

	return c.JSON(response)
}
