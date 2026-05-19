package app

import (
	"log"
	"os"

	. "github.com/seatsurfing/seatsurfing/server/repository"
)

func MigrateDailyBookingReportFromEnv() {
	enabled := os.Getenv("DAILY_BOOKING_REPORT_ENABLED") == "1"
	recipients := os.Getenv("DAILY_BOOKING_REPORT_RECIPIENTS")
	if !enabled && recipients == "" {
		return
	}

	orgs, err := GetOrganizationRepository().GetAll()
	if err != nil || len(orgs) == 0 {
		return
	}
	orgID := orgs[0].ID

	currentRecipients, _ := GetSettingsRepository().Get(orgID, SettingDailyBookingReportRecipients.Name)
	currentEnabled, _ := GetSettingsRepository().GetBool(orgID, SettingDailyBookingReportEnabled.Name)
	if currentRecipients != "" || currentEnabled {
		return
	}

	if enabled {
		if err := GetSettingsRepository().Set(orgID, SettingDailyBookingReportEnabled.Name, "1"); err != nil {
			log.Println("daily booking report migrate:", err)
			return
		}
	}
	if recipients != "" {
		if err := GetSettingsRepository().Set(orgID, SettingDailyBookingReportRecipients.Name, recipients); err != nil {
			log.Println("daily booking report migrate:", err)
			return
		}
	}
	log.Println("daily booking report: migrated config from env to org settings; remove DAILY_BOOKING_REPORT_* from env")
}
