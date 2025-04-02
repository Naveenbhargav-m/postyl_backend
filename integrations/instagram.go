package integrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Constants for Instagram Graph API
const (
	BaseURL         = "https://graph.facebook.com/v17.0"
	InstagramAPIURL = "https://api.instagram.com/oauth/access_token"
)

// InstagramClient handles Instagram API operations
type InstagramClient struct {
	AppID       string
	AppSecret   string
	RedirectURI string
	AccessToken string
	UserID      string
	HTTPClient  *http.Client
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	UserID      int64  `json:"user_id"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
}

// MediaResponse represents the media creation response
type MediaResponse struct {
	ID        string `json:"id"`
	StatusURL string `json:"status_url,omitempty"`
}

// MediaInsights represents engagement metrics for a post
type MediaInsights struct {
	Engagement     int `json:"engagement"`
	Impressions    int `json:"impressions"`
	Reach          int `json:"reach"`
	Saved          int `json:"saved"`
	VideoViews     int `json:"video_views,omitempty"`
	Likes          int `json:"likes"`
	Comments       int `json:"comments"`
	Shares         int `json:"shares"`
	StoriesReplies int `json:"stories_replies,omitempty"`
}

// UserInsights represents user profile engagement metrics
type UserInsights struct {
	Followers      int `json:"followers_count"`
	FollowersDelta int `json:"followers_delta"`
	ProfileViews   int `json:"profile_views"`
	Reach          int `json:"reach"`
	Impressions    int `json:"impressions"`
	WebsiteClicks  int `json:"website_clicks,omitempty"`
}

// NewInstagramClient creates a new Instagram API client
func NewInstagramClient(appID, appSecret, redirectURI string) *InstagramClient {
	return &InstagramClient{
		AppID:       appID,
		AppSecret:   appSecret,
		RedirectURI: redirectURI,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAuthURL generates the OAuth URL to authorize the app
func (c *InstagramClient) GetAuthURL() string {
	params := url.Values{}
	params.Add("client_id", c.AppID)
	params.Add("redirect_uri", c.RedirectURI)
	params.Add("scope", "user_profile,user_media,instagram_graph_user_profile,instagram_graph_user_media")
	params.Add("response_type", "code")

	return fmt.Sprintf("https://api.instagram.com/oauth/authorize?%s", params.Encode())
}

// GetAccessToken exchanges the authorization code for an access token
func (c *InstagramClient) GetAccessToken(code string) (*TokenResponse, error) {
	params := url.Values{}
	params.Add("client_id", c.AppID)
	params.Add("client_secret", c.AppSecret)
	params.Add("grant_type", "authorization_code")
	params.Add("redirect_uri", c.RedirectURI)
	params.Add("code", code)

	req, err := http.NewRequest("POST", InstagramAPIURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get access token: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	c.AccessToken = tokenResp.AccessToken
	c.UserID = fmt.Sprintf("%d", tokenResp.UserID)

	return &tokenResp, nil
}

// GetLongLivedAccessToken exchanges short-lived token for a long-lived one
func (c *InstagramClient) GetLongLivedAccessToken() (*TokenResponse, error) {
	if c.AccessToken == "" {
		return nil, errors.New("no access token available")
	}

	params := url.Values{}
	params.Add("grant_type", "ig_exchange_token")
	params.Add("client_secret", c.AppSecret)
	params.Add("access_token", c.AccessToken)

	url := fmt.Sprintf("%s/access_token?%s", BaseURL, params.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get long lived token: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	c.AccessToken = tokenResp.AccessToken

	return &tokenResp, nil
}

// RefreshAccessToken refreshes a long-lived access token
func (c *InstagramClient) RefreshAccessToken() (*TokenResponse, error) {
	if c.AccessToken == "" {
		return nil, errors.New("no access token available")
	}

	params := url.Values{}
	params.Add("grant_type", "ig_refresh_token")
	params.Add("access_token", c.AccessToken)

	url := fmt.Sprintf("%s/refresh_access_token?%s", BaseURL, params.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to refresh token: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	c.AccessToken = tokenResp.AccessToken

	return &tokenResp, nil
}

// PostImage uploads and publishes an image to Instagram
func (c *InstagramClient) PostImage(imagePath, caption string) (*MediaResponse, error) {
	if c.AccessToken == "" || c.UserID == "" {
		return nil, errors.New("access token and user ID are required")
	}

	// Step 1: Upload the image to get a container ID
	params := url.Values{}
	params.Add("image_url", imagePath) // You can also use a URL directly
	params.Add("caption", caption)
	params.Add("access_token", c.AccessToken)

	uploadURL := fmt.Sprintf("%s/%s/media?%s", BaseURL, c.UserID, params.Encode())

	req, err := http.NewRequest("POST", uploadURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create media container: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	var mediaResp MediaResponse
	if err := json.NewDecoder(resp.Body).Decode(&mediaResp); err != nil {
		return nil, err
	}

	// Step 2: Publish the container
	publishParams := url.Values{}
	publishParams.Add("creation_id", mediaResp.ID)
	publishParams.Add("access_token", c.AccessToken)

	publishURL := fmt.Sprintf("%s/%s/media_publish?%s", BaseURL, c.UserID, publishParams.Encode())

	pubReq, err := http.NewRequest("POST", publishURL, nil)
	if err != nil {
		return nil, err
	}

	pubResp, err := c.HTTPClient.Do(pubReq)
	if err != nil {
		return nil, err
	}
	defer pubResp.Body.Close()

	if pubResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(pubResp.Body)
		return nil, fmt.Errorf("failed to publish media: %s, status: %d", string(bodyBytes), pubResp.StatusCode)
	}

	var publishedMedia MediaResponse
	if err := json.NewDecoder(pubResp.Body).Decode(&publishedMedia); err != nil {
		return nil, err
	}

	return &publishedMedia, nil
}

// PostReel uploads and publishes a reel to Instagram
func (c *InstagramClient) PostReel(
	videoPath, caption, coverImagePath string,
	shareToFeed bool,
) (*MediaResponse, error) {
	if c.AccessToken == "" || c.UserID == "" {
		return nil, errors.New("access token and user ID are required")
	}

	// Step 1: Upload video to get a container ID
	params := url.Values{}
	params.Add("media_type", "REELS")
	params.Add("video_url", videoPath) // You can use a URL directly
	params.Add("caption", caption)
	params.Add("access_token", c.AccessToken)

	if coverImagePath != "" {
		params.Add("thumb_url", coverImagePath)
	}

	if shareToFeed {
		params.Add("share_to_feed", "true")
	}

	uploadURL := fmt.Sprintf("%s/%s/media?%s", BaseURL, c.UserID, params.Encode())

	req, err := http.NewRequest("POST", uploadURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create reel container: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	var mediaResp MediaResponse
	if err := json.NewDecoder(resp.Body).Decode(&mediaResp); err != nil {
		return nil, err
	}

	// Step 2: Check status until ready
	if mediaResp.StatusURL != "" {
		err = c.waitForMediaProcessing(mediaResp.StatusURL)
		if err != nil {
			return nil, err
		}
	}

	// Step 3: Publish the container
	publishParams := url.Values{}
	publishParams.Add("creation_id", mediaResp.ID)
	publishParams.Add("access_token", c.AccessToken)

	publishURL := fmt.Sprintf("%s/%s/media_publish?%s", BaseURL, c.UserID, publishParams.Encode())

	pubReq, err := http.NewRequest("POST", publishURL, nil)
	if err != nil {
		return nil, err
	}

	pubResp, err := c.HTTPClient.Do(pubReq)
	if err != nil {
		return nil, err
	}
	defer pubResp.Body.Close()

	if pubResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(pubResp.Body)
		return nil, fmt.Errorf("failed to publish reel: %s, status: %d", string(bodyBytes), pubResp.StatusCode)
	}

	var publishedMedia MediaResponse
	if err := json.NewDecoder(pubResp.Body).Decode(&publishedMedia); err != nil {
		return nil, err
	}

	return &publishedMedia, nil
}

// waitForMediaProcessing checks media status until ready
func (c *InstagramClient) waitForMediaProcessing(statusURL string) error {
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(2 * time.Second)

		statusReq, err := http.NewRequest("GET", statusURL, nil)
		if err != nil {
			return err
		}

		statusResp, err := c.HTTPClient.Do(statusReq)
		if err != nil {
			return err
		}

		bodyBytes, _ := io.ReadAll(statusResp.Body)
		statusResp.Body.Close()

		var statusData map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &statusData); err != nil {
			return err
		}

		status, ok := statusData["status_code"].(string)
		if !ok {
			return errors.New("invalid status response")
		}

		if status == "FINISHED" {
			return nil
		} else if status == "ERROR" {
			return fmt.Errorf("media processing failed: %s", string(bodyBytes))
		}
	}

	return errors.New("media processing timed out")
}

// PostCarousel uploads and publishes multiple images/videos as a carousel
func (c *InstagramClient) PostCarousel(mediaPaths []string, caption string) (*MediaResponse, error) {
	if c.AccessToken == "" || c.UserID == "" {
		return nil, errors.New("access token and user ID are required")
	}

	// Step 1: Create container for each media item
	childrenIDs := []string{}

	for _, mediaPath := range mediaPaths {
		mediaType := "IMAGE"
		paramName := "image_url"

		if strings.HasSuffix(strings.ToLower(mediaPath), ".mp4") {
			mediaType = "VIDEO"
			paramName = "video_url"
		}

		params := url.Values{}
		params.Add("media_type", mediaType)
		params.Add(paramName, mediaPath)
		params.Add("is_carousel_item", "true")
		params.Add("access_token", c.AccessToken)

		uploadURL := fmt.Sprintf("%s/%s/media?%s", BaseURL, c.UserID, params.Encode())

		req, err := http.NewRequest("POST", uploadURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf(
				"failed to create media container: %s, status: %d",
				string(bodyBytes),
				resp.StatusCode,
			)
		}

		var mediaResp MediaResponse
		if err := json.NewDecoder(resp.Body).Decode(&mediaResp); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		// Wait for processing if needed
		if mediaResp.StatusURL != "" {
			err = c.waitForMediaProcessing(mediaResp.StatusURL)
			if err != nil {
				return nil, err
			}
		}

		childrenIDs = append(childrenIDs, mediaResp.ID)
	}

	// Step 2: Create carousel container
	carouselParams := url.Values{}
	carouselParams.Add("media_type", "CAROUSEL")
	carouselParams.Add("caption", caption)
	carouselParams.Add("access_token", c.AccessToken)
	carouselParams.Add("children", strings.Join(childrenIDs, ","))

	carouselURL := fmt.Sprintf("%s/%s/media?%s", BaseURL, c.UserID, carouselParams.Encode())

	carReq, err := http.NewRequest("POST", carouselURL, nil)
	if err != nil {
		return nil, err
	}

	carResp, err := c.HTTPClient.Do(carReq)
	if err != nil {
		return nil, err
	}
	defer carResp.Body.Close()

	if carResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(carResp.Body)
		return nil, fmt.Errorf(
			"failed to create carousel container: %s, status: %d",
			string(bodyBytes),
			carResp.StatusCode,
		)
	}

	var carouselResp MediaResponse
	if err := json.NewDecoder(carResp.Body).Decode(&carouselResp); err != nil {
		return nil, err
	}

	// Step 3: Publish the carousel
	publishParams := url.Values{}
	publishParams.Add("creation_id", carouselResp.ID)
	publishParams.Add("access_token", c.AccessToken)

	publishURL := fmt.Sprintf("%s/%s/media_publish?%s", BaseURL, c.UserID, publishParams.Encode())

	pubReq, err := http.NewRequest("POST", publishURL, nil)
	if err != nil {
		return nil, err
	}

	pubResp, err := c.HTTPClient.Do(pubReq)
	if err != nil {
		return nil, err
	}
	defer pubResp.Body.Close()

	if pubResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(pubResp.Body)
		return nil, fmt.Errorf("failed to publish carousel: %s, status: %d", string(bodyBytes), pubResp.StatusCode)
	}

	var publishedMedia MediaResponse
	if err := json.NewDecoder(pubResp.Body).Decode(&publishedMedia); err != nil {
		return nil, err
	}

	return &publishedMedia, nil
}

// GetMediaInsights retrieves insights for a specific media item
func (c *InstagramClient) GetMediaInsights(mediaID string) (*MediaInsights, error) {
	if c.AccessToken == "" {
		return nil, errors.New("access token is required")
	}

	params := url.Values{}
	params.Add("metric", "engagement,impressions,reach,saved,video_views,likes,comments,shares")
	params.Add("access_token", c.AccessToken)

	insightsURL := fmt.Sprintf("%s/%s/insights?%s", BaseURL, mediaID, params.Encode())

	req, err := http.NewRequest("GET", insightsURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get media insights: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	// Parse the complex Instagram Insights response
	type InsightsData struct {
		Data []struct {
			Name   string `json:"name"`
			Period string `json:"period"`
			Values []struct {
				Value int `json:"value"`
			} `json:"values"`
		} `json:"data"`
	}

	var insightsData InsightsData
	if err := json.NewDecoder(resp.Body).Decode(&insightsData); err != nil {
		return nil, err
	}

	// Map the insight values to our struct
	insights := &MediaInsights{}
	for _, metric := range insightsData.Data {
		if len(metric.Values) == 0 {
			continue
		}

		value := metric.Values[0].Value

		switch metric.Name {
		case "engagement":
			insights.Engagement = value
		case "impressions":
			insights.Impressions = value
		case "reach":
			insights.Reach = value
		case "saved":
			insights.Saved = value
		case "video_views":
			insights.VideoViews = value
		case "likes":
			insights.Likes = value
		case "comments":
			insights.Comments = value
		case "shares":
			insights.Shares = value
		case "stories_replies":
			insights.StoriesReplies = value
		}
	}

	return insights, nil
}

// GetUserInsights retrieves insights for the user's profile
func (c *InstagramClient) GetUserInsights(period string) (*UserInsights, error) {
	if c.AccessToken == "" || c.UserID == "" {
		return nil, errors.New("access token and user ID are required")
	}

	if period == "" {
		period = "day" // Other options: week, month
	}

	params := url.Values{}
	params.Add("metric", "follower_count,profile_views,reach,impressions,website_clicks")
	params.Add("period", period)
	params.Add("access_token", c.AccessToken)

	insightsURL := fmt.Sprintf("%s/%s/insights?%s", BaseURL, c.UserID, params.Encode())

	req, err := http.NewRequest("GET", insightsURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user insights: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	// Parse the complex Instagram Insights response
	type InsightsData struct {
		Data []struct {
			Name   string `json:"name"`
			Period string `json:"period"`
			Values []struct {
				Value int `json:"value"`
			} `json:"values"`
		} `json:"data"`
	}

	var insightsData InsightsData
	if err := json.NewDecoder(resp.Body).Decode(&insightsData); err != nil {
		return nil, err
	}

	// Map the insight values to our struct
	insights := &UserInsights{}
	for _, metric := range insightsData.Data {
		if len(metric.Values) == 0 {
			continue
		}

		value := metric.Values[0].Value

		switch metric.Name {
		case "follower_count":
			insights.Followers = value
		case "profile_views":
			insights.ProfileViews = value
		case "reach":
			insights.Reach = value
		case "impressions":
			insights.Impressions = value
		case "website_clicks":
			insights.WebsiteClicks = value
		}

		// Calculate follower growth if we have data points
		if metric.Name == "follower_count" && len(metric.Values) > 1 {
			previousValue := metric.Values[1].Value
			insights.FollowersDelta = value - previousValue
		}
	}

	return insights, nil
}

// GetUserEngagement retrieves overall engagement metrics
func (c *InstagramClient) GetUserEngagement(days int) (map[string]interface{}, error) {
	if c.AccessToken == "" || c.UserID == "" {
		return nil, errors.New("access token and user ID are required")
	}

	if days <= 0 {
		days = 30 // Default to 30 days
	}

	// Get recent media first
	params := url.Values{}
	params.Add("fields", "id,media_type,timestamp")
	params.Add("limit", fmt.Sprintf("%d", days))
	params.Add("access_token", c.AccessToken)

	mediaURL := fmt.Sprintf("%s/%s/media?%s", BaseURL, c.UserID, params.Encode())

	req, err := http.NewRequest("GET", mediaURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get media: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	type MediaData struct {
		Data []struct {
			ID        string `json:"id"`
			MediaType string `json:"media_type"`
			Timestamp string `json:"timestamp"`
		} `json:"data"`
		Paging struct {
			Cursors struct {
				Before string `json:"before"`
				After  string `json:"after"`
			} `json:"cursors"`
		} `json:"paging"`
	}

	var mediaData MediaData
	if err := json.NewDecoder(resp.Body).Decode(&mediaData); err != nil {
		return nil, err
	}

	// Get insights for each media
	totalEngagement := 0
	totalImpressions := 0
	totalReach := 0
	totalLikes := 0
	totalComments := 0
	totalSaved := 0
	mediaCount := len(mediaData.Data)

	for _, media := range mediaData.Data {
		insights, err := c.GetMediaInsights(media.ID)
		if err != nil {
			continue // Skip if we can't get insights for this media
		}

		totalEngagement += insights.Engagement
		totalImpressions += insights.Impressions
		totalReach += insights.Reach
		totalLikes += insights.Likes
		totalComments += insights.Comments
		totalSaved += insights.Saved
	}

	// Get user insights
	userInsights, err := c.GetUserInsights("day")
	if err != nil {
		// Continue even if we can't get user insights
		userInsights = &UserInsights{}
	}

	// Calculate averages and engagement rate
	var avgEngagement, avgImpressions, avgReach, avgLikes, avgComments, engagementRate float64

	if mediaCount > 0 {
		avgEngagement = float64(totalEngagement) / float64(mediaCount)
		avgImpressions = float64(totalImpressions) / float64(mediaCount)
		avgReach = float64(totalReach) / float64(mediaCount)
		avgLikes = float64(totalLikes) / float64(mediaCount)
		avgComments = float64(totalComments) / float64(mediaCount)

		if userInsights.Followers > 0 {
			engagementRate = avgEngagement / float64(userInsights.Followers) * 100
		}
	}

	// Build comprehensive engagement report
	engagement := map[string]interface{}{
		"period_days":         days,
		"posts_analyzed":      mediaCount,
		"followers":           userInsights.Followers,
		"followers_delta":     userInsights.FollowersDelta,
		"profile_views":       userInsights.ProfileViews,
		"total_engagement":    totalEngagement,
		"total_impressions":   totalImpressions,
		"total_reach":         totalReach,
		"total_likes":         totalLikes,
		"total_comments":      totalComments,
		"total_saved":         totalSaved,
		"avg_engagement":      avgEngagement,
		"avg_impressions":     avgImpressions,
		"avg_reach":           avgReach,
		"avg_likes":           avgLikes,
		"avg_comments":        avgComments,
		"engagement_rate":     engagementRate,
		"engagement_per_post": avgEngagement,
		"most_engaging_day":   getMostEngagingDay(mediaData.Data),
		"engagement_trend":    getEngagementTrend(mediaData.Data),
	}

	return engagement, nil
}

// Helper function to find most engaging day
func getMostEngagingDay(mediaData []struct {
	ID        string `json:"id"`
	MediaType string `json:"media_type"`
	Timestamp string `json:"timestamp"`
}) string {
	dayCount := make(map[string]int)

	for _, media := range mediaData {
		t, err := time.Parse(time.RFC3339, media.Timestamp)
		if err != nil {
			continue
		}

		day := t.Weekday().String()
		dayCount[day]++
	}

	maxCount := 0
	maxDay := "Unknown"

	for day, count := range dayCount {
		if count > maxCount {
			maxCount = count
			maxDay = day
		}
	}

	return maxDay
}

// Helper function to calculate engagement trend
func getEngagementTrend(mediaData []struct {
	ID        string `json:"id"`
	MediaType string `json:"media_type"`
	Timestamp string `json:"timestamp"`
}) string {
	if len(mediaData) < 3 {
		return "Not enough data"
	}

	// This is a placeholder - in a real implementation
	// you would analyze the trend of engagement over time
	return "Stable" // Could be "Rising", "Falling", or "Stable"
}
