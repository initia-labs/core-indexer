package dto

// ModuleResponse represents the response for a module
type ModuleResponse struct {
	ModuleName    string `json:"module_name"`
	Digest        string `json:"digest"`
	IsVerified    bool   `json:"is_verified"`
	Address       string `json:"address"`
	Height        int64  `json:"height"`
	LatestUpdated string `json:"latest_updated"`
	IsRepublished bool   `json:"is_republished"`
}

// ModulesResponse represents the response for a list of modules
type ModulesResponse struct {
	Modules    []ModuleResponse   `json:"modules"`
	Pagination PaginationResponse `json:"pagination"`
}

// ModuleHistory represents a module history
type ModuleHistory struct {
	Height        int64  `json:"height"`
	LatestUpdated string `json:"latest_updated"`
}

// ModuleHistoriesResponse represents the response for a list of module histories
type ModuleHistoriesResponse struct {
	ModuleHistories []ModuleHistory    `json:"module_histories"`
	Pagination      PaginationResponse `json:"pagination"`
}
