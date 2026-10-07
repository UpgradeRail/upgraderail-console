import { describe, expect, it } from "vitest";
import { decodeRouteParam } from "./route-param";

describe("decodeRouteParam", () => {
  it("decodes the colon in indexed ids exactly once", () => {
    expect(decodeRouteParam("ctrl%3Afleet1")).toBe("ctrl:fleet1");
    expect(encodeURIComponent(decodeRouteParam("ctrl%3Afleet1"))).toBe("ctrl%3Afleet1");
  });
  it("leaves plain and malformed values alone", () => {
    expect(decodeRouteParam("test-fleet-1")).toBe("test-fleet-1");
    expect(decodeRouteParam("%E0%A4%A")).toBe("%E0%A4%A");
  });
});
