import type { Finding } from "./analysis";

export type DiffCategoryId =
  | "functions_added"
  | "functions_removed"
  | "inputs_changed"
  | "returns_changed"
  | "types"
  | "errors"
  | "events"
  | "metadata"
  | "runtime"
  | "other";

export type DiffChange = "added" | "removed" | "changed" | "info";

export type DiffEntry = { change: DiffChange; finding: Finding; symbol: string };

export type DiffCategory = { id: DiffCategoryId; label: string; entries: DiffEntry[] };

const labels: Record<DiffCategoryId, string> = {
  functions_added: "Added functions",
  functions_removed: "Removed functions",
  inputs_changed: "Changed inputs",
  returns_changed: "Changed returns",
  types: "Changed public types",
  errors: "Changed errors",
  events: "Changed events",
  metadata: "Metadata differences",
  runtime: "Runtime findings",
  other: "Other findings",
};

const order: DiffCategoryId[] = [
  "functions_added", "functions_removed", "inputs_changed", "returns_changed",
  "types", "errors", "events", "metadata", "runtime", "other",
];

// Rule codes are documented in the Engine's docs/analysis-rules.md.
// SPEC010 is a limitation notice, not an interface difference, and is shown in
// the evidence section instead.
const EXCLUDED = new Set(["SPEC010"]);

function classify(code: string): { category: DiffCategoryId; change: DiffChange } {
  switch (code) {
    case "FUNC001": return { category: "functions_removed", change: "removed" };
    case "FUNC002": return { category: "inputs_changed", change: "changed" };
    case "FUNC003": return { category: "returns_changed", change: "changed" };
    case "FUNC004": return { category: "functions_added", change: "added" };
    case "META001": return { category: "metadata", change: "changed" };
  }
  const family = /^(TYPE|ERROR|EVENT)(\d{3})$/.exec(code);
  if (family) {
    const category = family[1] === "TYPE" ? "types" : family[1] === "ERROR" ? "errors" : "events";
    const change: DiffChange = family[2] === "001" ? "removed" : family[2] === "002" ? "changed" : "added";
    return { category, change };
  }
  if (/^(AUTH|SIM|RESOURCE)\d{3}$/.test(code)) return { category: "runtime", change: "info" };
  return { category: "other", change: "info" };
}

const symbols: Record<DiffChange, string> = { added: "+ ADDED", removed: "− REMOVED", changed: "~ CHANGED", info: "• NOTE" };

export function buildDiff(findings: Finding[]): DiffCategory[] {
  const categories = new Map<DiffCategoryId, DiffEntry[]>(order.map((id) => [id, []]));
  for (const finding of findings) {
    if (EXCLUDED.has(finding.code)) continue;
    const { category, change } = classify(finding.code);
    categories.get(category)!.push({ change, finding, symbol: symbols[change] });
  }
  return order.map((id) => ({ id, label: labels[id], entries: categories.get(id)! }));
}

export function subjectOf(finding: Finding): string {
  const reference = finding.evidence.find((e) => e.reference)?.reference;
  return reference ?? finding.title;
}
