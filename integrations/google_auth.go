package integrations

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GoogleOAuthConfig holds the configuration for Google OAuth
type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// GoogleToken represents an OAuth token
type GoogleToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Expiry       time.Time
}

// GoogleUserInfo represents the user information returned from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// NewGoogleOAuth creates a new Google OAuth config
func NewGoogleOAuth(clientID, clientSecret, redirectURL string, scopes []string) *GoogleOAuthConfig {
	// If no scopes provided, use the default profile and email scopes
	if len(scopes) == 0 {
		scopes = []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
		}
	}

	return &GoogleOAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}
}

// GenerateStateToken creates a random state token to prevent CSRF attacks
func GenerateStateToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetLoginURL returns the Google OAuth login URL
func (g *GoogleOAuthConfig) GetLoginURL(state string) string {
	authURL := "https://accounts.google.com/o/oauth2/auth"

	// Build query parameters
	params := url.Values{}
	params.Add("client_id", g.ClientID)
	params.Add("redirect_uri", g.RedirectURL)
	params.Add("response_type", "code")
	params.Add("scope", strings.Join(g.Scopes, " "))
	params.Add("state", state)
	params.Add("access_type", "online")

	return authURL + "?" + params.Encode()
}

// ExchangeCodeForToken exchanges the authorization code for an access token
func (g *GoogleOAuthConfig) ExchangeCodeForToken(ctx context.Context, code string) (*GoogleToken, error) {
	tokenURL := "https://oauth2.googleapis.com/token"

	// Build the form data
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", g.ClientID)
	data.Set("client_secret", g.ClientSecret)
	data.Set("redirect_uri", g.RedirectURL)
	data.Set("grant_type", "authorization_code")

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var token GoogleToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Set the expiry time
	if token.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return &token, nil
}

// GetUserInfo retrieves the Google user info using the access token
func GetUserInfo(ctx context.Context, token *GoogleToken) (*GoogleUserInfo, error) {
	// Make a request to the userinfo endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user info request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// RefreshToken refreshes an expired access token
func (g *GoogleOAuthConfig) RefreshToken(ctx context.Context, refreshToken string) (*GoogleToken, error) {
	tokenURL := "https://oauth2.googleapis.com/token"

	// Build the form data
	data := url.Values{}
	data.Set("client_id", g.ClientID)
	data.Set("client_secret", g.ClientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send refresh token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("refresh token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var token GoogleToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Set the expiry time
	if token.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	// Preserve the refresh token if a new one wasn't provided
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}

	return &token, nil
}

// VerifyIDToken verifies and decodes a Google ID token
func VerifyIDToken(ctx context.Context, idToken string) (map[string]interface{}, error) {
	// Google's tokeninfo endpoint for verifying ID tokens
	tokenInfoURL := "https://oauth2.googleapis.com/tokeninfo"

	// Build the query parameters
	params := url.Values{}
	params.Add("id_token", idToken)

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", tokenInfoURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid ID token")
	}

	// Parse the response
	var tokenInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to decode token info: %w", err)
	}

	return tokenInfo, nil
}

// func main() {
// 	// Set up your Google OAuth configuration
// 	googleAuth := NewGoogleOAuth(
// 		os.Getenv("GOOGLE_CLIENT_ID"),
// 		os.Getenv("GOOGLE_CLIENT_SECRET"),
// 		"http://localhost:8080/auth/google/callback",
// 		nil, // Use default scopes
// 	)

// 	// Set up HTTP routes
// 	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
// 		// Generate a state token to prevent CSRF
// 		state, err := GenerateStateToken()
// 		if err != nil {
// 			http.Error(w, "Failed to generate state token", http.StatusInternalServerError)
// 			return
// 		}

// 		// Store the state in a cookie
// 		cookie := &http.Cookie{
// 			Name:     "oauthstate",
// 			Value:    state,
// 			Expires:  time.Now().Add(20 * time.Minute),
// 			HttpOnly: true,
// 			Path:     "/",
// 			Secure:   r.TLS != nil,
// 			SameSite: http.SameSiteLaxMode,
// 		}
// 		http.SetCookie(w, cookie)

// 		// Redirect to Google's consent page
// 		url := googleAuth.GetLoginURL(state)
// 		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
// 	})

// 	http.HandleFunc("/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
// 		// Get the state from cookie for validation
// 		stateCookie, err := r.Cookie("oauthstate")
// 		if err != nil {
// 			http.Error(w, "State not found", http.StatusBadRequest)
// 			return
// 		}

// 		// Verify state parameter to prevent CSRF attacks
// 		if r.FormValue("state") != stateCookie.Value {
// 			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
// 			return
// 		}

// 		// Exchange the authorization code for a token
// 		code := r.FormValue("code")
// 		token, err := googleAuth.ExchangeCodeForToken(r.Context(), code)
// 		if err != nil {
// 			http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
// 			return
// 		}

// 		// Get the user info
// 		userInfo, err := GetUserInfo(r.Context(), token)
// 		if err != nil {
// 			http.Error(w, "Failed to get user info", http.StatusInternalServerError)
// 			return
// 		}

// 		// Use the user info for authentication/registration
// 		fmt.Fprintf(w, "Welcome %s! Your email is %s", userInfo.Name, userInfo.Email)

// 		// In a real application, you would:
// 		// 1. Check if the user exists in your database
// 		// 2. Create a session or JWT token for the user
// 		// 3. Store refresh token securely if you need offline access
// 		// 4. Redirect to your application's main page
// 	})

// 	// Example of a protected route that requires authentication
// 	http.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
// 		// In a real app, you would verify the user's session/token here
// 		fmt.Fprintf(w, "This is a protected profile page")
// 	})

// 	// Start the server
// 	fmt.Println("Server is running on http://localhost:8080")
// 	http.ListenAndServe(":8080", nil)
// }
