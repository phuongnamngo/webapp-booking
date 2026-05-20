package router

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/seatsurfing/seatsurfing/server/config"
	"golang.org/x/oauth2"

	. "github.com/seatsurfing/seatsurfing/server/api"
	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/util"
)

type UserPreferencesRouter struct {
}

type ListCaldavCalendarsRequest struct {
	Provider string `json:"provider"`
	URL      string `json:"url" validate:"omitempty,url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ListCaldavCalendarsResponse struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type CaldavGoogleAuthURLResponse struct {
	URL string `json:"url"`
}

func redirectCaldavGoogleOAuthUI(w http.ResponseWriter, publicBase string, extra url.Values) {
	q := url.Values{}
	q.Set("tab", "integrations")
	for k, vals := range extra {
		for _, v := range vals {
			q.Add(k, v)
		}
	}
	SendTemporaryRedirect(w, publicBase+"/ui/preferences/?"+q.Encode())
}

func (router *UserPreferencesRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/caldav/google/auth-url", router.caldavGoogleAuthURL).Methods("GET")
	s.HandleFunc("/caldav/google/callback", router.caldavGoogleCallback).Methods("GET")
	s.HandleFunc("/caldav/listCalendars", router.caldavListCalendars).Methods("POST")
	s.HandleFunc("/{name}", router.getPreference).Methods("GET")
	s.HandleFunc("/{name}", router.setPreference).Methods("PUT")
	s.HandleFunc("/", router.getAll).Methods("GET")
	s.HandleFunc("/", router.setAll).Methods("PUT")
}

func (router *UserPreferencesRouter) caldavGoogleAuthURL(w http.ResponseWriter, r *http.Request) {
	if !CanCrypt() {
		log.Println("Error: CalDAV integration requires a valid crypt key (CRYPT_KEY).")
		SendInternalServerError(w)
		return
	}
	cfg := config.GetConfig()
	if cfg.GoogleCalDAVClientID == "" || cfg.GoogleCalDAVClientSecret == "" {
		SendServiceUnavailable(w)
		return
	}
	user := GetRequestUser(r)
	publicBase := GetRequestPublicBase(r)
	redirectURL := GoogleCalDAVRedirectURL(publicBase)
	authState := &AuthState{
		AuthProviderID: user.ID,
		Expiry:         time.Now().Add(10 * time.Minute),
		AuthStateType:  AuthCalDAVGoogleOAuth,
	}
	if err := GetAuthStateRepository().Create(authState); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	oauthCfg := NewGoogleCalDAVOAuthConfig(redirectURL)
	SendJSON(w, &CaldavGoogleAuthURLResponse{
		URL: oauthCfg.AuthCodeURL(authState.ID, oauth2.AccessTypeOffline, oauth2.ApprovalForce),
	})
}

func (router *UserPreferencesRouter) caldavGoogleCallback(w http.ResponseWriter, r *http.Request) {
	publicBase := GetRequestPublicBase(r)
	if !CanCrypt() {
		log.Println("Error: CalDAV integration requires a valid crypt key (CRYPT_KEY).")
		redirectCaldavGoogleOAuthUI(w, publicBase, url.Values{"caldav_oauth_error": {"crypt_key"}})
		return
	}
	cfg := config.GetConfig()
	if cfg.GoogleCalDAVClientID == "" || cfg.GoogleCalDAVClientSecret == "" {
		redirectCaldavGoogleOAuthUI(w, publicBase, url.Values{"caldav_oauth_error": {"not_configured"}})
		return
	}
	if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
		log.Printf("CalDAV Google OAuth user error: %s %s", oauthErr, r.URL.Query().Get("error_description"))
		redirectCaldavGoogleOAuthUI(w, publicBase, url.Values{"caldav_oauth_error": {oauthErr}})
		return
	}
	stateID := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if stateID == "" || code == "" {
		redirectCaldavGoogleOAuthUI(w, publicBase, url.Values{"caldav_oauth_error": {"missing_code"}})
		return
	}
	authState, err := GetAuthStateRepository().GetOne(stateID)
	if err != nil || authState == nil {
		redirectCaldavGoogleOAuthUI(w, publicBase, url.Values{"caldav_oauth_error": {"invalid_state"}})
		return
	}
	if authState.AuthStateType != AuthCalDAVGoogleOAuth || authState.Expiry.Before(time.Now()) {
		redirectCaldavGoogleOAuthUI(w, publicBase, url.Values{"caldav_oauth_error": {"invalid_state"}})
		return
	}
	userID := authState.AuthProviderID
	_ = GetAuthStateRepository().Delete(authState)

	redirectURL := GoogleCalDAVRedirectURL(publicBase)
	oauthCfg := NewGoogleCalDAVOAuthConfig(redirectURL)
	token, err := oauthCfg.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("CalDAV Google OAuth exchange failed (user=%s): %v", userID, err)
		redirectCaldavGoogleOAuthUI(w, publicBase, url.Values{"caldav_oauth_error": {"token_exchange"}})
		return
	}
	if token.RefreshToken == "" {
		log.Printf("CalDAV Google OAuth missing refresh token (user=%s)", userID)
		redirectCaldavGoogleOAuthUI(w, publicBase, url.Values{"caldav_oauth_error": {"no_refresh_token"}})
		return
	}
	email, err := FetchGoogleAccountEmail(context.Background(), token)
	if err != nil {
		log.Printf("CalDAV Google OAuth userinfo failed (user=%s): %v", userID, err)
		redirectCaldavGoogleOAuthUI(w, publicBase, url.Values{"caldav_oauth_error": {"userinfo"}})
		return
	}
	encryptedRefresh, err := EncryptString(token.RefreshToken)
	if err != nil {
		log.Println(err)
		redirectCaldavGoogleOAuthUI(w, publicBase, url.Values{"caldav_oauth_error": {"server"}})
		return
	}
	prefs := GetUserPreferencesRepository()
	_ = prefs.Set(userID, PreferenceCalDAVProvider.Name, CalDAVProviderGoogle)
	_ = prefs.Set(userID, PreferenceCalDAVOAuthRefresh.Name, encryptedRefresh)
	_ = prefs.Set(userID, PreferenceCalDAVGoogleEmail.Name, email)
	_ = prefs.Set(userID, PreferenceCalDAVURL.Name, "")
	_ = prefs.Set(userID, PreferenceCalDAVUser.Name, "")
	_ = prefs.Set(userID, PreferenceCalDAVPass.Name, "")

	redirectCaldavGoogleOAuthUI(w, publicBase, url.Values{"caldav": {"connected"}})
}

func (router *UserPreferencesRouter) caldavListCalendars(w http.ResponseWriter, r *http.Request) {
	if !CanCrypt() {
		log.Println("Error: CalDAV integration requires a valid crypt key (CRYPT_KEY).")
		SendInternalServerError(w)
		return
	}
	user := GetRequestUser(r)
	var m ListCaldavCalendarsRequest
	if UnmarshalBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}
	provider := m.Provider
	if provider == "" {
		provider = getUserCalDAVProvider(user.ID)
	}
	caldavClient := &CalDAVClient{}
	var err error
	if provider == CalDAVProviderGoogle {
		publicBase := GetRequestPublicBase(r)
		tokenSource, email, tokenErr := getGoogleCalDAVTokenSource(user.ID, publicBase)
		if tokenErr != nil {
			SendBadRequest(w)
			return
		}
		err = caldavClient.ConnectWithTokenSource(GoogleCalDAVPrincipalURL(email), tokenSource)
		if err != nil {
			log.Printf("CalDAV connect failed (user=%s, provider=google, email=%s): %v", user.ID, email, err)
			SendBadGateway(w)
			return
		}
	} else {
		if m.URL == "" || m.Username == "" || m.Password == "" {
			SendBadRequest(w)
			return
		}
		err = caldavClient.Connect(m.URL, m.Username, m.Password)
		if err != nil {
			log.Printf("CalDAV connect failed (user=%s, provider=generic, url=%s): %v", user.ID, m.URL, err)
			SendBadGateway(w)
			return
		}
	}
	calendars, err := caldavClient.ListCalendars()
	if err != nil {
		log.Printf("CalDAV list calendars failed (user=%s, provider=%s): %v", user.ID, provider, err)
		SendBadGateway(w)
		return
	}
	res := make([]*ListCaldavCalendarsResponse, 0)
	for _, calendar := range calendars {
		res = append(res, &ListCaldavCalendarsResponse{Path: calendar.Path, Name: calendar.Name})
	}
	SendJSON(w, res)
}

func (router *UserPreferencesRouter) getPreference(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	vars := mux.Vars(r)
	if !router.isValidPreferenceName(vars["name"]) {
		SendNotFound(w)
		return
	}
	value, err := GetUserPreferencesRepository().Get(user.ID, vars["name"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	SendJSON(w, value)
}

func (router *UserPreferencesRouter) setPreference(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	var value SetSettingsRequest
	if UnmarshalValidateBody(r, &value) != nil {
		SendBadRequest(w)
		return
	}
	vars := mux.Vars(r)
	if !router.isValidPreferenceName(vars["name"]) {
		SendNotFound(w)
		return
	}
	if !router.isValidPreferenceType(vars["name"], value.Value) {
		SendBadRequest(w)
		return
	}
	if !router.isValidPreferenceValue(vars["name"], value.Value, user) {
		SendBadRequest(w)
		return
	}
	err := router.doSetOne(user.ID, vars["name"], value.Value)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendUpdated(w)
}

func (router *UserPreferencesRouter) getAll(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	list, err := GetUserPreferencesRepository().GetAll(user.ID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	res := []*GetSettingsResponse{}
	for _, e := range list {
		m := router.copyToRestModel(e)
		res = append(res, m)
	}
	SendJSON(w, res)
}

func (router *UserPreferencesRouter) setAll(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	var list []GetSettingsResponse
	if err := UnmarshalBody(r, &list); err != nil {
		log.Println(err)
		SendBadRequest(w)
		return
	}
	for _, e := range list {
		if !router.isValidPreferenceName(e.Name) {
			SendNotFound(w)
			return
		}
		if !router.isValidPreferenceType(e.Name, e.Value) {
			SendBadRequest(w)
			return
		}
		if !router.isValidPreferenceValue(e.Name, e.Value, user) {
			SendBadRequest(w)
			return
		}
		err := router.doSetOne(user.ID, e.Name, e.Value)
		if err != nil {
			log.Println(err)
			SendInternalServerError(w)
			return
		}
	}
	SendUpdated(w)
}

func (router *UserPreferencesRouter) doSetOne(userID, name, value string) error {
	if router.getPreferenceType(name) == SettingTypeEncryptedString {
		var err error
		value, err = EncryptString(value)
		if err != nil {
			return err
		}
	}
	err := GetUserPreferencesRepository().Set(userID, name, value)
	return err
}

func (router *UserPreferencesRouter) isValidPreferenceName(name string) bool {
	if name == PreferenceEnterTime.Name ||
		name == PreferenceWorkdayStart.Name ||
		name == PreferenceWorkdayEnd.Name ||
		name == PreferenceWorkdays.Name ||
		name == PreferenceBookedColor.Name ||
		name == PreferenceBuddyBookedColor.Name ||
		name == PreferenceDisallowedColor.Name ||
		name == PreferenceSelfBookedColor.Name ||
		name == PreferencePartiallyBookedColor.Name ||
		name == PreferenceNotBookedColor.Name ||
		name == PreferenceLocation.Name ||
		name == PreferenceCalDAVURL.Name ||
		name == PreferenceCalDAVUser.Name ||
		name == PreferenceCalDAVPass.Name ||
		name == PreferenceCalDAVPath.Name ||
		name == PreferenceCalDAVProvider.Name ||
		name == PreferenceCalDAVOAuthRefresh.Name ||
		name == PreferenceCalDAVGoogleEmail.Name ||
		name == PreferenceMailNotifications.Name ||
		name == PreferenceApprovalNotifications.Name ||
		name == Preference24HourTime.Name ||
		name == PreferenceDateFormat.Name {
		return true
	}
	return false
}

func (router *UserPreferencesRouter) getPreferenceType(name string) SettingType {
	if name == PreferenceEnterTime.Name {
		return PreferenceEnterTime.Type
	}
	if name == PreferenceWorkdayStart.Name {
		return PreferenceWorkdayStart.Type
	}
	if name == PreferenceWorkdayEnd.Name {
		return PreferenceWorkdayEnd.Type
	}
	if name == PreferenceBookedColor.Name {
		return PreferenceBookedColor.Type
	}
	if name == PreferenceBuddyBookedColor.Name {
		return PreferenceBuddyBookedColor.Type
	}
	if name == PreferenceDisallowedColor.Name {
		return PreferenceDisallowedColor.Type
	}
	if name == PreferenceSelfBookedColor.Name {
		return PreferenceSelfBookedColor.Type
	}
	if name == PreferencePartiallyBookedColor.Name {
		return PreferencePartiallyBookedColor.Type
	}
	if name == PreferenceNotBookedColor.Name {
		return PreferenceNotBookedColor.Type
	}
	if name == PreferenceWorkdays.Name {
		return PreferenceWorkdays.Type
	}
	if name == PreferenceLocation.Name {
		return PreferenceLocation.Type
	}
	if name == PreferenceCalDAVURL.Name {
		return PreferenceCalDAVURL.Type
	}
	if name == PreferenceCalDAVUser.Name {
		return PreferenceCalDAVUser.Type
	}
	if name == PreferenceCalDAVPass.Name {
		return PreferenceCalDAVPass.Type
	}
	if name == PreferenceCalDAVPath.Name {
		return PreferenceCalDAVPath.Type
	}
	if name == PreferenceCalDAVProvider.Name {
		return PreferenceCalDAVProvider.Type
	}
	if name == PreferenceCalDAVOAuthRefresh.Name {
		return PreferenceCalDAVOAuthRefresh.Type
	}
	if name == PreferenceCalDAVGoogleEmail.Name {
		return PreferenceCalDAVGoogleEmail.Type
	}
	if name == PreferenceMailNotifications.Name {
		return PreferenceMailNotifications.Type
	}
	if name == PreferenceApprovalNotifications.Name {
		return PreferenceApprovalNotifications.Type
	}
	if name == Preference24HourTime.Name {
		return Preference24HourTime.Type
	}
	if name == PreferenceDateFormat.Name {
		return PreferenceDateFormat.Type
	}
	return 0
}

func (router *UserPreferencesRouter) isValidPreferenceType(name string, value string) bool {
	settingType := router.getPreferenceType(name)
	if settingType == 0 {
		return false
	}
	if settingType == SettingTypeString || settingType == SettingTypeEncryptedString {
		return true
	}
	if settingType == SettingTypeBool && (value == "1" || value == "0") {
		return true
	}
	if settingType == SettingTypeInt {
		if _, err := strconv.Atoi(value); err == nil {
			return true
		}
	}
	if settingType == SettingTypeIntArray {
		tokens := strings.Split(value, ",")
		ok := true
		for _, token := range tokens {
			if _, err := strconv.Atoi(token); err != nil {
				ok = false
			}
		}
		return ok
	}

	return false
}

func (router *UserPreferencesRouter) isValidPreferenceValue(name string, value string, user *User) bool {
	if name == PreferenceEnterTime.Name {
		i, _ := strconv.Atoi(value)
		if !(i == PreferenceEnterTimeNow || i == PreferenceEnterTimeNextDay || i == PreferenceEnterTimeNextWorkday) {
			return false
		}
	}
	if name == PreferenceWorkdayStart.Name {
		i, _ := strconv.Atoi(value)
		if i < 0 || i > 24 {
			return false
		}
	}
	if name == PreferenceWorkdayEnd.Name {
		i, _ := strconv.Atoi(value)
		if i < 0 || i > 24 {
			return false
		}
	}
	if name == PreferenceWorkdays.Name {
		tokens := strings.Split(value, ",")
		ok := true
		for _, token := range tokens {
			if workday, err := strconv.Atoi(token); err != nil || workday < 0 || workday > 6 {
				ok = false
			}
		}
		return ok
	}
	if name == PreferenceDateFormat.Name {
		switch value {
		case "Y-m-d", "d.m.Y", "m/d/Y", "d/m/Y":
			return true
		default:
			return false
		}
	}
	if name == PreferenceBookedColor.Name ||
		name == PreferenceNotBookedColor.Name ||
		name == PreferenceSelfBookedColor.Name ||
		name == PreferencePartiallyBookedColor.Name ||
		name == PreferenceBuddyBookedColor.Name ||
		name == PreferenceDisallowedColor.Name {
		return ValidateColorHex(value)
	}
	if name == PreferenceLocation.Name {
		if value == "" {
			return true
		}
		if !ValidateGUID(value) {
			return false
		}
		location, _ := GetLocationRepository().GetOne(value)
		return location.OrganizationID == user.OrganizationID
	}
	if name == PreferenceCalDAVProvider.Name {
		return value == "" || value == CalDAVProviderGoogle || value == CalDAVProviderGeneric
	}

	return len(value) <= 512
}

func (router *UserPreferencesRouter) copyToRestModel(e *UserPreference) *GetSettingsResponse {
	m := &GetSettingsResponse{}
	m.Name = e.Name
	if e.Name == PreferenceCalDAVOAuthRefresh.Name {
		m.Value = ""
		return m
	}
	if router.getPreferenceType(e.Name) == SettingTypeEncryptedString {
		var err error
		m.Value, err = DecryptString(e.Value)
		if err != nil {
			m.Value = ""
		}
	} else {
		m.Value = e.Value
	}
	return m
}
