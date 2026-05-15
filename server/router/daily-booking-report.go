package router

import (
	"fmt"
	"html"
	"strings"
	"time"

	. "github.com/seatsurfing/seatsurfing/server/config"
	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/util"
)

const dailyBookingReportTimezone = "Asia/Ho_Chi_Minh"

type DailyBookingReportService struct{}

type DailyBookingReportPreviewRow struct {
	UserName    string `json:"userName"`
	Area        string `json:"area"`
	Space       string `json:"space"`
	BookingTime string `json:"bookingTime"`
}

type DailyBookingReportPreview struct {
	Subject     string                         `json:"subject"`
	ReportDate  string                         `json:"reportDate"`
	Recipients  []string                       `json:"recipients"`
	RowCount    int                            `json:"rowCount"`
	Rows        []DailyBookingReportPreviewRow `json:"rows"`
	PreviewHTML string                         `json:"previewHtml"`
}

func GetDailyBookingReportService() *DailyBookingReportService {
	return &DailyBookingReportService{}
}

func (s *DailyBookingReportService) ParseRecipients(values []string) ([]string, error) {
	recipients := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if err := ValidateEmailAddress(trimmed); err != nil {
			return nil, fmt.Errorf("invalid daily booking report recipient %q: %w", trimmed, err)
		}
		recipients = append(recipients, trimmed)
	}
	if len(recipients) == 0 {
		return nil, fmt.Errorf("no daily booking report recipients configured")
	}
	return recipients, nil
}

func (s *DailyBookingReportService) getVietnamLocation() (*time.Location, error) {
	return time.LoadLocation(dailyBookingReportTimezone)
}

func (s *DailyBookingReportService) getVietnamNow(now time.Time) (time.Time, error) {
	loc, err := s.getVietnamLocation()
	if err != nil {
		return time.Time{}, err
	}
	return now.In(loc), nil
}

func (s *DailyBookingReportService) ShouldRun(now time.Time, lastSentDate string) (bool, string, error) {
	vietnamNow, err := s.getVietnamNow(now)
	if err != nil {
		return false, "", err
	}
	today := vietnamNow.Format("2006-01-02")
	if vietnamNow.Hour() < 8 {
		return false, today, nil
	}
	if lastSentDate == today {
		return false, today, nil
	}
	return true, today, nil
}

func (s *DailyBookingReportService) GetReportWindow(now time.Time) (string, time.Time, time.Time, error) {
	vietnamNow, err := s.getVietnamNow(now)
	if err != nil {
		return "", time.Time{}, time.Time{}, err
	}
	reportDate := vietnamNow.AddDate(0, 0, -1)
	start := time.Date(reportDate.Year(), reportDate.Month(), reportDate.Day(), 0, 0, 0, 0, reportDate.Location())
	end := start.Add(24 * time.Hour)
	return start.Format("2006-01-02"), start, end, nil
}

func (s *DailyBookingReportService) buildUserName(row *DailyBookingReportRow) string {
	name := strings.TrimSpace(row.UserFirstname + " " + row.UserLastname)
	if name == "" {
		return row.UserEmail
	}
	return name
}

func (s *DailyBookingReportService) buildBookingTime(row *DailyBookingReportRow) string {
	return row.Enter.Format("15:04") + " - " + row.Leave.Format("15:04")
}

func (s *DailyBookingReportService) buildHTML(reportDate string, rows []DailyBookingReportPreviewRow) string {
	var b strings.Builder
	b.WriteString("<div style=\"font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:1.6;color:#1f2937;\">")
	b.WriteString("<h1 style=\"margin:0 0 12px;font-size:24px;line-height:1.3;color:#111827;\">Daily booking report for " + html.EscapeString(reportDate) + "</h1>")
	b.WriteString("<p style=\"margin:0 0 20px;font-size:14px;\">Approved bookings for " + html.EscapeString(reportDate) + ".</p>")
	if len(rows) == 0 {
		b.WriteString("<p style=\"margin:0 0 20px;font-size:14px;\">No approved bookings for " + html.EscapeString(reportDate) + ".</p>{{footer}}</div>")
		return b.String()
	}
	b.WriteString("<table style=\"width:100%;border-collapse:collapse;font-size:14px;margin:0 0 20px;\">")
	b.WriteString("<thead><tr>")
	b.WriteString("<th style=\"border:1px solid #d1d5db;background-color:#f3f4f6;padding:12px 16px;text-align:left;\">User name</th>")
	b.WriteString("<th style=\"border:1px solid #d1d5db;background-color:#f3f4f6;padding:12px 16px;text-align:left;\">Area</th>")
	b.WriteString("<th style=\"border:1px solid #d1d5db;background-color:#f3f4f6;padding:12px 16px;text-align:left;\">Space</th>")
	b.WriteString("<th style=\"border:1px solid #d1d5db;background-color:#f3f4f6;padding:12px 16px;text-align:left;\">Booking time</th>")
	b.WriteString("</tr></thead><tbody>")
	for _, row := range rows {
		b.WriteString("<tr>")
		b.WriteString("<td style=\"border:1px solid #d1d5db;padding:12px 16px;vertical-align:top;\">" + html.EscapeString(row.UserName) + "</td>")
		b.WriteString("<td style=\"border:1px solid #d1d5db;padding:12px 16px;vertical-align:top;\">" + html.EscapeString(row.Area) + "</td>")
		b.WriteString("<td style=\"border:1px solid #d1d5db;padding:12px 16px;vertical-align:top;\">" + html.EscapeString(row.Space) + "</td>")
		b.WriteString("<td style=\"border:1px solid #d1d5db;padding:12px 16px;vertical-align:top;\">" + html.EscapeString(row.BookingTime) + "</td>")
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table>{{footer}}</div>")
	return b.String()
}

func (s *DailyBookingReportService) getOverrideWindow(overrideDate *time.Time) (string, time.Time, time.Time, error) {
	loc, err := s.getVietnamLocation()
	if err != nil {
		return "", time.Time{}, time.Time{}, err
	}
	start := time.Date(overrideDate.Year(), overrideDate.Month(), overrideDate.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)
	return start.Format("2006-01-02"), start, end, nil
}

func (s *DailyBookingReportService) BuildPreview(now time.Time, overrideDate *time.Time, overrideRecipients []string) (*DailyBookingReportPreview, error) {
	reportDate, start, end, err := s.GetReportWindow(now)
	if err != nil {
		return nil, err
	}
	if overrideDate != nil {
		reportDate, start, end, err = s.getOverrideWindow(overrideDate)
		if err != nil {
			return nil, err
		}
	}

	sourceRecipients := GetConfig().DailyBookingReportRecipients
	if len(overrideRecipients) > 0 {
		sourceRecipients = overrideRecipients
	}
	recipients, err := s.ParseRecipients(sourceRecipients)
	if err != nil {
		return nil, err
	}

	rows, err := GetBookingRepository().GetDailyBookingReportRows(start, end)
	if err != nil {
		return nil, err
	}

	previewRows := make([]DailyBookingReportPreviewRow, 0, len(rows))
	for _, row := range rows {
		previewRows = append(previewRows, DailyBookingReportPreviewRow{
			UserName:    s.buildUserName(row),
			Area:        row.Area,
			Space:       row.Space,
			BookingTime: s.buildBookingTime(row),
		})
	}

	return &DailyBookingReportPreview{
		Subject:     "Daily booking report for " + reportDate,
		ReportDate:  reportDate,
		Recipients:  recipients,
		RowCount:    len(previewRows),
		Rows:        previewRows,
		PreviewHTML: s.buildHTML(reportDate, previewRows),
	}, nil
}

func (s *DailyBookingReportService) Send(preview *DailyBookingReportPreview) error {
	var failures []string
	for _, recipient := range preview.Recipients {
		err := SendEmailWithBodyAndOrg(
			&MailAddress{Address: recipient, DisplayName: recipient},
			preview.Subject,
			preview.PreviewHTML,
			"en",
			"",
		)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", recipient, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("daily booking report send failed for %s", strings.Join(failures, "; "))
	}
	return nil
}

func (s *DailyBookingReportService) RunIfDue(now time.Time) error {
	if !GetConfig().DailyBookingReportEnabled {
		return nil
	}
	lastSentDate, _ := GetSettingsRepository().GetGlobalString(SettingDailyBookingReportLastSentDate.Name)
	shouldRun, today, err := s.ShouldRun(now, lastSentDate)
	if err != nil || !shouldRun {
		return err
	}

	preview, err := s.BuildPreview(now, nil, nil)
	if err != nil {
		return err
	}
	if err := s.Send(preview); err != nil {
		return err
	}
	return GetSettingsRepository().SetGlobal(SettingDailyBookingReportLastSentDate.Name, today)
}
