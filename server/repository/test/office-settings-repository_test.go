package test

import (
	"errors"
	"testing"
	"time"

	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/testutil"
)

func TestOfficeSettingsRepositoryUpsertCreatesAndUpdatesSingleton(t *testing.T) {
	ClearTestDB()
	repo := GetOfficeSettingsRepository()

	settings := &OfficeSettings{WorkStartTime: "08:00", WorkEndTime: "17:00"}
	if err := repo.Upsert(settings); err != nil {
		t.Fatalf("Failed to upsert office settings: %v", err)
	}
	CheckTestInt(t, 1, settings.ID)

	found, err := repo.Get()
	if err != nil {
		t.Fatalf("Failed to get office settings: %v", err)
	}
	CheckTestInt(t, 1, found.ID)
	CheckTestString(t, "08:00", found.WorkStartTime)
	CheckTestString(t, "17:00", found.WorkEndTime)

	updated := &OfficeSettings{ID: 99, WorkStartTime: "09:30", WorkEndTime: "18:45"}
	if err := repo.Upsert(updated); err != nil {
		t.Fatalf("Failed to update office settings: %v", err)
	}
	CheckTestInt(t, 1, updated.ID)

	found, err = repo.Get()
	if err != nil {
		t.Fatalf("Failed to get updated office settings: %v", err)
	}
	CheckTestInt(t, 1, found.ID)
	CheckTestString(t, "09:30", found.WorkStartTime)
	CheckTestString(t, "18:45", found.WorkEndTime)
}

func TestOfficeSettingsRepositoryMissingReturnsConfigurationError(t *testing.T) {
	ClearTestDB()
	repo := GetOfficeSettingsRepository()

	_, err := repo.Get()
	if !errors.Is(err, ErrOfficeSettingsMissing) {
		t.Fatalf("Expected ErrOfficeSettingsMissing, got %v", err)
	}
}

func TestOfficeSettingsRepositoryRejectsInvalidTimes(t *testing.T) {
	ClearTestDB()
	repo := GetOfficeSettingsRepository()

	invalid := []*OfficeSettings{
		nil,
		{WorkStartTime: "8:00", WorkEndTime: "17:00"},
		{WorkStartTime: "08:0", WorkEndTime: "17:00"},
		{WorkStartTime: "24:00", WorkEndTime: "17:00"},
		{WorkStartTime: "08:60", WorkEndTime: "17:00"},
		{WorkStartTime: "17:00", WorkEndTime: "17:00"},
		{WorkStartTime: "18:00", WorkEndTime: "17:00"},
	}

	for _, item := range invalid {
		if err := ValidateOfficeSettings(item); err == nil {
			t.Fatalf("Expected invalid office settings %+v to fail validation", item)
		}
		if err := repo.Upsert(item); err == nil {
			t.Fatalf("Expected invalid office settings %+v to fail upsert", item)
		}
	}
}

func TestOfficeSettingsClockValidationRequiresZeroPaddedHHMM(t *testing.T) {
	valid := []string{"00:00", "08:30", "23:59"}
	for _, value := range valid {
		CheckTestBool(t, true, IsValidOfficeClock(value))
	}

	invalid := []string{"0:00", "8:30", "08:3", "08:030", "24:00", "23:60", "-1:00", "12:00:00", "ab:cd"}
	for _, value := range invalid {
		CheckTestBool(t, false, IsValidOfficeClock(value))
	}
}

func TestOfficeSettingsParseClockUsesDateAndLocation(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("Failed to load location: %v", err)
	}
	date := time.Date(2030, time.September, 1, 14, 22, 33, 44, loc)

	parsed, err := ParseOfficeClockOnDate("08:30", date)
	if err != nil {
		t.Fatalf("Failed to parse office clock: %v", err)
	}

	CheckTestInt(t, 2030, parsed.Year())
	CheckTestInt(t, int(time.September), int(parsed.Month()))
	CheckTestInt(t, 1, parsed.Day())
	CheckTestInt(t, 8, parsed.Hour())
	CheckTestInt(t, 30, parsed.Minute())
	CheckTestInt(t, 0, parsed.Second())
	CheckTestInt(t, 0, parsed.Nanosecond())
	CheckTestString(t, loc.String(), parsed.Location().String())
}
