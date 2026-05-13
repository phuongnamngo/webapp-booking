import { afterEach, describe, expect, it, vi } from "vitest";
import { OfficeSettings, isValidOfficeHoursRange } from "./Settings";

describe("OfficeSettings", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("loads office hours from settings API", async () => {
    const response = {
      workStartTime: "08:00",
      workEndTime: "17:00",
    };
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(new Response(JSON.stringify(response)));

    const settings = await OfficeSettings.get();

    expect(fetchMock).toHaveBeenCalledWith(
      "/setting/office-hours",
      expect.objectContaining({ method: "GET" }),
    );
    expect(settings.workStartTime).toBe("08:00");
    expect(settings.workEndTime).toBe("17:00");
  });

  it("saves office hours to settings API", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(null, {
        status: 204,
      }),
    );

    await new OfficeSettings("08:30", "18:00").save();

    expect(fetchMock).toHaveBeenCalledWith(
      "/setting/office-hours",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({
          workStartTime: "08:30",
          workEndTime: "18:00",
        }),
      }),
    );
  });

  it("validates office hours before saving settings forms", () => {
    expect(isValidOfficeHoursRange("08:00", "17:00")).toBe(true);
    expect(isValidOfficeHoursRange("17:00", "08:00")).toBe(false);
    expect(isValidOfficeHoursRange("08:60", "17:00")).toBe(false);
    expect(isValidOfficeHoursRange("", "17:00")).toBe(false);
  });
});
