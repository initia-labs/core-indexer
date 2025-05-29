module github.com/initia-labs/core-indexer/api

go 1.23.6

toolchain go1.24.0

require (
	github.com/gofiber/fiber/v2 v2.52.0
	github.com/gofiber/swagger v0.1.14
	github.com/initia-labs/core-indexer/pkg v0.0.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/rs/zerolog v1.33.0
	github.com/swaggo/swag v1.16.3
)

replace github.com/initia-labs/core-indexer/pkg => ../pkg

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/go-openapi/jsonpointer v0.20.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/spec v0.20.9 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/swaggo/files/v2 v2.0.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.51.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/tools v0.16.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
