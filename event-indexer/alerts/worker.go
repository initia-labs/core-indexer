package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"
)

type Telegram struct {
	Token, ChatID string
	Client        *http.Client
}

func (t Telegram) Send(ctx context.Context, text string) error {
	text = telegramText(text)
	body, err := json.Marshal(map[string]any{"chat_id": t.ChatID, "text": text, "link_preview_options": map[string]bool{"is_disabled": true}})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+t.Token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Telegram request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.Client.Do(req)
	// HTTP errors include the URL, which contains the bot token.
	if err != nil {
		return fmt.Errorf("Telegram request failed")
	}
	defer resp.Body.Close()
	var result struct {
		OK        bool `json:"ok"`
		ErrorCode int  `json:"error_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("Telegram returned invalid JSON (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK || !result.OK {
		return fmt.Errorf("Telegram rejected message (HTTP %d, code %d)", resp.StatusCode, result.ErrorCode)
	}
	return nil
}

// Legacy rows have no trusted summary metadata. Keep their prefix rather than
// promoting arbitrary Amount/Event lines from free-form text.
func telegramText(text string) string {
	const notice = "\n[truncated; full message retained in database]"
	if utf8.RuneCountInString(text) <= 4096 {
		return text
	}
	return string([]rune(text)[:4096-utf8.RuneCountInString(notice)]) + notice
}

// Build the persisted preview from evaluated totals, before adding route names
// or event details. Never rediscover totals by parsing free-form message lines.
func alertPreview(header string, totals []string, details string) string {
	const limit = 4096
	full := header + "\n\n" + strings.Join(totals, "\n\n") + details
	if utf8.RuneCountInString(full) <= limit {
		return full
	}
	preview := header
	for i, total := range totals {
		notice := fmt.Sprintf("\n[%d totals omitted; full message retained in database]", len(totals)-i)
		if utf8.RuneCountInString(preview+"\n\n"+total+notice) > limit {
			return telegramText(preview + notice)
		}
		preview += "\n\n" + total
	}
	const notice = "\n[details truncated; full message retained in database]"
	remaining := limit - utf8.RuneCountInString(preview+notice)
	if remaining < 0 {
		return telegramText(preview)
	}
	runes := []rune(details)
	return preview + string(runes[:min(remaining, len(runes))]) + notice
}
