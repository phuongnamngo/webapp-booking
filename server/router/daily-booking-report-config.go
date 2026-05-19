package router

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/seatsurfing/seatsurfing/server/repository"
)

const defaultDailyBookingReportSendTime = "08:00"

type DailyBookingReportConfig struct {
	OrganizationID string
	Enabled        bool
	RecipientsCSV  string
	SendTime       string
}

func ResolveSendTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultDailyBookingReportSendTime
	}
	if _, _, err := parseSendTime(raw); err != nil {
		return defaultDailyBookingReportSendTime
	}
	return raw
}

func parseSendTime(value string) (hour int, minute int, err error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid send time %q", value)
	}
	hour, err = strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("invalid send time hour %q", value)
	}
	minute, err = strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid send time minute %q", value)
	}
	return hour, minute, nil
}

func LoadDailyBookingReportConfig(organizationID string) (*DailyBookingReportConfig, error) {
	enabled, err := GetSettingsRepository().GetBool(organizationID, SettingDailyBookingReportEnabled.Name)
	if err != nil {
		return nil, err
	}
	recipients, err := GetSettingsRepository().Get(organizationID, SettingDailyBookingReportRecipients.Name)
	if err != nil {
		return nil, err
	}
	sendTime, err := GetSettingsRepository().Get(organizationID, SettingDailyBookingReportSendTime.Name)
	if err != nil {
		return nil, err
	}
	return &DailyBookingReportConfig{
		OrganizationID: organizationID,
		Enabled:        enabled,
		RecipientsCSV:  recipients,
		SendTime:       ResolveSendTime(sendTime),
	}, nil
}

// LoadDailyBookingReportConfigForCron uses the first org with daily_booking_report_enabled=1 (single-tenant LNT).
func LoadDailyBookingReportConfigForCron() (*DailyBookingReportConfig, error) {
	orgIDs, err := GetSettingsRepository().GetOrgIDsByValue(SettingDailyBookingReportEnabled.Name, "1")
	if err != nil {
		return nil, err
	}
	if len(orgIDs) == 0 {
		return nil, nil
	}
	return LoadDailyBookingReportConfig(orgIDs[0])
}
