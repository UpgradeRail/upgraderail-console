import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { DiffView } from "@/components/diff-view";
import { parseFindings, type Finding } from "./analysis";
import { buildDiff } from "./diff";
// Real output of `upgraderail compare` on the contracts fleet fixtures.
import breaking from "./fixtures/engine-compare-breaking.json";
import compatible from "./fixtures/engine-compare-compatible.json";

const finding = (code: string, severity: Finding["severity"] = "BLOCKING"): Finding => ({
  code, severity, title: code, message: `${code} message`, evidence: [{ kind: "k", reference: `ref-${code}` }],
});
const byId = (findings: Finding[]) => Object.fromEntries(buildDiff(findings).map((c) => [c.id, c.entries]));

describe("buildDiff", () => {
  it("maps every documented Engine rule to its category and change", () => {
    const diff = byId([
      "FUNC001", "FUNC002", "FUNC003", "FUNC004", "TYPE001", "TYPE002", "TYPE003",
      "ERROR001", "ERROR002", "ERROR003", "EVENT001", "EVENT002", "EVENT003",
      "META001", "AUTH001", "AUTH002", "SIM001", "RESOURCE001", "NEW999",
    ].map((code) => finding(code)));
    expect(diff.functions_removed.map((e) => e.change)).toEqual(["removed"]);
    expect(diff.functions_added.map((e) => e.change)).toEqual(["added"]);
    expect(diff.inputs_changed).toHaveLength(1);
    expect(diff.returns_changed).toHaveLength(1);
    expect(diff.types.map((e) => e.change)).toEqual(["removed", "changed", "added"]);
    expect(diff.errors.map((e) => e.change)).toEqual(["removed", "changed", "added"]);
    expect(diff.events.map((e) => e.change)).toEqual(["removed", "changed", "added"]);
    expect(diff.metadata).toHaveLength(1);
    expect(diff.runtime).toHaveLength(4);
    expect(diff.other.map((e) => e.finding.code)).toEqual(["NEW999"]);
  });

  it("keeps the storage limitation notice out of the interface diff", () => {
    expect(buildDiff([finding("SPEC010", "INFO")]).every((c) => c.entries.length === 0)).toBe(true);
  });

  it("classifies real Engine output for a compatible and a breaking upgrade", () => {
    const ok = byId(parseFindings(compatible.findings));
    expect(ok.functions_added.map((e) => e.finding.evidence[0].reference)).toEqual(["is_positive"]);
    expect(ok.functions_removed).toHaveLength(0);
    const bad = byId(parseFindings(breaking.findings));
    expect(bad.functions_removed.map((e) => e.finding.evidence[0].reference)).toEqual(["__constructor", "get_value", "set_value"]);
    expect(bad.functions_added).toHaveLength(1);
  });
});

describe("DiffView", () => {
  it("renders symbols and text, not color alone", () => {
    const html = renderToStaticMarkup(React.createElement(DiffView, { findings: parseFindings(breaking.findings) }));
    expect(html).toContain("− REMOVED");
    expect(html).toContain("+ ADDED");
    expect(html).toContain("Removed functions (3)");
    expect(html).toContain("get_value");
    expect(html).toContain("Changed inputs (0)");
    expect(html).toContain("None reported.");
  });

  it("says no differences were reported rather than claiming identity", () => {
    const html = renderToStaticMarkup(React.createElement(DiffView, { findings: [] }));
    expect(html).toContain("NO DIFFERENCES REPORTED");
    expect(html).toContain("not that it was proven identical");
  });
});
