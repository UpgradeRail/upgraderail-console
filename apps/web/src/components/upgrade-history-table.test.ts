import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import UpgradeHistoryPage from "@/app/app/upgrades/history/page";
import { UpgradeHistoryTable, type UpgradeRecord } from "./upgrade-history-table";

const row: UpgradeRecord = {
  id: "tx:0", fleet_id: "ctrl:fleet1", proposal_id: "ctrl:7", old_wasm_hash: "aa".repeat(32),
  new_wasm_hash: "bb".repeat(32), manifest_hash: "cc".repeat(32), ledger_sequence: 1500, transaction_hash: "dd".repeat(32),
};

describe("UpgradeHistoryTable", () => {
  it("shows every projected field and links the fleet and proposal", () => {
    const html = renderToStaticMarkup(React.createElement(UpgradeHistoryTable, { upgrades: [row] }));
    for (const value of [row.old_wasm_hash, row.new_wasm_hash, row.manifest_hash, row.transaction_hash, "1500"]) expect(html).toContain(value);
    expect(html).toContain('href="/app/fleets/ctrl%3Afleet1"');
    expect(html).toContain('href="/app/upgrades/ctrl%3A7"');
  });

  it("does not invent timestamps or report links", () => {
    const html = renderToStaticMarkup(React.createElement(UpgradeHistoryTable, { upgrades: [row] }));
    expect(html).toContain("Not recorded");
    expect(html).toContain("No analysis linked");
  });

  it("renders an honest empty state", () => {
    const html = renderToStaticMarkup(React.createElement(UpgradeHistoryTable, { upgrades: [] }));
    expect(html).toContain("No executed upgrades are indexed");
    expect(html).not.toContain("<table");
  });

  it("renders the page in its loading state", () => {
    expect(renderToStaticMarkup(React.createElement(UpgradeHistoryPage))).toContain("Reading upgrade history");
  });
});
