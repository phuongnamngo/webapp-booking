import React from "react";
import { beforeEach, describe, expect, it } from "vitest";
import RuntimeConfig from "@/components/RuntimeConfig";
import type { NextRouter } from "next/router";
import Space, { type SpaceDayStatus } from "@/types/Space";
import { Search } from "@/pages/search";

describe("Search page", () => {
  beforeEach(() => {
    RuntimeConfig.resetInfos();
  });

  it("uses the 30-minute fallback in modal validation for spaces without a seat type when org minimum is disabled", () => {
    const space = createSpace();
    const search = createSearch();
    const enter = new Date("2030-09-02T09:00:00");
    const leave = new Date("2030-09-02T09:20:00");

    search.state = {
      ...search.state,
      enter,
      leave,
      modalEnter: enter,
      modalLeave: leave,
      selectedSpace: space,
      modalSpaceAvailable: true,
      dayStatusBySpaceId: {
        [space.id]: createDayStatus(space.id, "available", []),
      },
    };

    expect(search.getModalBookingError()).toBe(
      "errorMinBookingDurationMinutes:30",
    );
  });

  it("renders a zero badge count when day-status is full but has no actual bookings", () => {
    const space = createSpace();
    const search = createSearch();

    search.state = {
      ...search.state,
      dayStatusBySpaceId: {
        [space.id]: createDayStatus(space.id, "full", []),
      },
    };

    const element = search.renderListItem(space);
    const [, badge] = React.Children.toArray(
      (element as React.ReactElement).props.children,
    );

    expect(
      React.isValidElement<{ children: React.ReactNode }>(badge),
    ).toBe(true);
    if (!React.isValidElement<{ children: React.ReactNode }>(badge)) {
      throw new Error("Expected badge element");
    }

    expect(badge.props.children).toBe(0);
  });
});

function createSearch(): Search {
  return new Search({
    router: {} as NextRouter,
    t: (key: string, view?: object) =>
      key === "errorMinBookingDurationMinutes"
        ? `${key}:${(view as { num: number }).num}`
        : key,
  });
}

function createSpace(): Space {
  const space = new Space();
  space.id = "space-1";
  space.name = "Desk 1";
  space.allowed = true;
  space.enabled = true;
  space.available = true;
  space.locationId = "location-1";
  space.rawBookings = [];
  space.spaceType = null;
  return space;
}

function createDayStatus(
  spaceId: string,
  status: SpaceDayStatus["status"],
  bookings: { enter: string; leave: string }[],
): SpaceDayStatus {
  return {
    spaceId,
    status,
    officeStart: "2030-09-02T08:00:00+07:00",
    officeEnd: "2030-09-02T17:00:00+07:00",
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
