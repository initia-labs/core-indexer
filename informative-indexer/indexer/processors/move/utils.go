package move

import "fmt"

func makeNftsMapKey(collectionAddr, tokenId string) string {
	return fmt.Sprintf("%s:%s", collectionAddr, tokenId)
}
