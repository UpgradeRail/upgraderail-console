import { buildDiff, subjectOf } from "@/lib/diff";
import type { Finding } from "@/lib/analysis";

export function DiffView({ findings }: { findings: Finding[] }) {
  const categories = buildDiff(findings);
  const total = categories.reduce((sum, c) => sum + c.entries.length, 0);
  return (
    <section aria-labelledby="diff-heading">
      <span className="eyebrow">COMPATIBILITY DIFF</span>
      <h2 id="diff-heading" style={{ margin: "4px 0 8px", fontSize: "24px" }}>Interface and runtime differences</h2>
      <p>Derived only from Engine findings for current → candidate. A category with no entries means the Engine reported none, not that it was proven identical.</p>
      {total === 0 && <section className="empty-state"><span className="eyebrow">NO DIFFERENCES REPORTED</span><h2>No interface differences in the Engine findings</h2><p>Storage layout and authorization remain unproven; see the evidence states.</p></section>}
      {categories.map((category) => (
        <div key={category.id} style={{ marginTop: "20px" }} data-diff-category={category.id}>
          <h3 style={{ margin: "0 0 8px" }}>{category.label} ({category.entries.length})</h3>
          {category.entries.length === 0 ? (
            <p>None reported.</p>
          ) : (
            <ul style={{ listStyle: "none", padding: 0, margin: 0, display: "grid", gap: "8px" }}>
              {category.entries.map((entry, index) => (
                <li key={`${entry.finding.code}-${index}`} className="data-note" style={{ marginTop: 0 }}>
                  <strong>{entry.symbol}</strong> <code>{subjectOf(entry.finding)}</code>
                  <span> — {entry.finding.code} ({entry.finding.severity.toLowerCase()}): {entry.finding.message || entry.finding.title}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      ))}
    </section>
  );
}
