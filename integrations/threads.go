package integrations

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Thread represents a discussion thread
type Thread struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Reply represents a response to a thread
type Reply struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"thread_id"`
	Content   string    `json:"content"`
	AuthorID  string    `json:"author_id"`
	ParentID  string    `json:"parent_id,omitempty"` // For nested replies
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ThreadService handles thread-related API operations
type ThreadService struct {
	BaseURL    string
	HTTPClient *http.Client
	AuthToken  string
}

// NewThreadService creates a new thread service client
func NewThreadService(baseURL, authToken string) *ThreadService {
	return &ThreadService{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		AuthToken:  authToken,
	}
}

// CreateThread posts a new thread to the API
func (s *ThreadService) CreateThread(title, content, authorID string) (*Thread, error) {
	if title == "" {
		return nil, errors.New("title cannot be empty")
	}
	if content == "" {
		return nil, errors.New("content cannot be empty")
	}

	payload := map[string]string{
		"title":     title,
		"content":   content,
		"author_id": authorID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/threads", s.BaseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AuthToken))

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var thread Thread
	if err := json.NewDecoder(resp.Body).Decode(&thread); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &thread, nil
}

// GetThread retrieves a thread by ID
func (s *ThreadService) GetThread(threadID string) (*Thread, error) {
	if threadID == "" {
		return nil, errors.New("thread ID cannot be empty")
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/threads/%s", s.BaseURL, threadID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AuthToken))

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("thread not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var thread Thread
	if err := json.NewDecoder(resp.Body).Decode(&thread); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &thread, nil
}

// UpdateThread updates an existing thread
func (s *ThreadService) UpdateThread(threadID, title, content string) (*Thread, error) {
	if threadID == "" {
		return nil, errors.New("thread ID cannot be empty")
	}

	payload := map[string]string{}
	if title != "" {
		payload["title"] = title
	}
	if content != "" {
		payload["content"] = content
	}

	if len(payload) == 0 {
		return nil, errors.New("no update parameters provided")
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/threads/%s", s.BaseURL, threadID), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AuthToken))

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("thread not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var thread Thread
	if err := json.NewDecoder(resp.Body).Decode(&thread); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &thread, nil
}

// DeleteThread removes a thread
func (s *ThreadService) DeleteThread(threadID string) error {
	if threadID == "" {
		return errors.New("thread ID cannot be empty")
	}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/threads/%s", s.BaseURL, threadID), nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AuthToken))

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errors.New("thread not found")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListThreads retrieves all threads with optional pagination
func (s *ThreadService) ListThreads(page, limit int) ([]Thread, error) {
	url := fmt.Sprintf("%s/threads?page=%d&limit=%d", s.BaseURL, page, limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AuthToken))

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var threads []Thread
	if err := json.NewDecoder(resp.Body).Decode(&threads); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return threads, nil
}

// CreateReply posts a new reply to a thread
func (s *ThreadService) CreateReply(threadID, content, authorID, parentID string) (*Reply, error) {
	if threadID == "" {
		return nil, errors.New("thread ID cannot be empty")
	}
	if content == "" {
		return nil, errors.New("content cannot be empty")
	}

	payload := map[string]string{
		"thread_id": threadID,
		"content":   content,
		"author_id": authorID,
	}

	// Add parent ID for nested replies if provided
	if parentID != "" {
		payload["parent_id"] = parentID
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/threads/%s/replies", s.BaseURL, threadID),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AuthToken))

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("thread not found")
	}

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var reply Reply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &reply, nil
}

// GetReplies retrieves all replies for a thread
func (s *ThreadService) GetReplies(threadID string, page, limit int) ([]Reply, error) {
	if threadID == "" {
		return nil, errors.New("thread ID cannot be empty")
	}

	url := fmt.Sprintf("%s/threads/%s/replies?page=%d&limit=%d", s.BaseURL, threadID, page, limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AuthToken))

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("thread not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var replies []Reply
	if err := json.NewDecoder(resp.Body).Decode(&replies); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return replies, nil
}

// UpdateReply modifies an existing reply
func (s *ThreadService) UpdateReply(replyID, content string) (*Reply, error) {
	if replyID == "" {
		return nil, errors.New("reply ID cannot be empty")
	}
	if content == "" {
		return nil, errors.New("content cannot be empty")
	}

	payload := map[string]string{
		"content": content,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/replies/%s", s.BaseURL, replyID), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AuthToken))

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("reply not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var reply Reply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &reply, nil
}

// DeleteReply removes a reply
func (s *ThreadService) DeleteReply(replyID string) error {
	if replyID == "" {
		return errors.New("reply ID cannot be empty")
	}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/replies/%s", s.BaseURL, replyID), nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AuthToken))

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errors.New("reply not found")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// SearchThreads searches for threads by title or content
func (s *ThreadService) SearchThreads(query string, page, limit int) ([]Thread, error) {
	if query == "" {
		return nil, errors.New("search query cannot be empty")
	}

	url := fmt.Sprintf("%s/threads/search?q=%s&page=%d&limit=%d", s.BaseURL, query, page, limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.AuthToken))

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var threads []Thread
	if err := json.NewDecoder(resp.Body).Decode(&threads); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return threads, nil
}
