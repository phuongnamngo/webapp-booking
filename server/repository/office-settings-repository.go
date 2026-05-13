package repository

import (
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrOfficeSettingsMissing = errors.New("office settings are not configured")

type OfficeSettingsRepository struct {
}

type OfficeSettings struct {
	ID            int
	WorkStartTime string
	WorkEndTime   string
}

var officeClockPattern = regexp.MustCompile(`^\d{2}:\d{2}$`)
var officeSettingsRepository *OfficeSettingsRepository
var officeSettingsRepositoryOnce sync.Once

func GetOfficeSettingsRepository() *OfficeSettingsRepository {
	officeSettingsRepositoryOnce.Do(func() {
		officeSettingsRepository = &OfficeSettingsRepository{}
		createOfficeSettingsTable()
	})
	return officeSettingsRepository
}

func (r *OfficeSettingsRepository) RunSchemaUpgrade(curVersion, targetVersion int) {
	if curVersion < 42 {
		createOfficeSettingsTable()
	}
}

func createOfficeSettingsTable() {
	_, err := GetDatabase().DB().Exec("CREATE TABLE IF NOT EXISTS office_settings (" +
		"id INTEGER PRIMARY KEY CHECK (id = 1), " +
		"work_start_time VARCHAR NOT NULL, " +
		"work_end_time VARCHAR NOT NULL)")
	if err != nil {
		panic(err)
	}
}

func (r *OfficeSettingsRepository) Get() (*OfficeSettings, error) {
	e := &OfficeSettings{}
	err := GetDatabase().DB().QueryRow("SELECT id, work_start_time, work_end_time FROM office_settings WHERE id = 1").
		Scan(&e.ID, &e.WorkStartTime, &e.WorkEndTime)
	if err == sql.ErrNoRows {
		return nil, ErrOfficeSettingsMissing
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *OfficeSettingsRepository) Upsert(e *OfficeSettings) error {
	if err := ValidateOfficeSettings(e); err != nil {
		return err
	}
	_, err := GetDatabase().DB().Exec("INSERT INTO office_settings (id, work_start_time, work_end_time) "+
		"VALUES (1, $1, $2) "+
		"ON CONFLICT (id) DO UPDATE SET work_start_time = $1, work_end_time = $2",
		e.WorkStartTime, e.WorkEndTime)
	if err == nil {
		e.ID = 1
	}
	return err
}

func ValidateOfficeSettings(e *OfficeSettings) error {
	if e == nil || !IsValidOfficeClock(e.WorkStartTime) || !IsValidOfficeClock(e.WorkEndTime) || e.WorkStartTime >= e.WorkEndTime {
		return errors.New("invalid office settings")
	}
	return nil
}

func IsValidOfficeClock(value string) bool {
	if !officeClockPattern.MatchString(value) {
		return false
	}
	parts := strings.Split(value, ":")
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return false
	}
	minute, err := strconv.Atoi(parts[1])
	return err == nil && minute >= 0 && minute <= 59
}

func ParseOfficeClockOnDate(value string, date time.Time) (time.Time, error) {
	if !IsValidOfficeClock(value) {
		return time.Time{}, errors.New("invalid office clock")
	}
	parts := strings.Split(value, ":")
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, date.Location()), nil
}
