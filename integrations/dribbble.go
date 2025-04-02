package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// Client represents a Dribbble API client
type DribbbleClient struct {
	AccessToken string
	BaseURL     string
	HTTPClient  *http.Client
}

// Shot represents a Dribbble shot (post)
type Shot struct {
	ID          int64    `json:"id,omitempty"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	TeamID      int64    `json:"team_id,omitempty"`
	Low         string   `json:"low_profile,omitempty"`  // URL to low-res image
	Normal      string   `json:"normal,omitempty"`       // URL to normal-res image
	High        string   `json:"high_profile,omitempty"` // URL to high-res image
}

// Comment represents a comment on a Dribbble shot
type DribbbleComment struct {
	ID      int64  `json:"id,omitempty"`
	Body    string `json:"body"`
	ShotID  int64  `json:"shot_id,omitempty"`
	UserID  int64  `json:"user_id,omitempty"`
	Created string `json:"created_at,omitempty"`
}

// Stats represents statistics for a Dribbble shot
type DribbbleStats struct {
	Views       int `json:"views"`
	Likes       int `json:"likes"`
	Comments    int `json:"comments"`
	Rebounds    int `json:"rebounds"`
	Attachments int `json:"attachments"`
}

// NewClient creates a new Dribbble API client
func NewDribbbleClient(accessToken string) *DribbbleClient {
	return &DribbbleClient{
		AccessToken: accessToken,
		BaseURL:     "https://api.dribbble.com/v2",
		HTTPClient:  &http.Client{},
	}
}

// CreateShot uploads a new shot (post) to Dribbble
func (c *DribbbleClient) CreateShot(title, description string, tags []string, imagePath string) (*Shot, error) {
	endpoint := fmt.Sprintf("%s/shots", c.BaseURL)

	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %v", err)
	}
	defer file.Close()

	// Create a multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the image file
	part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file: %v", err)
	}

	// Add other form fields
	writer.WriteField("title", title)
	writer.WriteField("description", description)

	// Add tags
	for _, tag := range tags {
		writer.WriteField("tags[]", tag)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %v", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create shot. Status: %d, Response: %s", resp.StatusCode, string(responseBody))
	}

	// Parse the response
	var shot Shot
	err = json.NewDecoder(resp.Body).Decode(&shot)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &shot, nil
}

// ReplyToComment adds a reply to an existing comment on a shot
func (c *DribbbleClient) ReplyToComment(shotID int64, commentID int64, body string) (*Comment, error) {
	endpoint := fmt.Sprintf("%s/shots/%d/comments/%d/replies", c.BaseURL, shotID, commentID)

	// Create the request body
	requestBody, err := json.Marshal(map[string]string{
		"body": body,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"failed to reply to comment. Status: %d, Response: %s",
			resp.StatusCode,
			string(responseBody),
		)
	}

	// Parse the response
	var comment Comment
	err = json.NewDecoder(resp.Body).Decode(&comment)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &comment, nil
}

// GetShotStats retrieves statistics for a specific shot
func (c *DribbbleClient) GetShotStats(shotID int64) (*Stats, error) {
	endpoint := fmt.Sprintf("%s/shots/%d", c.BaseURL, shotID)

	// Create the request
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	// Send the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"failed to get shot stats. Status: %d, Response: %s",
			resp.StatusCode,
			string(responseBody),
		)
	}

	// Parse the response
	var shot struct {
		ID    int64 `json:"id"`
		Stats Stats `json:"statistics"`
	}

	err = json.NewDecoder(resp.Body).Decode(&shot)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &shot.Stats, nil
}

// ListShots fetches shots based on filters
func (c *DribbbleClient) ListShots(page, perPage int, timeframe string) ([]Shot, error) {
	endpoint := fmt.Sprintf("%s/shots?page=%d&per_page=%d&timeframe=%s",
		c.BaseURL, page, perPage, timeframe)

	// Create the request
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	// Send the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list shots. Status: %d, Response: %s", resp.StatusCode, string(responseBody))
	}

	// Parse the response
	var shots []Shot
	err = json.NewDecoder(resp.Body).Decode(&shots)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return shots, nil
}

// FollowUser follows a Dribbble user
func (c *DribbbleClient) FollowUser(userID int64) error {
	endpoint := fmt.Sprintf("%s/users/%d/follow", c.BaseURL, userID)

	// Create the request
	req, err := http.NewRequest("PUT", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	// Send the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to follow user. Status: %d, Response: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

// LikeShot likes a Dribbble shot
func (c *DribbbleClient) LikeShot(shotID int64) error {
	endpoint := fmt.Sprintf("%s/shots/%d/like", c.BaseURL, shotID)

	// Create the request
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	// Send the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to like shot. Status: %d, Response: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}
