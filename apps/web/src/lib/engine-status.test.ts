import { describe, expect, it } from "vitest";
import { isBlockingStatus } from "./engine-status";

describe("isBlockingStatus", () => {
  it("preserves the Engine blocker state", () => {
    expect(isBlockingStatus("BLOCKED")).toBe(true);
    expect(isBlockingStatus("READY_WITH_WARNINGS")).toBe(false);
    expect(isBlockingStatus("READY")).toBe(false);
  });
});
