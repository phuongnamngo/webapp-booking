package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type CalDAVClient struct {
	url        string
	httpClient webdav.HTTPClient
	client     *caldav.Client
	principal  string
	homeSet    string
}

type CalDAVCalendar struct {
	Path string
	Name string
}

type CalDAVEvent struct {
	ID       string
	Title    string
	Start    time.Time
	End      time.Time
	Location string
}

func (c *CalDAVClient) Connect(url, username, password string) error {
	httpClient := webdav.HTTPClientWithBasicAuth(http.DefaultClient, username, password)
	caldavClient, err := caldav.NewClient(httpClient, url)
	if err != nil {
		return err
	}
	principal, err := caldavClient.FindCurrentUserPrincipal(context.Background())
	if err != nil {
		return err
	}
	homeSet, err := caldavClient.FindCalendarHomeSet(context.Background(), principal)
	if err != nil {
		return err
	}
	c.url = url
	c.client = caldavClient
	c.httpClient = httpClient
	c.principal = principal
	c.homeSet = homeSet
	return nil
}

func (c *CalDAVClient) ConnectWithTokenSource(url string, src oauth2.TokenSource) error {
	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Source: src,
			Base:   http.DefaultTransport,
		},
	}
	caldavClient, err := caldav.NewClient(httpClient, url)
	if err != nil {
		return err
	}
	principal, err := caldavClient.FindCurrentUserPrincipal(context.Background())
	if err != nil {
		return err
	}
	homeSet, err := caldavClient.FindCalendarHomeSet(context.Background(), principal)
	if err != nil {
		return err
	}
	c.url = url
	c.client = caldavClient
	c.httpClient = httpClient
	c.principal = principal
	c.homeSet = homeSet
	return nil
}

func (c *CalDAVClient) ListCalendars() ([]*CalDAVCalendar, error) {
	calendars, err := c.client.FindCalendars(context.Background(), c.homeSet)
	if err != nil {
		return nil, err
	}
	res := make([]*CalDAVCalendar, 0)
	for _, calendar := range calendars {
		res = append(res, &CalDAVCalendar{
			Path: calendar.Path,
			Name: calendar.Name,
		})
	}
	return res, nil
}

func (c *CalDAVClient) CreateEvent(calendarPath string, e *CalDAVEvent) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	cal := c.GetCaldavEvent([]*CalDAVEvent{e})

	_, err := c.client.PutCalendarObject(context.Background(), path.Join(calendarPath, e.ID+".ics"), cal)
	return err
}

// resolveHref mirrors emersion/go-webdav internal.Client.ResolveHref so DELETE
// targets the same URL as PutCalendarObject (CreateEvent).
func (c *CalDAVClient) resolveHref(p string) (string, error) {
	u, err := url.Parse(c.url)
	if err != nil {
		return "", err
	}
	if u.Path == "" {
		u.Path = "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = path.Join(u.Path, p)
	}
	return (&url.URL{
		Scheme: u.Scheme,
		User:   u.User,
		Host:   u.Host,
		Path:   p,
	}).String(), nil
}

func (c *CalDAVClient) DeleteEvent(calendarPath string, e *CalDAVEvent) error {
	if e.ID == "" {
		return fmt.Errorf("caldav: event id is required")
	}
	objectPath := path.Join(calendarPath, e.ID+".ics")
	reqURL, err := c.resolveHref(objectPath)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req.WithContext(context.Background()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("caldav delete failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *CalDAVClient) GetCaldavEvent(events []*CalDAVEvent) *ical.Calendar {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropProductID, "-//lntbooking.io//LNT Booking//EN")
	cal.Props.SetText(ical.PropVersion, "2.0")

	for _, e := range events {
		event := ical.NewEvent()
		event.Props.SetText(ical.PropSummary, e.Title)
		event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())
		event.Props.SetDateTime(ical.PropDateTimeStart, e.Start)
		event.Props.SetDateTime(ical.PropDateTimeEnd, e.End)
		event.Props.SetText(ical.PropLocation, e.Location)
		event.Props.Del(ical.PropDuration)
		event.Props.SetText(ical.PropUID, e.ID)
		cal.Children = append(cal.Children, event.Component)
	}

	return cal
}
