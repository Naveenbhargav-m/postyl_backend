package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// TwitterClient handles authentication and requests to the Twitter API
type TwitterClient struct {
	BearerToken string
	APIKey      string
	APISecret   string
	AccessToken string
	TokenSecret string
	HTTPClient  *http.Client
	BaseURL     string
}

// NewTwitterClient creates a new Twitter API client
func NewTwitterClient(apiKey, apiSecret, accessToken, tokenSecret, bearerToken string) *TwitterClient {
	return &TwitterClient{
		BearerToken: bearerToken,
		APIKey:      apiKey,
		APISecret:   apiSecret,
		AccessToken: accessToken,
		TokenSecret: tokenSecret,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
		BaseURL:     "https://api.twitter.com/2",
	}
}

// Tweet represents a Twitter post
type Tweet struct {
	ID               string    `json:"id,omitempty"`
	Text             string    `json:"text"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	AuthorID         string    `json:"author_id,omitempty"`
	ConversationID   string    `json:"conversation_id,omitempty"`
	InReplyToTweetID string    `json:"in_reply_to_tweet_id,omitempty"`
}

// TweetResponse is the API response for a tweet
type TweetResponse struct {
	Data Tweet `json:"data"`
}

// TweetsResponse is the API response for multiple tweets
type TweetsResponse struct {
	Data []Tweet `json:"data"`
	Meta struct {
		ResultCount int    `json:"result_count"`
		NextToken   string `json:"next_token,omitempty"`
	} `json:"meta"`
}

// CreateTweet posts a new tweet
func (c *TwitterClient) CreateTweet(text string) (*Tweet, error) {
	endpoint := fmt.Sprintf("%s/tweets", c.BaseURL)

	payload := map[string]interface{}{
		"text": text,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling tweet: %v", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.BearerToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var tweetResp TweetResponse
	if err := json.NewDecoder(resp.Body).Decode(&tweetResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &tweetResp.Data, nil
}

// ReplyToTweet posts a reply to an existing tweet
func (c *TwitterClient) ReplyToTweet(inReplyToTweetID, text string) (*Tweet, error) {
	endpoint := fmt.Sprintf("%s/tweets", c.BaseURL)

	payload := map[string]interface{}{
		"text": text,
		"reply": map[string]string{
			"in_reply_to_tweet_id": inReplyToTweetID,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling reply: %v", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.BearerToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var tweetResp TweetResponse
	if err := json.NewDecoder(resp.Body).Decode(&tweetResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &tweetResp.Data, nil
}

// GetTweet retrieves a tweet by ID
func (c *TwitterClient) GetTweet(tweetID string) (*Tweet, error) {
	endpoint := fmt.Sprintf("%s/tweets/%s", c.BaseURL, tweetID)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.BearerToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var tweetResp TweetResponse
	if err := json.NewDecoder(resp.Body).Decode(&tweetResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &tweetResp.Data, nil
}

// DeleteTweet deletes a tweet by ID
func (c *TwitterClient) DeleteTweet(tweetID string) error {
	endpoint := fmt.Sprintf("%s/tweets/%s", c.BaseURL, tweetID)

	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.BearerToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// SearchRecentTweets searches for recent tweets matching a query
func (c *TwitterClient) SearchRecentTweets(query string, maxResults int) ([]Tweet, error) {
	endpoint := fmt.Sprintf("%s/tweets/search/recent", c.BaseURL)

	params := url.Values{}
	params.Add("query", query)
	if maxResults > 0 {
		params.Add("max_results", fmt.Sprintf("%d", maxResults))
	}

	req, err := http.NewRequest("GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.BearerToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var tweetsResp TweetsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tweetsResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return tweetsResp.Data, nil
}

// AutomatedTweeter handles scheduled posting
type AutomatedTweeter struct {
	Client       *TwitterClient
	PostInterval time.Duration
	Content      []string
	CurrentIndex int
	StopChan     chan struct{}
}

// NewAutomatedTweeter creates a new automated tweeting service
func NewAutomatedTweeter(client *TwitterClient, interval time.Duration, content []string) *AutomatedTweeter {
	return &AutomatedTweeter{
		Client:       client,
		PostInterval: interval,
		Content:      content,
		CurrentIndex: 0,
		StopChan:     make(chan struct{}),
	}
}

// Start begins the automated posting
func (at *AutomatedTweeter) Start() {
	ticker := time.NewTicker(at.PostInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if at.CurrentIndex >= len(at.Content) {
				at.CurrentIndex = 0
			}

			content := at.Content[at.CurrentIndex]
			_, err := at.Client.CreateTweet(content)
			if err != nil {
				fmt.Printf("Error posting automated tweet: %v\n", err)
			} else {
				fmt.Printf("Posted automated tweet: %s\n", content)
			}

			at.CurrentIndex++
		case <-at.StopChan:
			return
		}
	}
}

// Stop halts the automated posting
func (at *AutomatedTweeter) Stop() {
	close(at.StopChan)
}

// AutoReplier handles automated replies to tweets matching criteria
type AutoReplier struct {
	Client        *TwitterClient
	SearchQueries []string
	ReplyContent  string
	CheckInterval time.Duration
	StopChan      chan struct{}
	LastTweetIDs  map[string]string
}

// NewAutoReplier creates a new automatic reply service
func NewAutoReplier(client *TwitterClient, queries []string, replyContent string, interval time.Duration) *AutoReplier {
	return &AutoReplier{
		Client:        client,
		SearchQueries: queries,
		ReplyContent:  replyContent,
		CheckInterval: interval,
		StopChan:      make(chan struct{}),
		LastTweetIDs:  make(map[string]string),
	}
}

// Start begins the automated reply monitoring
func (ar *AutoReplier) Start() {
	ticker := time.NewTicker(ar.CheckInterval)
	defer ticker.Stop()

	// Initialize with empty last tweet IDs
	for _, query := range ar.SearchQueries {
		ar.LastTweetIDs[query] = ""
	}

	for {
		select {
		case <-ticker.C:
			for _, query := range ar.SearchQueries {
				tweets, err := ar.Client.SearchRecentTweets(query, 10)
				if err != nil {
					fmt.Printf("Error searching tweets for query '%s': %v\n", query, err)
					continue
				}

				if len(tweets) == 0 {
					continue
				}

				lastID := ar.LastTweetIDs[query]

				// Process tweets in reverse order (oldest first)
				for i := len(tweets) - 1; i >= 0; i-- {
					tweet := tweets[i]

					// Skip already processed tweets
					if lastID != "" && tweet.ID <= lastID {
						continue
					}

					// Update last seen tweet ID
					if i == 0 {
						ar.LastTweetIDs[query] = tweet.ID
					}

					// Reply to the tweet
					_, err := ar.Client.ReplyToTweet(tweet.ID, ar.ReplyContent)
					if err != nil {
						fmt.Printf("Error replying to tweet %s: %v\n", tweet.ID, err)
					} else {
						fmt.Printf("Replied to tweet %s matching query '%s'\n", tweet.ID, query)
					}

					// Add a small delay to avoid rate limiting
					time.Sleep(2 * time.Second)
				}
			}
		case <-ar.StopChan:
			return
		}
	}
}

// Stop halts the automated reply monitoring
func (ar *AutoReplier) Stop() {
	close(ar.StopChan)
}

// Example usage
func Example() {
	// Initialize client
	client := NewTwitterClient(
		"YOUR_API_KEY",
		"YOUR_API_SECRET",
		"YOUR_ACCESS_TOKEN",
		"YOUR_TOKEN_SECRET",
		"YOUR_BEARER_TOKEN",
	)

	// Post a tweet
	tweet, err := client.CreateTweet("Hello, Twitter! This is an automated post from my Go application.")
	if err != nil {
		fmt.Printf("Error posting tweet: %v\n", err)
		return
	}
	fmt.Printf("Tweet posted successfully with ID: %s\n", tweet.ID)

	// Set up automated tweets
	content := []string{
		"Automated tweet #1: Testing Twitter API with Go",
		"Automated tweet #2: Learn how to automate social media posting",
		"Automated tweet #3: Building Twitter bots responsibly",
	}
	autoTweeter := NewAutomatedTweeter(client, 6*time.Hour, content)

	// Start automated posting (in a goroutine for this example)
	go autoTweeter.Start()

	// Set up automated replies
	queries := []string{
		"#GoLang",
		"Twitter API",
	}
	replier := NewAutoReplier(
		client,
		queries,
		"Thanks for mentioning this topic! Check out our resources at example.com",
		15*time.Minute,
	)

	// Start automated replying (in a goroutine for this example)
	go replier.Start()

	// Run for some time
	time.Sleep(24 * time.Hour)

	// Stop automated processes
	autoTweeter.Stop()
	replier.Stop()
}
