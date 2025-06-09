package parser

type NftResource struct {
	Type string `json:"type"`
	Data struct {
		Description string `json:"description"`
		TokenID     string `json:"token_id"`
		URI         string `json:"uri"`
	} `json:"data"`
}

type CollectionResource struct {
	Type string `json:"type"`
	Data struct {
		Creator     string `json:"creator"`
		Description string `json:"description"`
		Name        string `json:"name"`
		Nfts        struct {
			Handle string `json:"handle"`
			Length string `json:"length"`
		} `json:"nfts"`
		URI string `json:"uri"`
	} `json:"data"`
}
