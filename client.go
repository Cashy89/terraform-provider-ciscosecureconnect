package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

type MerakiClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	maxRetries int
}

func NewClient(apiKey string, baseURL string) *MerakiClient {
	return &MerakiClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
		maxRetries: 3, // Default retry count
	}
}

// SetMaxRetries allows configuring the maximum number of retry attempts
func (c *MerakiClient) SetMaxRetries(maxRetries int) {
	c.maxRetries = maxRetries
}

// doRequestWithRetry handles the HTTP request with retry logic
func (c *MerakiClient) doRequestWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err // Network errors shouldn't be retried
		}
		
		// Success case
		if resp.StatusCode < 300 {
			return resp, nil
		}
		
		// Don't retry on client errors (4xx) except 429 (rate limit)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			return resp, nil
		}
		
		// Calculate backoff time (exponential with jitter)
		if attempt < c.maxRetries {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			// Add jitter (up to 1 second)
			jitter := time.Duration(time.Now().UnixNano()%1000) * time.Millisecond
			totalWait := backoff + jitter
			
			select {
			case <-time.After(totalWait):
				// Retry after backoff
			case <-ctx.Done():
				resp.Body.Close()
				return nil, ctx.Err()
			}
		}
		
		// Close the response body before retrying
		resp.Body.Close()
	}
	
	return resp, nil
}

// Updated CreateSecureConnectSite with retry logic
func (c *MerakiClient) CreateSecureConnectSite(ctx context.Context, orgID, siteID, regionType, regionID, regionName string) error {
	url := fmt.Sprintf("%s/organizations/%s/secureConnect/sites", c.baseURL, orgID)

	enrollment := map[string]string{
		"siteId":     siteID,
		"regionType": regionType,
	}
	if regionID != "" {
		enrollment["regionId"] = regionID
	}
	if regionName != "" {
		enrollment["regionName"] = regionName
	}

	body := map[string]interface{}{
		"enrollments": []interface{}{enrollment},
	}

	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Cisco-Meraki-API-Key", c.apiKey)

	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("create failed: %s", resp.Status)
	}

	return nil
}

// Updated DeleteSecureConnectSites with retry logic
func (c *MerakiClient) DeleteSecureConnectSites(ctx context.Context, orgID, siteID string) error {
	url := fmt.Sprintf("%s/organizations/%s/secureConnect/sites", c.baseURL, orgID)

	body := map[string]interface{}{
		"sites": []string{siteID},
	}

	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Cisco-Meraki-API-Key", c.apiKey)

	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("delete failed: %s", resp.Status)
	}

	return nil
}

// Updated GetSecureConnectSites with proper Link header pagination
func (c *MerakiClient) GetSecureConnectSites(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	var allSites []map[string]interface{}
	
	// Initial request
	url := fmt.Sprintf("%s/organizations/%s/secureConnect/sites?perPage=1000", c.baseURL, orgID)

	for {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Cisco-Meraki-API-Key", c.apiKey)

		resp, err := c.doRequestWithRetry(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("sending request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("bad response (%d): %s", resp.StatusCode, string(body))
		}

		// Try to unmarshal into the standard Meraki response format
		var apiResponse struct {
			Data []map[string]interface{} `json:"data"`
		}

		var currentSites []map[string]interface{}
		
		if err := json.Unmarshal(body, &apiResponse); err == nil {
			currentSites = apiResponse.Data
			allSites = append(allSites, currentSites...)
		} else {
			// If that fails, try direct array unmarshal (for backward compatibility)
			var directData []map[string]interface{}
			if err := json.Unmarshal(body, &directData); err == nil {
				currentSites = directData
				allSites = append(allSites, directData...)
			} else {
				return nil, fmt.Errorf("response doesn't match expected formats (wrapped or direct array)")
			}
		}

		// Check for next page in Link header
		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" {
			break // No more pages
		}

		// Parse Link header to find next page
		nextURL := ""
		links := strings.Split(linkHeader, ",")
		for _, link := range links {
			link = strings.TrimSpace(link)
			if strings.Contains(link, "rel=\"next\"") || strings.Contains(link, "rel=next") {
				// Extract URL from <url>; rel="next"
				start := strings.Index(link, "<")
				end := strings.Index(link, ">")
				if start != -1 && end != -1 && end > start {
					nextURL = link[start+1 : end]
					break
				}
			}
		}

		if nextURL == "" {
			break // No next page found
		}

		url = nextURL // Set URL for next iteration
		
		// Also break if we got fewer than 1000 items (reached last page)
		if len(currentSites) < 1000 {
			break
		}
	}

	return allSites, nil
}