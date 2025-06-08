package handlers

import (
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
// @Success 200 {object} dto.ModulesResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/module/v1/modules [get]
func (h *ModuleHandler) GetModules(c *fiber.Ctx) error {
	// Parse pagination parameters manually
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	// Get modules from service
	response, err := h.service.GetModules(*pagination)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetModuleById godoc
// @Summary Get module by id
// @Description Retrieve a module by id
// @Tags Module
// @Accept json
// @Produce json
// @Param vmAddress query string true "VM address"
// @Param name query string true "Name"
// @Success 200 {object} dto.ModuleResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/module/v1/modules/{vmAddress}/{name} [get]
func (h *ModuleHandler) GetModuleById(c *fiber.Ctx) error {
	vmAddress := c.Params("vmAddress")
	name := c.Params("name")

	response, err := h.service.GetModuleById(vmAddress, name)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}

// GetModuleHistories godoc
// @Summary Get module histories
// @Description Retrieve a list of module histories with pagination
// @Tags Module
// @Accept json
// @Produce json
// @Param vmAddress query string true "VM address"
// @Param name query string true "Name"
// @Success 200 {object} dto.ModuleHistoriesResponse
// @Failure 400 {object} apperror.Response
// @Failure 500 {object} apperror.Response
// @Router /indexer/module/v1/modules/{vmAddress}/{name}/histories [get]
func (h *ModuleHandler) GetModuleHistories(c *fiber.Ctx) error {
	vmAddress := c.Params("vmAddress")
	name := c.Params("name")

	// Parse pagination parameters manually
	pagination, err := dto.PaginationFromQuery(c)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	// Get module histories from service
	response, err := h.service.GetModuleHistories(*pagination, vmAddress, name)
	if err != nil {
		errResp := apperror.HandleError(err)
		return c.Status(errResp.Code).JSON(errResp)
	}

	return c.JSON(response)
}
