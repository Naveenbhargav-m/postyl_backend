package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Common types and interfaces
type SocialPlatform interface {
	CreatePost(ctx context.Context, post PostData) (string, error)
	ReplyToComment(ctx context.Context, postID, commentID, replyText string) (string, error)
	GetPostStats(ctx context.Context, postID string) (PostStats, error)
	SearchContent(ctx context.Context, query string) ([]ContentItem, error)
	DeleteContent(ctx context.Context, contentID string) error
	UpdateContent(ctx context.Context, contentID string, data UpdateData) error
}

type PostData struct {
	VideoPath    string
	Title        string
	Description  string
	Tags         []string
	Privacy      string // "public", "private", "unlisted"
	ScheduleTime *time.Time
}

type UpdateData struct {
	Title       *string
	Description *string
	Tags        *[]string
	Privacy     *string
}

type PostStats struct {
	Views        int64
	Likes        int64
	Comments     int64
	Shares       int64
	Engagement   float64
	Demographics map[string]interface{}
}

type ContentItem struct {
	ID          string
	Title       string
	Description string
	URL         string
	Author      string
	Stats       PostStats
}

// TikTok API Client
type TikTokClient struct {
	accessToken string
	apiKey      string
	baseURL     string
	httpClient  *http.Client
}

// NewTikTokClient creates a new TikTok API client
func NewTikTokClient(accessToken, apiKey string) *TikTokClient {
	return &TikTokClient{
		accessToken: accessToken,
		apiKey:      apiKey,
		baseURL:     "https://open-api.tiktok.com/v2",
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// CreatePost uploads a video to TikTok
func (c *TikTokClient) CreatePost(ctx context.Context, post PostData) (string, error) {
	// Open the video file
	file, err := os.Open(post.VideoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open video file: %w", err)
	}
	defer file.Close()

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("video", filepath.Base(post.VideoPath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err = io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add metadata
	_ = writer.WriteField("title", post.Title)
	_ = writer.WriteField("description", post.Description)

	for _, tag := range post.Tags {
		_ = writer.WriteField("tags", tag)
	}

	_ = writer.WriteField("privacy_level", post.Privacy)

	if post.ScheduleTime != nil {
		_ = writer.WriteField("schedule_time", post.ScheduleTime.Format(time.RFC3339))
	}

	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/video/upload/", body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("x-api-key", c.apiKey)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			VideoID string `json:"video_id"`
		} `json:"data"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data.VideoID, nil
}

// ReplyToComment posts a reply to a comment on TikTok
func (c *TikTokClient) ReplyToComment(ctx context.Context, postID, commentID, replyText string) (string, error) {
	data := map[string]string{
		"video_id":   postID,
		"comment_id": commentID,
		"text":       replyText,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/comment/reply/", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("reply failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			CommentID string `json:"comment_id"`
		} `json:"data"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data.CommentID, nil
}

// GetPostStats retrieves metrics for a TikTok post
func (c *TikTokClient) GetPostStats(ctx context.Context, postID string) (PostStats, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/video/stats/?video_id=%s", c.baseURL, postID),
		nil,
	)
	if err != nil {
		return PostStats{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return PostStats{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return PostStats{}, fmt.Errorf("stats request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Stats struct {
				ViewCount    int64 `json:"view_count"`
				LikeCount    int64 `json:"like_count"`
				CommentCount int64 `json:"comment_count"`
				ShareCount   int64 `json:"share_count"`
			} `json:"stats"`
			Demographics map[string]interface{} `json:"demographics"`
		} `json:"data"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return PostStats{}, fmt.Errorf("failed to decode response: %w", err)
	}

	engagement := float64(0)
	if result.Data.Stats.ViewCount > 0 {
		engagement = float64(
			result.Data.Stats.LikeCount+result.Data.Stats.CommentCount+result.Data.Stats.ShareCount,
		) / float64(
			result.Data.Stats.ViewCount,
		) * 100
	}

	return PostStats{
		Views:        result.Data.Stats.ViewCount,
		Likes:        result.Data.Stats.LikeCount,
		Comments:     result.Data.Stats.CommentCount,
		Shares:       result.Data.Stats.ShareCount,
		Engagement:   engagement,
		Demographics: result.Data.Demographics,
	}, nil
}

// SearchContent searches for content on TikTok
func (c *TikTokClient) SearchContent(ctx context.Context, query string) ([]ContentItem, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/search/videos/?query=%s", c.baseURL, query), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Videos []struct {
				ID          string `json:"id"`
				Title       string `json:"title"`
				Description string `json:"description"`
				URL         string `json:"share_url"`
				Author      struct {
					Username string `json:"username"`
				} `json:"author"`
				Stats struct {
					ViewCount    int64 `json:"view_count"`
					LikeCount    int64 `json:"like_count"`
					CommentCount int64 `json:"comment_count"`
					ShareCount   int64 `json:"share_count"`
				} `json:"stats"`
			} `json:"videos"`
		} `json:"data"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	items := make([]ContentItem, 0, len(result.Data.Videos))
	for _, v := range result.Data.Videos {
		engagement := float64(0)
		if v.Stats.ViewCount > 0 {
			engagement = float64(
				v.Stats.LikeCount+v.Stats.CommentCount+v.Stats.ShareCount,
			) / float64(
				v.Stats.ViewCount,
			) * 100
		}

		items = append(items, ContentItem{
			ID:          v.ID,
			Title:       v.Title,
			Description: v.Description,
			URL:         v.URL,
			Author:      v.Author.Username,
			Stats: PostStats{
				Views:      v.Stats.ViewCount,
				Likes:      v.Stats.LikeCount,
				Comments:   v.Stats.CommentCount,
				Shares:     v.Stats.ShareCount,
				Engagement: engagement,
			},
		})
	}

	return items, nil
}

// DeleteContent deletes a TikTok video
func (c *TikTokClient) DeleteContent(ctx context.Context, contentID string) error {
	data := map[string]string{
		"video_id": contentID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+"/video/delete/", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateContent updates a TikTok video's metadata
func (c *TikTokClient) UpdateContent(ctx context.Context, contentID string, data UpdateData) error {
	updateData := map[string]interface{}{
		"video_id": contentID,
	}

	if data.Title != nil {
		updateData["title"] = *data.Title
	}
	if data.Description != nil {
		updateData["description"] = *data.Description
	}
	if data.Tags != nil {
		updateData["tags"] = *data.Tags
	}
	if data.Privacy != nil {
		updateData["privacy_level"] = *data.Privacy
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", c.baseURL+"/video/update/", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// YouTube API Client
type YouTubeClient struct {
	accessToken string
	baseURL     string
	httpClient  *http.Client
}

// NewYouTubeClient creates a new YouTube API client
func NewYouTubeClient(accessToken string) *YouTubeClient {
	return &YouTubeClient{
		accessToken: accessToken,
		baseURL:     "https://www.googleapis.com/youtube/v3",
		httpClient:  &http.Client{Timeout: 60 * time.Second},
	}
}

// CreatePost uploads a video to YouTube
func (c *YouTubeClient) CreatePost(ctx context.Context, post PostData) (string, error) {
	// YouTube API requires a two-step process:
	// 1. Insert video metadata
	// 2. Upload video content

	// Step 1: Insert video metadata
	metaData := map[string]interface{}{
		"snippet": map[string]interface{}{
			"title":       post.Title,
			"description": post.Description,
			"tags":        post.Tags,
		},
		"status": map[string]interface{}{
			"privacyStatus": post.Privacy,
		},
	}

	if post.ScheduleTime != nil {
		metaData["status"].(map[string]interface{})["publishAt"] = post.ScheduleTime.Format(time.RFC3339)
	}

	jsonData, err := json.Marshal(metaData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Step 1: Create metadata request
	metaReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/videos?part=snippet,status",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create metadata request: %w", err)
	}

	metaReq.Header.Set("Content-Type", "application/json")
	metaReq.Header.Set("Authorization", "Bearer "+c.accessToken)

	// Step 2: Open the video file
	file, err := os.Open(post.VideoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open video file: %w", err)
	}
	defer file.Close()

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add metadata part
	metaPart, err := writer.CreateFormField("metadata")
	if err != nil {
		return "", fmt.Errorf("failed to create metadata field: %w", err)
	}
	if _, err = metaPart.Write(jsonData); err != nil {
		return "", fmt.Errorf("failed to write metadata: %w", err)
	}

	// Add file part
	filePart, err := writer.CreateFormFile("media", filepath.Base(post.VideoPath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err = io.Copy(filePart, file); err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// Create upload request
	uploadReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://www.googleapis.com/upload/youtube/v3/videos?uploadType=multipart&part=snippet,status",
		body,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %w", err)
	}

	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("Authorization", "Bearer "+c.accessToken)

	// Send request
	resp, err := c.httpClient.Do(uploadReq)
	if err != nil {
		return "", fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ID, nil
}

// ReplyToComment posts a reply to a comment on YouTube
func (c *YouTubeClient) ReplyToComment(ctx context.Context, postID, commentID, replyText string) (string, error) {
	data := map[string]interface{}{
		"snippet": map[string]interface{}{
			"parentId":     commentID,
			"textOriginal": replyText,
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/comments?part=snippet", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("reply failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ID, nil
}

// GetPostStats retrieves metrics for a YouTube video
func (c *YouTubeClient) GetPostStats(ctx context.Context, postID string) (PostStats, error) {
	// Fetch video statistics
	statsReq, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/videos?part=statistics&id=%s", c.baseURL, postID),
		nil,
	)
	if err != nil {
		return PostStats{}, fmt.Errorf("failed to create stats request: %w", err)
	}

	statsReq.Header.Set("Authorization", "Bearer "+c.accessToken)

	statsResp, err := c.httpClient.Do(statsReq)
	if err != nil {
		return PostStats{}, fmt.Errorf("stats request failed: %w", err)
	}
	defer statsResp.Body.Close()

	if statsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(statsResp.Body)
		return PostStats{}, fmt.Errorf("stats request failed with status %d: %s", statsResp.StatusCode, string(body))
	}

	var statsResult struct {
		Items []struct {
			Statistics struct {
				ViewCount     string `json:"viewCount"`
				LikeCount     string `json:"likeCount"`
				DislikeCount  string `json:"dislikeCount"`
				FavoriteCount string `json:"favoriteCount"`
				CommentCount  string `json:"commentCount"`
			} `json:"statistics"`
		} `json:"items"`
	}

	if err = json.NewDecoder(statsResp.Body).Decode(&statsResult); err != nil {
		return PostStats{}, fmt.Errorf("failed to decode stats response: %w", err)
	}

	if len(statsResult.Items) == 0 {
		return PostStats{}, fmt.Errorf("no stats found for video ID: %s", postID)
	}

	stats := statsResult.Items[0].Statistics

	// Fetch analytics data for demographics (requires YouTube Analytics API)
	// This is a placeholder as the actual demographics API requires more complex OAuth setup
	demographics := map[string]interface{}{
		"note": "To get full demographics, use the YouTube Analytics API",
	}

	// Parse string counts to int64
	viewCount, _ := parseInt64(stats.ViewCount)
	likeCount, _ := parseInt64(stats.LikeCount)
	commentCount, _ := parseInt64(stats.CommentCount)
	favoriteCount, _ := parseInt64(stats.FavoriteCount)

	// Calculate engagement
	engagement := float64(0)
	if viewCount > 0 {
		engagement = float64(likeCount+commentCount+favoriteCount) / float64(viewCount) * 100
	}

	return PostStats{
		Views:        viewCount,
		Likes:        likeCount,
		Comments:     commentCount,
		Shares:       favoriteCount, // YouTube uses "favorites" instead of shares
		Engagement:   engagement,
		Demographics: demographics,
	}, nil
}

// SearchContent searches for videos on YouTube
func (c *YouTubeClient) SearchContent(ctx context.Context, query string) ([]ContentItem, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/search?part=snippet&q=%s&type=video", c.baseURL, query),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
			Snippet struct {
				Title        string `json:"title"`
				Description  string `json:"description"`
				ChannelTitle string `json:"channelTitle"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	items := make([]ContentItem, 0, len(result.Items))

	// Batch fetch video statistics
	videoIDs := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		videoIDs = append(videoIDs, item.ID.VideoID)
	}

	// If we have videos, fetch their stats
	if len(videoIDs) > 0 {
		for _, item := range result.Items {
			// We'll only fetch basic info for this example
			videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.ID.VideoID)

			// In a real implementation, we would fetch stats for each video
			// But for this example, we'll just add placeholder stats
			items = append(items, ContentItem{
				ID:          item.ID.VideoID,
				Title:       item.Snippet.Title,
				Description: item.Snippet.Description,
				URL:         videoURL,
				Author:      item.Snippet.ChannelTitle,
				Stats: PostStats{
					Views:      0, // Would need separate API call for stats
					Likes:      0,
					Comments:   0,
					Shares:     0,
					Engagement: 0,
				},
			})
		}
	}

	return items, nil
}

// DeleteContent deletes a YouTube video
func (c *YouTubeClient) DeleteContent(ctx context.Context, contentID string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/videos?id=%s", c.baseURL, contentID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateContent updates a YouTube video's metadata
func (c *YouTubeClient) UpdateContent(ctx context.Context, contentID string, data UpdateData) error {
	updateData := map[string]interface{}{
		"id":      contentID,
		"snippet": map[string]interface{}{},
		"status":  map[string]interface{}{},
	}

	if data.Title != nil {
		updateData["snippet"].(map[string]interface{})["title"] = *data.Title
	}
	if data.Description != nil {
		updateData["snippet"].(map[string]interface{})["description"] = *data.Description
	}
	if data.Tags != nil {
		updateData["snippet"].(map[string]interface{})["tags"] = *data.Tags
	}
	if data.Privacy != nil {
		updateData["status"].(map[string]interface{})["privacyStatus"] = *data.Privacy
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"PUT",
		c.baseURL+"/videos?part=snippet,status",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper function to parse string to int64
func parseInt64(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
