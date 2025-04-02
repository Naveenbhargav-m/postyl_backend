package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RedditClient represents a Reddit API client
type RedditClient struct {
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	UserAgent    string
	AccessToken  string
	TokenExpiry  time.Time
	HTTPClient   *http.Client
}

// NewRedditClient creates a new Reddit API client
func NewRedditClient(clientID, clientSecret, username, password, userAgent string) *RedditClient {
	return &RedditClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Username:     username,
		Password:     password,
		UserAgent:    userAgent,
		HTTPClient:   &http.Client{},
	}
}

// Authenticate authenticates with Reddit API using OAuth
func (c *RedditClient) Authenticate() error {
	// Skip if we have a valid token
	if c.AccessToken != "" && time.Now().Before(c.TokenExpiry) {
		return nil
	}

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", c.Username)
	data.Set("password", c.Password)

	req, err := http.NewRequest("POST", "https://www.reddit.com/api/v1/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.ClientID, c.ClientSecret)
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed with status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}

	c.AccessToken = result.AccessToken
	c.TokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)

	return nil
}

// makeRequest makes an authenticated request to the Reddit API
func (c *RedditClient) makeRequest(method, endpoint string, body interface{}, query url.Values) ([]byte, error) {
	if err := c.Authenticate(); err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	baseURL := "https://oauth.reddit.com"
	fullURL := baseURL + endpoint

	if query != nil && len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	if method == "POST" || method == "PUT" || method == "PATCH" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// 1. CreatePost creates a new post in a subreddit
func (c *RedditClient) CreatePost(subreddit, title, content, kind string) (string, error) {
	// kind can be "self" for text post, "link" for link post, "image" for image, etc.
	data := map[string]string{
		"sr":    subreddit,
		"title": title,
		"kind":  kind,
	}

	// Add content based on post type
	if kind == "self" {
		data["text"] = content
	} else if kind == "link" {
		data["url"] = content
	}

	// Make the API call
	formData := url.Values{}
	for key, value := range data {
		formData.Add(key, value)
	}

	response, err := c.makeRequest("POST", "/api/submit", nil, formData)
	if err != nil {
		return "", err
	}

	// Parse response to get post ID
	var result struct {
		JSON struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"json"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return "", err
	}

	return result.JSON.Data.ID, nil
}

// 2. ReplyToComment replies to a comment
func (c *RedditClient) ReplyToComment(commentID, text string) (string, error) {
	formData := url.Values{}
	formData.Add("api_type", "json")
	formData.Add("text", text)
	formData.Add("thing_id", commentID) // Must include prefix, like "t1_" for comments

	response, err := c.makeRequest("POST", "/api/comment", nil, formData)
	if err != nil {
		return "", err
	}

	var result struct {
		JSON struct {
			Data struct {
				Things []struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				} `json:"things"`
			} `json:"data"`
		} `json:"json"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return "", err
	}

	if len(result.JSON.Data.Things) == 0 {
		return "", fmt.Errorf("no comment ID returned")
	}

	return result.JSON.Data.Things[0].Data.ID, nil
}

// 3. GetSubredditStats gets stats about a subreddit
func (c *RedditClient) GetSubredditStats(subreddit string) (map[string]interface{}, error) {
	response, err := c.makeRequest("GET", "/r/"+subreddit+"/about", nil, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetPostStats gets stats about a specific post
func (c *RedditClient) GetPostStats(postID string) (map[string]interface{}, error) {
	// Make sure postID includes the t3_ prefix if not already present
	if !strings.HasPrefix(postID, "t3_") {
		postID = "t3_" + postID
	}

	response, err := c.makeRequest("GET", "/api/info", nil, url.Values{"id": {postID}})
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetUserInfo gets information about a user
func (c *RedditClient) GetUserInfo(username string) (map[string]interface{}, error) {
	response, err := c.makeRequest("GET", "/user/"+username+"/about", nil, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetComments gets comments from a post
func (c *RedditClient) GetComments(postID, subreddit string) ([]interface{}, error) {
	// Remove t3_ prefix if present
	postID = strings.TrimPrefix(postID, "t3_")

	response, err := c.makeRequest("GET", "/r/"+subreddit+"/comments/"+postID, nil, nil)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Vote upvotes or downvotes a post or comment
// dir should be 1 for upvote, -1 for downvote, 0 for removing vote
func (c *RedditClient) Vote(id string, dir int) error {
	formData := url.Values{}
	formData.Add("id", id) // Must include prefix, like "t3_" for posts
	formData.Add("dir", fmt.Sprintf("%d", dir))

	_, err := c.makeRequest("POST", "/api/vote", nil, formData)
	return err
}

// SearchPosts searches for posts
func (c *RedditClient) SearchPosts(query, subreddit string, limit int) ([]interface{}, error) {
	params := url.Values{}
	params.Add("q", query)
	params.Add("limit", fmt.Sprintf("%d", limit))

	endpoint := "/search"
	if subreddit != "" {
		endpoint = "/r/" + subreddit + "/search"
	}

	response, err := c.makeRequest("GET", endpoint, nil, params)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Children []interface{} `json:"children"`
		} `json:"data"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, err
	}

	return result.Data.Children, nil
}
