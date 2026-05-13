package test

import (
	"testing"

	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/testutil"
)

func TestSpaceTypeRepositoryCRUD(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	repo := GetSpaceTypeRepository()

	spaceType := &SpaceType{
		OrganizationID:     org.ID,
		Name:               "Work seat",
		BookingMode:        SpaceTypeBookingModeFixedSlots,
		MinDurationMinutes: 0,
		Enabled:            true,
	}
	CheckTestBool(t, true, repo.Create(spaceType) == nil)

	slot := &SpaceTypeSlot{
		SpaceTypeID: spaceType.ID,
		Label:       "Morning",
		StartTime:   "08:00",
		EndTime:     "12:00",
		Enabled:     true,
		SortOrder:   1,
	}
	CheckTestBool(t, true, repo.CreateSlot(slot) == nil)

	found, err := repo.GetOne(spaceType.ID)
	CheckTestBool(t, true, err == nil)
	CheckTestString(t, spaceType.ID, found.ID)
	CheckTestString(t, "Work seat", found.Name)
	CheckTestString(t, SpaceTypeBookingModeFixedSlots, found.BookingMode)
	CheckTestInt(t, 0, found.MinDurationMinutes)
	CheckTestBool(t, true, found.Enabled)

	slots, err := repo.GetSlots(spaceType.ID, false)
	CheckTestBool(t, true, err == nil)
	CheckTestInt(t, 1, len(slots))
	CheckTestString(t, slot.ID, slots[0].ID)
	CheckTestString(t, "08:00", slots[0].StartTime)

	spaceType.Name = "Meeting room"
	spaceType.BookingMode = SpaceTypeBookingModeFlexibleTime
	spaceType.MinDurationMinutes = 30
	CheckTestBool(t, true, repo.Update(spaceType) == nil)

	slot.Label = "Full day"
	slot.StartTime = "08:00"
	slot.EndTime = "17:00"
	slot.SortOrder = 2
	CheckTestBool(t, true, repo.UpdateSlot(slot) == nil)

	found, err = repo.GetOne(spaceType.ID)
	CheckTestBool(t, true, err == nil)
	CheckTestString(t, "Meeting room", found.Name)
	CheckTestString(t, SpaceTypeBookingModeFlexibleTime, found.BookingMode)
	CheckTestInt(t, 30, found.MinDurationMinutes)

	slots, err = repo.GetSlots(spaceType.ID, true)
	CheckTestBool(t, true, err == nil)
	CheckTestInt(t, 1, len(slots))
	CheckTestString(t, "Full day", slots[0].Label)
	CheckTestString(t, "17:00", slots[0].EndTime)

	CheckTestBool(t, true, repo.DeleteSlot(slot.ID) == nil)
	slots, err = repo.GetSlots(spaceType.ID, false)
	CheckTestBool(t, true, err == nil)
	CheckTestInt(t, 0, len(slots))

	CheckTestBool(t, true, repo.Delete(spaceType.ID) == nil)
	list, err := repo.GetAll(org.ID)
	CheckTestBool(t, true, err == nil)
	CheckTestInt(t, 0, len(list))
}

func TestSpaceTypeRepositoryRejectsInvalidSlot(t *testing.T) {
	invalidSlot := &SpaceTypeSlot{
		Label:     "Invalid",
		StartTime: "12:00",
		EndTime:   "08:00",
		Enabled:   true,
	}

	CheckTestBool(t, false, ValidateSpaceTypeSlot(invalidSlot) == nil)
}

func TestSpaceTypeDeleteInUseFails(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("test.com")
	location := &Location{OrganizationID: org.ID, Name: "HQ", Enabled: true}
	CheckTestBool(t, true, GetLocationRepository().Create(location) == nil)

	spaceType := &SpaceType{
		OrganizationID: org.ID,
		Name:           "Work seat",
		BookingMode:    SpaceTypeBookingModeFixedSlots,
		Enabled:        true,
	}
	CheckTestBool(t, true, GetSpaceTypeRepository().Create(spaceType) == nil)

	space := &Space{
		Name:        "A1",
		LocationID:  location.ID,
		SpaceTypeID: spaceType.ID,
		Enabled:     true,
	}
	CheckTestBool(t, true, GetSpaceRepository().Create(space) == nil)

	CheckTestBool(t, false, GetSpaceTypeRepository().Delete(spaceType.ID) == nil)
}
