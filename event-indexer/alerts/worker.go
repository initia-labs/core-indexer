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

// Bound at send time so an oversized persisted message can also be retried.
func telegramText(text string) string {
	const limit = 4096
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	var fields [2]string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "Amount:") && fields[0] == "" {
			fields[0] = line
		}
		if strings.HasPrefix(line, "Event:") {
			fields[1] = line
		}
	}
	var important string
	for _, line := range fields {
		if line == "" {
			continue
		}
		runes := []rune(line)
		if len(runes) > 1024 {
			line = string(runes[:1023]) + "…"
		}
		important += line + "\n"
	}
	const notice = "\n[truncated; full message retained in database]"
	remaining := limit - utf8.RuneCountInString(important+notice)
	return important + string([]rune(text)[:remaining]) + notice
}
