package docs

// @title Core Indexer API
// @version 1.0
// @description Core Indexer API for NFT collections
// @host localhost:8080
// @BasePath /api/v1

// @tag.name NFT
// @tag.description NFT collection operations

// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @query.collection.param.pagination.limit query int false "Limit the number of results (default: 10, max: 100)" default(10) minimum(1) maximum(100)
// @query.collection.param.pagination.offset query int false "Offset for pagination (default: 0)" default(0) minimum(0)
// @query.collection.param.pagination.key query string false "Base64 encoded key for cursor-based pagination"
// @query.collection.param.pagination.reverse query boolean false "Reverse the order of results (default: true)" default(true)
// @query.collection.param.pagination.count_total query boolean false "Count total number of results (default: false)" default(false)
// @query.collection.param.search query string false "Search term for filtering collections by name or ID"

// @response.collection.success 200 {object} dto.NFTCollectionsResponse "Successfully retrieved NFT collections"
// @response.collection.error.400 {object} dto.ErrorResponse "Invalid request parameters"
// @response.collection.error.500 {object} dto.ErrorResponse "Internal server error"

// @response.collection.single.success 200 {object} dto.NFTCollectionResponse "Successfully retrieved NFT collection"
// @response.collection.single.error.404 {object} dto.ErrorResponse "Collection not found"
// @response.collection.single.error.500 {object} dto.ErrorResponse "Internal server error"

// @response.error.401 {object} dto.ErrorResponse "Unauthorized"
// @response.error.403 {object} dto.ErrorResponse "Forbidden"
// @response.error.404 {object} dto.ErrorResponse "Not found"
// @response.error.500 {object} dto.ErrorResponse "Internal server error"

// @response.error.schema {object} dto.ErrorResponse
// @response.error.schema.example {"error":"Error message","code":400}

// @response.collection.schema {object} dto.NFTCollectionsResponse
// @response.collection.schema.example {
//   "collections": [
//     {
//       "id": "collection1",
//       "name": "My Collection",
//       "uri": "https://example.com/collection1",
//       "description": "A collection of NFTs",
//       "creator": "creator1"
//     }
//   ],
//   "pagination": {
//     "next_key": "base64encodedkey",
//     "total": 100
//   }
// }

// @response.collection.single.schema {object} dto.NFTCollectionResponse
// @response.collection.single.schema.example {
//   "collection": {
//     "object_addr": "0x123",
//     "collection": {
//       "creator": "creator1",
//       "description": "A collection of NFTs",
//       "name": "My Collection",
//       "uri": "https://example.com/collection1",
//       "nfts": {
//         "handle": "0x456",
//         "length": "10"
//       }
//     }
//   }
// }
