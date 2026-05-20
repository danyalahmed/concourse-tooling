package resource

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	gitHubAPIBase = "https://api.github.com"
	requestTimeout = 10 * time.Second
	jwtExpiry      = 10 * time.Minute
)

type GitHubClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{Timeout: requestTimeout},
		baseURL:    gitHubAPIBase,
	}
}

func (c *GitHubClient) GenerateInstallationToken(src Source) (string, error) {
	key, err := parsePrivateKey([]byte(src.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("parsing private key: %w", err)
	}

	jwtToken, err := c.generateJWT(src.AppID, key)
	if err != nil {
		return "", fmt.Errorf("generating JWT: %w", err)
	}

	return c.exchangeJWT(jwtToken, src.InstallationID)
}

func parsePrivateKey(pemData []byte) (any, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in private key")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}

	key2, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err2 == nil {
		return key2, nil
	}

	return nil, fmt.Errorf("PKCS1: %w; PKCS8: %w", err, err2)
}

func (c *GitHubClient) generateJWT(appID string, privateKey any) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(jwtExpiry).Unix(),
		"iss": appID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func (c *GitHubClient) exchangeJWT(jwtToken, installationID string) (string, error) {
	url := fmt.Sprintf("%s/app/installations/%s/access_tokens", c.baseURL, installationID)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling GitHub API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	return result.Token, nil
}
