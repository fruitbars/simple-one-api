import { describe, expect, it } from "vitest";
import { chartPointPath, compactNumber, formatDuration, rangeQuery } from "./StatisticsPanel";

describe("statistics presentation helpers", () => {
  const now = new Date("2026-08-21T12:00:00.000Z");

  it("starts today at local midnight and uses daily buckets for longer ranges", () => {
    const day = rangeQuery("today", now);
    expect(day.bucket).toBe("hour");
    expect(day.from.getHours()).toBe(0);
    expect(day.from.getMinutes()).toBe(0);
    expect(day.from.getSeconds()).toBe(0);
    expect(day.from.getDate()).toBe(now.getDate());

    const month = rangeQuery("30d", now);
    expect(month.bucket).toBe("day");
    expect(month.to.getTime() - month.from.getTime()).toBe(30 * 24 * 60 * 60 * 1000);
  });

  it("keeps small metrics exact and compacts large values", () => {
    expect(compactNumber(9999)).toBe("9,999");
    expect(compactNumber(12500)).toMatch(/1[.。]?3万|12[,.]?5K/i);
  });

  it("formats latency as readable durations instead of compact counts", () => {
    expect(formatDuration(850)).toBe("850 ms");
    expect(formatDuration(1700)).toBe("1.7 秒");
    expect(formatDuration(10_000)).toBe("10 秒");
    expect(formatDuration(17_000)).toBe("17 秒");
    expect(formatDuration(65_900)).toBe("1 分 6 秒");
    expect(formatDuration(3_600_000)).toBe("1 小时");
    expect(formatDuration(Number.NaN)).toBe("-");
  });

  it("creates a visible centered marker for a single trend point", () => {
    expect(chartPointPath(5, 10, 40, 10, 900, 180)).toBe("M 490 100 l 0.01 0");
    expect(chartPointPath(5, 10, 40, 10, 900, 180, -16)).toBe("M 474 100 l 0.01 0");
  });
});
