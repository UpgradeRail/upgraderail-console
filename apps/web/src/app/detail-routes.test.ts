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

  it("instantiates route page elements without throw", async () => {
    const fleetPage = await AppFleetDetailPage();
    expect(fleetPage).toBeDefined();

    const proposalPage = await AppProposalDetailPage();
    expect(proposalPage).toBeDefined();

    const analysisPage = await AppFleetAnalysisPage();
    expect(analysisPage).toBeDefined();

    const exploreFleetPage = await ExploreFleetDetailPage();
    expect(exploreFleetPage).toBeDefined();

    const exploreProposalPage = await ExploreProposalDetailPage();
    expect(exploreProposalPage).toBeDefined();
  });
});
