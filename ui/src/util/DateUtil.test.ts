import { describe, it, expect } from "vitest";
import DateUtil from "./DateUtil";

describe("DateUtil", () => {
  describe("setHoursToMin", () => {
    it("should return new Date", () => {
      const date = new Date();
      const dateTimestamp = date.getTime();
      const dateHoursToMin = DateUtil.setHoursToMin(date);

      expect(dateTimestamp).toBe(date.getTime());
      expect(dateHoursToMin.getHours()).toBe(0);
    });
  });

  describe("getTodayStart", () => {
    it("should return date with time 00:00:00.000", () => {
      const todayStart = DateUtil.getTodayStart();

      expect(todayStart.getHours()).toBe(0);
      expect(todayStart.getMinutes()).toBe(0);
      expect(todayStart.getSeconds()).toBe(0);
      expect(todayStart.getMilliseconds()).toBe(0);
    });
  });

  describe("setSecondsToMin", () => {
    it("should zero seconds and milliseconds without mutating source date", () => {
      const date = new Date("2030-09-01T17:00:37.500");
      const normalized = DateUtil.setSecondsToMin(date);

      expect(date.getSeconds()).toBe(37);
      expect(date.getMilliseconds()).toBe(500);
      expect(normalized.getSeconds()).toBe(0);
      expect(normalized.getMilliseconds()).toBe(0);
      expect(normalized.getHours()).toBe(17);
      expect(normalized.getMinutes()).toBe(0);
    });
  });

  describe("isSameDate", () => {
    it("should return true if date1=date2", () => {
      const date = new Date();
      expect(DateUtil.isSameDay(date, date)).toBe(true);
    });
  });
});
