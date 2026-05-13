import SpaceType, { type SpaceTypeSlot } from "@/types/SpaceType";
import type { SpaceDayStatus } from "@/types/Space";
import DateUtil from "@/util/DateUtil";

export function getEnabledSlots(spaceType?: SpaceType | null): SpaceTypeSlot[] {
  return (spaceType?.slots || [])
    .filter((slot) => slot.enabled)
    .sort((a, b) => a.sortOrder - b.sortOrder);
}

export function usesLegacyTimeRangeFallback(
  spaceType?: SpaceType | null,
): boolean {
  return !spaceType || !spaceType.enabled;
}

export function isFixedSlotSpace(spaceType?: SpaceType | null): boolean {
  return !!spaceType && spaceType.enabled && spaceType.bookingMode === "fixed_slots";
}

export function getEffectiveMinimumBookingDurationMinutes(
  spaceType: SpaceType | null | undefined,
  minBookingDurationMinutes: number,
): number {
  if (isFixedSlotSpace(spaceType)) {
    return 0;
  }
  if (usesLegacyTimeRangeFallback(spaceType)) {
    return minBookingDurationMinutes > 0 ? minBookingDurationMinutes : 30;
  }
  return Math.max(
    minBookingDurationMinutes,
    spaceType?.enabled && spaceType.bookingMode === "flexible_time"
      ? spaceType.minDurationMinutes
      : 0,
  );
}

export function slotToDateRange(
  slot: SpaceTypeSlot,
  selectedDate: Date,
): [Date, Date] {
  const enter = new Date(selectedDate);
  const leave = new Date(selectedDate);
  const [startHour, startMinute] = slot.startTime
    .split(":")
    .map((value) => Number.parseInt(value, 10));
  const [endHour, endMinute] = slot.endTime
    .split(":")
    .map((value) => Number.parseInt(value, 10));
  enter.setHours(startHour, startMinute, 0, 0);
  leave.setHours(endHour, endMinute, 0, 0);
  return [enter, leave];
}

export function getDurationMinutes(enter: Date, leave: Date): number {
  return Math.floor((leave.getTime() - enter.getTime()) / 60000);
}

export function getAvailableFixedSlots(
  spaceType: SpaceType | null | undefined,
  selectedDate: Date,
  dayStatus?: SpaceDayStatus,
): SpaceTypeSlot[] {
  if (!isFixedSlotSpace(spaceType)) {
    return [];
  }
  const enabledSlots = getEnabledSlots(spaceType);
  if (!dayStatus) {
    return enabledSlots;
  }
  return enabledSlots.filter((slot) => {
    const [enter, leave] = slotToDateRange(slot, selectedDate);
    return (
      isBookingRangeInBookableFuture(enter, dayStatus) &&
      isBookingRangeWithinOfficeHours(enter, leave, dayStatus) &&
      !doesBookingRangeOverlapDayStatus(enter, leave, dayStatus)
    );
  });
}

export function getEffectiveDayStatusStatus(
  spaceType: SpaceType | null | undefined,
  selectedDate: Date,
  dayStatus: SpaceDayStatus | undefined,
  minBookingDurationMinutes: number,
  now: Date = new Date(),
): SpaceDayStatus["status"] | undefined {
  if (!dayStatus) {
    return undefined;
  }
  if (dayStatus.status === "full") {
    return dayStatus.status;
  }
  if (isFixedSlotSpace(spaceType)) {
    return getAvailableFixedSlots(spaceType, selectedDate, dayStatus).length > 0
      ? dayStatus.status
      : "full";
  }
  const requiredMinutes = getEffectiveMinimumBookingDurationMinutes(
    spaceType,
    minBookingDurationMinutes,
  );
  return hasBookableTimeRange(dayStatus, requiredMinutes, now)
    ? dayStatus.status
    : "full";
}

export function hasBookableTimeRange(
  dayStatus: SpaceDayStatus | undefined,
  minBookingDurationMinutes: number,
  now: Date = new Date(),
): boolean {
  const bookableWindow = getBookableWindow(dayStatus, now);
  if (!bookableWindow) {
    return false;
  }
  const [bookableStart, bookableEnd] = bookableWindow;
  const minimumDurationMs = Math.max(minBookingDurationMinutes, 0) * 60000;
  let cursor = bookableStart;
  const bookings = [...(dayStatus?.bookings || [])].sort((left, right) =>
    left.enter.localeCompare(right.enter),
  );
  for (const booking of bookings) {
    const bookingEnter = parseDayStatusDate(booking.enter);
    const bookingLeave = parseDayStatusDate(booking.leave);
    if (bookingLeave.getTime() <= cursor.getTime()) {
      continue;
    }
    if (bookingEnter.getTime() >= bookableEnd.getTime()) {
      break;
    }
    if (bookingEnter.getTime() > cursor.getTime()) {
      const gapMs = bookingEnter.getTime() - cursor.getTime();
      if (gapMs >= minimumDurationMs) {
        return true;
      }
    }
    if (bookingLeave.getTime() > cursor.getTime()) {
      cursor = bookingLeave;
    }
  }
  return bookableEnd.getTime() - cursor.getTime() >= minimumDurationMs;
}

export function isBookingRangeInBookableFuture(
  enter: Date,
  dayStatus?: SpaceDayStatus,
  now: Date = new Date(),
): boolean {
  if (!DateUtil.isSameDay(enter, now)) {
    return true;
  }
  const officeHours = getOfficeHoursRange(enter, dayStatus);
  if (officeHours && now.getTime() >= officeHours[1].getTime()) {
    return false;
  }
  return enter.getTime() >= now.getTime();
}

export function getBookableWindow(
  dayStatus?: SpaceDayStatus,
  now: Date = new Date(),
): [Date, Date] | null {
  if (!dayStatus) {
    return null;
  }
  const officeDate = parseDayStatusDate(dayStatus.officeStart);
  const officeHours = getOfficeHoursRange(officeDate, dayStatus);
  if (!officeHours) {
    return null;
  }
  const [officeStart, officeEnd] = officeHours;
  let bookableStart = officeStart;
  if (DateUtil.isSameDay(now, officeStart)) {
    if (now.getTime() >= officeEnd.getTime()) {
      return null;
    }
    if (now.getTime() > officeStart.getTime()) {
      bookableStart = now;
    }
  }
  if (bookableStart.getTime() >= officeEnd.getTime()) {
    return null;
  }
  return [bookableStart, officeEnd];
}

export function isBookingRangeWithinOfficeHours(
  enter: Date,
  leave: Date,
  dayStatus?: SpaceDayStatus,
): boolean {
  const officeHours = getOfficeHoursRange(enter, dayStatus);
  if (!officeHours) {
    return true;
  }
  const [officeStart, officeEnd] = officeHours;
  return (
    enter.getTime() >= officeStart.getTime() &&
    leave.getTime() <= officeEnd.getTime()
  );
}

export function doesBookingRangeOverlapDayStatus(
  enter: Date,
  leave: Date,
  dayStatus?: SpaceDayStatus,
): boolean {
  if (!dayStatus) {
    return false;
  }
  return dayStatus.bookings.some((booking) => {
    const bookingEnter = parseDayStatusDate(booking.enter);
    const bookingLeave = parseDayStatusDate(booking.leave);
    return (
      enter.getTime() < bookingLeave.getTime() &&
      leave.getTime() > bookingEnter.getTime()
    );
  });
}

export function getOfficeHoursRange(
  selectedDate: Date,
  dayStatus?: SpaceDayStatus,
): [Date, Date] | null {
  if (!dayStatus) {
    return null;
  }
  const officeStart = timeOnDate(selectedDate, dayStatus.officeStart);
  const officeEnd = timeOnDate(selectedDate, dayStatus.officeEnd);
  if (!officeStart || !officeEnd || officeEnd <= officeStart) {
    return null;
  }
  return [officeStart, officeEnd];
}

function timeOnDate(selectedDate: Date, time: string): Date | null {
  const match = /^(\d{2}):(\d{2})$/.exec(time);
  const parsed = match ? null : parseDayStatusDate(time);
  if (!match && Number.isNaN(parsed?.getTime())) {
    return null;
  }
  const hours = match
    ? Number.parseInt(match[1], 10)
    : (parsed as Date).getHours();
  const minutes = match
    ? Number.parseInt(match[2], 10)
    : (parsed as Date).getMinutes();
  if (hours > 23 || minutes > 59) {
    return null;
  }
  const result = new Date(selectedDate);
  result.setHours(hours, minutes, 0, 0);
  return result;
}

export function parseDayStatusDate(value: string): Date {
  return new Date(value.replace(/(?:Z|[+-]\d{2}:\d{2})$/, ""));
}
