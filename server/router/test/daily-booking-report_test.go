package test

import (
	"strings"
	"testing"
	"time"

	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/router"
	. "github.com/seatsurfing/seatsurfing/server/testutil"
)

func TestDailyBookingReportParseRecipientsTrimsWhitespaceAndDropsEmpty(t *testing.T) {
	service := GetDailyBookingReportService()

	recipients, err := service.ParseRecipients([]string{" a@test.com ", "", "  ", "b@test.com "})

	CheckTestIsNil(t, err)
	CheckTestInt(t, 2, len(recipients))
	CheckTestString(t, "a@test.com", recipients[0])
	CheckTestString(t, "b@test.com", recipients[1])
}

func TestDailyBookingReportParseRecipientsRejectsInvalidEmail(t *testing.T) {
	service := GetDailyBookingReportService()

	_, err := service.ParseRecipients([]string{"boss@test.com", "not-an-email"})

	CheckTestBool(t, true, err != nil)
}

func TestDailyBookingReportResolveSendTimeDefaultsToEightAM(t *testing.T) {
	CheckTestString(t, "08:00", ResolveSendTime(""))
	CheckTestString(t, "08:00", ResolveSendTime("   "))
	CheckTestString(t, "08:00", ResolveSendTime("invalid"))
}

func TestDailyBookingReportShouldRunUsesConfiguredSendTime(t *testing.T) {
	service := GetDailyBookingReportService()
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	CheckTestIsNil(t, err)

	before := time.Date(2026, 5, 14, 9, 29, 0, 0, loc)
	shouldRun, _, err := service.ShouldRun(before, "", "09:30")
	CheckTestIsNil(t, err)
	CheckTestBool(t, false, shouldRun)

	at := time.Date(2026, 5, 14, 9, 30, 0, 0, loc)
	shouldRun, _, err = service.ShouldRun(at, "", "09:30")
	CheckTestIsNil(t, err)
	CheckTestBool(t, true, shouldRun)
}

func TestDailyBookingReportShouldRunAtEightAMVietnamTimeOnlyOncePerDay(t *testing.T) {
	service := GetDailyBookingReportService()
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	CheckTestIsNil(t, err)

	beforeEight := time.Date(2026, 5, 14, 7, 59, 0, 0, loc)
	shouldRun, today, err := service.ShouldRun(beforeEight, "", "08:00")
	CheckTestIsNil(t, err)
	CheckTestBool(t, false, shouldRun)
	CheckTestString(t, "2026-05-14", today)

	atEight := time.Date(2026, 5, 14, 8, 0, 0, 0, loc)
	shouldRun, today, err = service.ShouldRun(atEight, "", "08:00")
	CheckTestIsNil(t, err)
	CheckTestBool(t, true, shouldRun)
	CheckTestString(t, "2026-05-14", today)

	afterEight := time.Date(2026, 5, 14, 8, 5, 0, 0, loc)
	shouldRun, today, err = service.ShouldRun(afterEight, "2026-05-14", "08:00")
	CheckTestIsNil(t, err)
	CheckTestBool(t, false, shouldRun)
	CheckTestString(t, "2026-05-14", today)
}

func TestDailyBookingReportReportWindowUsesYesterdayInVietnamTime(t *testing.T) {
	service := GetDailyBookingReportService()
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	CheckTestIsNil(t, err)

	now := time.Date(2026, 5, 14, 9, 0, 0, 0, loc)
	reportDate, start, end, err := service.GetReportWindow(now)

	CheckTestIsNil(t, err)
	CheckTestString(t, "2026-05-13", reportDate)
	CheckTestString(t, "2026-05-13 00:00", start.Format("2006-01-02 15:04"))
	CheckTestString(t, "2026-05-14 00:00", end.Format("2006-01-02 15:04"))
}

func TestDailyBookingReportBuildPreviewContainsRows(t *testing.T) {
	ClearTestDB()
	service := GetDailyBookingReportService()
	org := CreateTestOrg("test.com")
	user := CreateTestUserInOrgWithName(org, "john.doe@test.com", UserRoleUser)
	user.Firstname = "John"
	user.Lastname = "Doe"
	CheckTestIsNil(t, GetUserRepository().Update(user))

	location := &Location{
		Name:           "Floor 5",
		OrganizationID: org.ID,
	}
	CheckTestIsNil(t, GetLocationRepository().Create(location))
	space := &Space{
		Name:       "Desk A01",
		LocationID: location.ID,
	}
	CheckTestIsNil(t, GetSpaceRepository().Create(space))

	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	CheckTestIsNil(t, err)
	enter := time.Date(2026, 5, 13, 9, 0, 0, 0, loc)
	leave := time.Date(2026, 5, 13, 18, 0, 0, 0, loc)
	booking := &Booking{
		UserID:   user.ID,
		SpaceID:  space.ID,
		Enter:    enter,
		Leave:    leave,
		Approved: true,
	}
	CheckTestIsNil(t, GetBookingRepository().Create(booking))

	overrideDate := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	preview, err := service.BuildPreview(time.Date(2026, 5, 14, 9, 0, 0, 0, loc), &overrideDate, []string{"boss@test.com"}, "")

	CheckTestIsNil(t, err)
	CheckTestString(t, "Daily booking report for 2026-05-13", preview.Subject)
	CheckTestInt(t, 1, preview.RowCount)
	CheckTestInt(t, 1, len(preview.Rows))
	CheckTestString(t, "boss@test.com", preview.Recipients[0])
	CheckTestString(t, "John Doe", preview.Rows[0].UserName)
	CheckTestString(t, "Floor 5", preview.Rows[0].Area)
	CheckTestString(t, "Desk A01", preview.Rows[0].Space)
	CheckTestString(t, "09:00 - 18:00", preview.Rows[0].BookingTime)
	CheckTestBool(t, true, strings.Contains(preview.PreviewHTML, "User name"))
	CheckTestBool(t, true, strings.Contains(preview.PreviewHTML, "Space"))
	CheckTestBool(t, true, strings.Contains(preview.PreviewHTML, "Booking time"))
	CheckTestBool(t, true, strings.Contains(preview.PreviewHTML, "09:00 - 18:00"))
	CheckTestBool(t, true, strings.Contains(preview.PreviewHTML, "font-size:14px"))
	CheckTestBool(t, true, strings.Contains(preview.PreviewHTML, "border-collapse:collapse"))
	CheckTestBool(t, true, strings.Contains(preview.PreviewHTML, "padding:12px 16px"))
	CheckTestBool(t, true, strings.Contains(preview.PreviewHTML, "John Doe"))
	CheckTestBool(t, true, strings.Contains(preview.PreviewHTML, "Floor 5"))
	CheckTestBool(t, true, strings.Contains(preview.PreviewHTML, "Desk A01"))
}

func TestDailyBookingReportBuildPreviewShowsEmptyStateWhenNoRows(t *testing.T) {
	ClearTestDB()
	service := GetDailyBookingReportService()
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	CheckTestIsNil(t, err)

	overrideDate := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	preview, err := service.BuildPreview(time.Date(2026, 5, 14, 9, 0, 0, 0, loc), &overrideDate, []string{"boss@test.com"}, "")

	CheckTestIsNil(t, err)
	CheckTestInt(t, 0, preview.RowCount)
	CheckTestBool(t, true, strings.Contains(preview.PreviewHTML, "No approved bookings for 2026-05-13."))
}
