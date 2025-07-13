package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TelegramConfig struct {
	BotToken      string        `yaml:"bot_token"`
	ChatID        int64         `yaml:"chat_id"`
	Enabled       bool          `yaml:"enabled"`
	Timeout       time.Duration `yaml:"timeout"`
	RetryAttempts int           `yaml:"retry_attempts"`
}

type TelegramClient struct {
	config TelegramConfig
	client *http.Client
	apiURL string
}

type SendMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type TelegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	ErrorCode   int    `json:"error_code,omitempty"`
}

func NewTelegramClient(config TelegramConfig) *TelegramClient {
	return &TelegramClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		apiURL: fmt.Sprintf("https://api.telegram.org/bot%s", config.BotToken),
	}
}

func (t *TelegramClient) SendMessage(ctx context.Context, text string) error {
	if !t.config.Enabled {
		return fmt.Errorf("telegram client is disabled")
	}

	if t.config.ChatID == 0 {
		return fmt.Errorf("telegram chat_id is not configured")
	}

	message := SendMessageRequest{
		ChatID:    t.config.ChatID,
		Text:      text,
		ParseMode: "HTML",
	}

	var lastErr error
	for attempt := 1; attempt <= t.config.RetryAttempts; attempt++ {
		if err := t.sendMessageAttempt(ctx, message); err != nil {
			lastErr = err
			if attempt < t.config.RetryAttempts {
				delay := time.Duration(attempt) * time.Second
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("failed to send telegram message after %d attempts: %w",
		t.config.RetryAttempts, lastErr)
}

func (t *TelegramClient) sendMessageAttempt(ctx context.Context, message SendMessageRequest) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.apiURL+"/sendMessage", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var telegramResp TelegramResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if !telegramResp.OK {
		return fmt.Errorf("telegram API error %d: %s", telegramResp.ErrorCode, telegramResp.Description)
	}

	return nil
}

func (t *TelegramClient) IsEnabled() bool {
	return t.config.Enabled && t.config.ChatID != 0 && t.config.BotToken != ""
}
