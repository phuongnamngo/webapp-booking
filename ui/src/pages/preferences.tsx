import React from "react";
import Loading from "../components/Loading";
import { Alert, Button, Form, Modal } from "react-bootstrap";
import { NextRouter } from "next/router";
import { IoLinkOutline } from "react-icons/io5";
import NavBar from "@/components/NavBar";
import withReadyRouter from "@/components/withReadyRouter";
import RuntimeConfig from "@/components/RuntimeConfig";
import { TranslationFunc, withTranslation } from "@/components/withTranslation";
import Ajax from "@/util/Ajax";
import UserPreference from "@/types/UserPreference";
import Location from "@/types/Location";
import RedirectUtil from "@/util/RedirectUtil";
import Session from "@/types/Session";
import JwtDecoder from "@/util/JwtDecoder";
import Formatting from "@/util/Formatting";
import Validation from "@/util/Validation";
import TotpSettings from "@/components/TotpSettings";
import PasskeySettings from "@/components/PasskeySettings";
import SaveButton from "@/components/SaveButton";
import Passkey from "@/types/Passkey";
import RendererUtils from "@/util/RendererUtils";

interface State {
  loading: boolean;
  submitting: boolean;
  saved: boolean;
  error: boolean;
  enterTime: number;
  workdayStart: number;
  workdayEnd: number;
  workdays: boolean[];
  booked: string;
  notBooked: string;
  selfBooked: string;
  partiallyBooked: string;
  buddyBooked: string;
  disallowed: string;
  locationId: string;
  changePassword: boolean;
  password: string;
  activeTab: string;
  caldavUrl: string;
  caldavUser: string;
  caldavPass: string;
  caldavCalendar: string;
  caldavCalendars: any[];
  caldavCalendarsLoaded: boolean;
  caldavError: boolean;
  caldavProvider: string;
  caldavGoogleEmail: string;
  caldavLegacyGoogleReconnect: boolean;
  /** OAuth callback error code from `caldav_oauth_error` query (server redirect). */
  caldavOAuthError: string | null;
  mailNotifications: boolean;
  use24HourTime: boolean;
  dateFormat: string;
  activeSessions: Session[];
  currentSessionId: string;
  showPasswordChangedModal: boolean;
}

interface Props {
  router: NextRouter;
  t: TranslationFunc;
}

const COLOR_BOOKED: string = "#ff453a";
const COLOR_NOT_BOOKED: string = "#30d158";
const COLOR_SELF_BOOKED: string = "#b825de";
const COLOR_PARTIALLY_BOOKED: string = "#ff9100";
const COLOR_BUDDY_BOOKED: string = "#2415c5";
const COLOR_DISALLOWED: string = "#eeeeee";

class Preferences extends React.Component<Props, State> {
  locations: Location[];

  constructor(props: any) {
    super(props);
    this.locations = [];
    this.state = {
      loading: true,
      submitting: false,
      saved: false,
      error: false,
      enterTime: 0,
      workdayStart: 0,
      workdayEnd: 0,
      workdays: [],
      booked: COLOR_BOOKED,
      notBooked: COLOR_NOT_BOOKED,
      selfBooked: COLOR_SELF_BOOKED,
      partiallyBooked: COLOR_PARTIALLY_BOOKED,
      buddyBooked: COLOR_BUDDY_BOOKED,
      disallowed: COLOR_DISALLOWED,
      locationId: "",
      changePassword: false,
      password: "",
      activeTab: "tab-bookings",
      caldavUrl: "",
      caldavUser: "",
      caldavPass: "",
      caldavCalendar: "",
      caldavCalendars: [],
      caldavCalendarsLoaded: false,
      caldavError: false,
      caldavProvider: UserPreference.CALDAV_PROVIDER_GOOGLE,
      caldavGoogleEmail: "",
      caldavLegacyGoogleReconnect: false,
      caldavOAuthError: null,
      mailNotifications: false,
      use24HourTime: true,
      dateFormat: "Y-m-d",
      activeSessions: [],
      currentSessionId: "",
      showPasswordChangedModal: false,
    };
  }

  componentDidMount = () => {
    if (!Ajax.hasAccessToken()) {
      RedirectUtil.toLogin(this.props.router);
      return;
    }
    const tabParam = this.props.router.query.tab;
    if (tabParam === "security") {
      this.setState({ activeTab: "tab-security" });
    }
    if (tabParam === "integrations") {
      this.setState({ activeTab: "tab-integrations" });
    }
    const promises = [
      this.loadPreferences(),
      this.loadLocations(),
      this.loadActiveSessions(),
    ];
    Promise.all(promises).then(() => {
      this.setState({ loading: false }, () => {
        const q = this.props.router.query;
        const oauthErrRaw = q.caldav_oauth_error;
        const cleanIntegrationsUrl = () => {
          void this.props.router.replace(
            {
              pathname: this.props.router.pathname,
              query: { tab: "integrations" },
            },
            undefined,
            { shallow: true },
          );
        };
        if (oauthErrRaw) {
          const code = Array.isArray(oauthErrRaw)
            ? oauthErrRaw[0]
            : oauthErrRaw;
          this.setState({
            activeTab: "tab-integrations",
            caldavOAuthError: code || null,
          });
          cleanIntegrationsUrl();
          return;
        }
        if (q.caldav === "connected") {
          this.setState({ activeTab: "tab-integrations" });
          this.loadPreferences().then(() => {
            this.listGoogleCalDavCalendars();
            cleanIntegrationsUrl();
          });
        }
      });
    });
  };

  getCaldavOAuthErrorMessage = (code: string): string => {
    const keys: Record<string, string> = {
      access_denied: "caldavOAuthErrorAccessDenied",
      invalid_state: "caldavOAuthErrorInvalidState",
      missing_code: "caldavOAuthErrorMissingCode",
      token_exchange: "caldavOAuthErrorTokenExchange",
      no_refresh_token: "caldavOAuthErrorNoRefreshToken",
      userinfo: "caldavOAuthErrorUserinfo",
      crypt_key: "caldavOAuthErrorCryptKey",
      not_configured: "caldavOAuthErrorNotConfigured",
      server: "caldavOAuthErrorServer",
    };
    const k = keys[code] ?? "caldavOAuthErrorUnknown";
    return k === "caldavOAuthErrorUnknown"
      ? this.props.t(k, { code })
      : this.props.t(k);
  };

  loadActiveSessions = async (): Promise<void> => {
    const accessTokenPayload = JwtDecoder.getPayload(
      Ajax.PERSISTER.readCredentialsFromLocalStorage().accessToken,
    );
    const self = this;
    return new Promise<void>(function (resolve, reject) {
      Session.list()
        .then((sessions) => {
          self.setState({
            activeSessions: sessions,
            currentSessionId: accessTokenPayload.sid,
          });
          resolve();
        })
        .catch((e) => reject(e));
    });
  };

  loadPreferences = async (): Promise<void> => {
    const self = this;
    return new Promise<void>(function (resolve, reject) {
      UserPreference.list()
        .then((list) => {
          const state: any = {};
          list.forEach((s) => {
            if (typeof window !== "undefined") {
              if (s.name === UserPreference.PREF_ENTER_TIME)
                state.enterTime = window.parseInt(s.value);
              if (s.name === UserPreference.PREF_WORKDAY_START)
                state.workdayStart = window.parseInt(s.value);
              if (s.name === UserPreference.PREF_WORKDAY_END)
                state.workdayEnd = window.parseInt(s.value);
            }
            if (s.name === UserPreference.PREF_WORKDAYS) {
              state.workdays = [];
              for (let i = 0; i <= 6; i++) {
                state.workdays[i] = false;
              }
              s.value.split(",").forEach((val) => (state.workdays[val] = true));
            }
            if (s.name === UserPreference.PREF_BOOKED_COLOR)
              state.booked = s.value;
            if (s.name === UserPreference.PREF_NOT_BOOKED_COLOR)
              state.notBooked = s.value;
            if (s.name === UserPreference.PREF_SELF_BOOKED_COLOR)
              state.selfBooked = s.value;
            if (s.name === UserPreference.PREF_PARTIALLY_BOOKED_COLOR)
              state.partiallyBooked = s.value;
            if (s.name === UserPreference.PREF_BUDDY_BOOKED_COLOR)
              state.buddyBooked = s.value;
            if (s.name === UserPreference.PREF_DISALLOWED_COLOR)
              state.disallowed = s.value;
            if (s.name === UserPreference.PREF_LOCATION_ID)
              state.locationId = s.value;
            if (s.name === UserPreference.PREF_CALDAV_URL)
              state.caldavUrl = s.value;
            if (s.name === UserPreference.PREF_CALDAV_USER)
              state.caldavUser = s.value;
            if (s.name === UserPreference.PREF_CALDAV_PASS)
              state.caldavPass = s.value;
            if (s.name === UserPreference.PREF_CALDAV_PATH)
              state.caldavCalendar = s.value;
            if (s.name === UserPreference.PREF_CALDAV_PROVIDER)
              state.caldavProvider =
                s.value || UserPreference.CALDAV_PROVIDER_GOOGLE;
            if (s.name === UserPreference.PREF_CALDAV_GOOGLE_EMAIL)
              state.caldavGoogleEmail = s.value;
            if (s.name === UserPreference.PREF_MAIL_NOTIFICATIONS)
              state.mailNotifications = s.value === "1";
            if (s.name === UserPreference.PREF_USE_24_HOUR_TIME)
              state.use24HourTime = s.value === "1";
            if (s.name === UserPreference.PREF_DATE_FORMAT)
              state.dateFormat = s.value;
          });
          state.caldavLegacyGoogleReconnect =
            ((state.caldavUrl as string) || "").includes(
              "googleusercontent.com",
            ) &&
            !(state.caldavGoogleEmail as string) &&
            !!(state.caldavPass as string);
          self.setState(
            {
              ...self.state,
              ...state,
            },
            () => resolve(),
          );
        })
        .catch((e) => reject(e));
    });
  };

  loadLocations = async (): Promise<void> => {
    const self = this;
    return new Promise<void>(function (resolve, reject) {
      Location.list()
        .then((list) => {
          self.locations = list;
          resolve();
        })
        .catch((e) => reject(e));
    });
  };

  onSubmit = (e: any) => {
    e.preventDefault();
    this.setState({
      submitting: true,
      saved: false,
      error: false,
      caldavError: false,
    });
    const workdays: string[] = [];
    this.state.workdays.forEach((val, day) => {
      if (val) {
        workdays.push(day.toString());
      }
    });
    const payload = [
      new UserPreference("enter_time", this.state.enterTime.toString()),
      new UserPreference("workday_start", this.state.workdayStart.toString()),
      new UserPreference("workday_end", this.state.workdayEnd.toString()),
      new UserPreference("workdays", workdays.join(",")),
      new UserPreference(
        "mail_notifications",
        this.state.mailNotifications ? "1" : "0",
      ),
      new UserPreference(
        "use_24_hour_time",
        this.state.use24HourTime ? "1" : "0",
      ),
      new UserPreference("location_id", this.state.locationId),
      new UserPreference("date_format", this.state.dateFormat),
    ];
    UserPreference.setAll(payload)
      .then(() => {
        RuntimeConfig.loadUserPreferences().then(() => {
          this.setState({
            submitting: false,
            saved: true,
          });
        });
      })
      .catch(() => {
        this.setState({
          submitting: false,
          error: true,
        });
      });
  };

  onSubmitSecurity = async (e: any) => {
    e.preventDefault();
    if (!this.state.changePassword) {
      return;
    }
    this.setState({
      submitting: true,
      saved: false,
      error: false,
      caldavError: false,
    });
    const payload = {
      password: this.state.password,
    };

    await Ajax.putData("/user/me/password", payload);
    this.setState({ submitting: false, showPasswordChangedModal: true });
  };

  onSubmitColors = (e: any) => {
    e.preventDefault();
    this.setState({
      submitting: true,
      saved: false,
      error: false,
      caldavError: false,
    });
    const workdays: string[] = [];
    this.state.workdays.forEach((val, day) => {
      if (val) {
        workdays.push(day.toString());
      }
    });
    const payload = [
      new UserPreference("booked_color", this.state.booked),
      new UserPreference("not_booked_color", this.state.notBooked),
      new UserPreference("self_booked_color", this.state.selfBooked),
      new UserPreference("partially_booked_color", this.state.partiallyBooked),
      new UserPreference("buddy_booked_color", this.state.buddyBooked),
      new UserPreference("disallowed_color", this.state.disallowed),
    ];
    UserPreference.setAll(payload)
      .then(() => {
        this.setState({
          submitting: false,
          saved: true,
        });
      })
      .catch(() => {
        this.setState({
          submitting: false,
          error: true,
        });
      });
  };

  resetColors = () => {
    this.setState({
      booked: COLOR_BOOKED,
      notBooked: COLOR_NOT_BOOKED,
      selfBooked: COLOR_SELF_BOOKED,
      partiallyBooked: COLOR_PARTIALLY_BOOKED,
      buddyBooked: COLOR_BUDDY_BOOKED,
      disallowed: COLOR_DISALLOWED,
    });
  };

  onWorkdayCheck = (day: number, checked: boolean) => {
    const workdays = this.state.workdays.map((val, i) =>
      i === day ? checked : val,
    );
    this.setState({
      workdays: workdays,
    });
  };

  listGoogleCalDavCalendars = () => {
    this.setState({
      submitting: true,
      saved: false,
      error: false,
      caldavError: false,
      caldavCalendarsLoaded: false,
    });
    Ajax.postData("/preference/caldav/listCalendars", {
      provider: UserPreference.CALDAV_PROVIDER_GOOGLE,
    })
      .then((res) => {
        this.setState({
          caldavCalendarsLoaded: true,
          caldavCalendars: res.json,
          caldavCalendar:
            res.json && res.json.length > 0
              ? res.json[0].path
              : this.state.caldavCalendar,
          submitting: false,
        });
      })
      .catch(() => {
        this.setState({
          submitting: false,
          caldavError: true,
        });
      });
  };

  connectGoogleCalDav = () => {
    this.setState({
      caldavError: false,
      error: false,
    });
    Ajax.get("/preference/caldav/google/auth-url")
      .then((res) => {
        if (res.json && res.json.url) {
          window.location.href = res.json.url;
        } else {
          this.setState({ caldavError: true });
        }
      })
      .catch(() => {
        this.setState({ caldavError: true });
      });
  };

  connectCalDav = () => {
    this.setState({
      submitting: true,
      saved: false,
      error: false,
      caldavError: false,
      caldavCalendarsLoaded: false,
    });
    const payload = {
      provider: UserPreference.CALDAV_PROVIDER_GENERIC,
      url: this.state.caldavUrl,
      username: this.state.caldavUser,
      password: this.state.caldavPass,
    };
    Ajax.postData("/preference/caldav/listCalendars", payload)
      .then((res) => {
        this.setState({
          caldavCalendarsLoaded: true,
          caldavCalendars: res.json,
          caldavCalendar:
            res.json && res.json.length > 0 ? res.json[0].path : "",
          submitting: false,
        });
      })
      .catch(() => {
        this.setState({
          submitting: false,
          caldavError: true,
        });
      });
  };

  disconnectCalDav = () => {
    this.setState({
      submitting: true,
      saved: false,
      error: false,
      caldavError: false,
      caldavCalendarsLoaded: false,
    });
    const payload = [
      new UserPreference("caldav_provider", ""),
      new UserPreference("caldav_oauth_refresh", ""),
      new UserPreference("caldav_google_email", ""),
      new UserPreference("caldav_url", ""),
      new UserPreference("caldav_user", ""),
      new UserPreference("caldav_pass", ""),
      new UserPreference("caldav_path", ""),
    ];
    UserPreference.setAll(payload)
      .then(() => {
        this.setState({
          submitting: false,
          saved: true,
          caldavProvider: UserPreference.CALDAV_PROVIDER_GOOGLE,
          caldavGoogleEmail: "",
          caldavUrl: "",
          caldavUser: "",
          caldavPass: "",
          caldavCalendar: "",
          caldavCalendars: [],
          caldavLegacyGoogleReconnect: false,
        });
      })
      .catch(() => {
        this.setState({
          submitting: false,
          error: true,
        });
      });
  };

  saveCaldavSettings = (e: any) => {
    e.preventDefault();
    this.setState({
      submitting: true,
      saved: false,
      error: false,
      caldavError: false,
    });
    const payload = [
      new UserPreference(
        "caldav_provider",
        this.state.caldavProvider,
      ),
      new UserPreference("caldav_path", this.state.caldavCalendar),
    ];
    if (this.state.caldavProvider === UserPreference.CALDAV_PROVIDER_GENERIC) {
      payload.push(
        new UserPreference("caldav_url", this.state.caldavUrl),
        new UserPreference("caldav_user", this.state.caldavUser),
        new UserPreference("caldav_pass", this.state.caldavPass),
      );
    }
    UserPreference.setAll(payload)
      .then(() => {
        this.setState({
          submitting: false,
          saved: true,
        });
      })
      .catch(() => {
        this.setState({
          submitting: false,
          error: true,
        });
      });
  };

  renderBookingColor(
    stateKey: keyof Pick<
      State,
      | "notBooked"
      | "selfBooked"
      | "booked"
      | "partiallyBooked"
      | "buddyBooked"
      | "disallowed"
    >,
    labelKey: string,
  ) {
    const id = `color${RendererUtils.capitalize(stateKey)}`;
    return (
      <div className="preferences-color-row" key={id}>
        <label htmlFor={id} className="preferences-color-swatches">
          <Form.Control
            type="color"
            id={id}
            value={this.state[stateKey]}
            onChange={(e: any) =>
              this.setState({ [stateKey]: e.target.value } as any)
            }
            aria-label={this.props.t(labelKey)}
          />
        </label>
        <span className="preferences-color-label">
          {this.props.t(labelKey)}
        </span>
      </div>
    );
  }

  renderPrefsSubnavTab(tabKey: string, label: string): React.ReactElement {
    const selected = this.state.activeTab === tabKey;
    return (
      <button
        type="button"
        key={tabKey}
        role="tab"
        aria-selected={selected}
        id={`prefs-subnav-${tabKey}`}
        className={
          "preferences-subnav-tab" +
          (selected ? " preferences-subnav-tab-active" : "")
        }
        onClick={() =>
          this.setState({
            activeTab: tabKey,
            error: false,
            saved: false,
          })
        }
      >
        <span className="preferences-subnav-tab-label">{label}</span>
        <span className="preferences-subnav-tab-line" aria-hidden />
      </button>
    );
  }

  render() {
    if (this.state.loading) {
      return <Loading />;
    }

    let hint = <></>;
    if (this.state.saved) {
      hint = (
        <Alert variant="success" className="preferences-alert">
          {this.props.t("entryUpdated")}
        </Alert>
      );
    } else if (this.state.error) {
      hint = (
        <Alert variant="danger" className="preferences-alert">
          {this.props.t("errorSave")}
        </Alert>
      );
    } else if (this.state.caldavError) {
      hint = (
        <Alert variant="danger" className="preferences-alert">
          {this.props.t("errorCaldav")}
        </Alert>
      );
    }

    const credentials = Ajax.PERSISTER.readCredentialsFromLocalStorage();
    const profilePageUrl = credentials.profilePageUrl;

    return (
      <>
        <NavBar />
        <div className="container-center-top preferences-layout">
          <div className="preferences-shell">
            <nav
              className="preferences-subnav-bar"
              aria-label={this.props.t("preferences")}
              role="tablist"
            >
              {this.renderPrefsSubnavTab(
                "tab-bookings",
                this.props.t("bookings"),
              )}
              {this.renderPrefsSubnavTab("tab-style", this.props.t("style"))}
              {!RuntimeConfig.INFOS.idpLogin
                ? this.renderPrefsSubnavTab(
                  "tab-security",
                  this.props.t("security"),
                )
                : null}
              {RuntimeConfig.INFOS.idpLogin && profilePageUrl
                ? this.renderPrefsSubnavTab(
                  "tab-idp",
                  this.props.t("security"),
                )
                : null}
              {this.renderPrefsSubnavTab(
                "tab-integrations",
                this.props.t("integrations"),
              )}
            </nav>
            {hint}

            {/* -------- */}
            {/* BOOKINGS */}
            {/* -------- */}

            <Form
              onSubmit={this.onSubmit}
              hidden={this.state.activeTab !== "tab-bookings"}
            >
              <div className="preferences-panel">
                <Form.Group className="preferences-field">
                  <Form.Label htmlFor="enterTime">
                    {this.props.t("notice")}
                  </Form.Label>
                  <Form.Select
                    id="enterTime"
                    value={this.state.enterTime}
                    onChange={(e: any) =>
                      this.setState({ enterTime: e.target.value })
                    }
                  >
                    <option value="1">{this.props.t("earliestPossible")}</option>
                    <option value="2">{this.props.t("nextDay")}</option>
                    <option value="3">{this.props.t("nextWorkday")}</option>
                  </Form.Select>
                </Form.Group>
                <Form.Group className="preferences-field">
                  <Form.Label htmlFor="workdayStart">
                    {this.props.t("workingHours")}
                  </Form.Label>
                  <div className="preferences-working-hours-row">
                    <Form.Control
                      type="number"
                      id="workdayStart"
                      className="preferences-hour-input"
                      value={this.state.workdayStart}
                      onChange={(e: any) =>
                        this.setState({
                          workdayStart:
                            typeof window !== "undefined"
                              ? window.parseInt(e.target.value, 10)
                              : 0,
                        })
                      }
                      min={0}
                      max={23}
                    />
                    <Form.Label
                      htmlFor="workdayEnd"
                      className="preferences-working-hours-sep"
                    >
                      {this.props.t("to")}
                    </Form.Label>
                    <Form.Control
                      type="number"
                      id="workdayEnd"
                      className="preferences-hour-input"
                      value={this.state.workdayEnd}
                      onChange={(e: any) =>
                        this.setState({
                          workdayEnd:
                            typeof window !== "undefined"
                              ? window.parseInt(e.target.value, 10)
                              : 0,
                        })
                      }
                      min={this.state.workdayStart + 1}
                      max={23}
                    />
                  </div>
                </Form.Group>
                <Form.Group className="preferences-field preferences-workdays-block">
                  <Form.Label>{this.props.t("workdays")}</Form.Label>
                  <div className="preferences-workdays-row">
                    {[0, 1, 2, 3, 4, 5, 6].map((day) => (
                      <label
                        key={"workday-" + day}
                        className="preferences-workday-pill"
                      >
                        <input
                          type="checkbox"
                          checked={!!this.state.workdays[day]}
                          onChange={(e: any) =>
                            this.onWorkdayCheck(day, e.target.checked)
                          }
                        />
                        <span className="preferences-workday-box">
                          <span className="preferences-workday-check" />
                        </span>
                        <span>{this.props.t("workday-" + day)}</span>
                      </label>
                    ))}
                  </div>
                </Form.Group>
                <Form.Group className="preferences-field preferences-workdays-block">
                  <Form.Label htmlFor="mailNotifications-inline">
                    {this.props.t("mailNotifications")}
                  </Form.Label>
                  <label
                    htmlFor="mailNotifications-inline"
                    className="preferences-inline-check"
                  >
                    <input
                      type="checkbox"
                      id="mailNotifications-inline"
                      checked={this.state.mailNotifications}
                      onChange={(e: any) =>
                        this.setState({
                          mailNotifications: e.target.checked,
                        })
                      }
                    />
                    <span className="preferences-workday-box">
                      <span className="preferences-workday-check" />
                    </span>
                    <span>{this.props.t("mailNotifications")}</span>
                  </label>
                </Form.Group>
                <Form.Group className="preferences-field preferences-workdays-block">
                  <Form.Label htmlFor="use24HourTime-inline">
                    {this.props.t("timeFormat")}
                  </Form.Label>
                  <label
                    htmlFor="use24HourTime-inline"
                    className="preferences-inline-check"
                  >
                    <input
                      type="checkbox"
                      id="use24HourTime-inline"
                      checked={this.state.use24HourTime}
                      onChange={(e: any) =>
                        this.setState({ use24HourTime: e.target.checked })
                      }
                    />
                    <span className="preferences-workday-box">
                      <span className="preferences-workday-check" />
                    </span>
                    <span>{this.props.t("use24HourTime")}</span>
                  </label>
                </Form.Group>
                <Form.Group className="preferences-field">
                  <Form.Label htmlFor="dateFormat">
                    {this.props.t("dateFormat")}
                  </Form.Label>
                  <Form.Select
                    id="dateFormat"
                    value={this.state.dateFormat}
                    onChange={(e: any) =>
                      this.setState({ dateFormat: e.target.value })
                    }
                  >
                    <option value="Y-m-d">Y-m-d</option>
                    <option value="d.m.Y">d.m.Y</option>
                    <option value="m/d/Y">m/d/Y</option>
                    <option value="d/m/Y">d/m/Y</option>
                  </Form.Select>
                </Form.Group>
                <Form.Group className="preferences-field">
                  <Form.Label htmlFor="preferredLocation">
                    {this.props.t("preferredLocation")}
                  </Form.Label>
                  <Form.Select
                    id="preferredLocation"
                    value={this.state.locationId}
                    onChange={(e: any) =>
                      this.setState({ locationId: e.target.value })
                    }
                  >
                    <option value="">({this.props.t("none")})</option>
                    {this.locations.map((location) => (
                      <option
                        key={"location-" + location.id}
                        value={location.id}
                      >
                        {location.name}
                      </option>
                    ))}
                  </Form.Select>
                </Form.Group>
                <div>
                  <SaveButton
                    submitting={this.state.submitting}
                    className="preferences-btn-save"
                  />
                </div>
              </div>
            </Form>

            {/* ----- */}
            {/* STYLE */}
            {/* ----- */}

            <Form
              onSubmit={this.onSubmitColors}
              hidden={this.state.activeTab !== "tab-style"}
              className="form-colors"
            >
              <div className="preferences-panel">
                <h3 className="preferences-section-title">
                  {this.props.t("bookingcolors")}
                </h3>
                <div className="preferences-color-list">
                  {this.renderBookingColor("booked", "colorAlreadyBooked")}
                  {this.renderBookingColor("notBooked", "colorNotBooked")}
                  {this.renderBookingColor("selfBooked", "colorSelfBooked")}
                  {this.renderBookingColor(
                    "partiallyBooked",
                    "colorPartiallyBooked",
                  )}
                  {!RuntimeConfig.INFOS.disableBuddies &&
                    this.renderBookingColor(
                      "buddyBooked",
                      "colorBuddyBooked",
                    )}
                  {this.renderBookingColor("disallowed", "colorDisallowed")}
                </div>
                <div className="preferences-btn-row">
                  <Button
                    type="button"
                    className="preferences-btn-neutral"
                    variant="dark"
                    onClick={() => this.resetColors()}
                  >
                    {this.props.t("reset")}
                  </Button>
                  <SaveButton
                    submitting={this.state.submitting}
                    className="preferences-btn-save"
                  />
                </div>
              </div>
            </Form>

            {/* -------- */}
            {/* SECURITY */}
            {/* -------- */}

            <div hidden={this.state.activeTab !== "tab-security"}>
              <div className="preferences-panel">
                <Form onSubmit={this.onSubmitSecurity}>
                  <section className="preferences-security-block">
                    <h3 className="preferences-section-title">
                      {this.props.t("password")}
                    </h3>
                    <div className="preferences-field">
                      <Form.Check
                        type="checkbox"
                        id="check-changePassword"
                        label={this.props.t("passwordChange")}
                        checked={this.state.changePassword}
                        onChange={(e: any) =>
                          this.setState({ changePassword: e.target.checked })
                        }
                      />
                      <Form.Control
                        type="password"
                        autoComplete="new-password"
                        value={this.state.password}
                        onChange={(e: any) =>
                          this.setState({ password: e.target.value })
                        }
                        required={this.state.changePassword}
                        disabled={!this.state.changePassword}
                        minLength={Validation.PASSWORD_MIN_LENGTH}
                        maxLength={Validation.PASSWORD_MAX_LENGTH}
                        pattern={Validation.PASSWORD_PATTERN}
                        title={this.props.t("passwordRequirements")}
                      />
                    </div>
                    <div>
                      <SaveButton
                        submitting={this.state.submitting}
                        disabled={!this.state.changePassword}
                        className="preferences-btn-save"
                      />
                    </div>
                  </section>
                </Form>
                <TotpSettings
                  hidden={RuntimeConfig.INFOS.idpLogin}
                  className="preferences-security-block"
                  t={this.props.t}
                />
                <PasskeySettings
                  hidden={RuntimeConfig.INFOS.idpLogin}
                  className="preferences-security-block"
                  t={this.props.t}
                  onPasskeyAdded={() => {
                    RuntimeConfig.INFOS.hasPasskeys = true;
                  }}
                  onPasskeyDeleted={() => {
                    Passkey.list().then((passkeys) => {
                      RuntimeConfig.INFOS.hasPasskeys = passkeys.length > 0;
                    });
                  }}
                />
                <div className="preferences-sessions-wrap">
                  <h3 className="preferences-sessions-head">
                    {this.props.t("activeSessions")}
                  </h3>
                  {this.state.activeSessions.length === 0 ? (
                    <p>{this.props.t("noActiveSessions")}</p>
                  ) : (
                    <>
                      <div
                        className="preferences-sessions-list"
                        role="table"
                      >
                        <div
                          className="preferences-sessions-header"
                          role="row"
                        >
                          <span role="columnheader">
                            {this.props.t("device")}
                          </span>
                          <span role="columnheader">
                            {this.props.t("created")} (UTC)
                          </span>
                          <span role="columnheader" aria-label="Actions" />
                        </div>
                        {this.state.activeSessions.map((session, idx) => (
                          <React.Fragment key={"session-" + session.id}>
                            <div
                              className="preferences-sessions-divider"
                              aria-hidden
                            />
                            <div
                              className="preferences-sessions-row"
                              role="row"
                            >
                              <span role="cell">
                                {session.device}
                                {session.id === this.state.currentSessionId
                                  ? " *"
                                  : ""}
                              </span>
                              <span role="cell">
                                {Formatting.getFormatterShort(false).format(
                                  new Date(session.created),
                                )}
                              </span>
                              <span role="cell">
                                <button
                                  type="button"
                                  className="preferences-link-logout"
                                  onClick={() => {
                                    session
                                      .delete()
                                      .then(() => this.loadActiveSessions())
                                      .catch(() => RuntimeConfig.logOut());
                                  }}
                                >
                                  {this.props.t("logout")}
                                </button>
                              </span>
                            </div>
                          </React.Fragment>
                        ))}
                      </div>
                      <p className="preferences-footnote">
                        * {this.props.t("thisSession")}
                      </p>
                      <div>
                        <Button
                          hidden={this.state.activeSessions?.length <= 1}
                          type="button"
                          className="preferences-btn-neutral"
                          variant="dark"
                          onClick={() => {
                            const others = this.state.activeSessions.filter(
                              (s) => s.id !== this.state.currentSessionId,
                            );
                            Promise.all(others.map((s) => s.delete()))
                              .then(() => this.loadActiveSessions())
                              .catch(() => RuntimeConfig.logOut());
                          }}
                        >
                          {this.props.t("logoutOthers")}
                        </Button>
                      </div>
                    </>
                  )}
                </div>
              </div>
            </div>

            {/* --- */}
            {/* IDP */}
            {/* --- */}

            <div hidden={this.state.activeTab !== "tab-idp"}>
              <div className="preferences-panel">
                <div className="text-end">
                  <a
                    href={profilePageUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="btn preferences-btn-neutral btn-sm mb-2"
                  >
                    <IoLinkOutline className="feather me-1" />
                    {this.props.t("manageProfile")}
                  </a>
                </div>
              </div>
              <iframe
                src={profilePageUrl}
                style={{ width: "100%", height: "100vh", borderWidth: 0 }}
                id="idp-profilepage-iframe"
              ></iframe>
            </div>

            {/* ------------ */}
            {/* INTEGRATIONS */}
            {/* ------------ */}

            <Form
              onSubmit={this.saveCaldavSettings}
              hidden={this.state.activeTab !== "tab-integrations"}
            >
              <div className="preferences-panel">
                <h3 className="preferences-section-title">
                  {this.props.t("caldavCalendar")}
                </h3>
                {this.state.caldavOAuthError ? (
                  <Alert
                    variant="danger"
                    className="preferences-alert"
                    dismissible
                    onClose={() =>
                      this.setState({ caldavOAuthError: null })
                    }
                  >
                    {this.getCaldavOAuthErrorMessage(
                      this.state.caldavOAuthError,
                    )}
                  </Alert>
                ) : null}
                {this.state.caldavLegacyGoogleReconnect ? (
                  <Alert variant="warning" className="preferences-alert">
                    {this.props.t("caldavReconnectGoogle")}
                  </Alert>
                ) : null}
                <Form.Group className="preferences-field">
                  <Form.Label>{this.props.t("caldavCalendar")}</Form.Label>
                  <div>
                    <Form.Check
                      inline
                      type="radio"
                      id="caldavProviderGoogle"
                      label={this.props.t("caldavProviderGoogle")}
                      name="caldavProvider"
                      checked={
                        this.state.caldavProvider ===
                        UserPreference.CALDAV_PROVIDER_GOOGLE
                      }
                      onChange={() =>
                        this.setState({
                          caldavProvider: UserPreference.CALDAV_PROVIDER_GOOGLE,
                          caldavCalendarsLoaded: false,
                          caldavCalendars: [],
                        })
                      }
                    />
                    <Form.Check
                      inline
                      type="radio"
                      id="caldavProviderGeneric"
                      label={this.props.t("caldavProviderGeneric")}
                      name="caldavProvider"
                      checked={
                        this.state.caldavProvider ===
                        UserPreference.CALDAV_PROVIDER_GENERIC
                      }
                      onChange={() =>
                        this.setState({
                          caldavProvider:
                            UserPreference.CALDAV_PROVIDER_GENERIC,
                          caldavCalendarsLoaded: false,
                          caldavCalendars: [],
                        })
                      }
                    />
                  </div>
                </Form.Group>
                {this.state.caldavProvider ===
                UserPreference.CALDAV_PROVIDER_GOOGLE ? (
                  <>
                    <p className="preferences-footnote">
                      {this.state.caldavGoogleEmail
                        ? this.props.t("caldavConnectedAs", {
                            email: this.state.caldavGoogleEmail,
                          })
                        : this.props.t("caldavNotConnected")}
                    </p>
                    <div className="preferences-btn-row mb-3">
                      <Button
                        type="button"
                        className="preferences-btn-neutral"
                        variant="dark"
                        disabled={this.state.submitting}
                        onClick={() => this.connectGoogleCalDav()}
                      >
                        {this.props.t("caldavConnectGoogle")}
                      </Button>
                      {this.state.caldavGoogleEmail ? (
                        <Button
                          type="button"
                          className="preferences-btn-neutral"
                          variant="dark"
                          disabled={this.state.submitting}
                          onClick={() => this.listGoogleCalDavCalendars()}
                        >
                          {this.props.t("connect")}
                        </Button>
                      ) : null}
                    </div>
                  </>
                ) : (
                  <>
                    <Form.Group className="preferences-field">
                      <Form.Label htmlFor="caldavUrl">
                        {this.props.t("caldavUrl")}
                      </Form.Label>
                      <Form.Control
                        id="caldavUrl"
                        type="url"
                        value={this.state.caldavUrl}
                        onChange={(e: any) =>
                          this.setState({
                            caldavUrl: e.target.value,
                            caldavCalendarsLoaded: false,
                          })
                        }
                      />
                    </Form.Group>
                    <Form.Group className="preferences-field">
                      <Form.Label htmlFor="caldavUser">
                        {this.props.t("username")}
                      </Form.Label>
                      <Form.Control
                        id="caldavUser"
                        type="text"
                        value={this.state.caldavUser}
                        onChange={(e: any) =>
                          this.setState({
                            caldavUser: e.target.value,
                            caldavCalendarsLoaded: false,
                          })
                        }
                      />
                    </Form.Group>
                    <Form.Group className="preferences-field">
                      <Form.Label htmlFor="caldavPass">
                        {this.props.t("password")}
                      </Form.Label>
                      <Form.Control
                        id="caldavPass"
                        type="password"
                        value={this.state.caldavPass}
                        onChange={(e: any) =>
                          this.setState({
                            caldavPass: e.target.value,
                            caldavCalendarsLoaded: false,
                          })
                        }
                      />
                    </Form.Group>
                    <div className="preferences-btn-row mb-3">
                      <Button
                        type="button"
                        className="preferences-btn-neutral"
                        variant="dark"
                        disabled={
                          this.state.submitting ||
                          this.state.caldavUrl === "" ||
                          this.state.caldavUser === "" ||
                          this.state.caldavPass === ""
                        }
                        onClick={() => this.connectCalDav()}
                      >
                        {this.props.t("connect")}
                      </Button>
                    </div>
                  </>
                )}
                <Form.Group className="preferences-field">
                  <Form.Label htmlFor="caldavCalendar">
                    {this.props.t("calendar")}
                  </Form.Label>
                  <Form.Select
                    id="caldavCalendar"
                    className="preferences-calendar-select"
                    value={this.state.caldavCalendar}
                    onChange={(e: any) =>
                      this.setState({ caldavCalendar: e.target.value })
                    }
                    disabled={!this.state.caldavCalendarsLoaded}
                  >
                    {this.state.caldavCalendars.map((cal) => (
                      <option key={cal.path} value={cal.path}>
                        {cal.name}
                      </option>
                    ))}
                  </Form.Select>
                </Form.Group>
                <div className="preferences-btn-row">
                  <Button
                    type="button"
                    className="preferences-btn-neutral"
                    variant="dark"
                    disabled={
                      this.state.submitting ||
                      (this.state.caldavProvider ===
                        UserPreference.CALDAV_PROVIDER_GENERIC &&
                        (this.state.caldavUrl === "" ||
                          this.state.caldavUser === "" ||
                          this.state.caldavPass === "")) ||
                      (this.state.caldavProvider ===
                        UserPreference.CALDAV_PROVIDER_GOOGLE &&
                        !this.state.caldavGoogleEmail) ||
                      this.state.caldavCalendar === ""
                    }
                    onClick={() => this.disconnectCalDav()}
                  >
                    {this.props.t("disconnect")}
                  </Button>
                  <SaveButton
                    submitting={this.state.submitting}
                    className="preferences-btn-save"
                    disabled={
                      !(
                        this.state.caldavCalendarsLoaded &&
                        this.state.caldavCalendar != ""
                      ) || this.state.submitting
                    }
                  />
                </div>
              </div>
            </Form>
          </div>
        </div>
        <Modal
          show={this.state.showPasswordChangedModal}
          onHide={() => { }}
          backdrop="static"
          keyboard={false}
        >
          <Modal.Body>
            <p>{this.props.t("passwordChangedLoginAgain")}</p>
          </Modal.Body>
          <Modal.Footer>
            <Button variant="primary" onClick={() => window.location.reload()}>
              {this.props.t("ok")}
            </Button>
          </Modal.Footer>
        </Modal>
      </>
    );
  }
}

export default withTranslation(withReadyRouter(Preferences as any));
