package router

import (
	"context"
	"errors"

	"golang.org/x/oauth2"

	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/util"
)

func getGoogleCalDAVTokenSource(userID, publicBase string) (oauth2.TokenSource, string, error) {
	refreshEnc, err := GetUserPreferencesRepository().Get(userID, PreferenceCalDAVOAuthRefresh.Name)
	if err != nil || refreshEnc == "" {
		return nil, "", errors.New("google caldav not connected")
	}
	email, err := GetUserPreferencesRepository().Get(userID, PreferenceCalDAVGoogleEmail.Name)
	if err != nil || email == "" {
		return nil, "", errors.New("google caldav email missing")
	}
	refresh, err := DecryptString(refreshEnc)
	if err != nil || refresh == "" {
		return nil, "", errors.New("google caldav refresh token invalid")
	}
	oauthCfg := NewGoogleCalDAVOAuthConfig(GoogleCalDAVRedirectURL(publicBase))
	token := &oauth2.Token{RefreshToken: refresh}
	return oauthCfg.TokenSource(context.Background(), token), email, nil
}

func getUserCalDAVProvider(userID string) string {
	provider, err := GetUserPreferencesRepository().Get(userID, PreferenceCalDAVProvider.Name)
	if err != nil || provider == "" {
		return CalDAVProviderGeneric
	}
	return provider
}
