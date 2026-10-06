import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import ConsolePage from "./app/page";
import ActivityPage from "./app/activity/page";

describe("dashboard and activity pages", () => {
  it("exports valid ConsolePage and ActivityPage components", () => {
    expect(typeof ConsolePage).toBe("function");
    expect(typeof ActivityPage).toBe("function");
  });

  it("renders ConsolePage in initial loading state", () => {
    const html = renderToStaticMarkup(React.createElement(ConsolePage));
    expect(html).toContain("OVERVIEW");
    expect(html).toContain("Operations");
    expect(html).toContain("Reading controller telemetry");
  });

  it("renders ActivityPage in initial loading state", () => {
    const html = renderToStaticMarkup(React.createElement(ActivityPage));
    expect(html).toContain("ACTIVITY");
    expect(html).toContain("Event history");
    expect(html).toContain("Reading event journal");
  });
});
