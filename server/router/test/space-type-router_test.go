package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/testutil"
)

type spaceTypeResponse struct {
	ID                 string                  `json:"id"`
	OrganizationID     string                  `json:"organizationId"`
	Name               string                  `json:"name"`
	BookingMode        string                  `json:"bookingMode"`
	MinDurationMinutes int                     `json:"minDurationMinutes"`
	Enabled            bool                    `json:"enabled"`
	Slots              []spaceTypeSlotResponse `json:"slots"`
}

type spaceTypeSlotResponse struct {
	ID          string `json:"id"`
	SpaceTypeID string `json:"spaceTypeId"`
	Label       string `json:"label"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sortOrder"`
}

func TestSpaceTypeCRUD(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	admin := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(admin.ID)

	payload := `{"name":"Work seat","bookingMode":"fixed_slots","minDurationMinutes":0,"enabled":true}`
	req := NewHTTPRequest("POST", "/space-type/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	id := res.Header().Get("X-Object-Id")

	slotPayload := `{"label":"Morning","startTime":"08:00","endTime":"12:00","enabled":true,"sortOrder":1}`
	req = NewHTTPRequest("POST", "/space-type/"+id+"/slot/", loginResponse.UserID, bytes.NewBufferString(slotPayload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	req = NewHTTPRequest("GET", "/space-type/"+id, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var body spaceTypeResponse
	json.Unmarshal(res.Body.Bytes(), &body)
	CheckTestString(t, "Work seat", body.Name)
	CheckTestString(t, SpaceTypeBookingModeFixedSlots, body.BookingMode)
	CheckTestInt(t, 1, len(body.Slots))
	CheckTestString(t, "Morning", body.Slots[0].Label)
	CheckTestString(t, "08:00", body.Slots[0].StartTime)

	payload = `{"name":"Meeting room","bookingMode":"flexible_time","minDurationMinutes":30,"enabled":true}`
	req = NewHTTPRequest("PUT", "/space-type/"+id, loginResponse.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	slotPayload = `{"label":"Full day","startTime":"08:00","endTime":"17:00","enabled":true,"sortOrder":2}`
	req = NewHTTPRequest("PUT", "/space-type/"+id+"/slot/"+body.Slots[0].ID, loginResponse.UserID, bytes.NewBufferString(slotPayload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("GET", "/space-type/", loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var list []spaceTypeResponse
	json.Unmarshal(res.Body.Bytes(), &list)
	CheckTestInt(t, 1, len(list))
	CheckTestString(t, "Meeting room", list[0].Name)

	req = NewHTTPRequest("DELETE", "/space-type/"+id+"/slot/"+body.Slots[0].ID, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	req = NewHTTPRequest("DELETE", "/space-type/"+id, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)
}

func TestSpaceTypeCreateForbiddenForNormalUser(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrg(org)

	payload := `{"name":"Work seat","bookingMode":"fixed_slots","minDurationMinutes":0,"enabled":true}`
	req := NewHTTPRequest("POST", "/space-type/", user.ID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestSpaceTypeRejectsInvalidSlot(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	admin := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(admin.ID)

	payload := `{"name":"Work seat","bookingMode":"fixed_slots","minDurationMinutes":0,"enabled":true}`
	req := NewHTTPRequest("POST", "/space-type/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	id := res.Header().Get("X-Object-Id")

	slotPayload := `{"label":"Invalid","startTime":"12:00","endTime":"08:00","enabled":true,"sortOrder":1}`
	req = NewHTTPRequest("POST", "/space-type/"+id+"/slot/", loginResponse.UserID, bytes.NewBufferString(slotPayload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)
}

func TestSpaceTypeDeleteInUseReturnsBadRequest(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	admin := CreateTestUserOrgAdmin(org)
	loginResponse := LoginTestUser(admin.ID)

	payload := `{"name":"Work seat","bookingMode":"fixed_slots","minDurationMinutes":0,"enabled":true}`
	req := NewHTTPRequest("POST", "/space-type/", loginResponse.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	id := res.Header().Get("X-Object-Id")

	location := &Location{OrganizationID: org.ID, Name: "HQ", Enabled: true}
	CheckTestBool(t, true, GetLocationRepository().Create(location) == nil)
	space := &Space{Name: "A1", LocationID: location.ID, SpaceTypeID: id, Enabled: true}
	CheckTestBool(t, true, GetSpaceRepository().Create(space) == nil)

	req = NewHTTPRequest("DELETE", "/space-type/"+id, loginResponse.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusBadRequest, res.Code)
}
