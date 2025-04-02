package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Common structs and interfaces
type MessagingService interface {
	CreatePost(content string, channelID string) (string, error)
	ReplyToComment(postID string, content string) (string, error)
	GetPostStats(postID string) (interface{}, error)
	GetCommunityStats(communityID string) (interface{}, error)
}

// ==================== WhatsApp Business API ====================

type WhatsAppClient struct {
	AccessToken   string
	PhoneNumberID string
	BaseURL       string
}

func NewWhatsAppClient(accessToken, phoneNumberID string) *WhatsAppClient {
	return &WhatsAppClient{
		AccessToken:   accessToken,
		PhoneNumberID: phoneNumberID,
		BaseURL:       "https://graph.facebook.com/v17.0",
	}
}

// CreatePost sends a message to a WhatsApp user
func (w *WhatsAppClient) CreatePost(content string, recipientPhone string) (string, error) {
	url := fmt.Sprintf("%s/%s/messages", w.BaseURL, w.PhoneNumberID)

	requestBody, err := json.Marshal(map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                recipientPhone,
		"type":              "text",
		"text": map[string]string{
			"body": content,
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Extract message ID
	if messages, ok := result["messages"].([]interface{}); ok && len(messages) > 0 {
		if message, ok := messages[0].(map[string]interface{}); ok {
			if id, ok := message["id"].(string); ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("failed to extract message ID")
}

// ReplyToComment replies to a specific message in WhatsApp
func (w *WhatsAppClient) ReplyToComment(messageID string, content string) (string, error) {
	url := fmt.Sprintf("%s/%s/messages", w.BaseURL, w.PhoneNumberID)

	// In WhatsApp Business API, we need recipient phone
	parts := struct {
		RecipientPhone string `json:"phone"`
		MessageID      string `json:"messageID"`
	}{}

	// In a real implementation, you would retrieve the recipient phone from the messageID
	// For this example, we assume it's provided in the messageID string as "phone:messageID"
	fmt.Sscanf(messageID, "%s:%s", &parts.RecipientPhone, &parts.MessageID)

	requestBody, err := json.Marshal(map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                parts.RecipientPhone,
		"type":              "text",
		"text": map[string]string{
			"body": content,
		},
		"context": map[string]string{
			"message_id": parts.MessageID,
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Extract reply message ID
	if messages, ok := result["messages"].([]interface{}); ok && len(messages) > 0 {
		if message, ok := messages[0].(map[string]interface{}); ok {
			if id, ok := message["id"].(string); ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("failed to extract reply message ID")
}

// GetPostStats gets message status information for a WhatsApp message
func (w *WhatsAppClient) GetPostStats(messageID string) (interface{}, error) {
	// WhatsApp Business API doesn't provide direct stats endpoint for a specific message
	// Instead we can use the messages status webhook
	// This function would retrieve stored message stats from your database

	// Mock implementation
	return map[string]interface{}{
		"message_id": messageID,
		"status":     "delivered",
		"timestamp":  "2023-08-01T12:34:56Z",
	}, nil
}

// GetCommunityStats gets statistics for a WhatsApp Business Account
func (w *WhatsAppClient) GetCommunityStats(wabaID string) (interface{}, error) {
	// For WhatsApp, we'd use the Insights API
	url := fmt.Sprintf("%s/%s/insights", w.BaseURL, wabaID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+w.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Additional WhatsApp functionalities
func (w *WhatsAppClient) SendMediaMessage(recipientPhone, mediaType, mediaURL string) (string, error) {
	url := fmt.Sprintf("%s/%s/messages", w.BaseURL, w.PhoneNumberID)

	requestBody, err := json.Marshal(map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                recipientPhone,
		"type":              mediaType, // image, audio, document, video
		mediaType: map[string]string{
			"link": mediaURL,
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Extract message ID
	if messages, ok := result["messages"].([]interface{}); ok && len(messages) > 0 {
		if message, ok := messages[0].(map[string]interface{}); ok {
			if id, ok := message["id"].(string); ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("failed to extract message ID")
}

// ==================== Telegram API ====================

type TelegramClient struct {
	BotToken string
	BaseURL  string
}

func NewTelegramClient(botToken string) *TelegramClient {
	return &TelegramClient{
		BotToken: botToken,
		BaseURL:  "https://api.telegram.org/bot",
	}
}

// CreatePost sends a message to a Telegram chat
func (t *TelegramClient) CreatePost(content string, chatID string) (string, error) {
	url := fmt.Sprintf("%s%s/sendMessage", t.BaseURL, t.BotToken)

	requestBody, err := json.Marshal(map[string]interface{}{
		"chat_id": chatID,
		"text":    content,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Check if request was successful
	if ok, exists := result["ok"].(bool); !exists || !ok {
		return "", fmt.Errorf("telegram API error: %v", result["description"])
	}

	// Extract message ID
	if resultData, ok := result["result"].(map[string]interface{}); ok {
		if messageID, ok := resultData["message_id"].(float64); ok {
			return fmt.Sprintf("%f", messageID), nil
		}
	}

	return "", fmt.Errorf("failed to extract message ID")
}

// ReplyToComment replies to a message in Telegram
func (t *TelegramClient) ReplyToComment(messageID string, content string) (string, error) {
	// In Telegram, we need both the chat_id and message_id
	parts := struct {
		ChatID    string
		MessageID string
	}{}

	// In a real implementation, you would retrieve the chat_id from the messageID
	// For this example, we assume it's provided in the messageID string as "chatID:messageID"
	fmt.Sscanf(messageID, "%s:%s", &parts.ChatID, &parts.MessageID)

	url := fmt.Sprintf("%s%s/sendMessage", t.BaseURL, t.BotToken)

	requestBody, err := json.Marshal(map[string]interface{}{
		"chat_id":             parts.ChatID,
		"text":                content,
		"reply_to_message_id": parts.MessageID,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Check if request was successful
	if ok, exists := result["ok"].(bool); !exists || !ok {
		return "", fmt.Errorf("telegram API error: %v", result["description"])
	}

	// Extract reply message ID
	if resultData, ok := result["result"].(map[string]interface{}); ok {
		if replyMessageID, ok := resultData["message_id"].(float64); ok {
			return fmt.Sprintf("%s:%f", parts.ChatID, replyMessageID), nil
		}
	}

	return "", fmt.Errorf("failed to extract reply message ID")
}

// GetPostStats gets information about a message in Telegram
func (t *TelegramClient) GetPostStats(messageID string) (interface{}, error) {
	// Telegram API doesn't have a direct endpoint for message stats
	// For group/channel posts, we can get view count via getMessages method
	parts := struct {
		ChatID    string
		MessageID string
	}{}

	fmt.Sscanf(messageID, "%s:%s", &parts.ChatID, &parts.MessageID)

	url := fmt.Sprintf("%s%s/getMessages", t.BaseURL, t.BotToken)

	requestBody, err := json.Marshal(map[string]interface{}{
		"chat_id":     parts.ChatID,
		"message_ids": []string{parts.MessageID},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Extract message info
	if resultData, ok := result["result"].([]interface{}); ok && len(resultData) > 0 {
		if message, ok := resultData[0].(map[string]interface{}); ok {
			return message, nil
		}
	}

	return nil, fmt.Errorf("failed to get message stats")
}

// GetCommunityStats gets stats for a Telegram channel or group
func (t *TelegramClient) GetCommunityStats(chatID string) (interface{}, error) {
	url := fmt.Sprintf("%s%s/getChatMembersCount", t.BaseURL, t.BotToken)

	requestBody, err := json.Marshal(map[string]interface{}{
		"chat_id": chatID,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %s", string(body))
	}

	var memberCountResult map[string]interface{}
	if err := json.Unmarshal(body, &memberCountResult); err != nil {
		return nil, err
	}

	// Get chat information
	chatInfoURL := fmt.Sprintf("%s%s/getChat", t.BaseURL, t.BotToken)

	chatInfoRequestBody, err := json.Marshal(map[string]interface{}{
		"chat_id": chatID,
	})
	if err != nil {
		return nil, err
	}

	chatInfoReq, err := http.NewRequest("POST", chatInfoURL, bytes.NewBuffer(chatInfoRequestBody))
	if err != nil {
		return nil, err
	}

	chatInfoReq.Header.Set("Content-Type", "application/json")

	chatInfoResp, err := client.Do(chatInfoReq)
	if err != nil {
		return nil, err
	}
	defer chatInfoResp.Body.Close()

	chatInfoBody, err := ioutil.ReadAll(chatInfoResp.Body)
	if err != nil {
		return nil, err
	}

	var chatInfoResult map[string]interface{}
	if err := json.Unmarshal(chatInfoBody, &chatInfoResult); err != nil {
		return nil, err
	}

	// Combine results
	stats := map[string]interface{}{
		"member_count": memberCountResult["result"],
		"chat_info":    chatInfoResult["result"],
	}

	return stats, nil
}

// Additional Telegram functionalities
func (t *TelegramClient) SendMediaMessage(chatID, mediaType, mediaURL, caption string) (string, error) {
	var endpoint string
	switch mediaType {
	case "photo":
		endpoint = "sendPhoto"
	case "video":
		endpoint = "sendVideo"
	case "document":
		endpoint = "sendDocument"
	case "audio":
		endpoint = "sendAudio"
	default:
		return "", fmt.Errorf("unsupported media type: %s", mediaType)
	}

	url := fmt.Sprintf("%s%s/%s", t.BaseURL, t.BotToken, endpoint)

	requestBody, err := json.Marshal(map[string]interface{}{
		"chat_id": chatID,
		mediaType: mediaURL,
		"caption": caption,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Extract message ID
	if resultData, ok := result["result"].(map[string]interface{}); ok {
		if messageID, ok := resultData["message_id"].(float64); ok {
			return fmt.Sprintf("%s:%f", chatID, messageID), nil
		}
	}

	return "", fmt.Errorf("failed to extract message ID")
}

// ==================== Slack API ====================

type SlackClient struct {
	BotToken string
	BaseURL  string
}

func NewSlackClient(botToken string) *SlackClient {
	return &SlackClient{
		BotToken: botToken,
		BaseURL:  "https://slack.com/api",
	}
}

// CreatePost sends a message to a Slack channel
func (s *SlackClient) CreatePost(content string, channelID string) (string, error) {
	url := fmt.Sprintf("%s/chat.postMessage", s.BaseURL)

	requestBody, err := json.Marshal(map[string]interface{}{
		"channel": channelID,
		"text":    content,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.BotToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Check if request was successful
	if ok, exists := result["ok"].(bool); !exists || !ok {
		return "", fmt.Errorf("slack API error: %v", result["error"])
	}

	// Extract message timestamp (used as ID in Slack)
	if ts, ok := result["ts"].(string); ok {
		return fmt.Sprintf("%s:%s", channelID, ts), nil
	}

	return "", fmt.Errorf("failed to extract message timestamp")
}

// ReplyToComment replies to a thread in Slack
func (s *SlackClient) ReplyToComment(threadID string, content string) (string, error) {
	// In Slack, we need both the channel_id and thread_ts
	parts := struct {
		ChannelID string
		ThreadTS  string
	}{}

	// Extract channel and thread timestamp
	fmt.Sscanf(threadID, "%s:%s", &parts.ChannelID, &parts.ThreadTS)

	url := fmt.Sprintf("%s/chat.postMessage", s.BaseURL)

	requestBody, err := json.Marshal(map[string]interface{}{
		"channel":   parts.ChannelID,
		"text":      content,
		"thread_ts": parts.ThreadTS,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.BotToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Check if request was successful
	if ok, exists := result["ok"].(bool); !exists || !ok {
		return "", fmt.Errorf("slack API error: %v", result["error"])
	}

	// Extract reply timestamp
	if ts, ok := result["ts"].(string); ok {
		return fmt.Sprintf("%s:%s", parts.ChannelID, ts), nil
	}

	return "", fmt.Errorf("failed to extract reply timestamp")
}

// GetPostStats gets information about a message or thread in Slack
func (s *SlackClient) GetPostStats(messageID string) (interface{}, error) {
	// Extract channel and thread timestamp
	parts := struct {
		ChannelID string
		MessageTS string
	}{}

	fmt.Sscanf(messageID, "%s:%s", &parts.ChannelID, &parts.MessageTS)

	// Get message information
	url := fmt.Sprintf("%s/conversations.history", s.BaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("channel", parts.ChannelID)
	q.Add("latest", parts.MessageTS)
	q.Add("limit", "1")
	q.Add("inclusive", "true")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+s.BotToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Check if request was successful
	if ok, exists := result["ok"].(bool); !exists || !ok {
		return nil, fmt.Errorf("slack API error: %v", result["error"])
	}

	// Get replies if it's a thread
	threadUrl := fmt.Sprintf("%s/conversations.replies", s.BaseURL)

	threadReq, err := http.NewRequest("GET", threadUrl, nil)
	if err != nil {
		return nil, err
	}

	tq := threadReq.URL.Query()
	tq.Add("channel", parts.ChannelID)
	tq.Add("ts", parts.MessageTS)
	threadReq.URL.RawQuery = tq.Encode()

	threadReq.Header.Set("Authorization", "Bearer "+s.BotToken)

	threadResp, err := client.Do(threadReq)
	if err != nil {
		return nil, err
	}
	defer threadResp.Body.Close()

	threadBody, err := ioutil.ReadAll(threadResp.Body)
	if err != nil {
		return nil, err
	}

	var threadResult map[string]interface{}
	if err := json.Unmarshal(threadBody, &threadResult); err != nil {
		return nil, err
	}

	// Combine results
	stats := map[string]interface{}{
		"message": result["messages"].([]interface{})[0],
	}

	if threadResult["ok"].(bool) {
		stats["thread_replies"] = threadResult["messages"]
		stats["thread_reply_count"] = len(threadResult["messages"].([]interface{})) - 1
	}

	return stats, nil
}

// GetCommunityStats gets information about a Slack channel
func (s *SlackClient) GetCommunityStats(channelID string) (interface{}, error) {
	// Get channel info
	infoUrl := fmt.Sprintf("%s/conversations.info", s.BaseURL)

	infoReq, err := http.NewRequest("GET", infoUrl, nil)
	if err != nil {
		return nil, err
	}

	q := infoReq.URL.Query()
	q.Add("channel", channelID)
	infoReq.URL.RawQuery = q.Encode()

	infoReq.Header.Set("Authorization", "Bearer "+s.BotToken)

	client := &http.Client{}
	infoResp, err := client.Do(infoReq)
	if err != nil {
		return nil, err
	}
	defer infoResp.Body.Close()

	infoBody, err := ioutil.ReadAll(infoResp.Body)
	if err != nil {
		return nil, err
	}

	if infoResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %s", string(infoBody))
	}

	var infoResult map[string]interface{}
	if err := json.Unmarshal(infoBody, &infoResult); err != nil {
		return nil, err
	}

	// Check if request was successful
	if ok, exists := infoResult["ok"].(bool); !exists || !ok {
		return nil, fmt.Errorf("slack API error: %v", infoResult["error"])
	}

	// Get member count
	membersUrl := fmt.Sprintf("%s/conversations.members", s.BaseURL)

	membersReq, err := http.NewRequest("GET", membersUrl, nil)
	if err != nil {
		return nil, err
	}

	mq := membersReq.URL.Query()
	mq.Add("channel", channelID)
	membersReq.URL.RawQuery = mq.Encode()

	membersReq.Header.Set("Authorization", "Bearer "+s.BotToken)

	membersResp, err := client.Do(membersReq)
	if err != nil {
		return nil, err
	}
	defer membersResp.Body.Close()

	membersBody, err := ioutil.ReadAll(membersResp.Body)
	if err != nil {
		return nil, err
	}

	var membersResult map[string]interface{}
	if err := json.Unmarshal(membersBody, &membersResult); err != nil {
		return nil, err
	}

	// Combine results
	stats := map[string]interface{}{
		"channel_info": infoResult["channel"],
	}

	if membersResult["ok"].(bool) {
		stats["member_count"] = len(membersResult["members"].([]interface{}))
	}

	return stats, nil
}

// // Additional Slack functionalities
// func (s *SlackClient) SendMediaMessage(channelID, text string, files []string) (string, error) {
//     url :=
