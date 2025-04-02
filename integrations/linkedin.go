package integrations

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"postly.com/integrations/types"
)

// Constants for LinkedIn API
const (
	LinkedinBaseURL = "https://api.linkedin.com/v2"
	AuthURL         = "https://www.linkedin.com/oauth/v2/authorization"
	TokenURL        = "https://www.linkedin.com/oauth/v2/accessToken"
	UGCPostURL      = "https://api.linkedin.com/v2/ugcPosts"
	ShareURL        = "https://api.linkedin.com/v2/shares"
	AssetUploadURL  = "https://api.linkedin.com/v2/assets"
	MediaUploadURL  = "https://api.linkedin.com/mediaUpload"
)

// LinkedInClient handles LinkedIn API operations
type LinkedInClient struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AccessToken  string
	UserID       string
	HTTPClient   *http.Client
}

// UserProfile represents a LinkedIn user profile

// NewLinkedInClient creates a new LinkedIn API client
func NewLinkedInClient(clientID, clientSecret, redirectURI string) *LinkedInClient {
	return &LinkedInClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *LinkedInClient) GetAuthURL(scopes []byte) string {
	scopesStr := []string{}
	_ = json.Unmarshal(scopes, &scopesStr)
	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", c.ClientID)
	params.Add("redirect_uri", c.RedirectURI)
	params.Add("scope", strings.Join(scopesStr, " "))
	// params.Add("state", generateRandomState())

	return fmt.Sprintf("%s?%s", AuthURL, params.Encode())
}

// GetAccessToken exchanges the authorization code for an access token
func (c *LinkedInClient) GetAccessToken(code string) (*TokenResponse, error) {
	params := url.Values{}
	params.Add("grant_type", "authorization_code")
	params.Add("code", code)
	params.Add("redirect_uri", c.RedirectURI)
	params.Add("client_id", c.ClientID)
	params.Add("client_secret", c.ClientSecret)

	req, err := http.NewRequest("POST", TokenURL, strings.NewReader(params.Encode()))
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

	return &tokenResp, nil
}

// RefreshAccessToken refreshes an access token using refresh token
func (c *LinkedInClient) RefreshAccessToken(refreshToken string) (*TokenResponse, error) {
	params := url.Values{}
	params.Add("grant_type", "refresh_token")
	params.Add("refresh_token", refreshToken)
	params.Add("client_id", c.ClientID)
	params.Add("client_secret", c.ClientSecret)

	req, err := http.NewRequest("POST", TokenURL, strings.NewReader(params.Encode()))
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
		return nil, fmt.Errorf("failed to refresh access token: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	c.AccessToken = tokenResp.AccessToken

	return &tokenResp, nil
}

// GetUserProfile retrieves the authenticated user's profile
func (c *LinkedInClient) GetUserProfile() ([]byte, error) {
	if c.AccessToken == "" {
		return nil, errors.New("access token is required")
	}

	// LinkedIn API requires specific fields to be requested
	params := url.Values{}
	params.Add("projection", "(id,firstName,lastName,profilePicture,headline,email,industry)")

	profileURL := fmt.Sprintf("%s/me?%s", BaseURL, params.Encode())

	req, err := http.NewRequest("GET", profileURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get profile: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	// LinkedIn returns a complex nested JSON structure
	// This is simplified for readability
	var rawProfile map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawProfile); err != nil {
		return nil, err
	}

	// Extract the necessary fields from the complex structure
	profile := &types.LinkedInUserProfile{
		ID: rawProfile["id"].(string),
	}

	// Set the user ID in the client
	c.UserID = profile.ID

	// Parse other fields from the complex structure
	// In a real implementation, you would handle the nested localized fields properly
	if firstName, ok := rawProfile["firstName"].(map[string]interface{}); ok {
		if localized, ok := firstName["localized"].(map[string]interface{}); ok {
			for _, v := range localized {
				profile.FirstName = v.(string)
				break
			}
		}
	}

	if lastName, ok := rawProfile["lastName"].(map[string]interface{}); ok {
		if localized, ok := lastName["localized"].(map[string]interface{}); ok {
			for _, v := range localized {
				profile.LastName = v.(string)
				break
			}
		}
	}

	if headline, ok := rawProfile["headline"].(map[string]interface{}); ok {
		if localized, ok := headline["localized"].(map[string]interface{}); ok {
			for _, v := range localized {
				profile.Headline = v.(string)
				break
			}
		}
	}

	return json.Marshal(profile)
}

// GetCompanyPages retrieves company pages administered by the user
func (c *LinkedInClient) GetCompanyPages() ([]byte, error) {
	if c.AccessToken == "" {
		return nil, errors.New("access token is required")
	}

	orgURL := fmt.Sprintf("%s/organizationAcls?q=roleAssignee&role=ADMINISTRATOR", BaseURL)

	req, err := http.NewRequest("GET", orgURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get company pages: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	type OrganizationResponse struct {
		Elements []struct {
			OrganizationTarget string `json:"organizationTarget"`
			Role               string `json:"role"`
		} `json:"elements"`
	}

	var orgResp OrganizationResponse
	if err := json.NewDecoder(resp.Body).Decode(&orgResp); err != nil {
		return nil, err
	}

	// Retrieve details for each company page
	var companyPages []types.LinkedInCompanyPage

	for _, org := range orgResp.Elements {
		orgID := org.OrganizationTarget

		// Get organization details
		orgDetailsURL := fmt.Sprintf("%s/organizations/%s", BaseURL, orgID)

		detailsReq, err := http.NewRequest("GET", orgDetailsURL, nil)
		if err != nil {
			continue
		}

		detailsReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

		detailsResp, err := c.HTTPClient.Do(detailsReq)
		if err != nil {
			continue
		}

		var pageDetails map[string]interface{}
		if err := json.NewDecoder(detailsResp.Body).Decode(&pageDetails); err != nil {
			detailsResp.Body.Close()
			continue
		}
		detailsResp.Body.Close()

		page := types.LinkedInCompanyPage{
			ID: orgID,
		}

		// Extract relevant fields from the response
		if name, ok := pageDetails["name"].(string); ok {
			page.Name = name
		}

		if description, ok := pageDetails["description"].(map[string]interface{}); ok {
			if localized, ok := description["localized"].(map[string]interface{}); ok {
				for _, v := range localized {
					page.Description = v.(string)
					break
				}
			}
		}

		companyPages = append(companyPages, page)
	}

	return json.Marshal(companyPages)
}

// CreateTextPost creates a simple text post
func (c *LinkedInClient) CreateTextPost(input []byte) ([]byte, error) {
	var text, authorType, authorID string
	inputmap := map[string]interface{}{}
	json.Unmarshal(input, &inputmap)
	text, _ = inputmap["text"].(string)
	authorType, _ = inputmap["author_type"].(string)
	authorID, _ = inputmap["author_id"].(string)
	if c.AccessToken == "" {
		return nil, errors.New("access token is required")
	}

	if authorType == "" {
		authorType = "person"
	}

	if authorID == "" && authorType == "person" {
		// If no author ID is provided and type is person, use the authenticated user
		if c.UserID == "" {
			// Try to get the user profile if we don't have the ID
			inp, err := c.GetUserProfile()
			var profile types.LinkedInUserProfile
			_ = json.Unmarshal(inp, &profile)
			if err != nil {
				return nil, fmt.Errorf("could not determine user ID: %v", err)
			}
			authorID = profile.ID
		} else {
			authorID = c.UserID
		}
	}

	// Prepare the UGC post request
	postData := map[string]interface{}{
		"author":         fmt.Sprintf("urn:li:%s:%s", authorType, authorID),
		"lifecycleState": "PUBLISHED",
		"specificContent": map[string]interface{}{
			"com.linkedin.ugc.ShareContent": map[string]interface{}{
				"shareCommentary": map[string]interface{}{
					"text": text,
				},
				"shareMediaCategory": "NONE",
			},
		},
		"visibility": map[string]interface{}{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}

	postJSON, err := json.Marshal(postData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", UGCPostURL, bytes.NewBuffer(postJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Restli-Protocol-Version", "2.0.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create post: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	var postResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&postResp); err != nil {
		return nil, err
	}

	postID, ok := postResp["id"].(string)
	if !ok {
		return nil, errors.New("invalid post response, no ID found")
	}

	out := types.LinkedInPostResponse{
		ID: postID,
	}
	return json.Marshal(out)

}

// InitiateImageUpload prepares an image upload
func (c *LinkedInClient) InitiateImageUpload(imageType string) (string, map[string]interface{}, error) {
	if c.AccessToken == "" {
		return "", nil, errors.New("access token is required")
	}

	// Define the asset request
	assetData := map[string]interface{}{
		"registerUploadRequest": map[string]interface{}{
			"recipes": []string{
				"urn:li:digitalmediaRecipe:feedshare-image",
			},
			"owner": fmt.Sprintf("urn:li:person:%s", c.UserID),
			"serviceRelationships": []map[string]interface{}{
				{
					"relationshipType": "OWNER",
					"identifier":       "urn:li:userGeneratedContent",
				},
			},
		},
	}

	assetJSON, err := json.Marshal(assetData)
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("POST", AssetUploadURL, bytes.NewBuffer(assetJSON))
	if err != nil {
		return "", nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf(
			"failed to initiate image upload: %s, status: %d",
			string(bodyBytes),
			resp.StatusCode,
		)
	}

	var uploadResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", nil, err
	}

	// Get the upload URL and asset URN
	value, ok := uploadResp["value"].(map[string]interface{})
	if !ok {
		return "", nil, errors.New("invalid upload response structure")
	}

	uploadMechanism, ok := value["uploadMechanism"].(map[string]interface{})
	if !ok {
		return "", nil, errors.New("invalid upload mechanism")
	}

	uploadURL, ok := uploadMechanism["com.linkedin.digitalmedia.uploading.MediaUploadHttpRequest"].(map[string]interface{})["uploadUrl"].(string)
	fmt.Println(uploadURL)
	if !ok {
		return "", nil, errors.New("could not find upload URL")
	}

	asset, ok := value["asset"].(string)
	if !ok {
		return "", nil, errors.New("could not find asset URN")
	}

	return asset, uploadMechanism, nil
}

// UploadImage uploads an image to LinkedIn
func (c *LinkedInClient) UploadImage(imagePath string) (string, error) {
	if c.AccessToken == "" {
		return "", errors.New("access token is required")
	}

	// First, initiate the upload
	assetURN, uploadMechanism, err := c.InitiateImageUpload("image")
	if err != nil {
		return "", err
	}

	// Get the upload URL
	uploadURL, ok := uploadMechanism["com.linkedin.digitalmedia.uploading.MediaUploadHttpRequest"].(map[string]interface{})["uploadUrl"].(string)
	if !ok {
		return "", errors.New("could not find upload URL")
	}

	// Read the image file
	file, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fileContents, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// Upload the image
	uploadReq, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(fileContents))
	if err != nil {
		return "", err
	}

	resp, err := c.HTTPClient.Do(uploadReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated &&
		resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload image: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	return assetURN, nil
}

// CreateImagePost creates a post with an image
func (c *LinkedInClient) CreateImagePost(
	input []byte,
) ([]byte, error) {
	if c.AccessToken == "" {
		return nil, errors.New("access token is required")
	}

	var text,
		imageAssetURN,
		authorType,
		authorID string

	inputmap := map[string]interface{}{}
	json.Unmarshal(input, &inputmap)
	text, _ = inputmap["text"].(string)
	imageAssetURN, _ = inputmap["image_url"].(string)
	authorType, _ = inputmap["author_type"].(string)
	authorID, _ = inputmap["author_id"].(string)
	if authorType == "" {
		authorType = "person"
	}

	if authorID == "" && authorType == "person" {
		if c.UserID == "" {
			profileData, err := c.GetUserProfile()
			if err != nil {
				return nil, fmt.Errorf("could not determine user ID: %v", err)
			}
			profile := types.LinkedInUserProfile{}
			json.Unmarshal(profileData, &profile)
			authorID = profile.ID
		} else {
			authorID = c.UserID
		}
	}

	// Prepare the UGC post request with image
	postData := map[string]interface{}{
		"author":         fmt.Sprintf("urn:li:%s:%s", authorType, authorID),
		"lifecycleState": "PUBLISHED",
		"specificContent": map[string]interface{}{
			"com.linkedin.ugc.ShareContent": map[string]interface{}{
				"shareCommentary": map[string]interface{}{
					"text": text,
				},
				"shareMediaCategory": "IMAGE",
				"media": []map[string]interface{}{
					{
						"status": "READY",
						"description": map[string]interface{}{
							"text": "Image description",
						},
						"media": imageAssetURN,
						"title": map[string]interface{}{
							"text": "Image title",
						},
					},
				},
			},
		},
		"visibility": map[string]interface{}{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}

	postJSON, err := json.Marshal(postData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", UGCPostURL, bytes.NewBuffer(postJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Restli-Protocol-Version", "2.0.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create image post: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	var postResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&postResp); err != nil {
		return nil, err
	}

	postID, ok := postResp["id"].(string)
	if !ok {
		return nil, errors.New("invalid post response, no ID found")
	}

	output := types.LinkedInPostResponse{
		ID: postID,
	}
	return json.Marshal(output)
}

// PostWithImage is a convenience function that handles both image upload and post creation
func (c *LinkedInClient) PostWithImage(input []byte) ([]byte, error) {
	// First upload the image
	inputmap := map[string]interface{}{}
	json.Unmarshal(input, &inputmap)
	imagepath, _ := inputmap["image_path"].(string)
	assetURN, err := c.UploadImage(imagepath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %v", err)
	}
	inputmap["image_url"] = assetURN
	// Then create the post with the image
	bytes, _ := json.Marshal(inputmap)
	return c.CreateImagePost(bytes)
}

// InitiateVideoUpload prepares a video upload
func (c *LinkedInClient) InitiateVideoUpload() ([]byte, error) {
	if c.AccessToken == "" {
		return nil, errors.New("access token is required")
	}

	// Define the asset request for video
	assetData := map[string]interface{}{
		"registerUploadRequest": map[string]interface{}{
			"recipes": []string{
				"urn:li:digitalmediaRecipe:feedshare-video",
			},
			"owner": fmt.Sprintf("urn:li:person:%s", c.UserID),
			"serviceRelationships": []map[string]interface{}{
				{
					"relationshipType": "OWNER",
					"identifier":       "urn:li:userGeneratedContent",
				},
			},
		},
	}

	assetJSON, err := json.Marshal(assetData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", AssetUploadURL, bytes.NewBuffer(assetJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"failed to initiate video upload: %s, status: %d",
			string(bodyBytes),
			resp.StatusCode,
		)
	}

	var uploadResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return nil, err
	}

	// Get the upload URL and asset URN
	value, ok := uploadResp["value"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid upload response structure")
	}

	uploadMechanism, ok := value["uploadMechanism"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid upload mechanism")
	}

	asset, ok := value["asset"].(string)
	if !ok {
		return nil, errors.New("could not find asset URN")
	}
	data := map[string]interface{}{
		"asset":            asset,
		"upload_mechanism": uploadMechanism,
	}
	return json.Marshal(data)
}

// UploadVideo uploads a video to LinkedIn
func (c *LinkedInClient) UploadVideo(videoPath string) (string, error) {
	if c.AccessToken == "" {
		return "", errors.New("access token is required")
	}

	// First, initiate the upload
	assetURN := ""
	var err error
	uploadMechanism := map[string]interface{}{}
	videoData := []byte{}
	videoData, err = c.InitiateVideoUpload()
	if err != nil {
		return "", err
	}
	resp1 := map[string]interface{}{}
	json.Unmarshal(videoData, &resp1)
	assetURN, _ = resp1["asset"].(string)
	uploadMechanism, _ = resp1["upload_mechanism"].(map[string]interface{})
	// For videos, we need to use a multipart upload approach
	// This example assumes single-part upload for simplicity
	uploadInfo, ok := uploadMechanism["com.linkedin.digitalmedia.uploading.MediaUploadHttpRequest"].(map[string]interface{})
	if !ok {
		return "", errors.New("invalid upload mechanism format")
	}

	uploadURL, ok := uploadInfo["uploadUrl"].(string)
	if !ok {
		return "", errors.New("could not find upload URL")
	}

	// Read the video file
	file, err := os.Open(videoPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fileContents, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// Upload the video
	uploadReq, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(fileContents))
	if err != nil {
		return "", err
	}

	resp, err := c.HTTPClient.Do(uploadReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated &&
		resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload video: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	return assetURN, nil
}

// CreateVideoPost creates a post with a video
func (c *LinkedInClient) CreateVideoPost(
	input []byte,
) ([]byte, error) {
	if c.AccessToken == "" {
		return nil, errors.New("access token is required")
	}
	var text,
		vidoeAssetURL,
		authorType,
		authorID string

	inputmap := map[string]interface{}{}
	json.Unmarshal(input, &inputmap)
	text, _ = inputmap["text"].(string)
	vidoeAssetURL, _ = inputmap["video_url"].(string)
	authorType, _ = inputmap["author_type"].(string)
	authorID, _ = inputmap["author_id"].(string)

	if authorType == "" {
		authorType = "person"
	}

	if authorID == "" && authorType == "person" {
		if c.UserID == "" {
			profileData, err := c.GetUserProfile()
			if err != nil {
				return nil, fmt.Errorf("could not determine user ID: %v", err)
			}
			profile := types.LinkedInUserProfile{}
			json.Unmarshal(profileData, &profile)
			authorID = profile.ID
		} else {
			authorID = c.UserID
		}
	}

	// Prepare the UGC post request with video
	postData := map[string]interface{}{
		"author":         fmt.Sprintf("urn:li:%s:%s", authorType, authorID),
		"lifecycleState": "PUBLISHED",
		"specificContent": map[string]interface{}{
			"com.linkedin.ugc.ShareContent": map[string]interface{}{
				"shareCommentary": map[string]interface{}{
					"text": text,
				},
				"shareMediaCategory": "VIDEO",
				"media": []map[string]interface{}{
					{
						"status": "READY",
						"description": map[string]interface{}{
							"text": "Video description",
						},
						"media": vidoeAssetURL,
						"title": map[string]interface{}{
							"text": "Video title",
						},
					},
				},
			},
		},
		"visibility": map[string]interface{}{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}

	postJSON, err := json.Marshal(postData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", UGCPostURL, bytes.NewBuffer(postJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Restli-Protocol-Version", "2.0.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create video post: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	repmap := map[string]interface{}{}
	docoder := json.NewDecoder(req.Body)
	docoder.Decode(&repmap)
	return nil, nil
}
