package alerts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigRejectsTrailingContent(t *testing.T) {
	const config = `{"chain":"test-1","assets":{"USDC":{"decimals":6,"threshold":"100000"}},"layerzero":[{"type_tag":"0x1::oft_core::OftSent","asset":"USDC"}]}`
	for _, suffix := range []string{" \n\t", `{}`, `garbage`} {
		path := filepath.Join(t.TempDir(), "alerts.json")
		if err := os.WriteFile(path, []byte(config+suffix), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := LoadConfig(path)
		if (err == nil) != (suffix == " \n\t") {
			t.Fatalf("suffix %q: %v", suffix, err)
		}
	}
}
