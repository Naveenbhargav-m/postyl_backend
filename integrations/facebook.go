package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	FacebookAPIBaseURL = "https://graph.facebook.com/v18.0"
)

// Client represents a Facebook API client
type FaceBookClient struct {
	AccessToken string
	HTTPClient  *http.Client
}

// NewClient creates a new Facebook API client
func NewFaceBookClient(accessToken string) *FaceBookClient {
	return &FaceBookClient{
		AccessToken: accessToken,
		HTTPClient:  &http.Client{},
	}
}

// Response represents a general Facebook API response
type Response struct {
	ID      string `json:"id,omitempty"`
	Success bool   `json:"success,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error represents a Facebook API error
type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}

// CreatePost creates a new post on a Facebook page or profile
// pageID can be "me" for posting on the user's own timeline
func (c *FaceBookClient) CreatePost(pageID, message string, link string) (*Response, error) {
	endpoint := fmt.Sprintf("%s/%s/feed", FacebookAPIBaseURL, pageID)

	data := url.Values{}
	data.Set("access_token", c.AccessToken)
	data.Set("message", message)

	if link != "" {
		data.Set("link", link)
	}

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return &result, fmt.Errorf("Facebook API error: %s", result.Error.Message)
	}

	return &result, nil
}

// CreateScheduledPost creates a post scheduled for future publication
func (c *FaceBookClient) CreateScheduledPost(pageID, message string, scheduledTime int64) (*Response, error) {
	endpoint := fmt.Sprintf("%s/%s/feed", FacebookAPIBaseURL, pageID)

	data := url.Values{}
	data.Set("access_token", c.AccessToken)
	data.Set("message", message)
	data.Set("published", "false")
	data.Set("scheduled_publish_time", fmt.Sprintf("%d", scheduledTime))

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return &result, fmt.Errorf("Facebook API error: %s", result.Error.Message)
	}

	return &result, nil
}

// UploadPhoto uploads a photo to a Facebook page or profile
func (c *FaceBookClient) UploadPhoto(pageID, message, photoPath string) (*Response, error) {
	endpoint := fmt.Sprintf("%s/%s/photos", FacebookAPIBaseURL, pageID)

	file, err := os.Open(photoPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	_ = writer.WriteField("access_token", c.AccessToken)
	if message != "" {
		_ = writer.WriteField("message", message)
	}

	// Add the file
	part, err := writer.CreateFormFile("source", filepath.Base(photoPath))
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return &result, fmt.Errorf("Facebook API error: %s", result.Error.Message)
	}

	return &result, nil
}

// CommentOnPost adds a comment to a post
func (c *FaceBookClient) CommentOnPost(postID, message string) (*Response, error) {
	endpoint := fmt.Sprintf("%s/%s/comments", FacebookAPIBaseURL, postID)

	data := url.Values{}
	data.Set("access_token", c.AccessToken)
	data.Set("message", message)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return &result, fmt.Errorf("Facebook API error: %s", result.Error.Message)
	}

	return &result, nil
}

// ReplyToComment adds a reply to a specific comment
func (c *FaceBookClient) ReplyToComment(commentID, message string) (*Response, error) {
	// Replying to a comment is the same as commenting on a post in the API
	// The commentID becomes the "post" that we're commenting on
	return c.CommentOnPost(commentID, message)
}

// Comment represents a Facebook comment
type Comment struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_time"`
	From      struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"from"`
}

// CommentsResponse represents the response from getting comments
type CommentsResponse struct {
	Data   []Comment `json:"data"`
	Paging struct {
		Cursors struct {
			Before string `json:"before"`
			After  string `json:"after"`
		} `json:"cursors"`
		Next string `json:"next,omitempty"`
	} `json:"paging"`
	Error *Error `json:"error,omitempty"`
}

// GetComments gets comments on a post
func (c *FaceBookClient) GetComments(postID string, limit int) (*CommentsResponse, error) {
	endpoint := fmt.Sprintf("%s/%s/comments", FacebookAPIBaseURL, postID)

	data := url.Values{}
	data.Set("access_token", c.AccessToken)
	if limit > 0 {
		data.Set("limit", fmt.Sprintf("%d", limit))
	}

	req, err := http.NewRequest("GET", endpoint+"?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CommentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return &result, fmt.Errorf("Facebook API error: %s", result.Error.Message)
	}

	return &result, nil
}

// PostInsights represents insights for a post
type PostInsights struct {
	Data []struct {
		Name   string                   `json:"name"`
		Period string                   `json:"period"`
		Values []map[string]interface{} `json:"values"`
		Title  string                   `json:"title"`
		ID     string                   `json:"id"`
	} `json:"data"`
	Error *Error `json:"error,omitempty"`
}

// GetPostInsights gets insights (stats) for a post
func (c *FaceBookClient) GetPostInsights(postID string) (*PostInsights, error) {
	endpoint := fmt.Sprintf("%s/%s/insights", FacebookAPIBaseURL, postID)

	data := url.Values{}
	data.Set("access_token", c.AccessToken)
	data.Set(
		"metric",
		"post_impressions,post_impressions_unique,post_reactions_by_type_total,post_clicks,post_engaged_users",
	)

	req, err := http.NewRequest("GET", endpoint+"?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result PostInsights
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return &result, fmt.Errorf("Facebook API error: %s", result.Error.Message)
	}

	return &result, nil
}

// PageInsights represents insights for a page
type PageInsights struct {
	Data []struct {
		Name   string                   `json:"name"`
		Period string                   `json:"period"`
		Values []map[string]interface{} `json:"values"`
		Title  string                   `json:"title"`
		ID     string                   `json:"id"`
	} `json:"data"`
	Error *Error `json:"error,omitempty"`
}

// GetPageInsights gets insights (stats) for a page
func (c *FaceBookClient) GetPageInsights(pageID string, metrics []string, period string) (*PageInsights, error) {
	endpoint := fmt.Sprintf("%s/%s/insights", FacebookAPIBaseURL, pageID)

	data := url.Values{}
	data.Set("access_token", c.AccessToken)

	if len(metrics) == 0 {
		// Default metrics if none provided
		metrics = []string{
			"page_impressions",
			"page_impressions_unique",
			"page_engaged_users",
			"page_fan_adds",
		}
	}

	data.Set("metric", strings.Join(metrics, ","))

	if period != "" {
		data.Set("period", period) // day, week, month, etc.
	}

	req, err := http.NewRequest("GET", endpoint+"?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result PageInsights
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return &result, fmt.Errorf("Facebook API error: %s", result.Error.Message)
	}

	return &result, nil
}

// Page represents a Facebook page
type Page struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	CategoryList []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"category_list"`
	About           string `json:"about"`
	Description     string `json:"description"`
	Fan_count       int    `json:"fan_count"`
	Followers_count int    `json:"followers_count"`
	Link            string `json:"link"`
	Error           *Error `json:"error,omitempty"`
}

// GetPageInfo gets information about a Facebook page
func (c *FaceBookClient) GetPageInfo(pageID string) (*Page, error) {
	endpoint := fmt.Sprintf("%s/%s", FacebookAPIBaseURL, pageID)

	data := url.Values{}
	data.Set("access_token", c.AccessToken)
	data.Set("fields", "id,name,category,category_list,about,description,fan_count,followers_count,link")

	req, err := http.NewRequest("GET", endpoint+"?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Page
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return &result, fmt.Errorf("Facebook API error: %s", result.Error.Message)
	}

	return &result, nil
}

// DeletePost deletes a post
func (c *FaceBookClient) DeletePost(postID string) (*Response, error) {
	endpoint := fmt.Sprintf("%s/%s", FacebookAPIBaseURL, postID)

	data := url.Values{}
	data.Set("access_token", c.AccessToken)

	req, err := http.NewRequest("DELETE", endpoint+"?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return &result, fmt.Errorf("Facebook API error: %s", result.Error.Message)
	}

	return &result, nil
}
