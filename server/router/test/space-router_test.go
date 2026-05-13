package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/router"
	. "github.com/seatsurfing/seatsurfing/server/testutil"
)

type spaceResponseWithType struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	SpaceTypeID string             `json:"spaceTypeId"`
	SpaceType   *spaceTypeResponse `json:"spaceType"`
}

type spaceDayStatusTestResponse struct {
	SpaceID       string                               `json:"spaceId"`
	Status        string                               `json:"status"`
	OfficeStart   time.Time                            `json:"officeStart"`
	OfficeEnd     time.Time                            `json:"officeEnd"`
	BookedMinutes int                                  `json:"bookedMinutes"`
	OfficeMinutes int                                  `json:"officeMinutes"`
	Bookings      []*spaceDayStatusBookingTestResponse `json:"bookings"`
}

type spaceDayStatusBookingTestResponse struct {
	ID          string    `json:"id"`
	RecurringID string    `json:"recurringId"`
	UserID      string    `json:"userId"`
	UserEmail   string    `json:"userEmail"`
	Enter       time.Time `json:"enter"`
	Leave       time.Time `json:"leave"`
	Subject     string    `json:"subject"`
}

func TestSpacesSameOrgForbidden(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user2 := CreateTestUserOrgAdmin(org)
	loginResponse2 := LoginTestUser(user2.ID)

	// Create location
	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", loginResponse2.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	id := res.Header().Get("X-Object-Id")

	// Create space
	payload = `{"name": "H234", "x": 50, "y": 100, "width": 200, "height": 300, "rotation": 90}`
	req = NewHTTPRequest("POST", "/location/"+id+"/space/", loginResponse2.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	spaceID := res.Header().Get("X-Object-Id")

	// Switch to non-admin user
	user := CreateTestUserInOrg(org)
	loginResponse := LoginTestUser(user.ID)

	payload = `{"name": "Location 1"}`
	req = NewHTTPRequest("POST", "/location/"+id+"/space/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)

	payload = `{"name": "Location 1"}`
	req = NewHTTPRequest("PUT", "/location/"+id+"/space/"+spaceID, loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)

	req = NewHTTPRequest("DELETE", "/location/"+id+"/space/"+spaceID, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestSpacesEmptyResult(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	// Create location
	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	id := res.Header().Get("X-Object-Id")

	// Get spaces
	req = NewHTTPRequest("GET", "/location/"+id+"/space/", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []string
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if len(resBody) != 0 {
		t.Fatalf("Expected empty array")
	}
}

func TestSpacesCRUD(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	// Create location
	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locationID := res.Header().Get("X-Object-Id")

	// 1. Create
	payload = `{"name": "H234", "x": 50, "y": 100, "width": 200, "height": 300, "rotation": 90, "enabled": true}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	id := res.Header().Get("X-Object-Id")

	// 2. Read
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/"+id, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody *GetSpaceResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestString(t, "H234", resBody.Name)
	CheckTestUint(t, 50, resBody.X)
	CheckTestUint(t, 100, resBody.Y)
	CheckTestUint(t, 200, resBody.Width)
	CheckTestUint(t, 300, resBody.Height)
	CheckTestUint(t, 90, resBody.Rotation)
	CheckTestBool(t, false, resBody.RequireSubject)
	CheckTestBool(t, true, resBody.Enabled)

	// 3. Update
	payload = `{"name": "H235", "x": 51, "y": 101, "width": 201, "height": 301, "rotation": 91, "requireSubject": true, "enabled": false}`
	req = NewHTTPRequest("PUT", "/location/"+locationID+"/space/"+id, loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	// Read
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/"+id, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody2 *GetSpaceResponse
	json.Unmarshal(res.Body.Bytes(), &resBody2)
	CheckTestString(t, "H235", resBody2.Name)
	CheckTestUint(t, 51, resBody2.X)
	CheckTestUint(t, 101, resBody2.Y)
	CheckTestUint(t, 201, resBody2.Width)
	CheckTestUint(t, 301, resBody2.Height)
	CheckTestUint(t, 91, resBody2.Rotation)
	CheckTestBool(t, true, resBody2.RequireSubject)
	CheckTestBool(t, false, resBody2.Enabled)

	// 4. Delete
	req = NewHTTPRequest("DELETE", "/location/"+locationID+"/space/"+id, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	// Read
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/"+id, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNotFound, res.Code)
}

func TestSpacesRejectForeignSpaceType(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	org2 := CreateTestOrg("test2.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)
	spaceType := &SpaceType{
		OrganizationID: org2.ID,
		Name:           "Foreign type",
		BookingMode:    SpaceTypeBookingModeFlexibleTime,
		Enabled:        true,
	}
	CheckTestBool(t, true, GetSpaceTypeRepository().Create(spaceType) == nil)

	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locationID := res.Header().Get("X-Object-Id")

	payload = `{"name": "H234", "x": 50, "y": 100, "width": 200, "height": 300, "rotation": 90, "enabled": true, "spaceTypeId": "` + spaceType.ID + `"}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)

	payload = `{"creates":[{"name": "H235", "x": 50, "y": 100, "width": 200, "height": 300, "rotation": 90, "enabled": true, "spaceTypeId": "` + spaceType.ID + `"}]}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/bulk", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody *BulkUpdateResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 1, len(resBody.Creates))
	CheckTestBool(t, false, resBody.Creates[0].Success)
}

func TestSpacesApproversCRUD(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureGroups.Name, "1")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	// Create group 1
	payload := `{"name": "G 1"}`
	req := NewHTTPRequest("POST", "/group/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	group1ID := res.Header().Get("X-Object-Id")

	// Create group 2
	payload = `{"name": "G 2"}`
	req = NewHTTPRequest("POST", "/group/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	group2ID := res.Header().Get("X-Object-Id")

	// Create location
	payload = `{"name": "Location 1"}`
	req = NewHTTPRequest("POST", "/location/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locationID := res.Header().Get("X-Object-Id")

	// Create space
	payload = `{"name": "H234", "x": 50, "y": 100, "width": 200, "height": 300, "rotation": 90}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	spaceID := res.Header().Get("X-Object-Id")

	// Add approvers
	approverIDs := []string{group1ID, group2ID}
	approverIDsJson, _ := json.Marshal(approverIDs)
	req = NewHTTPRequest("PUT", "/location/"+locationID+"/space/"+spaceID+"/approver", loginResponse.UserID, bytes.NewBuffer(approverIDsJson))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	// List approvers
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/"+spaceID+"/approver", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []*GetGroupResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 2, len(resBody))
	CheckTestString(t, group1ID, resBody[0].ID)
	CheckTestString(t, group2ID, resBody[1].ID)

	// Remove one approver
	approverIDs = []string{group1ID}
	approverIDsJson, _ = json.Marshal(approverIDs)
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/"+spaceID+"/approver/remove", loginResponse.UserID, bytes.NewBuffer(approverIDsJson))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	// List approvers
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/"+spaceID+"/approver", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody2 []*GetGroupResponse
	json.Unmarshal(res.Body.Bytes(), &resBody2)
	CheckTestInt(t, 1, len(resBody2))
	CheckTestString(t, group2ID, resBody2[0].ID)
}

func TestSpacesAllowedSpaceBookersCRUD(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	GetSettingsRepository().Set(org.ID, SettingFeatureGroups.Name, "1")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	// Create group 1
	payload := `{"name": "G 1"}`
	req := NewHTTPRequest("POST", "/group/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	group1ID := res.Header().Get("X-Object-Id")

	// Create group 2
	payload = `{"name": "G 2"}`
	req = NewHTTPRequest("POST", "/group/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	group2ID := res.Header().Get("X-Object-Id")

	// Create location
	payload = `{"name": "Location 1"}`
	req = NewHTTPRequest("POST", "/location/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locationID := res.Header().Get("X-Object-Id")

	// Create space
	payload = `{"name": "H234", "x": 50, "y": 100, "width": 200, "height": 300, "rotation": 90}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	spaceID := res.Header().Get("X-Object-Id")

	// Add allowed bookers
	approverIDs := []string{group1ID, group2ID}
	approverIDsJson, _ := json.Marshal(approverIDs)
	req = NewHTTPRequest("PUT", "/location/"+locationID+"/space/"+spaceID+"/allowedbooker", loginResponse.UserID, bytes.NewBuffer(approverIDsJson))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	// List allowed bookers
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/"+spaceID+"/allowedbooker", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []*GetGroupResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 2, len(resBody))
	CheckTestString(t, group1ID, resBody[0].ID)
	CheckTestString(t, group2ID, resBody[1].ID)

	// Remove one allowed booker
	approverIDs = []string{group1ID}
	approverIDsJson, _ = json.Marshal(approverIDs)
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/"+spaceID+"/allowedbooker/remove", loginResponse.UserID, bytes.NewBuffer(approverIDsJson))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	// List allowed bookers
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/"+spaceID+"/allowedbooker", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody2 []*GetGroupResponse
	json.Unmarshal(res.Body.Bytes(), &resBody2)
	CheckTestInt(t, 1, len(resBody2))
	CheckTestString(t, group2ID, resBody2[0].ID)
}

func TestSpacesBulkUpdate(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	// Create location
	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locationID := res.Header().Get("X-Object-Id")

	// 1. Create 3 spaces
	payload = `{
		"creates": [
			{"name": "H1", "x": 50, "y": 110, "width": 210, "height": 310, "rotation": 90},
			{"name": "H2", "x": 60, "y": 120, "width": 220, "height": 320, "rotation": 91},
			{"name": "H3", "x": 70, "y": 130, "width": 230, "height": 330, "rotation": 92}
		]
	}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/bulk", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody *BulkUpdateResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 3, len(resBody.Creates))
	CheckTestInt(t, 0, len(resBody.Updates))
	CheckTestInt(t, 0, len(resBody.Deletes))
	CheckTestBool(t, true, resBody.Creates[0].Success)
	CheckTestBool(t, true, resBody.Creates[1].Success)
	CheckTestBool(t, true, resBody.Creates[2].Success)

	// 2. Create, Update, Delete
	payload = `{
		"creates": [
			{"name": "H4", "x": 80, "y": 140, "width": 240, "height": 340, "rotation": 93}
		],
		"updates": [
			{"id": "` + resBody.Creates[1].ID + `", "name": "H2.2", "x": 69, "y": 129, "width": 229, "height": 329, "rotation": 99}
		],
		"deleteIds": [
			"` + resBody.Creates[2].ID + `"
		]
	}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/bulk", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestInt(t, 1, len(resBody.Creates))
	CheckTestInt(t, 1, len(resBody.Updates))
	CheckTestInt(t, 1, len(resBody.Deletes))
	CheckTestBool(t, true, resBody.Creates[0].Success)
	CheckTestBool(t, true, resBody.Updates[0].Success)
	CheckTestBool(t, true, resBody.Deletes[0].Success)

	// 3. List
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody2 []*GetSpaceResponse
	json.Unmarshal(res.Body.Bytes(), &resBody2)
	if len(resBody2) != 3 {
		t.Fatalf("Expected array with 3 elements")
	}
	CheckTestString(t, "H1", resBody2[0].Name)
	CheckTestString(t, "H2.2", resBody2[1].Name)
	CheckTestString(t, "H4", resBody2[2].Name)
}

func TestSpacesList(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	locationID, _, _, _ := createTestSpaces(t, loginResponse)

	req := NewHTTPRequest("GET", "/location/"+locationID+"/space/", loginResponse.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []*GetSpaceResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if len(resBody) != 3 {
		t.Fatalf("Expected array with 3 elements")
	}
	CheckTestString(t, "H234", resBody[0].Name)
	CheckTestString(t, "H235", resBody[1].Name)
	CheckTestString(t, "H236", resBody[2].Name)
}

func TestSpacesListReturnsSpaceType(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	spaceType := &SpaceType{
		OrganizationID: org.ID,
		Name:           "Work seat",
		BookingMode:    SpaceTypeBookingModeFixedSlots,
		Enabled:        true,
	}
	CheckTestBool(t, true, GetSpaceTypeRepository().Create(spaceType) == nil)
	CheckTestBool(t, true, GetSpaceTypeRepository().CreateSlot(&SpaceTypeSlot{
		SpaceTypeID: spaceType.ID,
		Label:       "Morning",
		StartTime:   "08:00",
		EndTime:     "12:00",
		Enabled:     true,
		SortOrder:   1,
	}) == nil)
	CheckTestBool(t, true, GetSpaceTypeRepository().CreateSlot(&SpaceTypeSlot{
		SpaceTypeID: spaceType.ID,
		Label:       "Disabled",
		StartTime:   "18:00",
		EndTime:     "19:00",
		Enabled:     false,
		SortOrder:   2,
	}) == nil)

	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locationID := res.Header().Get("X-Object-Id")

	payload = `{"name": "A1", "x": 50, "y": 100, "width": 200, "height": 300, "rotation": 90, "enabled": true, "spaceTypeId": "` + spaceType.ID + `"}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var list []spaceResponseWithType
	json.Unmarshal(res.Body.Bytes(), &list)
	CheckTestInt(t, 1, len(list))
	CheckTestString(t, spaceType.ID, list[0].SpaceTypeID)
	CheckTestString(t, "Work seat", list[0].SpaceType.Name)
	CheckTestString(t, SpaceTypeBookingModeFixedSlots, list[0].SpaceType.BookingMode)
	CheckTestInt(t, 1, len(list[0].SpaceType.Slots))
	CheckTestString(t, "Morning", list[0].SpaceType.Slots[0].Label)
}

func TestSpaceAvailabilityReturnsSpaceType(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	spaceType := &SpaceType{
		OrganizationID:     org.ID,
		Name:               "Meeting room",
		BookingMode:        SpaceTypeBookingModeFlexibleTime,
		MinDurationMinutes: 30,
		Enabled:            true,
	}
	CheckTestBool(t, true, GetSpaceTypeRepository().Create(spaceType) == nil)

	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locationID := res.Header().Get("X-Object-Id")

	payload = `{"name": "Meeting A", "x": 50, "y": 100, "width": 200, "height": 300, "rotation": 90, "enabled": true, "spaceTypeId": "` + spaceType.ID + `"}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	enter := "2030-09-01T08:30:00+02:00"
	leave := "2030-09-01T17:00:00+02:00"
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/availability?enter="+url.QueryEscape(enter)+"&leave="+url.QueryEscape(leave), loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var list []spaceResponseWithType
	json.Unmarshal(res.Body.Bytes(), &list)
	CheckTestInt(t, 1, len(list))
	CheckTestString(t, spaceType.ID, list[0].SpaceTypeID)
	CheckTestString(t, "Meeting room", list[0].SpaceType.Name)
	CheckTestString(t, SpaceTypeBookingModeFlexibleTime, list[0].SpaceType.BookingMode)
	CheckTestInt(t, 30, list[0].SpaceType.MinDurationMinutes)
}

func TestSpacesAvailabilityOuter(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, "5000")

	locationID, spaceID, _, _ := createTestSpaces(t, loginResponse)

	// Create booking
	payload := "{\"spaceId\": \"" + spaceID + "\", \"enter\": \"2030-09-01T06:00:00+02:00\", \"leave\": \"2030-09-01T18:00:00+02:00\"}"
	req := NewHTTPRequest("POST", "/booking/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	// Check
	enter := "2030-09-01T08:30:00+02:00"
	leave := "2030-09-01T17:00:00+02:00"
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/availability?enter="+url.QueryEscape(enter)+"&leave="+url.QueryEscape(leave), loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []*GetSpaceResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if len(resBody) != 3 {
		t.Fatalf("Expected array with 3 elements")
	}
	CheckTestString(t, "H234", resBody[0].Name)
	CheckTestString(t, "H235", resBody[1].Name)
	CheckTestString(t, "H236", resBody[2].Name)
	CheckTestBool(t, false, resBody[0].Available)
	CheckTestBool(t, true, resBody[1].Available)
	CheckTestBool(t, true, resBody[2].Available)
}

func TestSpacesAvailabilityInner(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, "5000")

	locationID, spaceID, _, _ := createTestSpaces(t, loginResponse)

	// Create booking
	payload := "{\"spaceId\": \"" + spaceID + "\", \"enter\": \"2030-09-01T09:00:00+02:00\", \"leave\": \"2030-09-01T11:00:00+02:00\"}"
	req := NewHTTPRequest("POST", "/booking/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	// Check
	enter := "2030-09-01T08:30:00+02:00"
	leave := "2030-09-01T17:00:00+02:00"
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/availability?enter="+url.QueryEscape(enter)+"&leave="+url.QueryEscape(leave), loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []*GetSpaceResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if len(resBody) != 3 {
		t.Fatalf("Expected array with 3 elements")
	}
	CheckTestString(t, "H234", resBody[0].Name)
	CheckTestString(t, "H235", resBody[1].Name)
	CheckTestString(t, "H236", resBody[2].Name)
	CheckTestBool(t, false, resBody[0].Available)
	CheckTestBool(t, true, resBody[1].Available)
	CheckTestBool(t, true, resBody[2].Available)
}

func TestSpacesAvailabilityStart(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, "5000")

	locationID, spaceID, _, _ := createTestSpaces(t, loginResponse)

	// Create booking
	payload := "{\"spaceId\": \"" + spaceID + "\", \"enter\": \"2030-09-01T07:00:00Z\", \"leave\": \"2030-09-01T09:00:00Z\"}"
	req := NewHTTPRequest("POST", "/booking/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	// Check
	enter := "2030-09-01T08:30:00+02:00"
	leave := "2030-09-01T17:00:00+02:00"
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/availability?enter="+url.QueryEscape(enter)+"&leave="+url.QueryEscape(leave), loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []*GetSpaceResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if len(resBody) != 3 {
		t.Fatalf("Expected array with 3 elements")
	}
	CheckTestString(t, "H234", resBody[0].Name)
	CheckTestString(t, "H235", resBody[1].Name)
	CheckTestString(t, "H236", resBody[2].Name)
	CheckTestBool(t, false, resBody[0].Available)
	CheckTestBool(t, true, resBody[1].Available)
	CheckTestBool(t, true, resBody[2].Available)
}

func TestSpacesAvailabilityEnd(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, "5000")

	locationID, spaceID, _, _ := createTestSpaces(t, loginResponse)

	// Create booking
	payload := "{\"spaceId\": \"" + spaceID + "\", \"enter\": \"2030-09-01T16:30:00+02:00\", \"leave\": \"2030-09-01T17:30:00+02:00\"}"
	req := NewHTTPRequest("POST", "/booking/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	// Check
	enter := "2030-09-01T08:30:00+02:00"
	leave := "2030-09-01T17:00:00+02:00"
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/availability?enter="+url.QueryEscape(enter)+"&leave="+url.QueryEscape(leave), loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []*GetSpaceResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if len(resBody) != 3 {
		t.Fatalf("Expected array with 3 elements")
	}
	CheckTestString(t, "H234", resBody[0].Name)
	CheckTestString(t, "H235", resBody[1].Name)
	CheckTestString(t, "H236", resBody[2].Name)
	CheckTestBool(t, false, resBody[0].Available)
	CheckTestBool(t, true, resBody[1].Available)
	CheckTestBool(t, true, resBody[2].Available)
}

func TestSpacesAvailabilityNoBookings(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)

	locationID, _, _, _ := createTestSpaces(t, loginResponse)

	enter := "2020-09-01T08:30:00+02:00"
	leave := "2020-09-01T17:00:00+02:00"
	req := NewHTTPRequest("GET", "/location/"+locationID+"/space/availability?enter="+url.QueryEscape(enter)+"&leave="+url.QueryEscape(leave), loginResponse.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []*GetSpaceResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if len(resBody) != 3 {
		t.Fatalf("Expected array with 3 elements")
	}
	CheckTestString(t, "H234", resBody[0].Name)
	CheckTestString(t, "H235", resBody[1].Name)
	CheckTestString(t, "H236", resBody[2].Name)
	CheckTestBool(t, true, resBody[0].Available)
	CheckTestBool(t, true, resBody[1].Available)
	CheckTestBool(t, true, resBody[2].Available)
}

func TestSpaceDayStatusReturnsAvailablePartiallyBookedAndFull(t *testing.T) {
	_, booker, loginResponse, location, availableSpace, partialSpace, fullSpace := setupSpaceDayStatusTest(t, true)
	createSpaceDayStatusBooking(t, booker, partialSpace, "2030-09-01T09:00:00+02:00", "2030-09-01T10:00:00+02:00", "partial")
	createSpaceDayStatusBooking(t, booker, fullSpace, "2030-09-01T08:00:00+02:00", "2030-09-01T11:59:59+02:00", "full")

	resBody := requestSpaceDayStatus(t, loginResponse, location.ID)

	available := findSpaceDayStatus(t, resBody, availableSpace.ID)
	CheckTestString(t, "available", available.Status)
	CheckTestInt(t, 0, available.BookedMinutes)
	CheckTestInt(t, 240, available.OfficeMinutes)
	CheckTestString(t, "2030-09-01T08:00:00+02:00", available.OfficeStart.Format(time.RFC3339))
	CheckTestString(t, "2030-09-01T12:00:00+02:00", available.OfficeEnd.Format(time.RFC3339))
	CheckTestInt(t, 0, len(available.Bookings))

	partial := findSpaceDayStatus(t, resBody, partialSpace.ID)
	CheckTestString(t, "partially_booked", partial.Status)
	CheckTestInt(t, 60, partial.BookedMinutes)
	CheckTestInt(t, 1, len(partial.Bookings))
	CheckTestString(t, booker.ID, partial.Bookings[0].UserID)
	CheckTestString(t, booker.Email, partial.Bookings[0].UserEmail)

	full := findSpaceDayStatus(t, resBody, fullSpace.ID)
	CheckTestString(t, "full", full.Status)
	CheckTestInt(t, 240, full.BookedMinutes)
}

func TestSpaceDayStatusClampsBookingsToOfficeHours(t *testing.T) {
	_, booker, loginResponse, location, space, _, _ := setupSpaceDayStatusTest(t, true)
	createSpaceDayStatusBooking(t, booker, space, "2030-09-01T06:00:00+02:00", "2030-09-01T14:00:00+02:00", "clamped")

	resBody := requestSpaceDayStatus(t, loginResponse, location.ID)

	status := findSpaceDayStatus(t, resBody, space.ID)
	CheckTestString(t, "full", status.Status)
	CheckTestInt(t, 240, status.BookedMinutes)
	CheckTestInt(t, 1, len(status.Bookings))
	CheckTestString(t, "2030-09-01T08:00:00+02:00", status.Bookings[0].Enter.Format(time.RFC3339))
	CheckTestString(t, "2030-09-01T12:00:00+02:00", status.Bookings[0].Leave.Format(time.RFC3339))
	CheckTestString(t, "clamped", status.Bookings[0].Subject)
}

func TestSpaceDayStatusMergesOverlappingBookingsBeforeTotalingMinutes(t *testing.T) {
	_, booker, loginResponse, location, space, _, _ := setupSpaceDayStatusTest(t, true)
	createSpaceDayStatusBooking(t, booker, space, "2030-09-01T09:00:00+02:00", "2030-09-01T11:00:00+02:00", "first")
	createSpaceDayStatusBooking(t, booker, space, "2030-09-01T10:00:00+02:00", "2030-09-01T12:00:00+02:00", "second")

	resBody := requestSpaceDayStatus(t, loginResponse, location.ID)

	status := findSpaceDayStatus(t, resBody, space.ID)
	CheckTestString(t, "partially_booked", status.Status)
	CheckTestInt(t, 180, status.BookedMinutes)
	CheckTestInt(t, 2, len(status.Bookings))
}

func TestSpaceDayStatusFixedSlotsWithNoRemainingSlotReturnsFull(t *testing.T) {
	org, booker, loginResponse, location, _, _, space := setupSpaceDayStatusTest(t, true)
	assignSpaceDayStatusSpaceType(t, org, space, SpaceTypeBookingModeFixedSlots, 0, []*SpaceTypeSlot{
		{Label: "Morning", StartTime: "08:00", EndTime: "09:00", Enabled: true, SortOrder: 1},
		{Label: "Late morning", StartTime: "10:00", EndTime: "11:00", Enabled: true, SortOrder: 2},
	})
	createSpaceDayStatusBooking(t, booker, space, "2030-09-01T08:00:00+02:00", "2030-09-01T09:00:00+02:00", "morning")
	createSpaceDayStatusBooking(t, booker, space, "2030-09-01T10:00:00+02:00", "2030-09-01T11:00:00+02:00", "late")

	resBody := requestSpaceDayStatus(t, loginResponse, location.ID)

	status := findSpaceDayStatus(t, resBody, space.ID)
	CheckTestString(t, "full", status.Status)
	CheckTestInt(t, 120, status.BookedMinutes)
	CheckTestInt(t, 2, len(status.Bookings))
}

func TestSpaceDayStatusFlexibleTimeWithoutRemainingGapReturnsFull(t *testing.T) {
	org, booker, loginResponse, location, _, _, space := setupSpaceDayStatusTest(t, true)
	assignSpaceDayStatusSpaceType(t, org, space, SpaceTypeBookingModeFlexibleTime, 90, nil)
	createSpaceDayStatusBooking(t, booker, space, "2030-09-01T08:00:00+02:00", "2030-09-01T09:30:00+02:00", "first")
	createSpaceDayStatusBooking(t, booker, space, "2030-09-01T10:45:00+02:00", "2030-09-01T12:00:00+02:00", "second")

	resBody := requestSpaceDayStatus(t, loginResponse, location.ID)

	status := findSpaceDayStatus(t, resBody, space.ID)
	CheckTestString(t, "full", status.Status)
	CheckTestInt(t, 165, status.BookedMinutes)
	CheckTestInt(t, 2, len(status.Bookings))
}

func TestSpaceDayStatusWithoutSpaceTypeUsesLegacyTimeRangeBookability(t *testing.T) {
	org, booker, loginResponse, location, _, partialSpace, fullSpace := setupSpaceDayStatusTest(t, true)
	CheckTestBool(t, true, GetSettingsRepository().Set(org.ID, SettingMinBookingDurationHours.Name, "1") == nil)
	createSpaceDayStatusBooking(t, booker, partialSpace, "2030-09-01T08:00:00+02:00", "2030-09-01T09:00:00+02:00", "partial")
	createSpaceDayStatusBooking(t, booker, fullSpace, "2030-09-01T08:00:00+02:00", "2030-09-01T11:30:00+02:00", "full")

	resBody := requestSpaceDayStatus(t, loginResponse, location.ID)

	partial := findSpaceDayStatus(t, resBody, partialSpace.ID)
	CheckTestString(t, "partially_booked", partial.Status)
	CheckTestInt(t, 60, partial.BookedMinutes)

	full := findSpaceDayStatus(t, resBody, fullSpace.ID)
	CheckTestString(t, "full", full.Status)
	CheckTestInt(t, 210, full.BookedMinutes)
}

func TestSpaceDayStatusWithoutSpaceTypeUsesFallbackThirtyMinuteMinimumDuration(t *testing.T) {
	org, booker, loginResponse, location, thirtyMinuteSpace, tooShortSpace, _ := setupSpaceDayStatusTest(t, true)
	CheckTestBool(t, true, GetSettingsRepository().Set(org.ID, SettingMinBookingDurationHours.Name, "0") == nil)
	createSpaceDayStatusBooking(t, booker, thirtyMinuteSpace, "2030-09-01T08:00:00+02:00", "2030-09-01T11:30:00+02:00", "thirty-min-gap")
	createSpaceDayStatusBooking(t, booker, tooShortSpace, "2030-09-01T08:00:00+02:00", "2030-09-01T11:31:00+02:00", "twenty-nine-min-gap")

	resBody := requestSpaceDayStatus(t, loginResponse, location.ID)

	thirtyMinute := findSpaceDayStatus(t, resBody, thirtyMinuteSpace.ID)
	CheckTestString(t, "partially_booked", thirtyMinute.Status)
	CheckTestInt(t, 210, thirtyMinute.BookedMinutes)

	tooShort := findSpaceDayStatus(t, resBody, tooShortSpace.ID)
	CheckTestString(t, "full", tooShort.Status)
	CheckTestInt(t, 211, tooShort.BookedMinutes)
}

func TestSpaceDayStatusFlexibleTimeWithZeroMinimumDoesNotUseLegacyThirtyMinuteFallback(t *testing.T) {
	org, booker, loginResponse, location, _, _, space := setupSpaceDayStatusTest(t, true)
	CheckTestBool(t, true, GetSettingsRepository().Set(org.ID, SettingMinBookingDurationHours.Name, "0") == nil)
	assignSpaceDayStatusSpaceType(t, org, space, SpaceTypeBookingModeFlexibleTime, 0, nil)
	createSpaceDayStatusBooking(t, booker, space, "2030-09-01T08:00:00+02:00", "2030-09-01T11:31:00+02:00", "twenty-nine-min-gap")

	resBody := requestSpaceDayStatus(t, loginResponse, location.ID)

	status := findSpaceDayStatus(t, resBody, space.ID)
	CheckTestString(t, "partially_booked", status.Status)
	CheckTestInt(t, 211, status.BookedMinutes)
}

func TestSpaceDayStatusCurrentDayFixedSlotAlreadyStartedReturnsFull(t *testing.T) {
	org, _, loginResponse, location, space := setupCurrentDaySpaceDayStatusTest(t, "UTC", "00:00", "23:59")
	assignSpaceDayStatusSpaceType(t, org, space, SpaceTypeBookingModeFixedSlots, 0, []*SpaceTypeSlot{
		{Label: "All day", StartTime: "00:00", EndTime: "23:59", Enabled: true, SortOrder: 1},
	})

	resBody := requestSpaceDayStatusForDate(t, loginResponse, location.ID, time.Now().UTC().Format("2006-01-02"))

	status := findSpaceDayStatus(t, resBody, space.ID)
	CheckTestString(t, "full", status.Status)
	CheckTestInt(t, 0, status.BookedMinutes)
}

func TestSpaceDayStatusCurrentDayFlexibleTimeWithoutRemainingTimeReturnsFull(t *testing.T) {
	org, _, loginResponse, location, space := setupCurrentDaySpaceDayStatusTest(t, "UTC", "00:00", "23:59")
	assignSpaceDayStatusSpaceType(t, org, space, SpaceTypeBookingModeFlexibleTime, 24*60, nil)

	resBody := requestSpaceDayStatusForDate(t, loginResponse, location.ID, time.Now().UTC().Format("2006-01-02"))

	status := findSpaceDayStatus(t, resBody, space.ID)
	CheckTestString(t, "full", status.Status)
	CheckTestInt(t, 0, status.BookedMinutes)
}

func TestSpaceDayStatusCurrentDayWithoutSpaceTypeTooShortRemainingGapReturnsFull(t *testing.T) {
	now := time.Now().UTC()
	org, _, loginResponse, location, space := setupCurrentDaySpaceDayStatusTest(
		t,
		"UTC",
		"00:00",
		now.Add(29*time.Minute).Format("15:04"),
	)
	CheckTestBool(t, true, GetSettingsRepository().Set(org.ID, SettingMinBookingDurationHours.Name, "0") == nil)

	resBody := requestSpaceDayStatusForDate(t, loginResponse, location.ID, now.Format("2006-01-02"))

	status := findSpaceDayStatus(t, resBody, space.ID)
	CheckTestString(t, "full", status.Status)
	CheckTestInt(t, 0, status.BookedMinutes)
}

func TestSpaceDayStatusCurrentDayWithoutSpaceTypeTooShortRemainingGapReturnsFullWithDailyBasisBooking(t *testing.T) {
	now := time.Now().UTC()
	org, _, loginResponse, location, space := setupCurrentDaySpaceDayStatusTest(
		t,
		"UTC",
		"00:00",
		now.Add(29*time.Minute).Format("15:04"),
	)
	CheckTestBool(t, true, GetSettingsRepository().Set(org.ID, SettingDailyBasisBooking.Name, "1") == nil)
	CheckTestBool(t, true, GetSettingsRepository().Set(org.ID, SettingMinBookingDurationHours.Name, "0") == nil)

	resBody := requestSpaceDayStatusForDate(t, loginResponse, location.ID, now.Format("2006-01-02"))

	status := findSpaceDayStatus(t, resBody, space.ID)
	CheckTestString(t, "full", status.Status)
	CheckTestInt(t, 0, status.BookedMinutes)
}

func TestSpaceDayStatusCurrentDayUsesLocationTimezoneForClipping(t *testing.T) {
	timezone, localNow := pickNonUTCCurrentDayTimezoneForSpaceDayStatusTest(t)
	org, _, loginResponse, location, space := setupCurrentDaySpaceDayStatusTest(
		t,
		timezone,
		"00:00",
		localNow.Add(29*time.Minute).Format("15:04"),
	)
	CheckTestBool(t, true, GetSettingsRepository().Set(org.ID, SettingMinBookingDurationHours.Name, "0") == nil)

	resBody := requestSpaceDayStatusForDate(t, loginResponse, location.ID, localNow.Format("2006-01-02"))

	status := findSpaceDayStatus(t, resBody, space.ID)
	CheckTestString(t, "full", status.Status)
	CheckTestInt(t, 0, status.BookedMinutes)
}

func TestSpaceDayStatusMissingOfficeSettingsReturnsBadRequest(t *testing.T) {
	_, _, loginResponse, location, _, _, _ := setupSpaceDayStatusTest(t, false)

	req := NewHTTPRequest("GET", "/location/"+location.ID+"/space/day-status?date=2030-09-01", loginResponse.UserID, nil)
	res := ExecuteTestRequest(req)

	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)
	CheckTestString(t, "1013", res.Header().Get("X-Error-Code"))
}

func setupSpaceDayStatusTest(t *testing.T, withOfficeSettings bool) (*Organization, *User, *LoginResponse, *Location, *Space, *Space, *Space) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	booker := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(booker.ID)
	if withOfficeSettings {
		if err := GetOfficeSettingsRepository().Upsert(&OfficeSettings{WorkStartTime: "08:00", WorkEndTime: "12:00"}); err != nil {
			t.Fatalf("Could not create office settings: %v", err)
		}
	} else {
		GetDatabase().DB().Exec("DELETE FROM office_settings")
	}
	location := &Location{
		OrganizationID: org.ID,
		Name:           "Location 1",
		Timezone:       "Europe/Berlin",
		Enabled:        true,
	}
	if err := GetLocationRepository().Create(location); err != nil {
		t.Fatalf("Could not create location: %v", err)
	}
	availableSpace := createSpaceDayStatusSpace(t, location, "Available")
	partialSpace := createSpaceDayStatusSpace(t, location, "Partial")
	fullSpace := createSpaceDayStatusSpace(t, location, "Full")
	return org, booker, loginResponse, location, availableSpace, partialSpace, fullSpace
}

func setupCurrentDaySpaceDayStatusTest(t *testing.T, timezone, officeStart, officeEnd string) (*Organization, *User, *LoginResponse, *Location, *Space) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	booker := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(booker.ID)
	if err := GetOfficeSettingsRepository().Upsert(&OfficeSettings{WorkStartTime: officeStart, WorkEndTime: officeEnd}); err != nil {
		t.Fatalf("Could not create office settings: %v", err)
	}
	location := &Location{
		OrganizationID: org.ID,
		Name:           "Today location",
		Timezone:       timezone,
		Enabled:        true,
	}
	if err := GetLocationRepository().Create(location); err != nil {
		t.Fatalf("Could not create location: %v", err)
	}
	space := createSpaceDayStatusSpace(t, location, "Today space")
	return org, booker, loginResponse, location, space
}

func createSpaceDayStatusSpace(t *testing.T, location *Location, name string) *Space {
	space := &Space{
		LocationID: location.ID,
		Name:       name,
		Enabled:    true,
	}
	if err := GetSpaceRepository().Create(space); err != nil {
		t.Fatalf("Could not create space: %v", err)
	}
	return space
}

func assignSpaceDayStatusSpaceType(t *testing.T, org *Organization, space *Space, mode string, minDurationMinutes int, slots []*SpaceTypeSlot) *SpaceType {
	spaceType := &SpaceType{
		OrganizationID:     org.ID,
		Name:               space.Name + " type",
		BookingMode:        mode,
		MinDurationMinutes: minDurationMinutes,
		Enabled:            true,
	}
	if err := GetSpaceTypeRepository().Create(spaceType); err != nil {
		t.Fatalf("Could not create space type: %v", err)
	}
	for _, slot := range slots {
		slot.SpaceTypeID = spaceType.ID
		if err := GetSpaceTypeRepository().CreateSlot(slot); err != nil {
			t.Fatalf("Could not create space type slot: %v", err)
		}
	}
	space.SpaceTypeID = spaceType.ID
	if err := GetSpaceRepository().Update(space); err != nil {
		t.Fatalf("Could not update space type: %v", err)
	}
	return spaceType
}

func createSpaceDayStatusBooking(t *testing.T, user *User, space *Space, enter, leave, subject string) *Booking {
	booking := &Booking{
		UserID:   user.ID,
		SpaceID:  space.ID,
		Enter:    mustParseSpaceDayStatusTime(t, enter),
		Leave:    mustParseSpaceDayStatusTime(t, leave),
		Approved: true,
		Subject:  subject,
	}
	if err := GetBookingRepository().Create(booking); err != nil {
		t.Fatalf("Could not create booking: %v", err)
	}
	return booking
}

func mustParseSpaceDayStatusTime(t *testing.T, value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("Could not parse time %s: %v", value, err)
	}
	return parsed
}

func pickNonUTCCurrentDayTimezoneForSpaceDayStatusTest(t *testing.T) (string, time.Time) {
	t.Helper()
	utcNow := time.Now().UTC()
	utcDate := utcNow.Format("2006-01-02")
	candidates := []string{
		"Pacific/Kiritimati",
		"Pacific/Auckland",
		"Pacific/Apia",
		"Pacific/Pago_Pago",
		"Pacific/Honolulu",
	}
	for _, timezone := range candidates {
		location, err := time.LoadLocation(timezone)
		if err != nil {
			t.Fatalf("Could not load timezone %s: %v", timezone, err)
		}
		localNow := utcNow.In(location)
		if localNow.Format("2006-01-02") == utcDate {
			continue
		}
		localEnd := localNow.Add(29 * time.Minute)
		if localEnd.Year() != localNow.Year() || localEnd.Month() != localNow.Month() || localEnd.Day() != localNow.Day() {
			continue
		}
		return timezone, localNow
	}
	t.Fatalf("Could not find non-UTC timezone with shifted current day and stable remaining window")
	return "", time.Time{}
}

func requestSpaceDayStatus(t *testing.T, loginResponse *LoginResponse, locationID string) []*spaceDayStatusTestResponse {
	resBody := requestSpaceDayStatusForDate(t, loginResponse, locationID, "2030-09-01")
	CheckTestInt(t, 3, len(resBody))
	return resBody
}

func requestSpaceDayStatusForDate(t *testing.T, loginResponse *LoginResponse, locationID, date string) []*spaceDayStatusTestResponse {
	req := NewHTTPRequest("GET", "/location/"+locationID+"/space/day-status?date="+date, loginResponse.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []*spaceDayStatusTestResponse
	if err := json.Unmarshal(res.Body.Bytes(), &resBody); err != nil {
		t.Fatalf("Could not parse day status response: %v", err)
	}
	return resBody
}

func findSpaceDayStatus(t *testing.T, list []*spaceDayStatusTestResponse, spaceID string) *spaceDayStatusTestResponse {
	for _, item := range list {
		if item.SpaceID == spaceID {
			return item
		}
	}
	t.Fatalf("Could not find day status for space %s", spaceID)
	return nil
}

func createTestSpaces(t *testing.T, loginResponse *LoginResponse) (lID, s1ID, s2ID, s3ID string) {
	// Create location
	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locationID := res.Header().Get("X-Object-Id")

	// Create #1
	payload = `{"name": "H234", "x": 50, "y": 100, "width": 200, "height": 300, "rotation": 90}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	space1ID := res.Header().Get("X-Object-Id")

	// Create #2
	payload = `{"name": "H236", "x": 50, "y": 100, "width": 200, "height": 300, "rotation": 90}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	space2ID := res.Header().Get("X-Object-Id")

	// Create #3
	payload = `{"name": "H235", "x": 50, "y": 100, "width": 200, "height": 300, "rotation": 90}`
	req = NewHTTPRequest("POST", "/location/"+locationID+"/space/", loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	space3ID := res.Header().Get("X-Object-Id")

	return locationID, space1ID, space2ID, space3ID
}

func TestSpacesAvailabilitySingleSpace(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(user.ID)
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, "5000")

	locationID, spaceID1, spaceID2, _ := createTestSpaces(t, loginResponse)

	// Create booking
	payload := "{\"spaceId\": \"" + spaceID1 + "\", \"enter\": \"2030-09-01T07:00:00Z\", \"leave\": \"2030-09-01T09:00:00Z\"}"
	req := NewHTTPRequest("POST", "/booking/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	// Check #1
	enter := "2030-09-01T08:30:00+02:00"
	leave := "2030-09-01T17:00:00+02:00"
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/"+spaceID1+"/availability?enter="+url.QueryEscape(enter)+"&leave="+url.QueryEscape(leave), loginResponse.UserID, nil)

	/*
		payload = `{"enter": "2030-09-01T08:30:00Z", "leave": "2030-09-01T17:00:00Z"}`
		req = NewHTTPRequest("POST", "/location/"+locationID+"/space/"+spaceID1+"/availability", loginResponse.UserID, bytes.NewBufferString(payload))
	*/
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []*GetSpaceResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if len(resBody) != 1 {
		t.Fatalf("Expected array with 1 element")
	}
	CheckTestString(t, "H234", resBody[0].Name)
	CheckTestBool(t, false, resBody[0].Available)

	// Check #2
	enter = "2030-09-01T08:30:00+02:00"
	leave = "2030-09-01T17:00:00+02:00"
	req = NewHTTPRequest("GET", "/location/"+locationID+"/space/"+spaceID2+"/availability?enter="+url.QueryEscape(enter)+"&leave="+url.QueryEscape(leave), loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody2 []*GetSpaceResponse
	json.Unmarshal(res.Body.Bytes(), &resBody2)
	if len(resBody2) != 1 {
		t.Fatalf("Expected array with 1 element")
	}
	CheckTestString(t, "H236", resBody2[0].Name)
	CheckTestBool(t, true, resBody2[0].Available)
}

func TestSpacesListForbidden(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	org2 := CreateTestOrg("test2.com")
	admin := CreateTestUserOrgAdmin(org)
	user2 := CreateTestUserInOrg(org2)

	// Create location and space in org
	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", admin.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locID := res.Header().Get("X-Object-Id")

	// User from a different org tries to list spaces → 403
	req = NewHTTPRequest("GET", "/location/"+locID+"/space/", user2.ID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestSpacesAvailabilityForbidden(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	org2 := CreateTestOrg("test2.com")
	admin := CreateTestUserOrgAdmin(org)
	user2 := CreateTestUserInOrg(org2)

	// Create location and space in org
	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", admin.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locID := res.Header().Get("X-Object-Id")

	// User from a different org tries to get availability → 403
	enter := url.QueryEscape("2030-09-01T08:30:00+02:00")
	leave := url.QueryEscape("2030-09-01T17:00:00+02:00")
	req = NewHTTPRequest("GET", "/location/"+locID+"/space/availability?enter="+enter+"&leave="+leave, user2.ID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestSpacesBulkUpdateForbidden(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	admin := CreateTestUserOrgAdmin(org)
	user := CreateTestUserInOrg(org)

	// Create location
	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", admin.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locID := res.Header().Get("X-Object-Id")

	// Non-admin tries bulk update → 403
	req = NewHTTPRequest("POST", "/location/"+locID+"/space/bulk", user.ID, bytes.NewBufferString(`{"creates":[],"updates":[],"deleteIds":[]}`))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestSpacesApproverForbidden(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	admin := CreateTestUserOrgAdmin(org)
	user := CreateTestUserInOrg(org)

	// Create location and space
	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", admin.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locID := res.Header().Get("X-Object-Id")

	payload = `{"name": "Space 1", "x": 0, "y": 0, "width": 100, "height": 100, "rotation": 0}`
	req = NewHTTPRequest("POST", "/location/"+locID+"/space/", admin.ID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	spaceID := res.Header().Get("X-Object-Id")

	// Non-admin tries to get approvers → 403
	req = NewHTTPRequest("GET", "/location/"+locID+"/space/"+spaceID+"/approver", user.ID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)

	// Non-admin tries to set approvers → 403
	req = NewHTTPRequest("PUT", "/location/"+locID+"/space/"+spaceID+"/approver", user.ID, bytes.NewBufferString(`[]`))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestSpacesAllowedBookerForbidden(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	admin := CreateTestUserOrgAdmin(org)
	user := CreateTestUserInOrg(org)

	// Create location and space
	payload := `{"name": "Location 1"}`
	req := NewHTTPRequest("POST", "/location/", admin.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	locID := res.Header().Get("X-Object-Id")

	payload = `{"name": "Space 1", "x": 0, "y": 0, "width": 100, "height": 100, "rotation": 0}`
	req = NewHTTPRequest("POST", "/location/"+locID+"/space/", admin.ID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	spaceID := res.Header().Get("X-Object-Id")

	// Non-admin tries to get allowed bookers → 403
	req = NewHTTPRequest("GET", "/location/"+locID+"/space/"+spaceID+"/allowedbooker", user.ID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)

	// Non-admin tries to set allowed bookers → 403
	req = NewHTTPRequest("PUT", "/location/"+locID+"/space/"+spaceID+"/allowedbooker", user.ID, bytes.NewBufferString(`[]`))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestLocationAllowedBookerForbidden(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user1 := CreateTestUserInOrg(org)
	loginResponse1 := LoginTestUser(user1.ID)
	user2 := CreateTestUserInOrg(org)
	loginResponse2 := LoginTestUser(user2.ID)

	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, "5000")

	// create restricted location
	location, _ := CreateTestLocationAndSpace(org)
	group := CreateTestGroup(org, user2)
	GetLocationRepository().ReplaceAllowedBookers(location, []string{group.ID})

	enter := "2030-09-01T08:30:00+02:00"
	leave := "2030-09-01T17:00:00+02:00"

	// Check availability (for user 1)
	req := NewHTTPRequest("GET", "/location/"+location.ID+"/space/availability?enter="+url.QueryEscape(enter)+"&leave="+url.QueryEscape(leave), loginResponse1.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []*GetSpaceAvailabilityResponse
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestBool(t, false, resBody[0].IsAllowed)

	// Check availability (for user 2)
	req = NewHTTPRequest("GET", "/location/"+location.ID+"/space/availability?enter="+url.QueryEscape(enter)+"&leave="+url.QueryEscape(leave), loginResponse2.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	json.Unmarshal(res.Body.Bytes(), &resBody)
	CheckTestBool(t, true, resBody[0].IsAllowed)
}
