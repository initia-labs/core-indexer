package alerts

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
)

// Threshold is expressed in display tokens; comparisons use integer base units.
type Asset struct {
	Denom        string   `json:"denom"`
	PacketDenoms []string `json:"packet_denoms"`
	Decimals     int      `json:"decimals"`
	Threshold    string   `json:"threshold"`
	limit        *big.Int
}

type Route struct {
	Name    string   `json:"name"`
	Port    string   `json:"port"`
	Channel string   `json:"channel"`
	Assets  []string `json:"assets"`
}

type Escrow struct {
	Name    string   `json:"name"`
	Address string   `json:"address"`
	Assets  []string `json:"assets"`
}

type LayerZero struct {
	Name         string   `json:"name"`
	TypeTag      string   `json:"type_tag"`
	Asset        string   `json:"asset"`
	Destinations []uint32 `json:"destinations"`
}

type Config struct {
	Chain       string           `json:"chain"`
	TxURL       string           `json:"tx_url"` // Explorer prefix, followed by the tx hash.
	Assets      map[string]Asset `json:"assets"`
	IBC         []Route          `json:"ibc_routes"`
	Escrows     []Escrow         `json:"escrows"`
	LayerZero   []LayerZero      `json:"layerzero"`
	Workers     int              `json:"workers"`    // Concurrent DB evaluation workers.
	BatchSize   int              `json:"batch_size"` // Maximum event groups per DB transaction.
	StartHeight int64            `json:"start_height"`
}

func LoadConfig(path string) (Config, error) {
	var c Config
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return c, fmt.Errorf("rules file must contain exactly one JSON object")
	}
	return c, c.prepare()
}

func (c *Config) prepare() error {
	if c.Chain == "" {
		return fmt.Errorf("chain is required")
	}
	if c.Workers == 0 {
		c.Workers = 4
	}
	if c.Workers < 1 {
		return fmt.Errorf("workers must be positive")
	}
	if c.BatchSize == 0 {
		c.BatchSize = 100
	}
	if c.BatchSize < 1 {
		return fmt.Errorf("batch_size must be positive")
	}
	for id, a := range c.Assets {
		if a.Decimals < 0 || a.Decimals > 255 {
			return fmt.Errorf("asset %s: decimals must be between 0 and 255", id)
		}
		n, err := baseUnits(a.Threshold, a.Decimals)
		if err != nil || n.Sign() <= 0 {
			return fmt.Errorf("asset %s: threshold must be positive and fit its decimals", id)
		}
		a.limit = n
		c.Assets[id] = a
	}
	checkAssets := func(ids []string, packet bool) error {
		if len(ids) == 0 {
			return fmt.Errorf("assets must not be empty")
		}
		for _, id := range ids {
			a, ok := c.Assets[id]
			if !ok {
				return fmt.Errorf("unknown asset %q", id)
			}
			if packet && len(a.PacketDenoms) == 0 {
				return fmt.Errorf("asset %s: packet_denoms required for IBC", id)
			}
			if !packet && a.Denom == "" {
				return fmt.Errorf("asset %s: denom required for escrow", id)
			}
		}
		return nil
	}
	for _, r := range c.IBC {
		if r.Port == "" || r.Channel == "" {
			return fmt.Errorf("IBC route requires port and channel")
		}
		if err := checkAssets(r.Assets, true); err != nil {
			return err
		}
	}
	for i := range c.Escrows {
		e := &c.Escrows[i]
		e.Address = strings.ToLower(e.Address)
		if e.Address == "" {
			return fmt.Errorf("escrow address is required")
		}
		if err := checkAssets(e.Assets, false); err != nil {
			return err
		}
	}
	for _, l := range c.LayerZero {
		if !strings.HasSuffix(l.TypeTag, "::oft_core::OftSent") {
			return fmt.Errorf("LayerZero requires the full deployed ::oft_core::OftSent type tag")
		}
		if _, ok := c.Assets[l.Asset]; !ok {
			return fmt.Errorf("unknown LayerZero asset %q", l.Asset)
		}
	}
	if len(c.IBC)+len(c.Escrows)+len(c.LayerZero) == 0 {
		return fmt.Errorf("at least one monitored route or escrow is required")
	}
	return nil
}

func baseUnits(s string, decimals int) (*big.Int, error) {
	parts := strings.Split(s, ".")
	if len(parts) > 2 || parts[0] == "" {
		return nil, fmt.Errorf("invalid token amount")
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > decimals {
		return nil, fmt.Errorf("too many decimal places")
	}
	return unsigned(parts[0] + fraction + strings.Repeat("0", decimals-len(fraction)))
}

func unsigned(s string) (*big.Int, error) {
	if s == "" {
		return nil, fmt.Errorf("missing amount")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return nil, fmt.Errorf("invalid integer amount")
		}
	}
	n, _ := new(big.Int).SetString(s, 10)
	return n, nil
}

func display(n *big.Int, decimals int) string {
	s := n.String()
	if decimals == 0 {
		return s
	}
	if len(s) <= decimals {
		s = strings.Repeat("0", decimals+1-len(s)) + s
	}
	s = strings.TrimRight(s[:len(s)-decimals]+"."+s[len(s)-decimals:], "0")
	return strings.TrimSuffix(s, ".")
}
