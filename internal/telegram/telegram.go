package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var apiURL = func(botToken string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
}

type sendMessageReq struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type sendMessageResp struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

func Send(botToken, chatID, message string) error {
	body := sendMessageReq{
		ChatID: chatID,
		Text:   message,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Post(apiURL(botToken), "application/json", &buf)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	var apiResp sendMessageResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !apiResp.OK {
		if apiResp.Description != "" {
			return fmt.Errorf("telegram API error: %s", apiResp.Description)
		}
		return fmt.Errorf("telegram API error: unknown")
	}

	return nil
}
