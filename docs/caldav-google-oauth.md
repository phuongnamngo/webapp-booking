# Google Calendar CalDAV (OAuth 2.0)

## Overview

iDesk Booking supports two CalDAV integration modes:

- **Google Calendar** — OAuth 2.0 (recommended for `@gmail.com` / Google Workspace)
- **Other CalDAV server** — URL + username + password (Basic Auth, deprecated)

## CalDAV endpoint (Google)

The client must use the v2 discovery root **`https://apidata.googleusercontent.com/caldav/v2`** with OAuth. Google resolves `current-user-principal` from the bearer token at that root. Appending the user email to the path (for example `.../v2/user@gmail.com`) is wrong and leads to **404** on `DAV: current-user-principal`.

## Server configuration

Set in environment (or `.env` for local dev):

```bash
GOOGLE_CALDAV_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CALDAV_CLIENT_SECRET=your-client-secret
CRYPT_KEY=your-32-byte-encryption-key
PUBLIC_SCHEME=http
PUBLIC_PORT=8080
# When the Next.js UI runs on a port other than 3000 (e.g. 3003), set this so
# the Go dev server proxies /ui/* to the correct host:port and OAuth redirects
# land back in your app instead of a blank page.
DEV_UI_PROXY=localhost:3003
```

## Google Cloud Console setup

1. Create a project in [Google Cloud Console](https://console.cloud.google.com/).
2. Configure **OAuth consent screen** (Testing mode is fine for development).
3. Create **OAuth 2.0 Client ID** → Application type: **Web application**.
4. Add **Authorized redirect URI**:
   - Local: `http://localhost:8080/preference/caldav/google/callback`
   - Production: `https://your-domain/preference/caldav/google/callback`
5. Add test users while app is in Testing mode.
6. Scopes used: `https://www.googleapis.com/auth/calendar` and `https://www.googleapis.com/auth/userinfo.email` (needed to resolve the Google account email for CalDAV; add both on the OAuth consent screen if you restrict scopes).
7. Under **OAuth consent screen** → **Test users**: add every Google account that will connect (while app is in **Testing**). If the user is not listed, Google returns **403 `access_denied`**.
8. If you use a **Google Workspace** account, an admin may need to allow the OAuth client, or you must publish / verify the app per org policy.

## Google error: 403 access_denied

Common causes:

- App status is **Testing** but this Gmail / Workspace user is **not** in the Test users list → add them in [APIs & Services → OAuth consent screen → Test users](https://console.cloud.google.com/apis/credentials/consent).
- User clicked **Cancel** on the consent screen → try again and choose **Allow**.
- Workspace **blocks third-party OAuth** for this scope → IT must allow the client or you test with a personal `@gmail.com` account added as test user.

## User flow

1. Open **Preferences → Integrations**.
2. Select **Google Calendar**.
3. Click **Connect Google Calendar** → sign in and grant calendar access.
4. After redirect, click **Connect** to load calendars (or auto-load after OAuth).
5. Select a calendar → **Save**.

## Manual QA checklist

- [ ] Google OAuth connect completes and shows connected email
- [ ] Calendar list loads without 401
- [ ] Save persists selected calendar
- [ ] Approved booking appears in Google Calendar
- [ ] Booking update/delete syncs to Google Calendar
- [ ] **Other CalDAV server** path still works with URL/username/password
- [ ] Disconnect clears all CalDAV preferences
