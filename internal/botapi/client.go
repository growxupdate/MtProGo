package botapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is a tiny Telegram Bot API client built only with the Go standard library.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// New creates a Bot API client.
func New(token string) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("bot token is required")
	}
	return &Client{
		token:   token,
		baseURL: "https://api.telegram.org/bot" + token + "/",
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}, nil
}

// User is a Telegram Bot API user.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

// Chat is a Telegram Bot API chat.
type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username"`
}

// Message is a Telegram Bot API message subset.
type Message struct {
	MessageID int    `json:"message_id"`
	From      *User  `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

// Update is a Telegram Bot API update subset.
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

type apiResponse[T any] struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Result      T      `json:"result"`
}

// GetMe returns the current bot user.
func (c *Client) GetMe(ctx context.Context) (User, error) {
	var result apiResponse[User]
	if err := c.postForm(ctx, "getMe", url.Values{}, &result); err != nil {
		return User{}, err
	}
	if !result.OK {
		return User{}, errors.New(result.Description)
	}
	return result.Result, nil
}

// GetUpdates polls updates.
func (c *Client) GetUpdates(ctx context.Context, offset int, timeoutSeconds int) ([]Update, error) {
	form := url.Values{}
	if offset > 0 {
		form.Set("offset", strconv.Itoa(offset))
	}
	if timeoutSeconds > 0 {
		form.Set("timeout", strconv.Itoa(timeoutSeconds))
	}
	form.Set("allowed_updates", `["message"]`)

	var result apiResponse[[]Update]
	if err := c.postForm(ctx, "getUpdates", form, &result); err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, errors.New(result.Description)
	}
	return result.Result, nil
}

// SendMessage sends a text message.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, replyToMessageID int) error {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("text", text)
	if replyToMessageID > 0 {
		form.Set("reply_to_message_id", strconv.Itoa(replyToMessageID))
	}

	var result apiResponse[Message]
	if err := c.postForm(ctx, "sendMessage", form, &result); err != nil {
		return err
	}
	if !result.OK {
		return errors.New(result.Description)
	}
	return nil
}

func (c *Client) postForm(ctx context.Context, method string, form url.Values, out any) error {
	endpoint := c.baseURL + method
	body := bytes.NewBufferString(form.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram bot api http %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return json.Unmarshal(data, out)
}
