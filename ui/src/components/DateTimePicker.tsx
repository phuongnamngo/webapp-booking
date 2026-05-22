import React from "react";
import "flatpickr/dist/themes/airbnb.css";
import flatpickr from "flatpickr";
import Flatpickr from "react-flatpickr";
import { Instance } from "flatpickr/dist/types/instance";
import { Options } from "flatpickr/dist/types/options";
import { TranslationFunc, withTranslation } from "./withTranslation";
import RuntimeConfig from "./RuntimeConfig";
import { CustomLocale } from "flatpickr/dist/types/locale";
import { english as DefaultLocale } from "flatpickr/dist/l10n/default.js";
import DateUtil from "@/util/DateUtil";

interface State {
  locale: CustomLocale | undefined;
  isOpen: boolean;
  pickerSessionKey: string;
}

interface Props {
  t: TranslationFunc;
  id?: string;
  enableTime?: boolean | undefined;
  noCalendar?: boolean | undefined;
  required?: boolean | undefined;
  disabled?: boolean | undefined;
  /** Mount calendar inside nearest modal (avoids focus-trap / z-index issues). */
  contained?: boolean | undefined;
  value: Date;
  minDate?: Date | undefined;
  maxDate?: Date | undefined;
  onChange: (value: Date) => void;
}

class DateTimePicker extends React.Component<Props, State> {
  private flatpickrOptions: Options | null = null;
  private flatpickrOptionsKey = "";

  constructor(props: Props) {
    super(props);
    this.state = {
      locale: undefined,
      isOpen: false,
      pickerSessionKey: `closed-${props.value.getTime()}`,
    };
  }

  componentDidMount() {
    this.loadLocale();
  }

  componentDidUpdate(prevProps: Props) {
    if (
      !this.state.isOpen &&
      prevProps.value.getTime() !== this.props.value.getTime()
    ) {
      this.setState({
        pickerSessionKey: `closed-${this.props.value.getTime()}`,
      });
    }
    if (prevProps.disabled !== this.props.disabled && this.state.isOpen) {
      this.syncDisabledOnOpenInstance();
    }
  }

  shouldComponentUpdate(nextProps: Props, nextState: State): boolean {
    if (this.state.isOpen && nextState.isOpen) {
      return (
        nextProps.disabled !== this.props.disabled ||
        nextState.locale !== this.state.locale ||
        nextState.pickerSessionKey !== this.state.pickerSessionKey
      );
    }
    if (nextState.locale !== this.state.locale) {
      return true;
    }
    if (nextProps.disabled !== this.props.disabled) {
      return true;
    }
    if (nextProps.value.getTime() !== this.props.value.getTime()) {
      return true;
    }
    if (nextState.pickerSessionKey !== this.state.pickerSessionKey) {
      return true;
    }
    return (
      nextProps.id !== this.props.id ||
      nextProps.enableTime !== this.props.enableTime ||
      nextProps.noCalendar !== this.props.noCalendar ||
      nextProps.required !== this.props.required ||
      nextProps.contained !== this.props.contained ||
      nextProps.minDate?.getTime() !== this.props.minDate?.getTime() ||
      nextProps.maxDate?.getTime() !== this.props.maxDate?.getTime()
    );
  }

  syncDisabledOnOpenInstance = (): void => {
    const input = document.getElementById(
      this.props.id || "",
    ) as HTMLInputElement | null;
    if (input) {
      input.disabled = !!this.props.disabled;
    }
  };

  handlePickerOpen = (): void => {
    this.setState({
      isOpen: true,
      pickerSessionKey: `open-${this.props.value.getTime()}`,
    });
  };

  handlePickerClose = ([value]: Date[]): void => {
    const nextValue =
      value != null && value instanceof Date
        ? DateUtil.setSecondsToMin(value)
        : this.props.value;
    this.setState({
      isOpen: false,
      pickerSessionKey: `closed-${nextValue.getTime()}`,
    });
    if (value != null && value instanceof Date) {
      this.props.onChange(nextValue);
    }
  };

  getFlatpickrOptions(): Options {
    let formatting = this.props.noCalendar
      ? ""
      : RuntimeConfig.INFOS.dateFormat + " ";
    if (this.props.enableTime) {
      if (RuntimeConfig.INFOS.use24HourTime) {
        formatting += "H:i";
      } else {
        formatting += "h:i K";
      }
    }
    const minTime = this.props.minDate?.getTime() ?? "";
    const maxTime = this.props.maxDate?.getTime() ?? "";
    const optionsKey = [
      formatting,
      String(RuntimeConfig.INFOS.use24HourTime),
      minTime,
      maxTime,
      String(this.props.noCalendar),
      String(this.props.contained),
      this.state.locale ? "locale" : "no-locale",
    ].join("|");
    if (this.flatpickrOptions && this.flatpickrOptionsKey === optionsKey) {
      return this.flatpickrOptions;
    }
    this.flatpickrOptionsKey = optionsKey;
    const contained = this.props.contained;
    this.flatpickrOptions = {
      allowInput: false,
      dateFormat: formatting,
      time_24hr: RuntimeConfig.INFOS.use24HourTime,
      minDate: this.props.minDate,
      maxDate: this.props.maxDate,
      locale: this.state.locale,
      noCalendar: this.props.noCalendar,
      onReady: (_selectedDates, _dateStr, instance: Instance) => {
        if (!contained) {
          return;
        }
        const modalBody = instance.input
          .closest(".modal")
          ?.querySelector(".modal-body");
        if (modalBody instanceof HTMLElement) {
          instance.set("appendTo", modalBody);
        }
      },
    };
    return this.flatpickrOptions;
  }

  async loadLocale() {
    let lang = RuntimeConfig.getLanguage();
    if (lang.indexOf("-") !== -1) {
      lang = lang.split("-")[0];
    }
    if (lang === "en") {
      this.setState({ locale: DefaultLocale });
      return;
    }
    try {
      const localeModule = await import(`flatpickr/dist/l10n/${lang}.js`);
      const name = Object.keys(localeModule)[0];
      this.setState({ locale: localeModule[name] });
    } catch (error) {
      console.error(`Failed to load locale ${lang}:`, error);
      this.setState({ locale: DefaultLocale });
    }
  }

  render() {
    if (!this.state.locale) {
      return <></>;
    }
    const options = this.getFlatpickrOptions();
    const dateFormat =
      typeof options.dateFormat === "string" ? options.dateFormat : "";
    const defaultValue = flatpickr.formatDate(this.props.value, dateFormat);
    return (
      <Flatpickr
        key={this.state.pickerSessionKey}
        className="form-control"
        id={this.props.id}
        data-enable-time={this.props.enableTime}
        disabled={this.props.disabled}
        defaultValue={defaultValue}
        required={this.props.required}
        onOpen={this.handlePickerOpen}
        onClose={this.handlePickerClose}
        options={options}
      />
    );
  }
}

export default withTranslation(DateTimePicker as any);
