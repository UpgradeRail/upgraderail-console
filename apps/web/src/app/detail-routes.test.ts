import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import AppFleetDetailPage from "./app/fleets/[fleetId]/page";
import AppProposalDetailPage from "./app/upgrades/[proposalId]/page";
import AppFleetAnalysisPage from "./app/fleets/[fleetId]/analysis/page";
import ExploreFleetDetailPage from "./explore/fleets/[fleetId]/page";
import ExploreProposalDetailPage from "./explore/upgrades/[proposalId]/page";

describe("dynamic detail routes", () => {
  it("exports valid page components for app fleet and proposal detail routes", () => {
    expect(typeof AppFleetDetailPage).toBe("function");
    expect(typeof AppProposalDetailPage).toBe("function");
    expect(typeof AppFleetAnalysisPage).toBe("function");
  });

  it("exports valid page components for explore detail routes", () => {
    expect(typeof ExploreFleetDetailPage).toBe("function");
    expect(typeof ExploreProposalDetailPage).toBe("function");
  });

  it("renders every route page through React without throwing", () => {
    const pages = [
      AppFleetDetailPage,
      AppProposalDetailPage,
      AppFleetAnalysisPage,
      ExploreFleetDetailPage,
      ExploreProposalDetailPage,
    ] as unknown as React.ComponentType<Record<string, string>>[];
    for (const Page of pages) {
      expect(() => renderToStaticMarkup(React.createElement(Page, { fleetId: "f", proposalId: "1" }))).not.toThrow();
    }
  });
});
