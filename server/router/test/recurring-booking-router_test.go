package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/router"
	. "github.com/seatsurfing/seatsurfing/server/testutil"
)

func TestRecurringBookingsPrecheckFeatureDisabled(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "0")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	user1 := CreateTestUserInOrg(org)

	l := &Location{
		Name:                  "Test",
		MaxConcurrentBookings: 2,
		OrganizationID:        org.ID,
		Enabled:               true,
	}
	GetLocationRepository().Create(l)
	s1 := &Space{Name: "Test 1", LocationID: l.ID, Enabled: true}
	GetSpaceRepository().Create(s1)

	payload := `{
	"spaceId": "` + s1.ID + `",
	"subject": "Test",
	"enter": "2030-08-28T09:00:00+02:00",
	"leave": "2030-08-28T15:00:00+02:00",
	"end": "2030-09-03T00:00:00Z",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/precheck", user1.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusPaymentRequired, res.Code)
}

func TestRecurringBookingsCreateFeatureDisabled(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "0")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	user1 := CreateTestUserInOrg(org)

	l := &Location{
		Name:                  "Test",
		MaxConcurrentBookings: 2,
		OrganizationID:        org.ID,
		Enabled:               true,
	}
	GetLocationRepository().Create(l)
	s1 := &Space{Name: "Test 1", LocationID: l.ID, Enabled: true}
	GetSpaceRepository().Create(s1)

	payload := `{
	"spaceId": "` + s1.ID + `",
	"subject": "Test",
	"enter": "2030-08-28T09:00:00+02:00",
	"leave": "2030-08-28T15:00:00+02:00",
	"end": "2030-09-03T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/", user1.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusPaymentRequired, res.Code)
}

func TestRecurringBookingsPrecheck(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	user1 := CreateTestUserInOrg(org)
	user2 := CreateTestUserInOrg(org)
	user3 := CreateTestUserInOrg(org)

	l := &Location{
		Name:                  "Test",
		MaxConcurrentBookings: 2,
		OrganizationID:        org.ID,
		Enabled:               true,
	}
	GetLocationRepository().Create(l)
	s1 := &Space{Name: "Test 1", LocationID: l.ID, Enabled: true}
	GetSpaceRepository().Create(s1)
	s2 := &Space{Name: "Test 2", LocationID: l.ID, Enabled: true}
	GetSpaceRepository().Create(s2)

	// Create booking 1
	payload := "{\"spaceId\": \"" + s1.ID + "\", \"enter\": \"2030-09-01T08:30:00+02:00\", \"leave\": \"2030-09-01T17:00:00+02:00\"}"
	req := NewHTTPRequest("POST", "/booking/", user1.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	// Create booking 2
	payload = "{\"spaceId\": \"" + s2.ID + "\", \"enter\": \"2030-09-02T07:30:00+02:00\", \"leave\": \"2030-09-02T12:00:00+02:00\"}"
	req = NewHTTPRequest("POST", "/booking/", user2.ID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	payload = `{
	"spaceId": "` + s1.ID + `",
	"subject": "Test",
	"enter": "2030-08-28T09:00:00+02:00",
	"leave": "2030-08-28T15:00:00+02:00",
	"end": "2030-09-03T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req = NewHTTPRequest("POST", "/recurring-booking/precheck", user3.ID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []CreateRecurringBookingResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 6, len(resBody))

	CheckTestBool(t, true, resBody[0].Success)  // 28
	CheckTestBool(t, true, resBody[1].Success)  // 29
	CheckTestBool(t, true, resBody[2].Success)  // 30
	CheckTestBool(t, true, resBody[3].Success)  // 31
	CheckTestBool(t, false, resBody[4].Success) // 01
	CheckTestBool(t, true, resBody[5].Success)  // 02
}

func TestRecurringBookingFixedSlotSpaceTypeRejectsNonMatchingSlot(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	createBookingTestOfficeSettings(t, "08:00", "17:00")
	user := CreateTestUserInOrg(org)
	_, space := createBookingTestSpaceWithType(t, org, SpaceTypeBookingModeFixedSlots, 0, []*SpaceTypeSlot{
		{Label: "Morning", StartTime: "08:00", EndTime: "12:00", Enabled: true, SortOrder: 1},
	})

	payload := `{
	"spaceId": "` + space.ID + `",
	"subject": "Test",
	"enter": "2030-09-01T08:30:00Z",
	"leave": "2030-09-01T12:00:00Z",
	"end": "2030-09-02T00:00:00Z",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/precheck", user.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []CreateRecurringBookingResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 1, len(resBody))
	CheckTestBool(t, false, resBody[0].Success)
	CheckTestInt(t, 1012, resBody[0].ErrorCode)
}

func TestRecurringBookingOutsideOfficeHoursRejected(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	createBookingTestOfficeSettings(t, "08:00", "17:00")
	user := CreateTestUserInOrg(org)
	_, space := createBookingTestSpaceWithType(t, org, SpaceTypeBookingModeFlexibleTime, 30, nil)

	payload := `{
	"spaceId": "` + space.ID + `",
	"subject": "Test",
	"enter": "2030-09-01T07:30:00+02:00",
	"leave": "2030-09-01T08:30:00+02:00",
	"end": "2030-09-02T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/precheck", user.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []CreateRecurringBookingResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 1, len(resBody))
	CheckTestBool(t, false, resBody[0].Success)
	CheckTestInt(t, 1014, resBody[0].ErrorCode)
}

func TestRecurringBookingCreateOutsideOfficeHoursRejected(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	createBookingTestOfficeSettings(t, "08:00", "17:00")
	user := CreateTestUserInOrg(org)
	_, space := createBookingTestSpaceWithType(t, org, SpaceTypeBookingModeFlexibleTime, 30, nil)

	payload := `{
	"spaceId": "` + space.ID + `",
	"subject": "Test",
	"enter": "2030-09-01T07:30:00+02:00",
	"leave": "2030-09-01T08:30:00+02:00",
	"end": "2030-09-02T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/", user.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	CheckTestString(t, "", res.Header().Get("X-Object-ID"))
	var resBody []CreateRecurringBookingResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 1, len(resBody))
	CheckTestBool(t, false, resBody[0].Success)
	CheckTestInt(t, 1014, resBody[0].ErrorCode)
}

func TestRecurringBookingWithoutSpaceTypeUsesLegacyTimeRangeFlow(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	createBookingTestOfficeSettings(t, "08:00", "17:00")
	user := CreateTestUserInOrg(org)

	location := &Location{Name: "Location 1", OrganizationID: org.ID, Enabled: true}
	CheckTestBool(t, true, GetLocationRepository().Create(location) == nil)
	space := &Space{Name: "No type", LocationID: location.ID, Enabled: true}
	CheckTestBool(t, true, GetSpaceRepository().Create(space) == nil)

	validPayload := `{
	"spaceId": "` + space.ID + `",
	"subject": "Valid",
	"enter": "2030-09-02T09:00:00Z",
	"leave": "2030-09-02T10:00:00Z",
	"end": "2030-09-03T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/precheck", user.ID, bytes.NewBufferString(validPayload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var validBody []CreateRecurringBookingResponse
	json.Unmarshal(res.Body.Bytes(), &validBody)
	CheckTestInt(t, 1, len(validBody))
	CheckTestBool(t, true, validBody[0].Success)

	bookingPayload := "{\"spaceId\": \"" + space.ID + "\", \"enter\": \"2030-09-01T09:00:00Z\", \"leave\": \"2030-09-01T10:00:00Z\"}"
	req = NewHTTPRequest("POST", "/booking/", user.ID, bytes.NewBufferString(bookingPayload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	overlapPayload := `{
	"spaceId": "` + space.ID + `",
	"subject": "Overlap",
	"enter": "2030-09-01T09:30:00Z",
	"leave": "2030-09-01T10:30:00Z",
	"end": "2030-09-02T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req = NewHTTPRequest("POST", "/recurring-booking/precheck", user.ID, bytes.NewBufferString(overlapPayload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var overlapBody []CreateRecurringBookingResponse
	json.Unmarshal(res.Body.Bytes(), &overlapBody)
	CheckTestInt(t, 1, len(overlapBody))
	CheckTestBool(t, false, overlapBody[0].Success)
	CheckTestInt(t, 1001, overlapBody[0].ErrorCode)

	outsideOfficePayload := `{
	"spaceId": "` + space.ID + `",
	"subject": "Outside",
	"enter": "2030-09-03T07:30:00Z",
	"leave": "2030-09-03T08:30:00Z",
	"end": "2030-09-04T00:00:00Z",
	"cadence": 1,
	"cycle": 1
	}`
	req = NewHTTPRequest("POST", "/recurring-booking/precheck", user.ID, bytes.NewBufferString(outsideOfficePayload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var outsideOfficeBody []CreateRecurringBookingResponse
	json.Unmarshal(res.Body.Bytes(), &outsideOfficeBody)
	CheckTestInt(t, 1, len(outsideOfficeBody))
	CheckTestBool(t, false, outsideOfficeBody[0].Success)
	CheckTestInt(t, 1014, outsideOfficeBody[0].ErrorCode)
}

func TestRecurringBookingsCreateDelete(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	user1 := CreateTestUserInOrg(org)
	user2 := CreateTestUserInOrg(org)
	user3 := CreateTestUserInOrg(org)

	l := &Location{
		Name:                  "Test",
		MaxConcurrentBookings: 2,
		OrganizationID:        org.ID,
		Enabled:               true,
	}
	GetLocationRepository().Create(l)
	s1 := &Space{Name: "Test 1", LocationID: l.ID, Enabled: true}
	GetSpaceRepository().Create(s1)
	s2 := &Space{Name: "Test 2", LocationID: l.ID, Enabled: true}
	GetSpaceRepository().Create(s2)

	// Create booking 1
	payload := "{\"spaceId\": \"" + s1.ID + "\", \"enter\": \"2030-09-01T08:30:00+02:00\", \"leave\": \"2030-09-01T17:00:00+02:00\"}"
	req := NewHTTPRequest("POST", "/booking/", user1.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	// Create booking 2
	payload = "{\"spaceId\": \"" + s2.ID + "\", \"enter\": \"2030-09-02T07:30:00+02:00\", \"leave\": \"2030-09-02T12:00:00+02:00\"}"
	req = NewHTTPRequest("POST", "/booking/", user2.ID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	payload = `{
	"spaceId": "` + s1.ID + `",
	"subject": "Test",
	"enter": "2030-08-28T09:00:00+02:00",
	"leave": "2030-08-28T15:00:00+02:00",
	"end": "2030-09-03T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req = NewHTTPRequest("POST", "/recurring-booking/", user3.ID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	var resBody []CreateRecurringBookingResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 6, len(resBody))

	CheckTestBool(t, true, resBody[0].Success)  // 28
	CheckTestBool(t, true, resBody[1].Success)  // 29
	CheckTestBool(t, true, resBody[2].Success)  // 30
	CheckTestBool(t, true, resBody[3].Success)  // 31
	CheckTestBool(t, false, resBody[4].Success) // 01
	CheckTestBool(t, true, resBody[5].Success)  // 02

	for idx, b := range resBody {
		if idx != 4 { // 01 is not created
			CheckTestBool(t, true, b.ID != "")
			booking, err := GetBookingRepository().GetOne(b.ID)
			CheckTestBool(t, true, err == nil)
			CheckTestBool(t, true, booking != nil)
		}
	}

	booking, _ := GetBookingRepository().GetOne(resBody[0].ID)
	req = NewHTTPRequest("DELETE", "/recurring-booking/"+string(booking.RecurringID), user3.ID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)
}

func TestRecurringBookingsGet(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	user1 := CreateTestUserInOrg(org)

	l := &Location{
		Name:                  "Test",
		MaxConcurrentBookings: 2,
		OrganizationID:        org.ID,
		Enabled:               true,
	}
	GetLocationRepository().Create(l)
	s1 := &Space{Name: "Test 1", LocationID: l.ID, Enabled: true}
	GetSpaceRepository().Create(s1)

	payload := `{
	"spaceId": "` + s1.ID + `",
	"subject": "Test",
	"enter": "2030-08-28T09:00:00+02:00",
	"leave": "2030-08-28T15:00:00+02:00",
	"end": "2030-08-30T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/", user1.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	id := res.Header().Get("X-Object-ID")

	// Get the recurring booking by ID
	req = NewHTTPRequest("GET", "/recurring-booking/"+id, user1.ID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody *GetRecurringBookingResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if resBody == nil {
		t.Fatal("Expected non-nil recurring booking response")
	}
	CheckTestString(t, s1.ID, resBody.SpaceID)
}

func TestRecurringBookingsGetNotFound(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user1 := CreateTestUserInOrg(org)

	req := NewHTTPRequest("GET", "/recurring-booking/nonexistent-id", user1.ID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNotFound, res.Code)
}

func TestRecurringBookingsGetForeign(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	user1 := CreateTestUserInOrg(org)
	user2 := CreateTestUserInOrg(org)

	l := &Location{
		Name:                  "Test",
		MaxConcurrentBookings: 2,
		OrganizationID:        org.ID,
		Enabled:               true,
	}
	GetLocationRepository().Create(l)
	s1 := &Space{Name: "Test 1", LocationID: l.ID, Enabled: true}
	GetSpaceRepository().Create(s1)

	// user1 creates a recurring booking
	payload := `{
	"spaceId": "` + s1.ID + `",
	"subject": "Test",
	"enter": "2030-08-28T09:00:00+02:00",
	"leave": "2030-08-28T15:00:00+02:00",
	"end": "2030-08-30T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/", user1.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	id := res.Header().Get("X-Object-ID")

	// user2 tries to get user1's recurring booking → should be forbidden
	req = NewHTTPRequest("GET", "/recurring-booking/"+id, user2.ID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestRecurringBookingsGetICal(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	user1 := CreateTestUserInOrg(org)

	l := &Location{
		Name:                  "Test",
		MaxConcurrentBookings: 2,
		OrganizationID:        org.ID,
		Enabled:               true}
	GetLocationRepository().Create(l)
	s1 := &Space{Name: "Test 1", LocationID: l.ID, Enabled: true}
	GetSpaceRepository().Create(s1)

	payload := `{
	"spaceId": "` + s1.ID + `",
	"subject": "Test",
	"enter": "2030-08-28T09:00:00+02:00",
	"leave": "2030-08-28T15:00:00+02:00",
	"end": "2030-08-30T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/", user1.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	id := res.Header().Get("X-Object-ID")

	// Get iCal for the recurring booking
	req = NewHTTPRequest("GET", "/recurring-booking/"+id+"/ical", user1.ID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	contentType := res.Header().Get("Content-Type")
	if contentType != "text/calendar" {
		t.Fatalf("Expected Content-Type text/calendar, got %s", contentType)
	}
}

func TestRecurringBookingsGetICalNotFound(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user1 := CreateTestUserInOrg(org)

	req := NewHTTPRequest("GET", "/recurring-booking/nonexistent-id/ical", user1.ID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNotFound, res.Code)
}

func TestRecurringBookingsDeleteNotFound(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user1 := CreateTestUserInOrg(org)

	req := NewHTTPRequest("DELETE", "/recurring-booking/nonexistent-id", user1.ID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNotFound, res.Code)
}

func TestRecurringBookingsDeleteForeign(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	user1 := CreateTestUserInOrg(org)
	user2 := CreateTestUserInOrg(org)

	l := &Location{
		Name:                  "Test",
		MaxConcurrentBookings: 2,
		OrganizationID:        org.ID,
		Enabled:               true,
	}
	GetLocationRepository().Create(l)
	s1 := &Space{Name: "Test 1", LocationID: l.ID, Enabled: true}
	GetSpaceRepository().Create(s1)

	// user1 creates a recurring booking
	payload := `{
	"spaceId": "` + s1.ID + `",
	"subject": "Test",
	"enter": "2030-08-28T09:00:00+02:00",
	"leave": "2030-08-28T15:00:00+02:00",
	"end": "2030-08-30T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/", user1.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	id := res.Header().Get("X-Object-ID")

	// user2 tries to delete user1's recurring booking → should be forbidden
	req = NewHTTPRequest("DELETE", "/recurring-booking/"+id, user2.ID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestCannotCreateRecurringBookingsInDisabledLocation(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	user1 := CreateTestUserInOrg(org)

	location, space := CreateTestLocationAndSpace(org)

	location.Enabled = false
	GetLocationRepository().Update(location)

	payload := `{
	"spaceId": "` + space.ID + `",
	"subject": "Test",
	"enter": "2030-08-28T09:00:00+02:00",
	"leave": "2030-08-28T15:00:00+02:00",
	"end": "2030-08-30T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/", user1.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)
}

func TestCannotCreateRecurringBookingsInDisabledSpace(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureRecurringBookings.Name, "1")
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, strconv.Itoa(365*10))
	GetSettingsRepository().Set(org.ID, SettingMaxBookingsPerUser.Name, "1000")
	user1 := CreateTestUserInOrg(org)

	_, space := CreateTestLocationAndSpace(org)

	space.Enabled = false
	GetSpaceRepository().Update(space)

	payload := `{
	"spaceId": "` + space.ID + `",
	"subject": "Test",
	"enter": "2030-08-28T09:00:00+02:00",
	"leave": "2030-08-28T15:00:00+02:00",
	"end": "2030-08-30T00:00:00+02:00",
	"cadence": 1,
	"cycle": 1
	}`
	req := NewHTTPRequest("POST", "/recurring-booking/", user1.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)
}
