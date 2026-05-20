package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/seatsurfing/seatsurfing/server/config"
)

const GoogleCalendarScope = "https://www.googleapis.com/auth/calendar"

// Required for https://www.googleapis.com/oauth2/v3/userinfo — calendar scope alone returns 401.
const GoogleUserinfoEmailScope = "https://www.googleapis.com/auth/userinfo.email"

// GoogleCalDAVRootURL is the CalDAV v2 discovery root. With OAuth, the account is
// determined by the bearer token; appending an email to the path breaks
// current-user-principal (404 from Google).
const GoogleCalDAVRootURL = "https://apidata.googleusercontent.com/caldav/v2"

// GoogleCalDAVPrincipalURL returns the CalDAV endpoint for the given Google account.
// email is ignored for the URL: Google resolves the principal from the OAuth token.
func GoogleCalDAVPrincipalURL(email string) string {
	_ = email
	return GoogleCalDAVRootURL
}

func NewGoogleCalDAVOAuthConfig(redirectURL string) *oauth2.Config {
	cfg := config.GetConfig()
	return &oauth2.Config{
		ClientID:     cfg.GoogleCalDAVClientID,
		ClientSecret: cfg.GoogleCalDAVClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{GoogleCalendarScope, GoogleUserinfoEmailScope},
		Endpoint:     google.Endpoint,
	}
}

func GoogleCalDAVRedirectURL(publicBase string) string {
	return publicBase + "/preference/caldav/google/callback"
}

func GetRequestPublicBase(r *http.Request) string {
	cfg := config.GetConfig()
	scheme := cfg.PublicScheme
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		host = fwd
	}
	return scheme + "://" + host
}

type googleUserInfo struct {
	Email string `json:"email"`
}

func FetchGoogleAccountEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo request failed: %s", resp.Status)
	}
	var info googleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return "", err
	}
	if info.Email == "" {
		return "", fmt.Errorf("userinfo response missing email")
	}
	return info.Email, nil
}
