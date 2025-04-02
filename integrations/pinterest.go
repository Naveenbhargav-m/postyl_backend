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
	"strings"
)

// Pinterest represents a Pinterest API Pinterest
type Pinterest struct {
	AccessToken   string
	BaseURL       string
	HTTPPinterest *http.Client
}

// Pin represents a Pinterest pin
type Pin struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Link        string `json:"link,omitempty"`
	BoardID     string `json:"board_id,omitempty"`
	MediaSource string `json:"media_source,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
}

// Comment represents a Pinterest comment
type PinterestComment struct {
	ID        string `json:"id,omitempty"`
	Text      string `json:"text,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	PinID     string `json:"pin_id,omitempty"`
}

// Stats represents statistics for a Pinterest resource
type Stats struct {
	Followers   int `json:"followers,omitempty"`
	Following   int `json:"following,omitempty"`
	Pins        int `json:"pins,omitempty"`
	Likes       int `json:"likes,omitempty"`
	Comments    int `json:"comments,omitempty"`
	Saves       int `json:"saves,omitempty"`
	Impressions int `json:"impressions,omitempty"`
	Engagements int `json:"engagements,omitempty"`
	Clicks      int `json:"clicks,omitempty"`
	VideoViews  int `json:"video_views,omitempty"`
}

// Board represents a Pinterest board
type Board struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Privacy     string `json:"privacy,omitempty"`
}

// NewPinterest creates a new Pinterest API Pinterest
func NewPinterest(accessToken string) *Pinterest {
	return &Pinterest{
		AccessToken:   accessToken,
		BaseURL:       "https://api.pinterest.com/v5",
		HTTPPinterest: &http.Client{},
	}
}

// -----------------------------------------------
// 1. Create Post (Pin) Functions
// -----------------------------------------------

// CreatePin creates a new pin on Pinterest
func (c *Pinterest) CreatePin(pin Pin) (*Pin, error) {
	url := fmt.Sprintf("%s/pins", c.BaseURL)

	pinJSON, err := json.Marshal(pin)
	if err != nil {
		return nil, fmt.Errorf("error marshaling pin: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(pinJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create pin: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result Pin
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UploadImageForPin uploads an image to Pinterest and returns a media ID
func (c *Pinterest) UploadImageForPin(imagePath string) (string, error) {
	url := fmt.Sprintf("%s/media", c.BaseURL)

	file, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(imagePath))
	if err != nil {
		return "", err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}

	writer.Close()

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload image: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result struct {
		MediaID string `json:"media_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.MediaID, nil
}

// -----------------------------------------------
// 2. Reply to Comment Functions
// -----------------------------------------------

// GetComments gets comments on a pin
func (c *Pinterest) GetComments(pinID string) ([]Comment, error) {
	url := fmt.Sprintf("%s/pins/%s/comments", c.BaseURL, pinID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get comments: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result struct {
		Items []Comment `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

// AddComment adds a comment to a pin
func (c *Pinterest) AddComment(pinID, text string) (*Comment, error) {
	url := fmt.Sprintf("%s/pins/%s/comments", c.BaseURL, pinID)

	payload := map[string]string{
		"text": text,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to add comment: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result Comment
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ReplyToComment adds a reply to an existing comment
// Note: In Pinterest's API, a reply is just another comment that references the parent comment
func (c *Pinterest) ReplyToComment(pinID, parentCommentID, text string) (*Comment, error) {
	url := fmt.Sprintf("%s/pins/%s/comments", c.BaseURL, pinID)

	payload := map[string]string{
		"text":              text,
		"parent_comment_id": parentCommentID,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to reply to comment: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result Comment
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// -----------------------------------------------
// 3. Get Stats Functions
// -----------------------------------------------

// GetPinStats gets analytics for a specific pin
func (c *Pinterest) GetPinStats(pinID string, timeframe string) (*Stats, error) {
	if timeframe == "" {
		timeframe = "30days" // Default timeframe
	}

	url := fmt.Sprintf("%s/pins/%s/analytics?timeframe=%s", c.BaseURL, pinID, timeframe)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get pin stats: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result Stats
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetBoardStats gets analytics for a specific board
func (c *Pinterest) GetBoardStats(boardID string, timeframe string) (*Stats, error) {
	if timeframe == "" {
		timeframe = "30days" // Default timeframe
	}

	url := fmt.Sprintf("%s/boards/%s/analytics?timeframe=%s", c.BaseURL, boardID, timeframe)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get board stats: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result Stats
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetUserStats gets analytics for the authenticated user account
func (c *Pinterest) GetUserStats(timeframe string) (*Stats, error) {
	if timeframe == "" {
		timeframe = "30days" // Default timeframe
	}

	url := fmt.Sprintf("%s/user/analytics?timeframe=%s", c.BaseURL, timeframe)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user stats: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result Stats
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// -----------------------------------------------
// 4. Authentication Functions
// -----------------------------------------------

// GetOAuthURL returns the URL to redirect users to for OAuth authorization
func GetOAuthURL(PinterestID, redirectURI, state string, scopes []string) string {
	baseURL := "https://www.pinterest.com/oauth/"
	scope := strings.Join(scopes, " ")

	return fmt.Sprintf("%s?Pinterest_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		baseURL, PinterestID, redirectURI, scope, state)
}

// ExchangeCodeForToken exchanges an authorization code for an access token
func ExchangeCodeForToken(PinterestID, PinterestSecret, code, redirectURI string) (string, error) {
	url := "https://api.pinterest.com/v5/oauth/token"

	data := map[string]string{
		"grant_type":       "authorization_code",
		"Pinterest_id":     PinterestID,
		"Pinterest_secret": PinterestSecret,
		"code":             code,
		"redirect_uri":     redirectURI,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	Pinterest := &http.Client{}
	resp, err := Pinterest.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to exchange code: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

// -----------------------------------------------
// 5. User Information Functions
// -----------------------------------------------

// GetUserInfo gets information about the authenticated user
func (c *Pinterest) GetUserInfo() (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/user_account", c.BaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// -----------------------------------------------
// 6. Searching for Pins
// -----------------------------------------------

// SearchPins searches for pins with the given query
func (c *Pinterest) SearchPins(query string, limit int) ([]Pin, error) {
	if limit <= 0 {
		limit = 25 // Default limit
	}

	url := fmt.Sprintf("%s/pins/search?query=%s&limit=%d", c.BaseURL, query, limit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to search pins: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result struct {
		Items []Pin `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

// -----------------------------------------------
// 7. Board Management Functions
// -----------------------------------------------

// CreateBoard creates a new board
func (c *Pinterest) CreateBoard(board Board) (*Board, error) {
	url := fmt.Sprintf("%s/boards", c.BaseURL)

	boardJSON, err := json.Marshal(board)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(boardJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create board: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result Board
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateBoard updates an existing board
func (c *Pinterest) UpdateBoard(boardID string, board Board) (*Board, error) {
	url := fmt.Sprintf("%s/boards/%s", c.BaseURL, boardID)

	boardJSON, err := json.Marshal(board)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(boardJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update board: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result Board
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetBoards gets all boards for the authenticated user
func (c *Pinterest) GetBoards() ([]Board, error) {
	url := fmt.Sprintf("%s/boards", c.BaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get boards: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result struct {
		Items []Board `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

// -----------------------------------------------
// 8. Follow/Unfollow Functions
// -----------------------------------------------

// FollowUser follows a user
func (c *Pinterest) FollowUser(username string) error {
	url := fmt.Sprintf("%s/user/follows/users/", c.BaseURL)

	data := map[string]string{
		"username": username,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to follow user: %s, status code: %d", string(body), resp.StatusCode)
	}

	return nil
}

// UnfollowUser unfollows a user
func (c *Pinterest) UnfollowUser(username string) error {
	url := fmt.Sprintf("%s/user/follows/users/%s", c.BaseURL, username)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPPinterest.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to unfollow user: %s, status code: %d", string(body), resp.StatusCode)
	}

	return nil
}
