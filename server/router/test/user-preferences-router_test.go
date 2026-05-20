package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/router"
	. "github.com/seatsurfing/seatsurfing/server/testutil"
)

func TestUserPreferencesCRUD(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)
	loginResponse := LoginTestUser(user.ID)

	payload := `{"value": "1"}`
	req := NewHTTPRequest("PUT", "/preference/"+PreferenceEnterTime.Name, loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/preference/"+PreferenceEnterTime.Name, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody string
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestString(t, "1", resBody)

	payload = `{"value": "2"}`
	req = NewHTTPRequest("PUT", "/preference/"+PreferenceEnterTime.Name, loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/preference/"+PreferenceEnterTime.Name, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody2 string
	json.Unmarshal(res.Body.Bytes(), &resBody2)
	CheckTestString(t, "2", resBody2)
}

func TestUserPreferencesCRUDMany(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)
	loginResponse := LoginTestUser(user.ID)
	GetDatabase().DB().Exec("TRUNCATE users_preferences")

	payload := `[{"name": "enter_time", "value": "1"}, {"name": "workday_start", "value": "5"}]`
	req := NewHTTPRequest("PUT", "/preference/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/preference/", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []GetSettingsResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 2, len(resBody))
	CheckTestString(t, PreferenceEnterTime.Name, resBody[0].Name)
	CheckTestString(t, PreferenceWorkdayStart.Name, resBody[1].Name)
	CheckTestString(t, "1", resBody[0].Value)
	CheckTestString(t, "5", resBody[1].Value)

	payload = `[{"name": "enter_time", "value": "2"}, {"name": "workday_start", "value": "3"}]`
	req = NewHTTPRequest("PUT", "/preference/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/preference/", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody2 []GetSettingsResponse
	json.Unmarshal(res.Body.Bytes(), &resBody2)
	CheckTestInt(t, 2, len(resBody2))
	CheckTestString(t, PreferenceEnterTime.Name, resBody2[0].Name)
	CheckTestString(t, PreferenceWorkdayStart.Name, resBody2[1].Name)
	CheckTestString(t, "2", resBody2[0].Value)
	CheckTestString(t, "3", resBody2[1].Value)
}

func TestUserPreferencesApprovalNotifications(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)
	loginResponse := LoginTestUser(user.ID)

	// Test default value (should be "0" = false)
	req := NewHTTPRequest("GET", "/preference/"+PreferenceApprovalNotifications.Name, loginResponse.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody string
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestString(t, "0", resBody)

	// Test setting to true
	payload := `{"value": "1"}`
	req = NewHTTPRequest("PUT", "/preference/"+PreferenceApprovalNotifications.Name, loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	// Verify it was set to true
	req = NewHTTPRequest("GET", "/preference/"+PreferenceApprovalNotifications.Name, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody2 string
	json.Unmarshal(res.Body.Bytes(), &resBody2)
	CheckTestString(t, "1", resBody2)

	// Test setting to false
	payload = `{"value": "0"}`
	req = NewHTTPRequest("PUT", "/preference/"+PreferenceApprovalNotifications.Name, loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	// Verify it was set to false
	req = NewHTTPRequest("GET", "/preference/"+PreferenceApprovalNotifications.Name, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody3 string
	json.Unmarshal(res.Body.Bytes(), &resBody3)
	CheckTestString(t, "0", resBody3)
}
func TestPreferencesGetNotFound(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)

	// Invalid preference name → 404
	req := NewHTTPRequest("GET", "/preference/nonexistentpreference", user.ID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNotFound, res.Code)
}

func TestPreferencesPutInvalid(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)

	// Invalid preference name → 404
	payload := `{"value": "somevalue"}`
	req := NewHTTPRequest("PUT", "/preference/nonexistentpreference", user.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNotFound, res.Code)
}

func TestPreferencesPutInvalidColor(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)

	// Invalid color values → 400
	payload := `[{"name": "booked_color", "value": "this is invalid"}, {"name": "disallowed_color", "value": "` + CreateTestString(1000) + `"}]`
	req := NewHTTPRequest("PUT", "/preference/", user.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)
}

func TestPreferencesForbiddenNoAuth(t *testing.T) {
	ClearTestDB()

	// No auth → 401
	req := NewHTTPRequest("GET", "/preference/", "", nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestPreferencesCalDavListCalendarsInvalidBody(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)

	// Empty body with no Google OAuth configured → 400
	req := NewHTTPRequest("POST", "/preference/caldav/listCalendars", user.ID, bytes.NewBufferString(`{}`))
	res := ExecuteTestRequest(req)
	if res.Code != http.StatusBadRequest && res.Code != http.StatusInternalServerError {
		t.Fatalf("Expected 400 or 500, got %d", res.Code)
	}
}

func TestCalDavGoogleAuthURLRequiresAuth(t *testing.T) {
	ClearTestDB()
	req := NewHTTPRequest("GET", "/preference/caldav/google/auth-url", "", nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusUnauthorized, res.Code)
}

func TestCalDavGoogleAuthURLNoClientConfigured(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)

	req := NewHTTPRequest("GET", "/preference/caldav/google/auth-url", user.ID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusServiceUnavailable, res.Code)
}

func TestCalDavListCalendarsGoogleNotConnected(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)

	GetUserPreferencesRepository().Set(user.ID, PreferenceCalDAVProvider.Name, CalDAVProviderGoogle)

	req := NewHTTPRequest("POST", "/preference/caldav/listCalendars", user.ID, bytes.NewBufferString(`{"provider":"google"}`))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)
}
