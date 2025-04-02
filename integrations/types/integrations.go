package types

type LinkedInUserProfile struct {
	ID              string `json:"id"`
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	ProfilePicture  string `json:"profilePicture"`
	Email           string `json:"email"`
	Headline        string `json:"headline"`
	Industry        string `json:"industry"`
	Country         string `json:"country"`
	CurrentPosition string `json:"currentPosition"`
}

// CompanyPage represents a LinkedIn company page
type LinkedInCompanyPage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
	Website     string `json:"website"`
	Industry    string `json:"industry"`
	Size        string `json:"size"`
	Followers   int    `json:"followers"`
}

// PostResponse represents a response from creating a post
type LinkedInPostResponse struct {
	ID     string `json:"id"`
	Status string `json:"status,omitempty"`
}

// PostMetrics represents engagement metrics for a post
type LinkedInPostMetrics struct {
	Impressions    int     `json:"impressions"`
	Clicks         int     `json:"clicks"`
	Likes          int     `json:"likes"`
	Comments       int     `json:"comments"`
	Shares         int     `json:"shares"`
	Engagement     int     `json:"engagement"`
	CTR            float64 `json:"ctr"`
	EngagementRate float64 `json:"engagementRate"`
}
