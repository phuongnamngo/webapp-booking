import { afterEach, describe, expect, it, vi } from "vitest";
import SpaceType, { SpaceTypeSlot } from "@/types/SpaceType";
import {
  doesBookingRangeOverlapDayStatus,
  getEffectiveDayStatusStatus,
  getAvailableFixedSlots,
  hasBookableTimeRange,
  getEnabledSlots,
  isBookingRangeInBookableFuture,
  isFixedSlotSpace,
  isBookingRangeWithinOfficeHours,
  slotToDateRange,
  usesLegacyTimeRangeFallback,
} from "./SpaceTypeBooking";
import { SpaceDayStatus } from "@/types/Space";

describe("SpaceTypeBooking", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("uses legacy time-range fallback when space type is missing", () => {
    expect(usesLegacyTimeRangeFallback(undefined)).toBe(true);
  });

  it("does not treat a fixed-slot space with zero remaining slots as legacy fallback", () => {
    const type = new SpaceType();
    type.bookingMode = "fixed_slots";
    type.slots = [];

    expect(usesLegacyTimeRangeFallback(type)).toBe(false);
    expect(isFixedSlotSpace(type)).toBe(true);
    expect(
      getAvailableFixedSlots(type, new Date("2030-09-01T00:00:00"), createDayStatus([
        {
          enter: "2030-09-01T08:00:00",
          leave: "2030-09-01T17:00:00",
        },
      ])),
    ).toEqual([]);
  });

  it("converts slot to dates on selected day", () => {
    const slot = new SpaceTypeSlot();
    slot.startTime = "08:00";
    slot.endTime = "12:00";

    const [enter, leave] = slotToDateRange(
      slot,
      new Date("2030-09-01T10:30:00"),
    );

    expect(enter.getHours()).toBe(8);
    expect(enter.getMinutes()).toBe(0);
    expect(leave.getHours()).toBe(12);
    expect(leave.getMinutes()).toBe(0);
  });

  it("filters and sorts enabled slots", () => {
    const type = new SpaceType();
    const later = new SpaceTypeSlot();
    later.enabled = true;
    later.sortOrder = 2;
    const disabled = new SpaceTypeSlot();
    disabled.enabled = false;
    disabled.sortOrder = 1;
    const earlier = new SpaceTypeSlot();
    earlier.enabled = true;
    earlier.sortOrder = 1;
    type.slots = [later, disabled, earlier];

    expect(getEnabledSlots(type)).toEqual([earlier, later]);
  });

  it("filters fixed slots by enabled state, office hours, and existing bookings", () => {
    const type = new SpaceType();
    type.bookingMode = "fixed_slots";
    const disabled = createSlot("disabled", "Disabled", "08:00", "09:00", 1);
    disabled.enabled = false;
    const outsideOfficeHours = createSlot(
      "outside",
      "Outside",
      "07:30",
      "08:30",
      2,
    );
    const overlapping = createSlot("overlap", "Overlap", "09:00", "10:00", 3);
    const adjacent = createSlot("adjacent", "Adjacent", "10:30", "11:00", 4);
    const available = createSlot("available", "Available", "11:00", "12:00", 5);
    type.slots = [
      available,
      disabled,
      outsideOfficeHours,
      overlapping,
      adjacent,
    ];

    const dayStatus = createDayStatus([
      {
        enter: "2030-09-01T09:30:00",
        leave: "2030-09-01T10:30:00",
      },
    ]);

    expect(
      getAvailableFixedSlots(type, new Date("2030-09-01T00:00:00"), dayStatus),
    ).toEqual([adjacent, available]);
  });

  it("filters out fixed slots that already started earlier today", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2030-09-01T14:00:00"));

    const type = new SpaceType();
    type.bookingMode = "fixed_slots";
    const morning = createSlot("morning", "Morning", "08:00", "11:00", 1);
    const afternoon = createSlot("afternoon", "Afternoon", "15:00", "17:00", 2);
    type.slots = [morning, afternoon];

    expect(
      getAvailableFixedSlots(type, new Date("2030-09-01T00:00:00"), createDayStatus([])),
    ).toEqual([afternoon]);
  });

  it("rejects same-day booking ranges that start before the current time", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2030-09-01T14:00:00"));

    expect(
      isBookingRangeInBookableFuture(
        new Date("2030-09-01T13:00:00"),
        createDayStatus([]),
      ),
    ).toBe(false);
    expect(
      isBookingRangeInBookableFuture(
        new Date("2030-09-01T15:00:00"),
        createDayStatus([]),
      ),
    ).toBe(true);
  });

  it("marks flexible-time day status as full when remaining time is shorter than minimum duration", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2030-09-01T14:50:00"));

    const type = new SpaceType();
    type.bookingMode = "flexible_time";
    type.minDurationMinutes = 30;
    const dayStatus = createDayStatus([]);
    dayStatus.officeEnd = "15:00";

    expect(hasBookableTimeRange(dayStatus, 30)).toBe(false);
    expect(
      getEffectiveDayStatusStatus(
        type,
        new Date("2030-09-01T00:00:00"),
        dayStatus,
        0,
      ),
    ).toBe("full");
  });

  it("marks same-day status as full when the remaining gap before a booking is shorter than minimum duration", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2030-09-01T15:20:00"));

    const type = new SpaceType();
    type.bookingMode = "flexible_time";
    type.minDurationMinutes = 30;
    const dayStatus = createDayStatus([
      {
        enter: "2030-09-01T15:45:00",
        leave: "2030-09-01T16:59:00",
      },
    ]);

    expect(hasBookableTimeRange(dayStatus, 30)).toBe(false);
    expect(
      getEffectiveDayStatusStatus(
        type,
        new Date("2030-09-01T00:00:00"),
        dayStatus,
        0,
      ),
    ).toBe("full");
  });

  it("uses 30-minute fallback for no-seat-type same-day bookability when org min is disabled", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2030-09-01T15:40:00"));

    const dayStatus = createDayStatus([
      {
        enter: "2030-09-01T16:00:00",
        leave: "2030-09-01T17:00:00",
      },
    ]);
    dayStatus.officeStart = "2030-09-01T08:00:00+07:00";
    dayStatus.officeEnd = "2030-09-01T17:00:00+07:00";

    expect(
      getEffectiveDayStatusStatus(
        undefined,
        new Date("2030-09-01T00:00:00"),
        dayStatus,
        0,
      ),
    ).toBe("full");
  });

  it("marks no-seat-type same-day status as full when remaining gap is shorter than fallback minimum", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2030-09-01T16:45:00"));

    const dayStatus = createDayStatus([]);
    dayStatus.officeStart = "2030-09-01T08:00:00+07:00";
    dayStatus.officeEnd = "2030-09-01T17:00:00+07:00";

    expect(
      getEffectiveDayStatusStatus(
        undefined,
        new Date("2030-09-01T00:00:00"),
        dayStatus,
        0,
      ),
    ).toBe("full");
  });

  it("keeps enabled slots when day status is missing", () => {
    const type = new SpaceType();
    type.bookingMode = "fixed_slots";
    const slot = createSlot("available", "Available", "07:00", "08:00", 1);
    type.slots = [slot];

    expect(
      getAvailableFixedSlots(type, new Date("2030-09-01T00:00:00"), undefined),
    ).toEqual([slot]);
  });

  it("validates booking ranges against office hours", () => {
    const dayStatus = createDayStatus([]);

    expect(
      isBookingRangeWithinOfficeHours(
        new Date("2030-09-01T08:00:00"),
        new Date("2030-09-01T17:00:00"),
        dayStatus,
      ),
    ).toBe(true);
    expect(
      isBookingRangeWithinOfficeHours(
        new Date("2030-09-01T07:59:00"),
        new Date("2030-09-01T09:00:00"),
        dayStatus,
      ),
    ).toBe(false);
    expect(
      isBookingRangeWithinOfficeHours(
        new Date("2030-09-01T16:00:00"),
        new Date("2030-09-01T17:01:00"),
        dayStatus,
      ),
    ).toBe(false);
  });

  it("validates office hours returned as backend timestamps", () => {
    const dayStatus = createDayStatus([]);
    dayStatus.officeStart = "2030-09-01T08:00:00+02:00";
    dayStatus.officeEnd = "2030-09-01T17:00:00+02:00";

    expect(
      isBookingRangeWithinOfficeHours(
        new Date("2030-09-01T08:00:00"),
        new Date("2030-09-01T17:00:00"),
        dayStatus,
      ),
    ).toBe(true);
    expect(
      isBookingRangeWithinOfficeHours(
        new Date("2030-09-01T07:59:00"),
        new Date("2030-09-01T09:00:00"),
        dayStatus,
      ),
    ).toBe(false);
  });

  it("detects overlapping booking ranges while allowing adjacent ranges", () => {
    const dayStatus = createDayStatus([
      {
        enter: "2030-09-01T09:00:00",
        leave: "2030-09-01T10:00:00",
      },
    ]);

    expect(
      doesBookingRangeOverlapDayStatus(
        new Date("2030-09-01T08:00:00"),
        new Date("2030-09-01T09:00:00"),
        dayStatus,
      ),
    ).toBe(false);
    expect(
      doesBookingRangeOverlapDayStatus(
        new Date("2030-09-01T09:30:00"),
        new Date("2030-09-01T10:30:00"),
        dayStatus,
      ),
    ).toBe(true);
  });
});

function createSlot(
  id: string,
  label: string,
  startTime: string,
  endTime: string,
  sortOrder: number,
): SpaceTypeSlot {
  const slot = new SpaceTypeSlot();
  slot.id = id;
  slot.label = label;
  slot.startTime = startTime;
  slot.endTime = endTime;
  slot.sortOrder = sortOrder;
  return slot;
}

function createDayStatus(
  bookings: { enter: string; leave: string }[],
): SpaceDayStatus {
  return {
    spaceId: "space-1",
    status: bookings.length ? "partially_booked" : "available",
    officeStart: "08:00",
    officeEnd: "17:00",
    bookedMinutes: 0,
    officeMinutes: 540,
    bookings: bookings.map((booking, index) => ({
      id: `booking-${index}`,
      recurringId: "",
      userId: "",
      userEmail: "",
      subject: "",
      enter: booking.enter,
      leave: booking.leave,
    })),
  };
}
