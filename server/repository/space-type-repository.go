package repository

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/lib/pq"
)

const (
	SpaceTypeBookingModeFlexibleTime = "flexible_time"
	SpaceTypeBookingModeFixedSlots   = "fixed_slots"
)

type SpaceTypeRepository struct {
}

type SpaceType struct {
	ID                 string
	OrganizationID     string
	Name               string
	BookingMode        string
	MinDurationMinutes int
	Enabled            bool
	Slots              []*SpaceTypeSlot
}

type SpaceTypeSlot struct {
	ID          string
	SpaceTypeID string
	Label       string
	StartTime   string
	EndTime     string
	Enabled     bool
	SortOrder   int
}

var spaceTypeRepository *SpaceTypeRepository
var spaceTypeRepositoryOnce sync.Once

func GetSpaceTypeRepository() *SpaceTypeRepository {
	spaceTypeRepositoryOnce.Do(func() {
		spaceTypeRepository = &SpaceTypeRepository{}
		_, err := GetDatabase().DB().Exec("CREATE TABLE IF NOT EXISTS space_types (" +
			"id uuid DEFAULT uuid_generate_v4(), " +
			"organization_id uuid NOT NULL, " +
			"name VARCHAR NOT NULL, " +
			"booking_mode VARCHAR NOT NULL DEFAULT '" + SpaceTypeBookingModeFlexibleTime + "', " +
			"min_duration_minutes INTEGER NOT NULL DEFAULT 0, " +
			"enabled BOOLEAN NOT NULL DEFAULT TRUE, " +
			"PRIMARY KEY (id))")
		if err != nil {
			panic(err)
		}
		_, err = GetDatabase().DB().Exec("CREATE TABLE IF NOT EXISTS space_type_time_slots (" +
			"id uuid DEFAULT uuid_generate_v4(), " +
			"space_type_id uuid NOT NULL, " +
			"label VARCHAR NOT NULL, " +
			"start_time VARCHAR NOT NULL, " +
			"end_time VARCHAR NOT NULL, " +
			"enabled BOOLEAN NOT NULL DEFAULT TRUE, " +
			"sort_order INTEGER NOT NULL DEFAULT 0, " +
			"PRIMARY KEY (id))")
		if err != nil {
			panic(err)
		}
	})
	return spaceTypeRepository
}

func (r *SpaceTypeRepository) RunSchemaUpgrade(curVersion, targetVersion int) {
	if curVersion < 41 {
		if _, err := GetDatabase().DB().Exec("ALTER TABLE spaces ADD COLUMN IF NOT EXISTS space_type_id uuid DEFAULT NULL"); err != nil {
			panic(err)
		}
	}
}

func IsValidSpaceTypeBookingMode(mode string) bool {
	return mode == SpaceTypeBookingModeFlexibleTime || mode == SpaceTypeBookingModeFixedSlots
}

func IsValidSpaceTypeSlotTime(value string) bool {
	if !regexp.MustCompile(`^\d{2}:\d{2}$`).MatchString(value) {
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

func ValidateSpaceTypeSlot(slot *SpaceTypeSlot) error {
	if slot == nil || strings.TrimSpace(slot.Label) == "" || !IsValidSpaceTypeSlotTime(slot.StartTime) || !IsValidSpaceTypeSlotTime(slot.EndTime) || slot.StartTime >= slot.EndTime {
		return errors.New("invalid space type slot")
	}
	return nil
}

func validateSpaceType(e *SpaceType) error {
	if e == nil || strings.TrimSpace(e.Name) == "" || !IsValidSpaceTypeBookingMode(e.BookingMode) || e.MinDurationMinutes < 0 {
		return errors.New("invalid space type")
	}
	return nil
}

func (r *SpaceTypeRepository) Create(e *SpaceType) error {
	if err := validateSpaceType(e); err != nil {
		return err
	}
	var id string
	err := GetDatabase().DB().QueryRow("INSERT INTO space_types "+
		"(organization_id, name, booking_mode, min_duration_minutes, enabled) "+
		"VALUES ($1, $2, $3, $4, $5) "+
		"RETURNING id",
		e.OrganizationID, e.Name, e.BookingMode, e.MinDurationMinutes, e.Enabled).Scan(&id)
	if err != nil {
		return err
	}
	e.ID = id
	return nil
}

func (r *SpaceTypeRepository) GetOne(id string) (*SpaceType, error) {
	e := &SpaceType{}
	err := GetDatabase().DB().QueryRow("SELECT id, organization_id, name, booking_mode, min_duration_minutes, enabled "+
		"FROM space_types "+
		"WHERE id = $1",
		id).Scan(&e.ID, &e.OrganizationID, &e.Name, &e.BookingMode, &e.MinDurationMinutes, &e.Enabled)
	if err != nil {
		return nil, err
	}
	e.Slots, err = r.GetSlots(e.ID, false)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *SpaceTypeRepository) GetAll(organizationID string) ([]*SpaceType, error) {
	var result []*SpaceType
	rows, err := GetDatabase().DB().Query("SELECT id, organization_id, name, booking_mode, min_duration_minutes, enabled "+
		"FROM space_types "+
		"WHERE organization_id = $1 "+
		"ORDER BY name", organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		e := &SpaceType{}
		err = rows.Scan(&e.ID, &e.OrganizationID, &e.Name, &e.BookingMode, &e.MinDurationMinutes, &e.Enabled)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (r *SpaceTypeRepository) GetAllByIDs(ids []string, enabledSlotsOnly bool) ([]*SpaceType, error) {
	var result []*SpaceType
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := GetDatabase().DB().Query("SELECT id, organization_id, name, booking_mode, min_duration_minutes, enabled "+
		"FROM space_types "+
		"WHERE id = ANY($1::uuid[]) "+
		"ORDER BY name", pq.StringArray(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		e := &SpaceType{}
		err = rows.Scan(&e.ID, &e.OrganizationID, &e.Name, &e.BookingMode, &e.MinDurationMinutes, &e.Enabled)
		if err != nil {
			return nil, err
		}
		e.Slots, err = r.GetSlots(e.ID, enabledSlotsOnly)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (r *SpaceTypeRepository) Update(e *SpaceType) error {
	if err := validateSpaceType(e); err != nil {
		return err
	}
	_, err := GetDatabase().DB().Exec("UPDATE space_types SET "+
		"organization_id = $1, "+
		"name = $2, "+
		"booking_mode = $3, "+
		"min_duration_minutes = $4, "+
		"enabled = $5 "+
		"WHERE id = $6",
		e.OrganizationID, e.Name, e.BookingMode, e.MinDurationMinutes, e.Enabled, e.ID)
	return err
}

func (r *SpaceTypeRepository) Delete(id string) error {
	count, err := GetSpaceRepository().CountBySpaceTypeID(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("space type is assigned to one or more spaces")
	}
	if _, err := GetDatabase().DB().Exec("DELETE FROM space_type_time_slots WHERE space_type_id = $1", id); err != nil {
		return err
	}
	_, err = GetDatabase().DB().Exec("DELETE FROM space_types WHERE id = $1", id)
	return err
}

func (r *SpaceTypeRepository) CreateSlot(e *SpaceTypeSlot) error {
	if err := ValidateSpaceTypeSlot(e); err != nil {
		return err
	}
	var id string
	err := GetDatabase().DB().QueryRow("INSERT INTO space_type_time_slots "+
		"(space_type_id, label, start_time, end_time, enabled, sort_order) "+
		"VALUES ($1, $2, $3, $4, $5, $6) "+
		"RETURNING id",
		e.SpaceTypeID, e.Label, e.StartTime, e.EndTime, e.Enabled, e.SortOrder).Scan(&id)
	if err != nil {
		return err
	}
	e.ID = id
	return nil
}

func (r *SpaceTypeRepository) GetSlots(spaceTypeID string, enabledOnly bool) ([]*SpaceTypeSlot, error) {
	var result []*SpaceTypeSlot
	query := "SELECT id, space_type_id, label, start_time, end_time, enabled, sort_order " +
		"FROM space_type_time_slots " +
		"WHERE space_type_id = $1 "
	if enabledOnly {
		query += "AND enabled IS TRUE "
	}
	query += "ORDER BY sort_order, start_time, label"

	rows, err := GetDatabase().DB().Query(query, spaceTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		e := &SpaceTypeSlot{}
		err = rows.Scan(&e.ID, &e.SpaceTypeID, &e.Label, &e.StartTime, &e.EndTime, &e.Enabled, &e.SortOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (r *SpaceTypeRepository) UpdateSlot(e *SpaceTypeSlot) error {
	if err := ValidateSpaceTypeSlot(e); err != nil {
		return err
	}
	_, err := GetDatabase().DB().Exec("UPDATE space_type_time_slots SET "+
		"space_type_id = $1, "+
		"label = $2, "+
		"start_time = $3, "+
		"end_time = $4, "+
		"enabled = $5, "+
		"sort_order = $6 "+
		"WHERE id = $7",
		e.SpaceTypeID, e.Label, e.StartTime, e.EndTime, e.Enabled, e.SortOrder, e.ID)
	return err
}

func (r *SpaceTypeRepository) DeleteSlot(id string) error {
	_, err := GetDatabase().DB().Exec("DELETE FROM space_type_time_slots WHERE id = $1", id)
	return err
}
