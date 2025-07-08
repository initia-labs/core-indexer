package db_test

// import (
// 	"fmt"
// 	"testing"

// 	"github.com/cometbft/cometbft/types"
// 	"github.com/stretchr/testify/require"
// 	"go.uber.org/mock/gomock"

// 	"github.com/alleslabs/initia-mono/generic-indexer/common"
// 	"github.com/alleslabs/initia-mono/generic-indexer/db"
// 	"github.com/alleslabs/initia-mono/generic-indexer/mocks"
// )

// func setupMockTx(t *testing.T) *mocks.MockTx {
// 	ctrl := gomock.NewController(t)
// 	return mocks.NewMockTx(ctrl)
// }

// func setupBlockMsg(num int) (*common.BlockMsg, *types.ValidatorSet, []types.PrivValidator) {
// 	valSet, privValidators := types.RandValidatorSet(num, 100)
// 	sigs := make([]types.CommitSig, 0)
// 	for _, pv := range privValidators {
// 		pk, _ := pv.GetPubKey()
// 		sigs = append(sigs, types.CommitSig{Signature: pk.Bytes(), ValidatorAddress: pk.Address(), BlockIDFlag: types.BlockIDFlagCommit})
// 	}
// 	//60299126
// 	proposer := "initvaloper1h8lpl5qcs5k5nngxvdum5v20jnww2lcku52n9d"
// 	lastCommit := &types.Commit{
// 		Signatures: sigs,
// 	}
// 	return &common.BlockMsg{
// 		Proposer:   &proposer,
// 		LastCommit: lastCommit,
// 	}, valSet, privValidators
// }

// func TestGenerateGetAccountsStmt(t *testing.T) {
// 	addresses := []string{
// 		"sei1q2k0p9x6f5jlvnsdzu7cj2k9jfp9ee22xr32u9",
// 		"sei1qdmt7sq86mawwq62gl3w9aheu3ak3vtqkasf6t",
// 		"sei1qnn8rvtlfya3u4qvryqtjnc6gum4ltajj84uut",
// 		"sei1pcv7znvrnpwqg4xhy8ww36tq08s9warugqtqln",
// 		"sei1r35u5wseduv34h77zx88l49dc4s3cmq2sjtm22",
// 		"sei1ytenacqmp6xzwwfa42shg3ch9vhzk6z5qz6x9m",
// 		"sei19t4d78pt5xg6xm0mf97q8ay7v0xwrr4j9x7mnp",
// 		"sei183xtf2wmcah9fh5kpdr47wlspv3w47c0lgvwvf",
// 		"sei186kxval3h94wazflk0r8ca0nk54ueetzj5ssxl",
// 		"sei1d0xlql0fykcfar75mjf3m68l2x8re753hl9fs9",
// 		"sei1d7ma9amuchzyflqrr2ushzljduq969js7vhd8c",
// 		"sei1wyqrf3n4cnzj5g5tppntvwxwnj6dr8sr3655x9",
// 		"sei1w94ynn6xxsjxau2svdd8cucdupwzyrw7z057fr",
// 		"sei1wxqxlmyp3h5sjgvz525hauyuqu9zavq4wxx0jv",
// 		"sei1waglkj7r9vh5xe3cx72799rud4msculnrgvlue",
// 		"sei1sq7x0r2mf3gvwr2l9amtlye0yd3c6dqatyxt4u",
// 		"sei13dmp202rnljk03jae79qxw8ywcj9kdff5vvp79",
// 		"sei1n4r5qfawp3wxkjgw8pmp28w6kcjhv9a02ah94h",
// 		"sei1n4g85ut9hhaksfmnwpje5lrec6x9tezn08mwfy",
// 		"sei15lu2zzp9ll0x6weth0y0wr3ur7rd4f9300g825",
// 		"sei1kv8khcjwg2e52fd7kdcpgx6hnwepg8rkwnwlc5",
// 		"sei1kn794s9nj5x8dr0ycuuusxptthexjys46dtcpk",
// 		"sei1hzhh8wmp7y46l0gxpehy5hhmpsh4mjdg2dqwgv",
// 		"sei1hsavr43a7m9zxal2s6s08yvy05yftm8fjwyrsm",
// 		"sei1hmvektdsvw3prajtuxem5z3ymrgha599eqcl7x",
// 		"sei1eexg4ys5zp2qv4f5ytut0qlrtx65tk6rcj9skh",
// 		"sei16wma8ngawcgxdwlut644f4jdmd96agsarjs3g4",
// 		"sei16mplw469mp3wt0edv0gd87wn66p0vy6n9d8ap5",
// 		"sei16aeqsqxy5slg6xhvfk62u78ucnaqs20r269kda",
// 		"sei1lr4klneaxm6djkllmclt8lzpvv0lr7tlex8ztw",
// 		"sei1ln9ydqs9ycy6jp2lxjw4gzxqj9un9wexlzs6ge",
// 		"sei1q0ejqj0mg76cq2885rf6qrwvtht3nqgdml46z7",
// 		"sei1q5nkn98602wld2kqkm4sh5fhdkfaap4y56939k",
// 	}

// 	result := db.GenerateGetAccountsStmt(addresses)
// 	require.Equal(t, result, "SELECT id, address FROM accounts a WHERE address in ('sei1q2k0p9x6f5jlvnsdzu7cj2k9jfp9ee22xr32u9','sei1qdmt7sq86mawwq62gl3w9aheu3ak3vtqkasf6t','sei1qnn8rvtlfya3u4qvryqtjnc6gum4ltajj84uut','sei1pcv7znvrnpwqg4xhy8ww36tq08s9warugqtqln','sei1r35u5wseduv34h77zx88l49dc4s3cmq2sjtm22','sei1ytenacqmp6xzwwfa42shg3ch9vhzk6z5qz6x9m','sei19t4d78pt5xg6xm0mf97q8ay7v0xwrr4j9x7mnp','sei183xtf2wmcah9fh5kpdr47wlspv3w47c0lgvwvf','sei186kxval3h94wazflk0r8ca0nk54ueetzj5ssxl','sei1d0xlql0fykcfar75mjf3m68l2x8re753hl9fs9','sei1d7ma9amuchzyflqrr2ushzljduq969js7vhd8c','sei1wyqrf3n4cnzj5g5tppntvwxwnj6dr8sr3655x9','sei1w94ynn6xxsjxau2svdd8cucdupwzyrw7z057fr','sei1wxqxlmyp3h5sjgvz525hauyuqu9zavq4wxx0jv','sei1waglkj7r9vh5xe3cx72799rud4msculnrgvlue','sei1sq7x0r2mf3gvwr2l9amtlye0yd3c6dqatyxt4u','sei13dmp202rnljk03jae79qxw8ywcj9kdff5vvp79','sei1n4r5qfawp3wxkjgw8pmp28w6kcjhv9a02ah94h','sei1n4g85ut9hhaksfmnwpje5lrec6x9tezn08mwfy','sei15lu2zzp9ll0x6weth0y0wr3ur7rd4f9300g825','sei1kv8khcjwg2e52fd7kdcpgx6hnwepg8rkwnwlc5','sei1kn794s9nj5x8dr0ycuuusxptthexjys46dtcpk','sei1hzhh8wmp7y46l0gxpehy5hhmpsh4mjdg2dqwgv','sei1hsavr43a7m9zxal2s6s08yvy05yftm8fjwyrsm','sei1hmvektdsvw3prajtuxem5z3ymrgha599eqcl7x','sei1eexg4ys5zp2qv4f5ytut0qlrtx65tk6rcj9skh','sei16wma8ngawcgxdwlut644f4jdmd96agsarjs3g4','sei16mplw469mp3wt0edv0gd87wn66p0vy6n9d8ap5','sei16aeqsqxy5slg6xhvfk62u78ucnaqs20r269kda','sei1lr4klneaxm6djkllmclt8lzpvv0lr7tlex8ztw','sei1ln9ydqs9ycy6jp2lxjw4gzxqj9un9wexlzs6ge','sei1q0ejqj0mg76cq2885rf6qrwvtht3nqgdml46z7','sei1q5nkn98602wld2kqkm4sh5fhdkfaap4y56939k')")

// 	addresses = []string{"sei1q5nkn98602wld2kqkm4sh5fhdkfaap4y56939k"}
// 	result = db.GenerateGetAccountsStmt(addresses)
// 	require.Equal(t, result, "SELECT id, address FROM accounts a WHERE address in ('sei1q5nkn98602wld2kqkm4sh5fhdkfaap4y56939k')")

// }

// func TestGenerateInsertAccountsStmt(t *testing.T) {
// 	addresses := []string{
// 		"sei1q2k0p9x6f5jlvnsdzu7cj2k9jfp9ee22xr32u9",
// 		"sei1qdmt7sq86mawwq62gl3w9aheu3ak3vtqkasf6t",
// 		"sei1qnn8rvtlfya3u4qvryqtjnc6gum4ltajj84uut",
// 		"sei1pcv7znvrnpwqg4xhy8ww36tq08s9warugqtqln",
// 		"sei1r35u5wseduv34h77zx88l49dc4s3cmq2sjtm22",
// 		"sei1ytenacqmp6xzwwfa42shg3ch9vhzk6z5qz6x9m",
// 		"sei19t4d78pt5xg6xm0mf97q8ay7v0xwrr4j9x7mnp",
// 		"sei183xtf2wmcah9fh5kpdr47wlspv3w47c0lgvwvf",
// 		"sei186kxval3h94wazflk0r8ca0nk54ueetzj5ssxl",
// 		"sei1d0xlql0fykcfar75mjf3m68l2x8re753hl9fs9",
// 		"sei1d7ma9amuchzyflqrr2ushzljduq969js7vhd8c",
// 		"sei1wyqrf3n4cnzj5g5tppntvwxwnj6dr8sr3655x9",
// 		"sei1w94ynn6xxsjxau2svdd8cucdupwzyrw7z057fr",
// 		"sei1wxqxlmyp3h5sjgvz525hauyuqu9zavq4wxx0jv",
// 		"sei1waglkj7r9vh5xe3cx72799rud4msculnrgvlue",
// 		"sei1sq7x0r2mf3gvwr2l9amtlye0yd3c6dqatyxt4u",
// 		"sei13dmp202rnljk03jae79qxw8ywcj9kdff5vvp79",
// 		"sei1n4r5qfawp3wxkjgw8pmp28w6kcjhv9a02ah94h",
// 		"sei1n4g85ut9hhaksfmnwpje5lrec6x9tezn08mwfy",
// 		"sei15lu2zzp9ll0x6weth0y0wr3ur7rd4f9300g825",
// 		"sei1kv8khcjwg2e52fd7kdcpgx6hnwepg8rkwnwlc5",
// 		"sei1kn794s9nj5x8dr0ycuuusxptthexjys46dtcpk",
// 		"sei1hzhh8wmp7y46l0gxpehy5hhmpsh4mjdg2dqwgv",
// 		"sei1hsavr43a7m9zxal2s6s08yvy05yftm8fjwyrsm",
// 		"sei1hmvektdsvw3prajtuxem5z3ymrgha599eqcl7x",
// 		"sei1eexg4ys5zp2qv4f5ytut0qlrtx65tk6rcj9skh",
// 		"sei16wma8ngawcgxdwlut644f4jdmd96agsarjs3g4",
// 		"sei16mplw469mp3wt0edv0gd87wn66p0vy6n9d8ap5",
// 		"sei16aeqsqxy5slg6xhvfk62u78ucnaqs20r269kda",
// 		"sei1lr4klneaxm6djkllmclt8lzpvv0lr7tlex8ztw",
// 		"sei1ln9ydqs9ycy6jp2lxjw4gzxqj9un9wexlzs6ge",
// 		"sei1q0ejqj0mg76cq2885rf6qrwvtht3nqgdml46z7",
// 		"sei1q5nkn98602wld2kqkm4sh5fhdkfaap4y56939k",
// 	}
// 	result := db.GenerateInsertAccountsStmt(addresses)
// 	fmt.Println(db.GenerateInsertAccountsStmt(addresses))
// 	require.Equal(t, result, "SELECT id, address FROM accounts a WHERE address in ('sei1q5nkn98602wld2kqkm4sh5fhdkfaap4y56939k')")

// }
