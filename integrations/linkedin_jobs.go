package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a LinkedIn API client
type Client struct {
	AccessToken string
	HTTPClient  *http.Client
	BaseURL     string
}

// JobPosting represents a LinkedIn job posting
type JobPosting struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Location        Location `json:"location"`
	CompanyID       string   `json:"companyID"`
	ExperienceLevel string   `json:"experienceLevel,omitempty"`
	EmploymentType  string   `json:"employmentType,omitempty"`
	WorkRemoteType  string   `json:"workRemoteType,omitempty"`
	SeniorityLevel  string   `json:"seniorityLevel,omitempty"`
	ApplicationURL  string   `json:"applicationUrl,omitempty"`
	ExpirationDate  string   `json:"expirationDate,omitempty"`
}

// Location represents a job location
type Location struct {
	Country    string `json:"country"`
	City       string `json:"city,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	RegionCode string `json:"regionCode,omitempty"`
	LineOne    string `json:"addressLineOne,omitempty"`
	LineTwo    string `json:"addressLineTwo,omitempty"`
}

// NewClient creates a new LinkedIn API client
func NewClient(accessToken string) *Client {
	return &Client{
		AccessToken: accessToken,
		HTTPClient: &http.Client{
			Timeout: time.Second * 30,
		},
		BaseURL: "https://api.linkedin.com/v2",
	}
}

// CreateJobPosting creates a new job posting on LinkedIn
func (c *Client) CreateJobPosting(jobPosting *JobPosting) (string, error) {
	url := fmt.Sprintf("%s/jobs", c.BaseURL)

	jobData, err := json.Marshal(jobPosting)
	if err != nil {
		return "", fmt.Errorf("error marshaling job posting: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jobData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}

	return result.ID, nil
}

// GetJobPosting fetches a job posting by ID
func (c *Client) GetJobPosting(jobID string) (*JobPosting, error) {
	url := fmt.Sprintf("%s/jobs/%s", c.BaseURL, jobID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	var jobPosting JobPosting
	if err := json.NewDecoder(resp.Body).Decode(&jobPosting); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &jobPosting, nil
}

// UpdateJobPosting updates an existing job posting
func (c *Client) UpdateJobPosting(jobID string, jobPosting *JobPosting) error {
	url := fmt.Sprintf("%s/jobs/%s", c.BaseURL, jobID)

	jobData, err := json.Marshal(jobPosting)
	if err != nil {
		return fmt.Errorf("error marshaling job posting: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jobData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	return nil
}

// DeleteJobPosting deletes a job posting
func (c *Client) DeleteJobPosting(jobID string) error {
	url := fmt.Sprintf("%s/jobs/%s", c.BaseURL, jobID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	return nil
}

// ListJobPostings fetches all job postings for a company
func (c *Client) ListJobPostings(companyID string, limit int, offset int) ([]JobPosting, error) {
	url := fmt.Sprintf("%s/jobs?companyId=%s&limit=%d&offset=%d", c.BaseURL, companyID, limit, offset)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s, status: %d", string(bodyBytes), resp.StatusCode)
	}

	var result struct {
		Elements []JobPosting `json:"elements"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return result.Elements, nil
}
