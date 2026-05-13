package router

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/mux"

	. "github.com/seatsurfing/seatsurfing/server/repository"
)

type SpaceRouter struct {
}

type SpaceAttributeValueRequest struct {
	AttributeID string `json:"attributeId"`
	Value       string `json:"value"`
}

type CreateSpaceRequest struct {
	Name                  string                       `json:"name" validate:"required,max=128"`
	X                     uint                         `json:"x"`
	Y                     uint                         `json:"y"`
	Width                 uint                         `json:"width"`
	Height                uint                         `json:"height"`
	Rotation              uint                         `json:"rotation"`
	RequireSubject        bool                         `json:"requireSubject"`
	Enabled               bool                         `json:"enabled"`
	SpaceTypeID           string                       `json:"spaceTypeId"`
	Attributes            []SpaceAttributeValueRequest `json:"attributes"`
	ApproverGroupIDs      []string                     `json:"approverGroupIds"`
	AllowedBookerGroupIDs []string                     `json:"allowedBookerGroupIds"`
}

type UpdateSpaceRequest struct {
	CreateSpaceRequest
	ID string `json:"id"`
}

type SpaceBulkUpdateRequest struct {
	Creates   []CreateSpaceRequest `json:"creates"`
	Updates   []UpdateSpaceRequest `json:"updates"`
	DeleteIDs []string             `json:"deleteIds"`
}

type BulkUpdateItemResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

type BulkUpdateResponse struct {
	Creates []BulkUpdateItemResponse `json:"creates"`
	Updates []BulkUpdateItemResponse `json:"updates"`
	Deletes []BulkUpdateItemResponse `json:"deletes"`
}

type GetSpaceResponse struct {
	ID         string                `json:"id"`
	Available  bool                  `json:"available"`
	LocationID string                `json:"locationId"`
	Location   *GetLocationResponse  `json:"location,omitempty"`
	SpaceType  *GetSpaceTypeResponse `json:"spaceType,omitempty"`
	CreateSpaceRequest
}

type GetSpaceAvailabilityBookingsResponse struct {
	BookingID   string    `json:"id"`
	RecurringID string    `json:"recurringId"`
	UserID      string    `json:"userId"`
	UserEmail   string    `json:"userEmail"`
	Enter       time.Time `json:"enter"`
	Leave       time.Time `json:"leave"`
	Subject     string    `json:"subject"`
}

type GetSpaceAvailabilityResponse struct {
	GetSpaceResponse
	Bookings           []*GetSpaceAvailabilityBookingsResponse `json:"bookings"`
	IsAllowed          bool                                    `json:"allowed"`
	IsApprovalRequired bool                                    `json:"approvalRequired"`
}

type GetSpaceAvailabilityRequest struct {
	Enter      time.Time         `json:"enter"`
	Leave      time.Time         `json:"leave"`
	SpaceID    string            `json:"spaceId"`
	Attributes []SearchAttribute `json:"attributes"`
}

const (
	SpaceDayStatusAvailable       = "available"
	SpaceDayStatusPartiallyBooked = "partially_booked"
	SpaceDayStatusFull            = "full"
)

type GetSpaceDayStatusResponse struct {
	SpaceID       string                                  `json:"spaceId"`
	Status        string                                  `json:"status"`
	OfficeStart   time.Time                               `json:"officeStart"`
	OfficeEnd     time.Time                               `json:"officeEnd"`
	BookedMinutes int                                     `json:"bookedMinutes"`
	OfficeMinutes int                                     `json:"officeMinutes"`
	Bookings      []*GetSpaceAvailabilityBookingsResponse `json:"bookings"`
}

type spaceDayStatusInterval struct {
	Enter time.Time
	Leave time.Time
}

func (router *SpaceRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/availability", router.getAvailability).Methods("GET")
	s.HandleFunc("/day-status", router.getDayStatus).Methods("GET")
	s.HandleFunc("/bulk", router.bulkUpdate).Methods("POST")
	s.HandleFunc("/{id}/approver/remove", router.removeApprovers).Methods("POST")
	s.HandleFunc("/{id}/approver", router.getApprovers).Methods("GET")
	s.HandleFunc("/{id}/approver", router.addApprovers).Methods("PUT")
	s.HandleFunc("/{id}/allowedbooker/remove", router.removeAllowedBookers).Methods("POST")
	s.HandleFunc("/{id}/allowedbooker", router.getAllowedBookers).Methods("GET")
	s.HandleFunc("/{id}/allowedbooker", router.addAllowedBookers).Methods("PUT")
	s.HandleFunc("/{id}/availability", router.getSingleSpaceAvailability).Methods("GET")
	s.HandleFunc("/{id}", router.getOne).Methods("GET")
	s.HandleFunc("/{id}", router.update).Methods("PUT")
	s.HandleFunc("/{id}", router.delete).Methods("DELETE")
	s.HandleFunc("/", router.create).Methods("POST")
	s.HandleFunc("/", router.getAll).Methods("GET")
}

func (router *SpaceRouter) getOne(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetSpaceRepository().GetOne(vars["id"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	location, err := GetLocationRepository().GetOne(e.LocationID)
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanAccessOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	attributes, err := GetSpaceAttributeValueRepository().GetAllForEntity(e.ID, SpaceAttributeValueEntityTypeSpace)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	approvers, err := GetSpaceRepository().GetAllApproversForSpaceList([]string{e.ID})
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	allowedBookers, err := GetSpaceRepository().GetAllAllowedBookersForSpaceList([]string{e.ID})
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	spaceTypes, err := router.loadSpaceTypeResponseMap([]string{e.SpaceTypeID}, location.OrganizationID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	res := router.copyToRestModel(e, attributes, approvers, allowedBookers, spaceTypes)
	SendJSON(w, res)
}

func (router *SpaceRouter) getSingleSpaceAvailability(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	router._getAvailability(vars["id"], w, r)
}

func (router *SpaceRouter) getAvailability(w http.ResponseWriter, r *http.Request) {
	router._getAvailability("", w, r)
}

func (router *SpaceRouter) _getAvailability(spaceID string, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	locationId := vars["locationId"]
	location, err := GetLocationRepository().GetOne(locationId)
	if err != nil {
		SendBadRequest(w)
		return
	}
	var enter, leave time.Time
	if !r.URL.Query().Has("enter") && !r.URL.Query().Has("leave") {
		tz := GetLocationRepository().GetTimezone(location)
		tzLocation, err := time.LoadLocation(tz)
		if err != nil || tzLocation == nil {
			log.Println("Error loading timezone:", tz, "Error:", err)
			SendInternalServerError(w)
			return
		}
		enter = time.Now().In(tzLocation).Add(time.Minute * -1)
		leave = time.Now().In(tzLocation).Add(time.Minute * +1)
	} else if r.URL.Query().Has("enter") && r.URL.Query().Has("leave") {
		var err error
		if enter, err = time.Parse(time.RFC3339Nano, r.URL.Query().Get("enter")); err != nil {
			SendBadRequest(w)
			return
		}
		if leave, err = time.Parse(time.RFC3339Nano, r.URL.Query().Get("leave")); err != nil {
			SendBadRequest(w)
			return
		}
		enter, err = GetLocationRepository().AttachTimezoneInformation(enter, location)
		if err != nil {
			SendInternalServerError(w)
			return
		}
		leave, err = GetLocationRepository().AttachTimezoneInformation(leave, location)
		if err != nil {
			SendInternalServerError(w)
			return
		}
	} else {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanAccessOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	var showNames bool = false
	if CanSpaceAdminOrg(user, location.OrganizationID) {
		showNames = true
	} else {
		showNames, _ = GetSettingsRepository().GetBool(location.OrganizationID, SettingShowNames.Name)
	}
	list, err := GetSpaceRepository().GetAllInTime(location.ID, enter, leave)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	spaceIds := []string{}
	for _, e := range list {
		spaceIds = append(spaceIds, e.Space.ID)
	}
	spaceTypes, err := router.loadSpaceTypeResponseMap(router.spaceTypeIDsFromAvailability(list), location.OrganizationID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	attributeValues, err := GetSpaceAttributeValueRepository().GetAllForEntityList(spaceIds, SpaceAttributeValueEntityTypeSpace)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	userGroups, err := GetGroupRepository().GetAllWhereUserIsMember(user.ID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	spaceAllowedBookers, err := GetSpaceRepository().GetAllAllowedBookersForSpaceList(spaceIds)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	locationAllowedBookers, err := GetLocationRepository().GetAllAllowedBookersForLocation(locationId)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	approvers, err := GetSpaceRepository().GetAllApproversForSpaceList(spaceIds)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	attributes := []SearchAttribute{}
	if r.URL.Query().Has("attributes") {
		json.Unmarshal([]byte(r.URL.Query().Get("attributes")), &attributes)
	}
	isAllowedToBookLocation := router.IsUserAllowedToBookLocation(locationAllowedBookers, userGroups)
	res := []*GetSpaceAvailabilityResponse{}
	for _, e := range list {
		if spaceID != "" && e.ID != spaceID {
			continue
		}
		if MatchesSearchAttributes(e.ID, &attributes, attributeValues) {
			m := &GetSpaceAvailabilityResponse{
				GetSpaceResponse: *router.copyToRestModel(&e.Space, attributeValues, approvers, spaceAllowedBookers, spaceTypes),
			}
			m.Available = e.Available
			m.IsAllowed = isAllowedToBookLocation && router.IsUserAllowedToBookSpace(&e.Space, spaceAllowedBookers, userGroups)
			m.IsApprovalRequired = router.IsApprovalRequired(&e.Space, approvers)
			m.Bookings = []*GetSpaceAvailabilityBookingsResponse{}
			for _, booking := range e.Bookings {
				var showName bool = showNames
				enter, _ := GetLocationRepository().AttachTimezoneInformation(booking.Enter, location)
				leave, _ := GetLocationRepository().AttachTimezoneInformation(booking.Leave, location)
				outUserId := ""
				outUserEmail := ""
				if showName || user.Email == booking.UserEmail {
					outUserId = booking.UserID
					outUserEmail = booking.UserEmail
				}
				entry := &GetSpaceAvailabilityBookingsResponse{
					BookingID:   booking.BookingID,
					RecurringID: booking.RecurringID,
					UserID:      outUserId,
					UserEmail:   outUserEmail,
					Enter:       enter,
					Leave:       leave,
					Subject:     booking.Subject,
				}
				m.Bookings = append(m.Bookings, entry)
			}
			res = append(res, m)
		}
	}
	SendJSON(w, res)
}

func (router *SpaceRouter) getDayStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	locationID := vars["locationId"]
	location, err := GetLocationRepository().GetOne(locationID)
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanAccessOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	dateValue := r.URL.Query().Get("date")
	if dateValue == "" {
		SendBadRequest(w)
		return
	}
	tz := GetLocationRepository().GetTimezone(location)
	tzLocation, err := time.LoadLocation(tz)
	if err != nil || tzLocation == nil {
		log.Println("Error loading timezone:", tz, "Error:", err)
		SendInternalServerError(w)
		return
	}
	date, err := time.ParseInLocation("2006-01-02", dateValue, tzLocation)
	if err != nil {
		SendBadRequest(w)
		return
	}
	officeSettings, err := GetOfficeSettingsRepository().Get()
	if err != nil {
		if errors.Is(err, ErrOfficeSettingsMissing) {
			SendBadRequestCode(w, ResponseCodeOfficeSettingsMissing)
			return
		}
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	officeStart, err := ParseOfficeClockOnDate(officeSettings.WorkStartTime, date)
	if err != nil {
		SendBadRequest(w)
		return
	}
	officeEnd, err := ParseOfficeClockOnDate(officeSettings.WorkEndTime, date)
	if err != nil {
		SendBadRequest(w)
		return
	}
	officeMinutes := int(officeEnd.Sub(officeStart).Minutes())
	if officeMinutes <= 0 {
		SendBadRequest(w)
		return
	}
	spaces, err := GetSpaceRepository().GetAll(location.ID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	spaceTypes, err := router.loadSpaceTypeMap(router.spaceTypeIDsFromSpaces(spaces), location.OrganizationID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	minBookingDurationHours, err := GetSettingsRepository().GetInt(location.OrganizationID, SettingMinBookingDurationHours.Name)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	dailyBasisBooking, err := GetSettingsRepository().GetBool(location.OrganizationID, SettingDailyBasisBooking.Name)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	spaceIDs := []string{}
	for _, space := range spaces {
		spaceIDs = append(spaceIDs, space.ID)
	}
	bookings, err := GetBookingRepository().GetAllForSpacesInTime(spaceIDs, officeStart, officeEnd)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	userByID, err := router.loadBookingUserMap(bookings)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	showNames := false
	if CanSpaceAdminOrg(user, location.OrganizationID) {
		showNames = true
	} else {
		showNames, _ = GetSettingsRepository().GetBool(location.OrganizationID, SettingShowNames.Name)
	}
	bookingsBySpace := map[string][]*GetSpaceAvailabilityBookingsResponse{}
	intervalsBySpace := map[string][]spaceDayStatusInterval{}
	for _, booking := range bookings {
		enter, leave, ok := router.clampBookingToOfficeHours(booking, officeStart, officeEnd, location)
		if !ok {
			continue
		}
		outUserID := ""
		outUserEmail := ""
		bookingUser := userByID[booking.UserID]
		if bookingUser != nil && (showNames || user.ID == booking.UserID || user.Email == bookingUser.Email) {
			outUserID = booking.UserID
			outUserEmail = bookingUser.Email
		}
		bookingsBySpace[booking.SpaceID] = append(bookingsBySpace[booking.SpaceID], &GetSpaceAvailabilityBookingsResponse{
			BookingID:   booking.ID,
			RecurringID: string(booking.RecurringID),
			UserID:      outUserID,
			UserEmail:   outUserEmail,
			Enter:       enter,
			Leave:       leave,
			Subject:     booking.Subject,
		})
		intervalsBySpace[booking.SpaceID] = append(intervalsBySpace[booking.SpaceID], spaceDayStatusInterval{Enter: enter, Leave: leave})
	}
	res := []*GetSpaceDayStatusResponse{}
	requestNow := time.Now()
	for _, space := range spaces {
		spaceBookings := bookingsBySpace[space.ID]
		if spaceBookings == nil {
			spaceBookings = []*GetSpaceAvailabilityBookingsResponse{}
		}
		mergedIntervals := normalizeSpaceDayStatusIntervals(intervalsBySpace[space.ID])
		bookedMinutes := sumSpaceDayStatusIntervalMinutes(mergedIntervals)
		hasBookableOption := false
		if bookableStart, bookableEnd, ok := getDayStatusBookableWindow(officeStart, officeEnd, dailyBasisBooking, requestNow); ok {
			hasBookableOption = router.hasDayStatusBookableOption(
				space,
				spaceTypes[space.SpaceTypeID],
				mergedIntervals,
				bookableStart,
				bookableEnd,
				minBookingDurationHours,
				dailyBasisBooking,
			)
		}
		status := getSpaceDayStatus(bookedMinutes, officeMinutes, hasBookableOption)
		res = append(res, &GetSpaceDayStatusResponse{
			SpaceID:       space.ID,
			Status:        status,
			OfficeStart:   officeStart,
			OfficeEnd:     officeEnd,
			BookedMinutes: bookedMinutes,
			OfficeMinutes: officeMinutes,
			Bookings:      spaceBookings,
		})
	}
	SendJSON(w, res)
}

func getSpaceDayStatus(bookedMinutes, officeMinutes int, hasBookableOption bool) string {
	if bookedMinutes <= 0 {
		if hasBookableOption {
			return SpaceDayStatusAvailable
		}
		return SpaceDayStatusFull
	}
	if bookedMinutes >= officeMinutes || !hasBookableOption {
		return SpaceDayStatusFull
	}
	return SpaceDayStatusPartiallyBooked
}

func (router *SpaceRouter) clampBookingToOfficeHours(booking *Booking, officeStart, officeEnd time.Time, location *Location) (time.Time, time.Time, bool) {
	enter, err := GetLocationRepository().AttachTimezoneInformation(booking.Enter, location)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	leave, err := GetLocationRepository().AttachTimezoneInformation(booking.Leave, location)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	if enter.Before(officeStart) {
		enter = officeStart
	}
	if leave.After(officeEnd) {
		leave = officeEnd
	}
	return enter, leave, enter.Before(leave)
}

func (router *SpaceRouter) loadBookingUserMap(bookings []*Booking) (map[string]*User, error) {
	userIDs := []string{}
	seen := map[string]bool{}
	for _, booking := range bookings {
		if booking.UserID == "" || seen[booking.UserID] {
			continue
		}
		seen[booking.UserID] = true
		userIDs = append(userIDs, booking.UserID)
	}
	res := map[string]*User{}
	if len(userIDs) == 0 {
		return res, nil
	}
	users, err := GetUserRepository().GetAllByIDs(userIDs)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		res[user.ID] = user
	}
	return res, nil
}

func normalizeSpaceDayStatusIntervals(intervals []spaceDayStatusInterval) []spaceDayStatusInterval {
	if len(intervals) == 0 {
		return nil
	}
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].Enter.Before(intervals[j].Enter)
	})
	currentEnter := intervals[0].Enter
	currentLeave := intervals[0].Leave
	result := []spaceDayStatusInterval{}
	for _, interval := range intervals[1:] {
		if interval.Enter.After(currentLeave) {
			result = append(result, spaceDayStatusInterval{Enter: currentEnter, Leave: currentLeave})
			currentEnter = interval.Enter
			currentLeave = interval.Leave
			continue
		}
		if interval.Leave.After(currentLeave) {
			currentLeave = interval.Leave
		}
	}
	result = append(result, spaceDayStatusInterval{Enter: currentEnter, Leave: currentLeave})
	return result
}

func mergeSpaceDayStatusIntervals(intervals []spaceDayStatusInterval) int {
	return sumSpaceDayStatusIntervalMinutes(normalizeSpaceDayStatusIntervals(intervals))
}

func sumSpaceDayStatusIntervalMinutes(intervals []spaceDayStatusInterval) int {
	total := 0
	for _, interval := range intervals {
		total += int(math.Ceil(interval.Leave.Sub(interval.Enter).Minutes()))
	}
	return total
}

func (router *SpaceRouter) hasDayStatusBookableOption(space *Space, spaceType *SpaceType, intervals []spaceDayStatusInterval, officeStart, officeEnd time.Time, minBookingDurationHours int, dailyBasisBooking bool) bool {
	effectiveType := getEffectiveDayStatusSpaceType(space, spaceType)
	if effectiveType != nil && effectiveType.BookingMode == SpaceTypeBookingModeFixedSlots {
		return router.hasDayStatusAvailableFixedSlot(effectiveType, intervals, officeStart, officeEnd)
	}
	minDuration := getDayStatusMinimumBookableDuration(effectiveType, minBookingDurationHours, dailyBasisBooking)
	return hasDayStatusBookableGap(intervals, officeStart, officeEnd, minDuration)
}

func getDayStatusBookableWindow(officeStart, officeEnd time.Time, dailyBasisBooking bool, now time.Time) (time.Time, time.Time, bool) {
	if !officeEnd.After(officeStart) {
		return time.Time{}, time.Time{}, false
	}
	current := now.In(officeStart.Location())
	if !isSameDayInLocation(current, officeStart) || current.Before(officeStart) {
		return officeStart, officeEnd, true
	}
	if !officeEnd.After(current) {
		return time.Time{}, time.Time{}, false
	}
	return current, officeEnd, true
}

func isSameDayInLocation(left, right time.Time) bool {
	return left.Year() == right.Year() && left.Month() == right.Month() && left.Day() == right.Day()
}

func getEffectiveDayStatusSpaceType(space *Space, spaceType *SpaceType) *SpaceType {
	if space == nil || space.SpaceTypeID == "" || spaceType == nil || !spaceType.Enabled {
		return nil
	}
	if spaceType.BookingMode != SpaceTypeBookingModeFixedSlots && spaceType.BookingMode != SpaceTypeBookingModeFlexibleTime {
		return nil
	}
	return spaceType
}

func (router *SpaceRouter) hasDayStatusAvailableFixedSlot(spaceType *SpaceType, intervals []spaceDayStatusInterval, officeStart, officeEnd time.Time) bool {
	for _, slot := range spaceType.Slots {
		if slot == nil || !slot.Enabled {
			continue
		}
		startHour, startMinute, ok := parseSpaceTypeSlotClock(slot.StartTime)
		if !ok {
			continue
		}
		endHour, endMinute, ok := parseSpaceTypeSlotClock(slot.EndTime)
		if !ok {
			continue
		}
		slotEnter := time.Date(officeStart.Year(), officeStart.Month(), officeStart.Day(), startHour, startMinute, 0, 0, officeStart.Location())
		slotLeave := time.Date(officeStart.Year(), officeStart.Month(), officeStart.Day(), endHour, endMinute, 0, 0, officeStart.Location())
		if !slotEnter.Before(slotLeave) || slotEnter.Before(officeStart) || slotLeave.After(officeEnd) {
			continue
		}
		if !doesSpaceDayStatusIntervalOverlap(intervals, slotEnter, slotLeave) {
			return true
		}
	}
	return false
}

func hasDayStatusBookableGap(intervals []spaceDayStatusInterval, officeStart, officeEnd time.Time, minDuration time.Duration) bool {
	if !officeEnd.After(officeStart) {
		return false
	}
	if minDuration <= 0 {
		minDuration = time.Second
	}
	cursor := officeStart
	for _, interval := range intervals {
		if interval.Enter.After(cursor) && interval.Enter.Sub(cursor) >= minDuration {
			return true
		}
		if interval.Leave.After(cursor) {
			cursor = interval.Leave
		}
	}
	return officeEnd.After(cursor) && officeEnd.Sub(cursor) >= minDuration
}

func getEffectiveLegacyMinimumBookableDuration(minBookingDurationHours int) time.Duration {
	if minBookingDurationHours > 0 {
		return time.Duration(minBookingDurationHours) * time.Hour
	}
	return 30 * time.Minute
}

func doesSpaceDayStatusIntervalOverlap(intervals []spaceDayStatusInterval, enter, leave time.Time) bool {
	for _, interval := range intervals {
		if enter.Before(interval.Leave) && leave.After(interval.Enter) {
			return true
		}
	}
	return false
}

func getDayStatusMinimumBookableDuration(spaceType *SpaceType, minBookingDurationHours int, dailyBasisBooking bool) time.Duration {
	required := time.Duration(minBookingDurationHours) * time.Hour
	if spaceType == nil {
		required = getEffectiveLegacyMinimumBookableDuration(minBookingDurationHours)
	}
	if spaceType != nil && spaceType.BookingMode == SpaceTypeBookingModeFlexibleTime {
		spaceTypeDuration := time.Duration(spaceType.MinDurationMinutes) * time.Minute
		if spaceTypeDuration > required {
			required = spaceTypeDuration
		}
	}
	if required > 0 && !dailyBasisBooking {
		required -= time.Second
	}
	return required
}

func (router *SpaceRouter) IsApprovalRequired(e *Space, approvers []*SpaceGroup) bool {
	for _, approver := range approvers {
		if approver.SpaceID == e.ID {
			return true
		}
	}
	return false
}

func (router *SpaceRouter) IsUserAllowedToBookSpace(e *Space, allowedBookers []*SpaceGroup, userGroups []*Group) bool {
	restricted := false
	for _, allowedBooker := range allowedBookers {
		if allowedBooker.SpaceID == e.ID {
			restricted = true
			for _, userGroup := range userGroups {
				if allowedBooker.GroupID == userGroup.ID {
					return true
				}
			}
		}
	}
	return !restricted
}

func (router *SpaceRouter) IsUserAllowedToBookLocation(allowedBookers []*LocationGroup, userGroups []*Group) bool {
	restricted := false
	for _, allowedBooker := range allowedBookers {
		restricted = true
		for _, userGroup := range userGroups {
			if allowedBooker.GroupID == userGroup.ID {
				return true
			}
		}
	}
	return !restricted
}

func (router *SpaceRouter) spaceTypeIDsFromSpaces(spaces []*Space) []string {
	ids := []string{}
	seen := map[string]bool{}
	for _, space := range spaces {
		if space.SpaceTypeID == "" || seen[space.SpaceTypeID] {
			continue
		}
		seen[space.SpaceTypeID] = true
		ids = append(ids, space.SpaceTypeID)
	}
	return ids
}

func (router *SpaceRouter) spaceTypeIDsFromAvailability(spaces []*SpaceAvailability) []string {
	ids := []string{}
	seen := map[string]bool{}
	for _, space := range spaces {
		if space.SpaceTypeID == "" || seen[space.SpaceTypeID] {
			continue
		}
		seen[space.SpaceTypeID] = true
		ids = append(ids, space.SpaceTypeID)
	}
	return ids
}

func (router *SpaceRouter) loadSpaceTypeResponseMap(ids []string, organizationID string) (map[string]*GetSpaceTypeResponse, error) {
	res := map[string]*GetSpaceTypeResponse{}
	filteredIDs := []string{}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		filteredIDs = append(filteredIDs, id)
	}
	if len(filteredIDs) == 0 {
		return res, nil
	}
	spaceTypes, err := GetSpaceTypeRepository().GetAllByIDs(filteredIDs, true)
	if err != nil {
		return nil, err
	}
	spaceTypeRouter := &SpaceTypeRouter{}
	for _, spaceType := range spaceTypes {
		if spaceType.OrganizationID != organizationID {
			continue
		}
		res[spaceType.ID] = spaceTypeRouter.copyToRestModel(spaceType)
	}
	return res, nil
}

func (router *SpaceRouter) loadSpaceTypeMap(ids []string, organizationID string) (map[string]*SpaceType, error) {
	res := map[string]*SpaceType{}
	filteredIDs := []string{}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		filteredIDs = append(filteredIDs, id)
	}
	if len(filteredIDs) == 0 {
		return res, nil
	}
	spaceTypes, err := GetSpaceTypeRepository().GetAllByIDs(filteredIDs, false)
	if err != nil {
		return nil, err
	}
	for _, spaceType := range spaceTypes {
		if spaceType.OrganizationID != organizationID {
			continue
		}
		res[spaceType.ID] = spaceType
	}
	return res, nil
}

func (router *SpaceRouter) isValidSpaceTypeID(spaceTypeID, organizationID string) bool {
	if spaceTypeID == "" {
		return true
	}
	spaceType, err := GetSpaceTypeRepository().GetOne(spaceTypeID)
	if err != nil {
		return false
	}
	return spaceType.OrganizationID == organizationID
}

func (router *SpaceRouter) bulkUpdate(w http.ResponseWriter, r *http.Request) {
	var m SpaceBulkUpdateRequest
	if UnmarshalValidateBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}
	vars := mux.Vars(r)
	location, err := GetLocationRepository().GetOne(vars["locationId"])
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	availableAttributes, err := GetSpaceAttributeRepository().GetAll(location.OrganizationID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}

	res := BulkUpdateResponse{
		Creates: []BulkUpdateItemResponse{},
		Updates: []BulkUpdateItemResponse{},
		Deletes: []BulkUpdateItemResponse{},
	}

	// Process deletes
	if m.DeleteIDs != nil {
		for _, deleteID := range m.DeleteIDs {
			e, err := GetSpaceRepository().GetOne(deleteID)
			if err != nil {
				res.Deletes = append(res.Deletes, BulkUpdateItemResponse{ID: deleteID, Success: false})
			} else {
				if err := GetSpaceRepository().Delete(e); err != nil {
					res.Deletes = append(res.Deletes, BulkUpdateItemResponse{ID: deleteID, Success: false})
				} else {
					res.Deletes = append(res.Deletes, BulkUpdateItemResponse{ID: deleteID, Success: true})
				}
			}
		}
	}

	// Process creates
	if m.Creates != nil {
		for _, mSpace := range m.Creates {
			if !router.isValidSpaceTypeID(mSpace.SpaceTypeID, location.OrganizationID) {
				res.Creates = append(res.Creates, BulkUpdateItemResponse{ID: "", Success: false})
				continue
			}
			e := router.copyFromRestModel(&mSpace)
			e.LocationID = vars["locationId"]
			if err := GetSpaceRepository().Create(e); err != nil {
				log.Println(err)
				res.Creates = append(res.Creates, BulkUpdateItemResponse{ID: "", Success: false})
			} else {
				if err := router.applySpaceAttributes(availableAttributes, e, &mSpace); err != nil {
					log.Println("Could not apply space attributes:", err)
				}
				if err := router.applyApprovers(e, &mSpace); err != nil {
					log.Println("Could not apply approvers:", err)
				}
				if err := router.applyAllowBookers(e, &mSpace); err != nil {
					log.Println("Could not apply allow bookers:", err)
				}
				res.Creates = append(res.Creates, BulkUpdateItemResponse{ID: e.ID, Success: true})
			}
		}
	}

	// Process updates
	if m.Updates != nil {
		for _, mSpace := range m.Updates {
			if !router.isValidSpaceTypeID(mSpace.SpaceTypeID, location.OrganizationID) {
				res.Updates = append(res.Updates, BulkUpdateItemResponse{ID: mSpace.ID, Success: false})
				continue
			}
			e := router.copyFromRestModel(&mSpace.CreateSpaceRequest)
			e.ID = mSpace.ID
			e.LocationID = vars["locationId"]
			if err := GetSpaceRepository().Update(e); err != nil {
				log.Println(err)
				res.Updates = append(res.Updates, BulkUpdateItemResponse{ID: "", Success: false})
			} else {
				if err := router.applySpaceAttributes(availableAttributes, e, &mSpace.CreateSpaceRequest); err != nil {
					log.Println("Could not apply space attributes:", err)
				}
				if err := router.applyApprovers(e, &mSpace.CreateSpaceRequest); err != nil {
					log.Println("Could not apply approvers:", err)
				}
				if err := router.applyAllowBookers(e, &mSpace.CreateSpaceRequest); err != nil {
					log.Println("Could not apply allow bookers:", err)
				}
				res.Updates = append(res.Updates, BulkUpdateItemResponse{ID: e.ID, Success: true})
			}
		}
	}
	SendJSON(w, res)
}

func (router *SpaceRouter) getAll(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	location, err := GetLocationRepository().GetOne(vars["locationId"])
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanAccessOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	list, err := GetSpaceRepository().GetAll(location.ID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	spaceIds := []string{}
	for _, e := range list {
		spaceIds = append(spaceIds, e.ID)
	}
	spaceTypes, err := router.loadSpaceTypeResponseMap(router.spaceTypeIDsFromSpaces(list), location.OrganizationID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	attributes, err := GetSpaceAttributeValueRepository().GetAllForEntityList(spaceIds, SpaceAttributeValueEntityTypeSpace)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	approvers, err := GetSpaceRepository().GetAllApproversForSpaceList(spaceIds)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	allowedBookers, err := GetSpaceRepository().GetAllAllowedBookersForSpaceList(spaceIds)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	res := []*GetSpaceResponse{}
	for _, e := range list {
		m := router.copyToRestModel(e, attributes, approvers, allowedBookers, spaceTypes)
		res = append(res, m)
	}
	SendJSON(w, res)
}

func (router *SpaceRouter) update(w http.ResponseWriter, r *http.Request) {
	var m CreateSpaceRequest
	if UnmarshalValidateBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}
	vars := mux.Vars(r)
	e := router.copyFromRestModel(&m)
	e.ID = vars["id"]
	e.LocationID = vars["locationId"]
	location, err := GetLocationRepository().GetOne(e.LocationID)
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	if !router.isValidSpaceTypeID(m.SpaceTypeID, location.OrganizationID) {
		SendBadRequest(w)
		return
	}
	if err := GetSpaceRepository().Update(e); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	availableAttributes, err := GetSpaceAttributeRepository().GetAll(location.OrganizationID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	router.applySpaceAttributes(availableAttributes, e, &m)
	SendUpdated(w)
}

func (router *SpaceRouter) delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetSpaceRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}
	location, err := GetLocationRepository().GetOne(e.LocationID)
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	if err := GetSpaceRepository().Delete(e); err != nil {
		SendInternalServerError(w)
		return
	}
	SendUpdated(w)
}

func (router *SpaceRouter) create(w http.ResponseWriter, r *http.Request) {
	var m CreateSpaceRequest
	if UnmarshalValidateBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}
	vars := mux.Vars(r)
	e := router.copyFromRestModel(&m)
	e.LocationID = vars["locationId"]
	location, err := GetLocationRepository().GetOne(e.LocationID)
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	if !router.isValidSpaceTypeID(m.SpaceTypeID, location.OrganizationID) {
		SendBadRequest(w)
		return
	}
	if err := GetSpaceRepository().Create(e); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	availableAttributes, err := GetSpaceAttributeRepository().GetAll(location.OrganizationID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	router.applySpaceAttributes(availableAttributes, e, &m)
	SendCreated(w, e.ID)
}

func (router *SpaceRouter) applySpaceAttributes(availableAttributes []*SpaceAttribute, space *Space, m *CreateSpaceRequest) error {
	existingSpaceAttributes, err := GetSpaceAttributeValueRepository().GetAllForEntity(space.ID, SpaceAttributeValueEntityTypeSpace)
	if err != nil {
		return err
	}
	// Check deletes
	for _, attribute := range existingSpaceAttributes {
		found := false
		for _, mAttribute := range m.Attributes {
			if attribute.AttributeID == mAttribute.AttributeID {
				found = true
				break
			}
		}
		if !found {
			if err := GetSpaceAttributeValueRepository().Delete(attribute.AttributeID, space.ID, SpaceAttributeValueEntityTypeSpace); err != nil {
				return err
			}
		}
	}
	// Check creates / updates
	for _, mAttribute := range m.Attributes {
		// Check if attribute is valid
		found := false
		for _, availableAttribute := range availableAttributes {
			if availableAttribute.ID == mAttribute.AttributeID {
				found = true
				break
			}
		}
		if found {
			if err := GetSpaceAttributeValueRepository().Set(mAttribute.AttributeID, space.ID, SpaceAttributeValueEntityTypeSpace, mAttribute.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (router *SpaceRouter) applyApprovers(space *Space, m *CreateSpaceRequest) error {
	existingApprovers, err := GetSpaceRepository().GetApproverGroupIDs(space.ID)
	if err != nil {
		return err
	}
	// Check deletes
	removes := []string{}
	for _, approver := range existingApprovers {
		found := false
		for _, mApprover := range m.ApproverGroupIDs {
			if approver == mApprover {
				found = true
				break
			}
		}
		if !found {
			removes = append(removes, approver)
		}
	}
	// Check creates
	adds := []string{}
	for _, mApprover := range m.ApproverGroupIDs {
		found := false
		for _, approver := range existingApprovers {
			if approver == mApprover {
				found = true
				break
			}
		}
		if !found {
			adds = append(adds, mApprover)
		}
	}
	if err := GetSpaceRepository().AddApprovers(space, adds); err != nil {
		return err
	}
	if err := GetSpaceRepository().RemoveApprovers(space, removes); err != nil {
		return err
	}
	return nil
}

func (router *SpaceRouter) applyAllowBookers(space *Space, m *CreateSpaceRequest) error {
	existingAllowBookers, err := GetSpaceRepository().GetAllowedBookersGroupIDs(space)
	if err != nil {
		return err
	}
	// Check deletes
	removes := []string{}
	for _, allowBooker := range existingAllowBookers {
		found := false
		for _, mAllowBooker := range m.AllowedBookerGroupIDs {
			if allowBooker == mAllowBooker {
				found = true
				break
			}
		}
		if !found {
			removes = append(removes, allowBooker)
		}
	}
	// Check creates
	adds := []string{}
	for _, mAllowBooker := range m.AllowedBookerGroupIDs {
		found := false
		for _, allowBooker := range existingAllowBookers {
			if allowBooker == mAllowBooker {
				found = true
				break
			}
		}
		if !found {
			adds = append(adds, mAllowBooker)
		}
	}
	if err := GetSpaceRepository().AddAllowedBookers(space, adds); err != nil {
		return err
	}
	if err := GetSpaceRepository().RemoveAllowedBookers(space, removes); err != nil {
		return err
	}
	return nil
}

func (router *SpaceRouter) addApprovers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	location, err := GetLocationRepository().GetOne(vars["locationId"])
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	e, err := GetSpaceRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}
	var approvers []string
	if UnmarshalBody(r, &approvers) != nil {
		SendBadRequest(w)
		return
	}
	ok, err := GetGroupRepository().GroupsExistAndBelongToOrg(location.OrganizationID, approvers)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	if !ok {
		SendBadRequest(w)
		return
	}
	if err := GetSpaceRepository().AddApprovers(e, approvers); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendUpdated(w)
}

func (router *SpaceRouter) removeApprovers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	location, err := GetLocationRepository().GetOne(vars["locationId"])
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	e, err := GetSpaceRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}
	var approvers []string
	if UnmarshalBody(r, &approvers) != nil {
		SendBadRequest(w)
		return
	}
	if err := GetSpaceRepository().RemoveApprovers(e, approvers); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendUpdated(w)
}

func (router *SpaceRouter) getApprovers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	location, err := GetLocationRepository().GetOne(vars["locationId"])
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	e, err := GetSpaceRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}
	approvers, err := GetSpaceRepository().GetApproverGroupIDs(e.ID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	groups, err := GetGroupRepository().GetAllByIDs(approvers)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	gr := &GroupRouter{}
	res := []*GetGroupResponse{}
	for _, e := range groups {
		m := gr.copyToRestModel(e)
		res = append(res, m)
	}
	SendJSON(w, res)
}

func (router *SpaceRouter) addAllowedBookers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	location, err := GetLocationRepository().GetOne(vars["locationId"])
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	e, err := GetSpaceRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}
	var approvers []string
	if UnmarshalBody(r, &approvers) != nil {
		SendBadRequest(w)
		return
	}
	ok, err := GetGroupRepository().GroupsExistAndBelongToOrg(location.OrganizationID, approvers)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	if !ok {
		SendBadRequest(w)
		return
	}
	if err := GetSpaceRepository().AddAllowedBookers(e, approvers); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendUpdated(w)
}

func (router *SpaceRouter) removeAllowedBookers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	location, err := GetLocationRepository().GetOne(vars["locationId"])
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	e, err := GetSpaceRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}
	var approvers []string
	if UnmarshalBody(r, &approvers) != nil {
		SendBadRequest(w)
		return
	}
	if err := GetSpaceRepository().RemoveAllowedBookers(e, approvers); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendUpdated(w)
}

func (router *SpaceRouter) getAllowedBookers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	location, err := GetLocationRepository().GetOne(vars["locationId"])
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, location.OrganizationID) {
		SendForbidden(w)
		return
	}
	e, err := GetSpaceRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}
	approvers, err := GetSpaceRepository().GetAllowedBookersGroupIDs(e)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	groups, err := GetGroupRepository().GetAllByIDs(approvers)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	gr := &GroupRouter{}
	res := []*GetGroupResponse{}
	for _, e := range groups {
		m := gr.copyToRestModel(e)
		res = append(res, m)
	}
	SendJSON(w, res)
}

func (router *SpaceRouter) copyFromRestModel(m *CreateSpaceRequest) *Space {
	e := &Space{}
	e.Name = m.Name
	e.X = m.X
	e.Y = m.Y
	e.Width = m.Width
	e.Height = m.Height
	e.Rotation = m.Rotation
	e.RequireSubject = m.RequireSubject
	e.Enabled = m.Enabled
	e.SpaceTypeID = m.SpaceTypeID
	return e
}

func (router *SpaceRouter) copyToRestModel(e *Space, attributes []*SpaceAttributeValue, approvers, allowedBookers []*SpaceGroup, spaceTypes map[string]*GetSpaceTypeResponse) *GetSpaceResponse {
	m := &GetSpaceResponse{}
	m.ID = e.ID
	m.LocationID = e.LocationID
	m.Name = e.Name
	m.X = e.X
	m.Y = e.Y
	m.Width = e.Width
	m.Height = e.Height
	m.Rotation = e.Rotation
	m.RequireSubject = e.RequireSubject
	m.Enabled = e.Enabled
	m.SpaceTypeID = e.SpaceTypeID
	if spaceTypes != nil && e.SpaceTypeID != "" {
		m.SpaceType = spaceTypes[e.SpaceTypeID]
	}
	if attributes != nil {
		m.Attributes = []SpaceAttributeValueRequest{}
		for _, attribute := range attributes {
			if attribute.EntityType == SpaceAttributeValueEntityTypeSpace {
				if attribute.EntityID == e.ID {
					m.Attributes = append(m.Attributes, SpaceAttributeValueRequest{AttributeID: attribute.AttributeID, Value: attribute.Value})
				}
			}
		}
	}
	if approvers != nil {
		m.ApproverGroupIDs = []string{}
		for _, approver := range approvers {
			if approver.SpaceID == e.ID {
				m.ApproverGroupIDs = append(m.ApproverGroupIDs, approver.GroupID)
			}
		}
	}
	if allowedBookers != nil {
		m.AllowedBookerGroupIDs = []string{}
		for _, allowedBooker := range allowedBookers {
			if allowedBooker.SpaceID == e.ID {
				m.AllowedBookerGroupIDs = append(m.AllowedBookerGroupIDs, allowedBooker.GroupID)
			}
		}
	}
	return m
}

func (router *SpaceRouter) appendAttributesToRestModel(m *GetSpaceResponse, attributes []*SpaceAttributeValue) {
	if attributes != nil {
		m.Attributes = []SpaceAttributeValueRequest{}
		for _, attribute := range attributes {
			if attribute.EntityType == SpaceAttributeValueEntityTypeSpace {
				if attribute.EntityID == m.ID {
					m.Attributes = append(m.Attributes, SpaceAttributeValueRequest{AttributeID: attribute.AttributeID, Value: attribute.Value})
				}
			}
		}
	}
}
