package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	toolsPkg "github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/spf13/viper"
)

type copilotOptions struct {
	reasoningEffort string
	extraHeaders    map[string]string
	bearerToken     string
}

type CopilotOption func(*copilotOptions)

type copilotClient struct {
	providerOptions providerClientOptions
	options         copilotOptions
	client          openai.Client
	httpClient      *http.Client
}

type CopilotClient ProviderClient

// CopilotTokenResponse represents the response from GitHub's token exchange endpoint
type CopilotTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

func (c *copilotClient) isAnthropicModel() bool {
	for _, modelId := range models.CopilotAnthropicModels {
		if c.providerOptions.model.ID == modelId {
			return true
		}
	}
	return false
}

func (c *copilotClient) getGitHubTokenScopes(githubToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create GitHub API request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+githubToken)
	req.Header.Set("User-Agent", "OpenCode/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	scopes := resp.Header.Get("X-OAuth-Scopes")
	return scopes, nil
}

// GitHub OAuth device flow response
type GitHubDeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// GitHub OAuth token response
type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// exchangeGitHubToken exchanges a GitHub token for a Copilot bearer token
// If the token is provided, it will try to use it directly
// Otherwise, it will start the GitHub device code flow
func (c *copilotClient) exchangeGitHubToken(githubToken string) (string, error) {
	// If a token is provided, try to use it directly
	if githubToken != "" {
		prefixLen := 10
		if len(githubToken) < prefixLen {
			prefixLen = len(githubToken)
		}
		logging.Debug("Exchanging GitHub token", "token_length", len(githubToken), "token_prefix", githubToken[:prefixLen])

		// Check GitHub token scopes first to verify it's a valid token
		scopes, err := c.getGitHubTokenScopes(githubToken)
		if err != nil {
			logging.Error("Failed to get GitHub token scopes", "error", err)
			// If we can't verify token scopes, just continue - the token exchange will fail if invalid
		} else {
			logging.Debug("GitHub token scopes", "scopes", scopes)
			if !strings.Contains(scopes, "copilot") {
				logging.Warn("GitHub token does not have copilot scope - token exchange may fail")
			}
		}

		// Attempt to exchange for a Copilot bearer token - match VS Code exactly
		req, err := http.NewRequest("GET", "https://api.github.com/copilot_internal/v2/token", nil)
		if err != nil {
			return "", fmt.Errorf("failed to create token exchange request: %w", err)
		}

		req.Header.Set("Authorization", "token "+githubToken) // Note: "token" not "Token"
		req.Header.Set("User-Agent", "GithubCopilot/1.133.0")
		req.Header.Set("Accept", "application/json")

		logging.Debug("Sending token exchange request to GitHub API")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			logging.Error("Failed HTTP request for token exchange", "error", err)
			// If we're not in non-interactive mode, try device flow
			if !viper.GetBool("non_interactive") && viper.GetString("prompt") == "" {
				logging.Info("Token exchange HTTP request failed, falling back to device code flow")
				return c.performDeviceCodeFlow()
			}
			return "", fmt.Errorf("failed to exchange GitHub token: %w", err)
		}
		defer resp.Body.Close()

		logging.Debug("Token exchange response received", "status", resp.StatusCode, "headers", resp.Header)
		
		// Check for HTTP errors
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			logging.Error("Token exchange failed", "status_code", resp.StatusCode, "body", string(body))
			
			// If we're not in non-interactive mode, try device flow
			if !viper.GetBool("non_interactive") && viper.GetString("prompt") == "" {
				logging.Info("Token exchange failed, falling back to device code flow")
				return c.performDeviceCodeFlow()
			}
			return "", fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Success! Read the response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logging.Error("Failed to read token response body", "error", err)
			return "", fmt.Errorf("failed to read token response: %w", err)
		}

		logging.Debug("Token exchange response body received, length", "bytes", len(body))
		
		var tokenResp CopilotTokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			logging.Error("Failed to decode token response", "error", err)
			return "", fmt.Errorf("failed to decode token response: %w", err)
		}

		if tokenResp.Token == "" {
			logging.Error("Received empty token from GitHub API")
			
			// If we're not in non-interactive mode, try device flow
			if !viper.GetBool("non_interactive") && viper.GetString("prompt") == "" {
				logging.Info("Received empty token, falling back to device code flow")
				return c.performDeviceCodeFlow()
			}
			return "", fmt.Errorf("received empty token from GitHub API")
		}

		prefixLen = 10
		if len(tokenResp.Token) < prefixLen {
			prefixLen = len(tokenResp.Token)
		}
		logging.Debug("Successfully obtained Copilot bearer token", 
			"token_prefix", tokenResp.Token[:prefixLen], 
			"expires_at", tokenResp.ExpiresAt)
			
		// Try saving the token for future use
		saveGitHubToken(githubToken)
			
		return tokenResp.Token, nil
	} else {
		// No token provided, use device code flow if we're not in non-interactive mode
		if !viper.GetBool("non_interactive") && viper.GetString("prompt") == "" {
			logging.Info("No GitHub token provided, starting device code flow")
			return c.performDeviceCodeFlow()
		}
		return "", fmt.Errorf("no GitHub token available and running in non-interactive mode")
	}
}

// saveGitHubToken saves the GitHub token to the standard location for future use
func saveGitHubToken(token string) {
	// Only save if we have a token
	if token == "" {
		return
	}
	
	// Get the config directory
	var configDir string
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		configDir = xdgConfig
	} else if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			configDir = localAppData
		} else {
			configDir = filepath.Join(os.Getenv("HOME"), "AppData", "Local")
		}
	} else {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	
	// Create the directory if it doesn't exist
	copilotDir := filepath.Join(configDir, "github-copilot")
	if err := os.MkdirAll(copilotDir, 0755); err != nil {
		logging.Error("Failed to create github-copilot directory", "error", err)
		return
	}
	
	// Create the hosts.json file
	hostsFile := filepath.Join(copilotDir, "hosts.json")
	
	// Create the JSON structure
	hostsData := map[string]map[string]interface{}{
		"github.com": {
			"oauth_token": token,
		},
	}
	
	// Marshal to JSON
	jsonData, err := json.MarshalIndent(hostsData, "", "  ")
	if err != nil {
		logging.Error("Failed to marshal hosts.json", "error", err)
		return
	}
	
	// Write the file
	if err := os.WriteFile(hostsFile, jsonData, 0600); err != nil {
		logging.Error("Failed to write hosts.json", "error", err)
		return
	}
	
	logging.Info("Saved GitHub token to hosts.json for future use", "path", hostsFile)
}

// performDeviceCodeFlow initiates the GitHub device code flow and returns a Copilot bearer token
func (c *copilotClient) performDeviceCodeFlow() (string, error) {
	// Step 1: Get a device code
	data := url.Values{}
	
	// Use the official GitHub Copilot client ID
	// This is used by multiple Copilot integrations including Neovim
	// The client ID is publicly visible in VS Code and Neovim plugins
	const copilotClientID = "Iv1.b507a08c87ecfe98"
	data.Set("client_id", copilotClientID)
	data.Set("scope", "user:email read:user copilot")
	
	fmt.Printf("🔐 Using GitHub Copilot client ID: %s\n", copilotClientID)
	fmt.Printf("🔐 Requesting device code for scopes: user:email read:user copilot\n")

	// Using the exact URL and headers from VS Code Copilot extension
	req, err := http.NewRequest("POST", "https://github.com/login/device/code", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create device code request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "OpenCode/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get device code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("device code request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read device code response: %w", err)
	}

	var deviceResp GitHubDeviceCodeResponse
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return "", fmt.Errorf("failed to parse device code response: %w", err)
	}

	// Step 2: Print instructions for the user
	fmt.Printf("\n🔑 GitHub Copilot Authentication Required\n\n")
	fmt.Printf("1. Visit: %s\n", deviceResp.VerificationURI)
	fmt.Printf("2. Enter code: %s\n\n", deviceResp.UserCode)
	fmt.Printf("Waiting for authentication... (expires in %d seconds)\n", deviceResp.ExpiresIn)
	fmt.Printf("Please complete authentication in your browser to continue.\n\n")

	// Step 3: Poll for the token
	tokenData := url.Values{}
	tokenData.Set("client_id", copilotClientID) // Use the same client ID as before
	tokenData.Set("device_code", deviceResp.DeviceCode)
	tokenData.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	// Add a slight delay before first poll
	time.Sleep(2 * time.Second)

	// Create a context with timeout based on expires_in
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(deviceResp.ExpiresIn)*time.Second)
	defer cancel()

	interval := deviceResp.Interval
	if interval < 5 {
		interval = 5 // Ensure minimum polling interval
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	fmt.Printf("⏳ Waiting for you to authorize the device...\n")
	pollAttempts := 0
	
	for {
		select {
		case <-ticker.C:
			pollAttempts++
			fmt.Printf("🔄 Checking authorization status... (attempt %d)\n", pollAttempts)
			
			// Make a request to check if the user has authorized
			tokenReq, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", 
				strings.NewReader(tokenData.Encode()))
			if err != nil {
				fmt.Printf("❌ Error creating token request: %v\n", err)
				return "", fmt.Errorf("failed to create token request: %w", err)
			}

			tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			tokenReq.Header.Set("User-Agent", "OpenCode/1.0")
			tokenReq.Header.Set("Accept", "application/json")

			tokenResp, err := c.httpClient.Do(tokenReq)
			if err != nil {
				fmt.Printf("❌ Error making token request: %v\n", err)
				return "", fmt.Errorf("failed token request: %w", err)
			}

			fmt.Printf("📊 Token poll response status: %d\n", tokenResp.StatusCode)
			
			tokenRespBody, err := io.ReadAll(tokenResp.Body)
			tokenResp.Body.Close()
			if err != nil {
				fmt.Printf("❌ Error reading token response: %v\n", err)
				return "", fmt.Errorf("failed to read token response: %w", err)
			}

			if tokenResp.StatusCode == http.StatusOK {
				fmt.Printf("✅ Received response from GitHub. Processing...\n")
				fmt.Printf("📃 Raw response: %s\n", string(tokenRespBody))
				
				// Check if we're getting an error response even with 200 status
				var errorCheck map[string]string
				if json.Unmarshal(tokenRespBody, &errorCheck) == nil {
					if errorVal, ok := errorCheck["error"]; ok {
						fmt.Printf("⚠️ Received error with 200 status: %s\n", errorVal)
						if errorVal == "authorization_pending" {
							fmt.Printf("⏳ Still waiting for authorization in browser...\n")
							continue
						}
					}
				}
				
				var tokenData GitHubTokenResponse
				if err := json.Unmarshal(tokenRespBody, &tokenData); err != nil {
					fmt.Printf("❌ Error parsing token response: %v\n", err)
					return "", fmt.Errorf("failed to parse token response: %w", err)
				}

				fmt.Printf("✅ Token data: %+v\n", tokenData)
				
				if tokenData.AccessToken != "" {
					fmt.Printf("✅ Successfully authenticated with GitHub!\n")
					fmt.Printf("✅ Token received and stored for future use\n")
					fmt.Printf("✅ Now exchanging for Copilot bearer token...\n")
					
					// Save the token for future use
					saveGitHubToken(tokenData.AccessToken)
					
					// Set environment variable for immediate use in this session
					os.Setenv("GITHUB_COPILOT_TOKEN", tokenData.AccessToken)
					logging.Info("Saved GitHub token and set environment variable for immediate use")
					
					// Direct exchange - don't call exchangeGitHubToken to avoid potential loop
					logging.Debug("Performing direct token exchange for GitHub token")
					
					// Create the request to exchange for a Copilot bearer token - use internal API
					req, err := http.NewRequest("GET", "https://api.github.com/copilot_internal/v2/token", nil)
					if err != nil {
						fmt.Printf("❌ Error creating exchange request: %v\n", err)
						return "", fmt.Errorf("failed to create exchange request: %w", err)
					}
					
					req.Header.Set("Authorization", "token "+tokenData.AccessToken) // lowercase "token"
					req.Header.Set("User-Agent", "GithubCopilot/1.133.0")
					req.Header.Set("Accept", "application/json")
					
					fmt.Printf("🔄 Requesting Copilot token from GitHub API...\n")
					resp, err := c.httpClient.Do(req)
					if err != nil {
						fmt.Printf("❌ Exchange request failed: %v\n", err)
						return "", fmt.Errorf("failed exchange request: %w", err)
					}
					defer resp.Body.Close()
					
					fmt.Printf("📊 Exchange response status: %d\n", resp.StatusCode)
					
					// Check for successful response
					if resp.StatusCode != http.StatusOK {
						body, _ := io.ReadAll(resp.Body)
						fmt.Printf("❌ Token exchange failed: %s\n", string(body))
						return "", fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
					}
					
					// Read the response body
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						fmt.Printf("❌ Failed to read exchange response: %v\n", err)
						return "", fmt.Errorf("failed to read response: %w", err)
					}
					
					// Parse the token response
					var tokenResp CopilotTokenResponse
					if err := json.Unmarshal(body, &tokenResp); err != nil {
						fmt.Printf("❌ Failed to parse exchange response: %v\n", err)
						return "", fmt.Errorf("failed to parse response: %w", err)
					}
					
					if tokenResp.Token == "" {
						fmt.Printf("❌ Received empty token from GitHub API\n")
						return "", fmt.Errorf("received empty token from GitHub API")
					}
					
					// Store the token for future use
					c.options.bearerToken = tokenResp.Token
					
					// Create a new OpenAI client specifically for Copilot with the bearer token
					baseURL := "https://api.githubcopilot.com"
					newClient := openai.NewClient(
						option.WithBaseURL(baseURL),
						option.WithAPIKey(tokenResp.Token),
						option.WithHeader("Editor-Version", "OpenCode/1.0"),
						option.WithHeader("Editor-Plugin-Version", "OpenCode/1.0"),
						option.WithHeader("Copilot-Integration-Id", "vscode-chat"),
						option.WithHeader("X-GitHub-Api-Version", "2022-11-28"),
					)
					
					// Replace the client in the current instance
					c.client = newClient
					
					fmt.Printf("✅ Successfully exchanged token for Copilot bearer token!\n")
					fmt.Printf("✅ Created new OpenAI client for GitHub Copilot\n")
					fmt.Printf("✅ You can now use OpenCode with GitHub Copilot\n")
					return tokenResp.Token, nil
				}
			} else if tokenResp.StatusCode == http.StatusBadRequest {
				// If it's still pending, continue polling
				var errorResp map[string]string
				if err := json.Unmarshal(tokenRespBody, &errorResp); err == nil {
					if errorResp["error"] == "authorization_pending" {
						// This is normal, just wait for next poll
						fmt.Printf("⏳ Authorization pending - waiting for you to approve in the browser...\n")
						continue
					} else if errorResp["error"] == "slow_down" {
						// Need to slow down polling
						interval += 5
						ticker.Reset(time.Duration(interval) * time.Second)
						fmt.Printf("⚠️ GitHub asked us to slow down polling. Increased interval to %d seconds.\n", interval)
						continue
					} else if errorResp["error"] == "expired_token" {
						fmt.Printf("❌ Device code expired. Please try again.\n")
						return "", fmt.Errorf("device code expired, please try again")
					} else {
						// Unknown error
						fmt.Printf("❓ Unknown error from GitHub: %s\n", errorResp["error"])
						fmt.Printf("❓ Error details: %s\n", string(tokenRespBody))
					}
				} else {
					// Error parsing JSON
					fmt.Printf("❌ Error parsing response: %v\n", err)
					fmt.Printf("❌ Raw response: %s\n", string(tokenRespBody))
				}
				return "", fmt.Errorf("token request failed with status %d: %s", 
					tokenResp.StatusCode, string(tokenRespBody))
			} else {
				return "", fmt.Errorf("token request failed with status %d: %s", 
					tokenResp.StatusCode, string(tokenRespBody))
			}

		case <-ctx.Done():
			return "", fmt.Errorf("authentication timed out after %d seconds", deviceResp.ExpiresIn)
		}
	}
}

// newCopilotClient creates a new client for GitHub Copilot
// Following the 4-step flow:
// 1. Check if Copilot is enabled in config (handled by validation)
// 2. Check for token in config folder
// 3. If no token, trigger login flow
// 4. With token ready, open OpenCode normally
func newCopilotClient(opts providerClientOptions) CopilotClient {
	logging.Debug("Creating new Copilot client", "model", opts.model)
	fmt.Printf("🔧 Creating new GitHub Copilot client for model: %s\n", opts.model.ID)
	
	copilotOpts := copilotOptions{
		reasoningEffort: "medium",
	}
	// Apply copilot-specific options
	for _, o := range opts.copilotOptions {
		o(&copilotOpts)
	}

	// Create HTTP client for token exchange
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	var bearerToken string

	// Step 1: Check if Copilot is enabled in config - already done by validation

	// Step 2: Check for token in config folder
	var githubToken string
	var useDeviceFlow bool
	
	logging.Info("Looking for GitHub Copilot token")
	fmt.Printf("🔍 Looking for GitHub Copilot authentication token...\n")
	
	// If bearer token is already provided, use it
	if copilotOpts.bearerToken != "" {
		logging.Debug("Using provided bearer token")
		fmt.Printf("✅ Using provided bearer token\n")
		bearerToken = copilotOpts.bearerToken
	} else {
		// Check for GitHub token in standard locations
		var err error
		logging.Debug("Checking for GitHub token in standard locations")
		githubToken, err = config.LoadGitHubToken()
		
		if err != nil {
			if err.Error() == "no_copilot_token" {
				// Special error indicating we need device flow
				useDeviceFlow = true
				logging.Info("No Copilot token found in config. Need to use device flow.")
				fmt.Printf("ℹ️ No GitHub Copilot token found in config\n")
			} else {
				logging.Error("Failed to load GitHub token", "error", err)
				fmt.Printf("❌ Error loading GitHub token: %v\n", err)
			}
		} else if githubToken != "" {
			prefixLen := 10
			if len(githubToken) < prefixLen {
				prefixLen = len(githubToken)
			}
			logging.Debug("Found GitHub token in config", "token_length", len(githubToken), "token_prefix", githubToken[:prefixLen])
			fmt.Printf("✅ Found GitHub token in config\n")
		} else {
			logging.Debug("GitHub token not found in config")
			fmt.Printf("ℹ️ No GitHub token found in config\n")
			useDeviceFlow = true
		}
		
		// Step 3: If no token, trigger login flow
		nonInteractiveFlag := viper.GetBool("non_interactive")
		cliNonInteractive := viper.GetString("prompt") != ""
		
		if useDeviceFlow && !nonInteractiveFlag && !cliNonInteractive {
			logging.Info("Starting GitHub Copilot authentication flow")
			fmt.Printf("🔑 Starting GitHub Copilot authentication flow\n")
			
			// Create temporary client for auth flow
			tempClient := &copilotClient{
				providerOptions: opts,
				options:         copilotOpts,
				httpClient:      httpClient,
			}
			
			// Use device code flow to get token
			var err error
			githubToken, err = tempClient.performDeviceCodeFlow()
			if err != nil {
				logging.Error("Device code authentication failed", "error", err)
				fmt.Printf("❌ Authentication failed: %v\n", err)
				
				// Return dummy client so app doesn't crash
				return createDummyClient(opts, copilotOpts, httpClient)
			}
		} else if useDeviceFlow {
			// Can't do auth flow in non-interactive mode
			logging.Error("Authentication required but running in non-interactive mode")
			fmt.Printf("❌ Authentication required but running in non-interactive mode\n")
			fmt.Printf("⚠️ Run OpenCode in interactive mode first to authenticate with Copilot\n")
			
			// Return dummy client
			return createDummyClient(opts, copilotOpts, httpClient)
		}
		
		// If we have a GitHub token but no bearer token, exchange for bearer token
		if githubToken != "" {
			// Create temporary client for token exchange
			tempClient := &copilotClient{
				providerOptions: opts,
				options:         copilotOpts,
				httpClient:      httpClient,
			}
			
			// Exchange GitHub token for bearer token
			var err error
			logging.Debug("Exchanging GitHub token for Copilot bearer token")
			fmt.Printf("🔄 Exchanging GitHub token for Copilot bearer token...\n")
			
			bearerToken, err = tempClient.exchangeGitHubToken(githubToken)
			if err != nil {
				logging.Error("Failed to exchange GitHub token", "error", err)
				fmt.Printf("❌ Failed to exchange GitHub token: %v\n", err)
				
				// Return dummy client
				return createDummyClient(opts, copilotOpts, httpClient)
			}
		} else {
			// No GitHub token and can't trigger auth flow
			logging.Error("No GitHub token available and cannot trigger authentication")
			fmt.Printf("❌ No GitHub token available and cannot trigger authentication\n")
			
			// Return dummy client
			return createDummyClient(opts, copilotOpts, httpClient)
		}
	}

	copilotOpts.bearerToken = bearerToken

	// Step 4: With token ready, create client and proceed with normal operation
	return createCopilotClient(opts, copilotOpts, httpClient, bearerToken)
}

// createDummyClient creates a placeholder client when authentication fails
func createDummyClient(opts providerClientOptions, options copilotOptions, httpClient *http.Client) *copilotClient {
	logging.Debug("Creating dummy Copilot client due to authentication issues")
	fmt.Printf("ℹ️ Creating temporary client due to authentication issues\n")
	
	dummyClient := openai.NewClient(
		option.WithBaseURL("https://api.githubcopilot.com"),
		option.WithAPIKey("dummy-for-initialization"),
	)
	
	return &copilotClient{
		providerOptions: opts,
		options:         options,
		client:          dummyClient,
		httpClient:      httpClient,
	}
}

// createCopilotClient creates a fully configured client for GitHub Copilot
func createCopilotClient(opts providerClientOptions, options copilotOptions, httpClient *http.Client, bearerToken string) *copilotClient {
	// GitHub Copilot API base URL
	baseURL := "https://api.githubcopilot.com"
	customURL := viper.GetString("providers.copilot.baseUrl")
	if customURL != "" {
		logging.Debug("Using custom baseUrl for Copilot from config", "baseUrl", customURL)
		fmt.Printf("🌐 Using custom baseUrl for Copilot: %s\n", customURL)
		baseURL = customURL
	}
	
	// Make sure baseURL is set
	if baseURL == "" {
		logging.Error("Missing baseURL for Copilot client")
		fmt.Printf("❌ Missing baseURL for Copilot client\n")
		return createDummyClient(opts, options, httpClient)
	}
	
	// If no bearer token, return dummy client
	if bearerToken == "" {
		logging.Error("No bearer token available for Copilot client")
		return createDummyClient(opts, options, httpClient)
	}

	// Create the proper client with all required options and headers
	prefixLen := 10
	if len(bearerToken) < prefixLen {
		prefixLen = len(bearerToken)
	}
	logging.Debug("Creating Copilot client with valid bearer token", 
		"baseURL", baseURL, 
		"model", opts.model.APIModel, 
		"token_length", len(bearerToken),
		"bearerToken_prefix", bearerToken[:prefixLen])
	fmt.Printf("✅ Creating Copilot client with valid bearer token\n")
	
	// Create OpenAI client with all required settings for Copilot
	openaiClientOptions := []option.RequestOption{
		option.WithBaseURL(baseURL),
		option.WithAPIKey(bearerToken), // Use bearer token as API key
		option.WithHeader("User-Agent", "GithubCopilot/1.133.0"),
		option.WithHeader("Editor-Version", "vscode/1.78.0"),
		option.WithHeader("Editor-Plugin-Version", "copilot-chat/0.8.0"),
		option.WithHeader("Accept", "application/json"),
		option.WithHeader("X-GitHub-Api-Version", "2022-11-28"), // Required GitHub API version
	}

	// Add any extra headers from options
	if options.extraHeaders != nil {
		for key, value := range options.extraHeaders {
			openaiClientOptions = append(openaiClientOptions, option.WithHeader(key, value))
		}
	}

	// Create client with proper headers
	client := openai.NewClient(openaiClientOptions...)
	fmt.Printf("✅ GitHub Copilot client created successfully\n")
	fmt.Printf("✅ Using model: %s (%s)\n", opts.model.Name, opts.model.APIModel)
	
	// Create and return the copilotClient
	return &copilotClient{
		providerOptions: opts,
		options: copilotOptions{
			reasoningEffort: options.reasoningEffort,
			extraHeaders: options.extraHeaders,
			bearerToken: bearerToken,
		},
		client:          client,
		httpClient:      httpClient,
	}
}

func (c *copilotClient) convertMessages(messages []message.Message) (copilotMessages []openai.ChatCompletionMessageParamUnion) {
	// Add system message first
	copilotMessages = append(copilotMessages, openai.SystemMessage(c.providerOptions.systemMessage))

	for _, msg := range messages {
		switch msg.Role {
		case message.User:
			var content []openai.ChatCompletionContentPartUnionParam
			textBlock := openai.ChatCompletionContentPartTextParam{Text: msg.Content().String()}
			content = append(content, openai.ChatCompletionContentPartUnionParam{OfText: &textBlock})

			for _, binaryContent := range msg.BinaryContent() {
				imageURL := openai.ChatCompletionContentPartImageImageURLParam{URL: binaryContent.String(models.ProviderCopilot)}
				imageBlock := openai.ChatCompletionContentPartImageParam{ImageURL: imageURL}
				content = append(content, openai.ChatCompletionContentPartUnionParam{OfImageURL: &imageBlock})
			}

			copilotMessages = append(copilotMessages, openai.UserMessage(content))

		case message.Assistant:
			assistantMsg := openai.ChatCompletionAssistantMessageParam{
				Role: "assistant",
			}

			if msg.Content().String() != "" {
				assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(msg.Content().String()),
				}
			}

			if len(msg.ToolCalls()) > 0 {
				assistantMsg.ToolCalls = make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls()))
				for i, call := range msg.ToolCalls() {
					assistantMsg.ToolCalls[i] = openai.ChatCompletionMessageToolCallParam{
						ID:   call.ID,
						Type: "function",
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      call.Name,
							Arguments: call.Input,
						},
					}
				}
			}

			copilotMessages = append(copilotMessages, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &assistantMsg,
			})

		case message.Tool:
			for _, result := range msg.ToolResults() {
				copilotMessages = append(copilotMessages,
					openai.ToolMessage(result.Content, result.ToolCallID),
				)
			}
		}
	}

	return
}

func (c *copilotClient) convertTools(tools []toolsPkg.BaseTool) []openai.ChatCompletionToolParam {
	copilotTools := make([]openai.ChatCompletionToolParam, len(tools))

	for i, tool := range tools {
		info := tool.Info()
		copilotTools[i] = openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        info.Name,
				Description: openai.String(info.Description),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": info.Parameters,
					"required":   info.Required,
				},
			},
		}
	}

	return copilotTools
}

func (c *copilotClient) finishReason(reason string) message.FinishReason {
	switch reason {
	case "stop":
		return message.FinishReasonEndTurn
	case "length":
		return message.FinishReasonMaxTokens
	case "tool_calls":
		return message.FinishReasonToolUse
	default:
		return message.FinishReasonUnknown
	}
}

func (c *copilotClient) preparedParams(messages []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolParam) openai.ChatCompletionNewParams {
	logging.Debug("Copilot preparedParams start", "modelID", c.providerOptions.model.ID, "apiModel", c.providerOptions.model.APIModel)
	
	// For Claude models, use the proper model name format
	apiModel := c.providerOptions.model.APIModel
	if c.isAnthropicModel() {
		logging.Debug("Using Claude model", "original_api_model", apiModel)
		fmt.Printf("📢 Using Claude model: %s through GitHub Copilot\n", apiModel)
		// The Claude models might need a special format for Copilot
		if strings.HasPrefix(apiModel, "claude-") {
			logging.Debug("Using Claude model with standard format", "api_model", apiModel)
			fmt.Printf("📢 Claude model name format: %s\n", apiModel)
		}
	} else {
		fmt.Printf("📢 Using non-Claude model: %s through GitHub Copilot\n", apiModel)
	}
	
	// Log important model details
	logging.Debug("Model details", 
		"name", c.providerOptions.model.Name,
		"context_window", c.providerOptions.model.ContextWindow,
		"max_tokens", c.providerOptions.maxTokens)
	
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(apiModel),
		Messages: messages,
		Tools:    tools,
	}

	if c.providerOptions.model.CanReason == true {
		params.MaxCompletionTokens = openai.Int(c.providerOptions.maxTokens)
		switch c.options.reasoningEffort {
		case "low":
			params.ReasoningEffort = shared.ReasoningEffortLow
		case "medium":
			params.ReasoningEffort = shared.ReasoningEffortMedium
		case "high":
			params.ReasoningEffort = shared.ReasoningEffortHigh
		default:
			params.ReasoningEffort = shared.ReasoningEffortMedium
		}
	} else {
		params.MaxTokens = openai.Int(c.providerOptions.maxTokens)
	}

	jsonData, err := json.Marshal(params)
	if err == nil {
		logging.Debug("Copilot request parameters", "params", string(jsonData))
	}
	return params
}

func (c *copilotClient) send(ctx context.Context, messages []message.Message, tools []toolsPkg.BaseTool) (response *ProviderResponse, err error) {
	params := c.preparedParams(c.convertMessages(messages), c.convertTools(tools))
	cfg := config.Get()
	var sessionId string
	requestSeqId := (len(messages) + 1) / 2
	
	// Always log parameters in debug mode
	jsonData, _ := json.Marshal(params)
	logging.Debug("Copilot API request parameters", "model", params.Model, "messages_count", len(params.Messages), "tools_count", len(params.Tools))
	logging.Debug("Copilot API full request", "params", string(jsonData))
	logging.Debug("Model being used for request", "model_id", c.providerOptions.model.ID, "api_model", c.providerOptions.model.APIModel)
	
	if cfg.Debug {
		if sid, ok := ctx.Value(toolsPkg.SessionIDContextKey).(string); ok {
			sessionId = sid
		}
		if sessionId != "" {
			filepath := logging.WriteRequestMessageJson(sessionId, requestSeqId, params)
			logging.Debug("Prepared messages", "filepath", filepath)
		}
	}

	attempts := 0
	for {
		attempts++
		fmt.Printf("🔄 Sending request to GitHub Copilot API...\n")
		logging.Debug("About to send Copilot API request", "model", string(params.Model), "max_tokens_param", params.MaxTokens, "max_completion_tokens_param", params.MaxCompletionTokens, "tools_count", len(tools))
		logging.Debug("Sending request to Copilot API", "baseURL", "https://api.githubcopilot.com", "model", params.Model)
		
		// Dump headers for debugging
		logging.Debug("Request is being made with OpenAI client", "client_type", fmt.Sprintf("%T", c.client))
		fmt.Printf("📋 Making request with client: %T\n", c.client)
		
		// Dump important request parameters
		fmt.Printf("📋 Request model: %s\n", params.Model)
		fmt.Printf("📋 Request max tokens: %d\n", c.providerOptions.maxTokens)
		
		// Make the API request
		copilotResponse, err := c.client.Chat.Completions.New(
			ctx,
			params,
		)
		logging.Debug("Received response from Copilot API", "error", err != nil)

		// If there is an error we are going to see if we can retry the call
		if err != nil {
			fmt.Printf("❌ Copilot API request failed: %v\n", err)
			logging.Error("Copilot API request failed", "error", err)
			var apierr *openai.Error
			if errors.As(err, &apierr) {
				fmt.Printf("❌ API Error: Status %d, Type %s\n", apierr.StatusCode, apierr.Type)
				fmt.Printf("❌ Error Response: %s\n", apierr.RawJSON())
				logging.Error("Copilot API error details", "status", apierr.StatusCode, "type", apierr.Type, "raw_json", apierr.RawJSON())
				
				// Check if this is a model not found error
				if apierr.StatusCode == 400 {
					if strings.Contains(string(apierr.RawJSON()), "model") {
						fmt.Printf("⚠️ This might be because the model '%s' is not available or not supported by GitHub Copilot.\n", params.Model)
						fmt.Printf("⚠️ Automatically trying 'copilot.gpt-4o' instead...\n")
						
						// If this was a Claude model, try with GPT-4o as fallback
						if c.isAnthropicModel() && attempts < 2 {
							logging.Info("Trying with GPT-4o model instead of Claude")
							params.Model = "gpt-4o"
							continue
						} else {
							fmt.Printf("⚠️ Try manually changing your config to use 'copilot.gpt-4o' instead.\n")
						}
					}
				}
			}
			
			retry, after, retryErr := c.shouldRetry(attempts, err)
			if retryErr != nil {
				fmt.Printf("❌ Cannot retry: %v\n", retryErr)
				logging.Error("Retry error", "error", retryErr)
				return nil, retryErr
			}
			if retry {
				fmt.Printf("⏳ Retrying in %d ms (attempt %d of %d)...\n", after, attempts, maxRetries)
				logging.WarnPersist(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d", attempts, maxRetries), logging.PersistTimeArg, time.Millisecond*time.Duration(after+100))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			return nil, retryErr
		} else {
			fmt.Printf("✅ Successful response from GitHub Copilot API!\n")
		}

		content := ""
		if copilotResponse.Choices[0].Message.Content != "" {
			content = copilotResponse.Choices[0].Message.Content
		}

		toolCalls := c.toolCalls(*copilotResponse)
		finishReason := c.finishReason(string(copilotResponse.Choices[0].FinishReason))

		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		return &ProviderResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			Usage:        c.usage(*copilotResponse),
			FinishReason: finishReason,
		}, nil
	}
}

func (c *copilotClient) stream(ctx context.Context, messages []message.Message, tools []toolsPkg.BaseTool) <-chan ProviderEvent {
	params := c.preparedParams(c.convertMessages(messages), c.convertTools(tools))
	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}

	cfg := config.Get()
	var sessionId string
	requestSeqId := (len(messages) + 1) / 2
	if cfg.Debug {
		if sid, ok := ctx.Value(toolsPkg.SessionIDContextKey).(string); ok {
			sessionId = sid
		}
		jsonData, _ := json.Marshal(params)
		if sessionId != "" {
			filepath := logging.WriteRequestMessageJson(sessionId, requestSeqId, params)
			logging.Debug("Prepared messages", "filepath", filepath)
		} else {
			logging.Debug("Prepared messages", "messages", string(jsonData))
		}

	}

	attempts := 0
	eventChan := make(chan ProviderEvent)

	go func() {
		for {
			attempts++
			copilotStream := c.client.Chat.Completions.NewStreaming(
				ctx,
				params,
			)

			acc := openai.ChatCompletionAccumulator{}
			currentContent := ""
			toolCalls := make([]message.ToolCall, 0)

			var currentToolCallId string
			var currentToolCall openai.ChatCompletionMessageToolCall
			var msgToolCalls []openai.ChatCompletionMessageToolCall
			for copilotStream.Next() {
				chunk := copilotStream.Current()
				acc.AddChunk(chunk)

				if cfg.Debug {
					logging.AppendToStreamSessionLogJson(sessionId, requestSeqId, chunk)
				}

				for _, choice := range chunk.Choices {
					if choice.Delta.Content != "" {
						eventChan <- ProviderEvent{
							Type:    EventContentDelta,
							Content: choice.Delta.Content,
						}
						currentContent += choice.Delta.Content
					}
				}

				if c.isAnthropicModel() {
					// Monkeypatch adapter for Sonnet-4 multi-tool use
					for _, choice := range chunk.Choices {
						if choice.Delta.ToolCalls != nil && len(choice.Delta.ToolCalls) > 0 {
							toolCall := choice.Delta.ToolCalls[0]
							// Detect tool use start
							if currentToolCallId == "" {
								if toolCall.ID != "" {
									currentToolCallId = toolCall.ID
									currentToolCall = openai.ChatCompletionMessageToolCall{
										ID:   toolCall.ID,
										Type: "function",
										Function: openai.ChatCompletionMessageToolCallFunction{
											Name:      toolCall.Function.Name,
											Arguments: toolCall.Function.Arguments,
										},
									}
								}
							} else {
								// Delta tool use
								if toolCall.ID == "" {
									currentToolCall.Function.Arguments += toolCall.Function.Arguments
								} else {
									// Detect new tool use
									if toolCall.ID != currentToolCallId {
										msgToolCalls = append(msgToolCalls, currentToolCall)
										currentToolCallId = toolCall.ID
										currentToolCall = openai.ChatCompletionMessageToolCall{
											ID:   toolCall.ID,
											Type: "function",
											Function: openai.ChatCompletionMessageToolCallFunction{
												Name:      toolCall.Function.Name,
												Arguments: toolCall.Function.Arguments,
											},
										}
									}
								}
							}
						}
						if choice.FinishReason == "tool_calls" {
							msgToolCalls = append(msgToolCalls, currentToolCall)
							acc.ChatCompletion.Choices[0].Message.ToolCalls = msgToolCalls
						}
					}
				}
			}

			err := copilotStream.Err()
			if err == nil || errors.Is(err, io.EOF) {
				if cfg.Debug {
					respFilepath := logging.WriteChatResponseJson(sessionId, requestSeqId, acc.ChatCompletion)
					logging.Debug("Chat completion response", "filepath", respFilepath)
				}
				// Stream completed successfully
				finishReason := c.finishReason(string(acc.ChatCompletion.Choices[0].FinishReason))
				if len(acc.ChatCompletion.Choices[0].Message.ToolCalls) > 0 {
					toolCalls = append(toolCalls, c.toolCalls(acc.ChatCompletion)...)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
				}

				eventChan <- ProviderEvent{
					Type: EventComplete,
					Response: &ProviderResponse{
						Content:      currentContent,
						ToolCalls:    toolCalls,
						Usage:        c.usage(acc.ChatCompletion),
						FinishReason: finishReason,
					},
				}
				close(eventChan)
				return
			}

			// If there is an error we are going to see if we can retry the call
			retry, after, retryErr := c.shouldRetry(attempts, err)
			if retryErr != nil {
				eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
				close(eventChan)
				return
			}
			// shouldRetry is not catching the max retries...
			// TODO: Figure out why
			if attempts > maxRetries {
				logging.Warn("Maximum retry attempts reached for rate limit", "attempts", attempts, "max_retries", maxRetries)
				retry = false
			}
			if retry {
				logging.WarnPersist(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d (paused for %d ms)", attempts, maxRetries, after), logging.PersistTimeArg, time.Millisecond*time.Duration(after+100))
				select {
				case <-ctx.Done():
					// context cancelled
					if ctx.Err() == nil {
						eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
					}
					close(eventChan)
					return
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
			close(eventChan)
			return
		}
	}()

	return eventChan
}

func (c *copilotClient) shouldRetry(attempts int, err error) (bool, int64, error) {
	var apierr *openai.Error
	if !errors.As(err, &apierr) {
		return false, 0, err
	}

	// Check for token expiration (401 Unauthorized)
	if apierr.StatusCode == 401 {
		// Try to refresh the bearer token
		var githubToken string

		// 1. Environment variable
		githubToken = os.Getenv("GITHUB_TOKEN")

		// 2. API key from options
		if githubToken == "" {
			githubToken = c.providerOptions.apiKey
		}

		// 3. Standard GitHub CLI/Copilot locations
		if githubToken == "" {
			var err error
			githubToken, err = config.LoadGitHubToken()
			if err != nil {
				logging.Debug("Failed to load GitHub token from standard locations during retry", "error", err)
			}
		}

		if githubToken != "" {
			newBearerToken, tokenErr := c.exchangeGitHubToken(githubToken)
			if tokenErr == nil {
				c.options.bearerToken = newBearerToken
				// Update the client with the new token
				// Note: This is a simplified approach. In a production system,
				// you might want to recreate the entire client with the new token
				logging.Info("Refreshed Copilot bearer token")
				
				// Recreate the entire client with the new token
				baseURL := "https://api.githubcopilot.com"
				c.client = openai.NewClient(
					option.WithBaseURL(baseURL),
					option.WithAPIKey(newBearerToken),
					option.WithHeader("Editor-Version", "OpenCode/1.0"),
					option.WithHeader("Editor-Plugin-Version", "OpenCode/1.0"),
					option.WithHeader("Copilot-Integration-Id", "vscode-chat"),
				)
				
				return true, 1000, nil // Retry immediately with new token
			}
			logging.Error("Failed to refresh Copilot bearer token", "error", tokenErr)
		}
		return false, 0, fmt.Errorf("authentication failed: %w", err)
	}
	logging.Debug("Copilot API Error", "status", apierr.StatusCode, "headers", apierr.Response.Header, "body", apierr.RawJSON())

	if apierr.StatusCode == 400 {
		// Special handling for 400 Bad Request
		logging.Error("Copilot API 400 Bad Request error", "error", err.Error(), "response_body", apierr.RawJSON())
		
		// Try to extract more details from the error
		detailedErr := fmt.Errorf("Copilot API 400 Bad Request: %s", apierr.Error())
		return false, 0, detailedErr
	} else if apierr.StatusCode != 429 && apierr.StatusCode != 500 {
		return false, 0, err
	}

	if apierr.StatusCode == 500 {
		logging.Warn("Copilot API returned 500 error, retrying", "error", err)
	}

	if attempts > maxRetries {
		return false, 0, fmt.Errorf("maximum retry attempts reached for rate limit: %d retries", maxRetries)
	}

	retryMs := 0
	retryAfterValues := apierr.Response.Header.Values("Retry-After")

	backoffMs := 2000 * (1 << (attempts - 1))
	jitterMs := int(float64(backoffMs) * 0.2)
	retryMs = backoffMs + jitterMs
	if len(retryAfterValues) > 0 {
		if _, err := fmt.Sscanf(retryAfterValues[0], "%d", &retryMs); err == nil {
			retryMs = retryMs * 1000
		}
	}
	return true, int64(retryMs), nil
}

func (c *copilotClient) toolCalls(completion openai.ChatCompletion) []message.ToolCall {
	var toolCalls []message.ToolCall

	if len(completion.Choices) > 0 && len(completion.Choices[0].Message.ToolCalls) > 0 {
		for _, call := range completion.Choices[0].Message.ToolCalls {
			toolCall := message.ToolCall{
				ID:       call.ID,
				Name:     call.Function.Name,
				Input:    call.Function.Arguments,
				Type:     "function",
				Finished: true,
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	return toolCalls
}

func (c *copilotClient) usage(completion openai.ChatCompletion) TokenUsage {
	cachedTokens := completion.Usage.PromptTokensDetails.CachedTokens
	inputTokens := completion.Usage.PromptTokens - cachedTokens

	return TokenUsage{
		InputTokens:         inputTokens,
		OutputTokens:        completion.Usage.CompletionTokens,
		CacheCreationTokens: 0, // GitHub Copilot doesn't provide this directly
		CacheReadTokens:     cachedTokens,
	}
}

func WithCopilotReasoningEffort(effort string) CopilotOption {
	return func(options *copilotOptions) {
		defaultReasoningEffort := "medium"
		switch effort {
		case "low", "medium", "high":
			defaultReasoningEffort = effort
		default:
			logging.Warn("Invalid reasoning effort, using default: medium")
		}
		options.reasoningEffort = defaultReasoningEffort
	}
}

func WithCopilotExtraHeaders(headers map[string]string) CopilotOption {
	return func(options *copilotOptions) {
		options.extraHeaders = headers
	}
}

func WithCopilotBearerToken(bearerToken string) CopilotOption {
	return func(options *copilotOptions) {
		options.bearerToken = bearerToken
	}
}