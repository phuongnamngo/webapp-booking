package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/router"
	. "github.com/seatsurfing/seatsurfing/server/testutil"
	. "github.com/seatsurfing/seatsurfing/server/util"
)

func TestSettingsForbidden(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)
	loginResponse := LoginTestUser(user.ID)

	payload := `{"value": "1"}`
	req := NewHTTPRequest("PUT", "/setting/"+SettingAllowAnyUser.Name, loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)

	req = NewHTTPRequest("GET", "/setting/"+SettingAllowAnyUser.Name, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)

	req = NewHTTPRequest("GET", "/setting/"+SettingConfluenceServerSharedSecret.Name, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)

	req = NewHTTPRequest("GET", "/setting/", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)

	payload = `[]`
	req = NewHTTPRequest("PUT", "/setting/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestSettingsReadPublic(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)
	loginResponse := LoginTestUser(user.ID)

	allowedSettings := []string{
		SettingDisableBuddies.Name,
		SettingMaxBookingsPerUser.Name,
		SettingMaxConcurrentBookingsPerUser.Name,
		SettingMaxDaysInAdvance.Name,
		SettingMaxBookingDurationHours.Name,
		SettingMaxHoursBeforeDelete.Name,
		SettingEnableMaxHourBeforeDelete.Name,
		SettingDailyBasisBooking.Name,
		SettingNoAdminRestrictions.Name,
		SettingShowNames.Name,
		SettingMinBookingDurationHours.Name,
		SettingAllowBookingsNonExistingUsers.Name,
		SettingDefaultTimezone.Name,
		SettingCustomLogoUrl.Name,
		SysSettingVersion,
		SysSettingOrgPrimaryDomain,
		SysSettingDisablePasswordLogin,
		SettingFeatureNoUserLimit.Name,
		SettingFeatureGroups.Name,
		SettingFeatureCustomDomains.Name,
		SettingAllowRecurringBookings.Name,
		SettingSubjectDefault.Name,
		SettingEnforceTOTP.Name,
	}
	forbiddenSettings := []string{
		SettingDatabaseVersion.Name,
		SettingAllowAnyUser.Name,
		SettingConfluenceServerSharedSecret.Name,
		SettingConfluenceAnonymous.Name,
		"max_hours_partially_booked",
		"max_hours_partially_booked_enabled",
	}

	for _, name := range allowedSettings {
		req := NewHTTPRequest("GET", "/setting/"+name, loginResponse.UserID, nil)
		res := ExecuteTestRequest(req)
		CheckTestResponseCode(t, http.StatusOK, res.Code)
	}

	for _, name := range forbiddenSettings {
		req := NewHTTPRequest("GET", "/setting/"+name, loginResponse.UserID, nil)
		res := ExecuteTestRequest(req)
		CheckTestResponseCode(t, http.StatusForbidden, res.Code)
	}

	req := NewHTTPRequest("GET", "/setting/", loginResponse.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []GetSettingsResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, len(allowedSettings), len(resBody))
	found := 0
	for _, name := range allowedSettings {
		for _, cur := range resBody {
			if name == cur.Name {
				found++
			}
		}
	}
	CheckTestInt(t, len(allowedSettings), found)
}

func TestSettingsReadAdmin(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	allowedSettings := []string{
		SettingDisableBuddies.Name,
		SettingMaxBookingsPerUser.Name,
		SettingMaxConcurrentBookingsPerUser.Name,
		SettingMaxDaysInAdvance.Name,
		SettingMaxBookingDurationHours.Name,
		SettingMaxHoursBeforeDelete.Name,
		SettingDailyBasisBooking.Name,
		SettingMinBookingDurationHours.Name,
		SettingNoAdminRestrictions.Name,
		SettingShowNames.Name,
		SettingEnableMaxHourBeforeDelete.Name,
		SettingAllowBookingsNonExistingUsers.Name,
		SettingAllowAnyUser.Name,
		SettingConfluenceServerSharedSecret.Name,
		SettingConfluenceAnonymous.Name,
		SettingFeatureNoUserLimit.Name,
		SettingFeatureGroups.Name,
		SettingFeatureCustomDomains.Name,
		SettingDefaultTimezone.Name,
		SettingCustomLogoUrl.Name,
		SysSettingOrgSignupDelete,
		SysSettingVersion,
		SysSettingAdminMenuItems,
		SysSettingAdminWelcomeScreens,
		SysSettingOrgPrimaryDomain,
		SysSettingDisablePasswordLogin,
		SettingBookingRetentionEnabled.Name,
		SettingBookingRetentionDays.Name,
		SettingAllowRecurringBookings.Name,
		SettingNewUserDefaultMailNotification.Name,
		SettingSubjectDefault.Name,
		SettingEnforceTOTP.Name,
		SettingTargetUtilizationHoursPerWeek.Name,
	}
	forbiddenSettings := []string{
		SettingDatabaseVersion.Name,
		"max_hours_partially_booked",
		"max_hours_partially_booked_enabled",
	}

	for _, name := range allowedSettings {
		req := NewHTTPRequest("GET", "/setting/"+name, loginResponse.UserID, nil)
		res := ExecuteTestRequest(req)
		CheckTestResponseCode(t, http.StatusOK, res.Code)
	}

	for _, name := range forbiddenSettings {
		req := NewHTTPRequest("GET", "/setting/"+name, loginResponse.UserID, nil)
		res := ExecuteTestRequest(req)
		CheckTestResponseCode(t, http.StatusForbidden, res.Code)
	}

	req := NewHTTPRequest("GET", "/setting/", loginResponse.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []GetSettingsResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, len(allowedSettings), len(resBody))
	found := 0
	for _, name := range allowedSettings {
		for _, cur := range resBody {
			if name == cur.Name {
				found++
			}
		}
	}
	CheckTestInt(t, len(allowedSettings), found)
}

func TestSettingsCRUD(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	payload := `{"value": "1"}`
	req := NewHTTPRequest("PUT", "/setting/"+SettingAllowAnyUser.Name, loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/setting/"+SettingAllowAnyUser.Name, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody string
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestString(t, "1", resBody)

	payload = `{"value": "0"}`
	req = NewHTTPRequest("PUT", "/setting/"+SettingAllowAnyUser.Name, loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/setting/"+SettingAllowAnyUser.Name, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody2 string
	json.Unmarshal(res.Body.Bytes(), &resBody2)
	CheckTestString(t, "0", resBody2)
}

func TestSettingsOfficeHoursCRUD(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	payload := `{"workStartTime": "08:00", "workEndTime": "17:00"}`
	req := NewHTTPRequest("PUT", "/setting/office-hours", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/setting/office-hours", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody OfficeSettingsResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestString(t, "08:00", resBody.WorkStartTime)
	CheckTestString(t, "17:00", resBody.WorkEndTime)
}

func TestSettingsOfficeHoursForbiddenForNonAdmin(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)
	loginResponse := LoginTestUser(user.ID)

	req := NewHTTPRequest("GET", "/setting/office-hours", loginResponse.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)

	payload := `{"workStartTime": "08:00", "workEndTime": "17:00"}`
	req = NewHTTPRequest("PUT", "/setting/office-hours", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestSettingsOfficeHoursRejectsInvalidTimes(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	payload := `{"workStartTime": "17:00", "workEndTime": "08:00"}`
	req := NewHTTPRequest("PUT", "/setting/office-hours", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)
}

func TestSettingsCRUDMany(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)
	GetDatabase().DB().Exec("TRUNCATE settings")

	payload := `[{"name": "allow_any_user", "value": "1"}, {"name": "max_bookings_per_user", "value": "5"}]`
	req := NewHTTPRequest("PUT", "/setting/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/setting/", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []GetSettingsResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 8, len(resBody))
	CheckTestString(t, SettingAllowAnyUser.Name, resBody[0].Name)
	CheckTestString(t, SettingMaxBookingsPerUser.Name, resBody[1].Name)
	CheckTestString(t, SysSettingOrgSignupDelete, resBody[2].Name)
	CheckTestString(t, SysSettingAdminWelcomeScreens, resBody[3].Name)
	CheckTestString(t, SysSettingAdminMenuItems, resBody[4].Name)
	CheckTestString(t, SysSettingVersion, resBody[5].Name)
	CheckTestString(t, "1", resBody[0].Value)
	CheckTestString(t, "5", resBody[1].Value)
	CheckTestString(t, GetProductVersion(), resBody[5].Value)

	payload = `[{"name": "allow_any_user", "value": "0"}, {"name": "max_bookings_per_user", "value": "3"}]`
	req = NewHTTPRequest("PUT", "/setting/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/setting/", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody2 []GetSettingsResponse
	json.Unmarshal(res.Body.Bytes(), &resBody2)
	CheckTestInt(t, 8, len(resBody2))
	CheckTestString(t, SettingAllowAnyUser.Name, resBody2[0].Name)
	CheckTestString(t, SettingMaxBookingsPerUser.Name, resBody2[1].Name)
	CheckTestString(t, SysSettingOrgSignupDelete, resBody2[2].Name)
	CheckTestString(t, SysSettingVersion, resBody2[5].Name)
	CheckTestString(t, "0", resBody2[0].Value)
	CheckTestString(t, "3", resBody2[1].Value)

}

func TestSettingsMaxHoursBeforeDelete(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)
	GetDatabase().DB().Exec("TRUNCATE settings")

	payload := `[{"name": "max_hours_before_delete", "value": "2"}]`
	req := NewHTTPRequest("PUT", "/setting/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/setting/", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody3 []GetSettingsResponse
	json.Unmarshal(res.Body.Bytes(), &resBody3)
	CheckTestInt(t, 7, len(resBody3))
	CheckTestString(t, SettingMaxHoursBeforeDelete.Name, resBody3[0].Name)
	CheckTestString(t, SysSettingOrgSignupDelete, resBody3[1].Name)
	CheckTestString(t, SysSettingVersion, resBody3[4].Name)
	CheckTestString(t, "2", resBody3[0].Value)
}

func TestSettingsMinHoursBookingDuration(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)
	GetDatabase().DB().Exec("TRUNCATE settings")

	payload := `[{"name": "min_booking_duration_hours", "value": "2"}]`
	req := NewHTTPRequest("PUT", "/setting/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/setting/", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody3 []GetSettingsResponse
	json.Unmarshal(res.Body.Bytes(), &resBody3)
	CheckTestInt(t, 7, len(resBody3))
	CheckTestString(t, SettingMinBookingDurationHours.Name, resBody3[0].Name)
	CheckTestString(t, SysSettingOrgSignupDelete, resBody3[1].Name)
	CheckTestString(t, SysSettingVersion, resBody3[4].Name)
	CheckTestString(t, "2", resBody3[0].Value)
}

func TestSettingsInvalidName(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	payload := `{"value": "1"}`
	req := NewHTTPRequest("PUT", "/setting/test123", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNotFound, res.Code)
}

func TestSettingsInvalidBool(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	payload := `{"value": "2"}`
	req := NewHTTPRequest("PUT", "/setting/"+SettingAllowAnyUser.Name, loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)
}

func TestSettingsInvalidInt(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	payload := `{"value": "test"}`
	req := NewHTTPRequest("PUT", "/setting/"+SettingMaxBookingsPerUser.Name, loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)
}

func TestSettingsInvalidTimezone(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	payload := `{"value": "Europe/Hamburg"}`
	req := NewHTTPRequest("PUT", "/setting/"+SettingDefaultTimezone.Name, loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)

	payload = `{"value": "Europe/Berlin"}`
	req = NewHTTPRequest("PUT", "/setting/"+SettingDefaultTimezone.Name, loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)
}

func TestSettingsGetTimezones(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)

	// getTimezones has NO permission check - any authenticated user can access
	req := NewHTTPRequest("GET", "/setting/timezones", user.ID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []string
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if len(resBody) == 0 {
		t.Fatal("Expected non-empty timezone list")
	}
}

func TestSettingsDailyBookingReportRouteForbiddenForNonAdmin(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)
	loginResponse := LoginTestUser(user.ID)

	req := NewHTTPRequest("POST", "/setting/test-daily-booking-report-email", loginResponse.UserID, bytes.NewBufferString(`{"previewOnly":true,"recipients":["boss@test.com"]}`))
	res := ExecuteTestRequest(req)

	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestSettingsDailyBookingReportPreviewReturnsSubjectRecipientsAndHtml(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	admin := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(admin.ID)

	user := CreateTestUserInOrgWithName(org, "john.doe@test.com", UserRoleUser)
	user.Firstname = "John"
	user.Lastname = "Doe"
	CheckTestIsNil(t, GetUserRepository().Update(user))

	location := &Location{Name: "Floor 5", OrganizationID: org.ID}
	CheckTestIsNil(t, GetLocationRepository().Create(location))
	space := &Space{Name: "Desk A01", LocationID: location.ID}
	CheckTestIsNil(t, GetSpaceRepository().Create(space))

	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	CheckTestIsNil(t, err)
	booking := &Booking{
		UserID:   user.ID,
		SpaceID:  space.ID,
		Enter:    time.Date(2026, 5, 13, 9, 0, 0, 0, loc),
		Leave:    time.Date(2026, 5, 13, 18, 0, 0, 0, loc),
		Approved: true,
	}
	CheckTestIsNil(t, GetBookingRepository().Create(booking))

	payload := `{"previewOnly":true,"date":"2026-05-13","recipients":["boss@test.com"]}`
	req := NewHTTPRequest("POST", "/setting/test-daily-booking-report-email", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)

	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var body struct {
		Subject     string   `json:"subject"`
		ReportDate  string   `json:"reportDate"`
		Recipients  []string `json:"recipients"`
		RowCount    int      `json:"rowCount"`
		PreviewHTML string   `json:"previewHtml"`
		Sent        bool     `json:"sent"`
	}
	json.Unmarshal(res.Body.Bytes(), &body)
	CheckTestString(t, "Daily booking report for 2026-05-13", body.Subject)
	CheckTestString(t, "2026-05-13", body.ReportDate)
	CheckTestInt(t, 1, len(body.Recipients))
	CheckTestString(t, "boss@test.com", body.Recipients[0])
	CheckTestInt(t, 1, body.RowCount)
	CheckTestBool(t, true, strings.Contains(body.PreviewHTML, "User name"))
	CheckTestBool(t, true, strings.Contains(body.PreviewHTML, "Space"))
	CheckTestBool(t, true, strings.Contains(body.PreviewHTML, "Booking time"))
	CheckTestBool(t, true, strings.Contains(body.PreviewHTML, "09:00 - 18:00"))
	CheckTestBool(t, true, strings.Contains(body.PreviewHTML, "font-size:14px"))
	CheckTestBool(t, true, strings.Contains(body.PreviewHTML, "border-collapse:collapse"))
	CheckTestBool(t, true, strings.Contains(body.PreviewHTML, "padding:12px 16px"))
	CheckTestBool(t, true, strings.Contains(body.PreviewHTML, "John Doe"))
	CheckTestBool(t, true, strings.Contains(body.PreviewHTML, "Floor 5"))
	CheckTestBool(t, true, strings.Contains(body.PreviewHTML, "Desk A01"))
	CheckTestBool(t, false, body.Sent)
}

func TestSettingsDailyBookingReportRouteRejectsInvalidDate(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	admin := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(admin.ID)

	req := NewHTTPRequest("POST", "/setting/test-daily-booking-report-email", loginResponse.UserID, bytes.NewBufferString(`{"previewOnly":true,"date":"13-05-2026","recipients":["boss@test.com"]}`))
	res := ExecuteTestRequest(req)

	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)
}

func TestSettingsDailyBookingReportSendUsesMockSendmail(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	admin := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(admin.ID)

	req := NewHTTPRequest("POST", "/setting/test-daily-booking-report-email", loginResponse.UserID, bytes.NewBufferString(`{"previewOnly":false,"date":"2026-05-13","recipients":["boss@test.com"]}`))
	res := ExecuteTestRequest(req)

	CheckTestResponseCode(t, http.StatusOK, res.Code)
	CheckTestBool(t, true, strings.Contains(SendMailMockContent, "No approved bookings for 2026-05-13."))
}
