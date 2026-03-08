import { describe, it, expect } from "vitest";
import { formatUSD, formatTokens, formatUptime } from "./format";

describe("formatUSD", () => {
  it("formats dollars with two decimals", () => {
    expect(formatUSD(1.5)).toBe("$1.50");
    expect(formatUSD(0)).toBe("$0.00");
    expect(formatUSD(123.456)).toBe("$123.46");
  });
});

describe("formatTokens", () => {
  it("returns '0' for zero", () => {
    expect(formatTokens(0)).toBe("0");
  });

  it("formats thousands as k", () => {
    expect(formatTokens(1500)).toBe("1.5k");
    expect(formatTokens(12400)).toBe("12.4k");
  });

  it("formats millions as M", () => {
    expect(formatTokens(1_500_000)).toBe("1.5M");
  });

  it("returns raw number for small values", () => {
    expect(formatTokens(500)).toBe("500");
  });
});

describe("formatUptime", () => {
  it("returns '-' for zero or negative", () => {
    expect(formatUptime(0)).toBe("-");
    expect(formatUptime(-1)).toBe("-");
  });

  it("formats seconds", () => {
    expect(formatUptime(45)).toBe("45s");
  });

  it("formats minutes and seconds", () => {
    expect(formatUptime(125)).toBe("2m 5s");
  });

  it("formats hours and minutes", () => {
    expect(formatUptime(3725)).toBe("1h 2m");
  });

  it("formats days and hours", () => {
    expect(formatUptime(90000)).toBe("1d 1h");
  });
});
