import { afterEach, describe, expect, it, vi } from "vitest";
import Space from "./Space";

describe("Space", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("loads day status for a selected local date", async () => {
    const response = [
      {
        spaceId: "space-1",
        status: "partially_booked",
        officeStart: "08:00",
        officeEnd: "17:00",
        bookedMinutes: 120,
        officeMinutes: 540,
        bookings: [],
      },
    ];
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(new Response(JSON.stringify(response)));

    const statuses = await Space.listDayStatus(
      "location-1",
      new Date(2030, 8, 2, 15, 30),
    );

    expect(fetchMock).toHaveBeenCalledWith(
      "/location/location-1/space/day-status?date=2030-09-02",
      expect.objectContaining({ method: "GET" }),
    );
    expect(statuses).toEqual(response);
  });
});
