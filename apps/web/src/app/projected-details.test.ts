import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import AppFleetDetailPage from "./app/fleets/[fleetId]/page";
import AppProposalDetailPage from "./app/upgrades/[proposalId]/page";
import ExploreFleetPage from "./explore/fleets/[fleetId]/page";
import ExploreProposalPage from "./explore/upgrades/[proposalId]/page";

type FleetPageProps = { fleetId?: string; params?: Promise<{ fleetId: string }> };
type ProposalPageProps = { proposalId?: string; params?: Promise<{ proposalId: string }> };

describe("projected detail pages", () => {
  it("renders AppFleetDetailPage with initial loading state", () => {
    const Component = AppFleetDetailPage as unknown as React.FC<FleetPageProps>;
    const html = renderToStaticMarkup(
      React.createElement(Component, {
        fleetId: "test-fleet-1",
      })
    );
    expect(html).toContain("FLEET");
    expect(html).toContain("Fleet detail");
    expect(html).toContain("Reading fleet projection");
  });

  it("renders ExploreFleetPage with initial loading state", () => {
    const Component = ExploreFleetPage as unknown as React.FC<FleetPageProps>;
    const html = renderToStaticMarkup(
      React.createElement(Component, {
        fleetId: "test-fleet-1",
      })
    );
    expect(html).toContain("PUBLIC FLEET EXPLORER");
    expect(html).toContain("Reading fleet projection");
  });

  it("renders AppProposalDetailPage with initial loading state", () => {
    const Component = AppProposalDetailPage as unknown as React.FC<ProposalPageProps>;
    const html = renderToStaticMarkup(
      React.createElement(Component, {
        proposalId: "test-prop-1",
      })
    );
    expect(html).toContain("PROPOSAL");
    expect(html).toContain("Proposal detail");
    expect(html).toContain("Reading proposal projection");
  });

  it("renders ExploreProposalPage with initial loading state", () => {
    const Component = ExploreProposalPage as unknown as React.FC<ProposalPageProps>;
    const html = renderToStaticMarkup(
      React.createElement(Component, {
        proposalId: "test-prop-1",
      })
    );
    expect(html).toContain("PUBLIC PROPOSAL EXPLORER");
    expect(html).toContain("Reading proposal projection");
  });
});
