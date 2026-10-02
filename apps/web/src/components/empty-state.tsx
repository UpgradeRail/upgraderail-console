import Link from "next/link";

export function UnavailableState({ title = "No indexed data yet", detail = "Connect an API and indexer to display verified controller activity." }: { title?: string; detail?: string }) {
  return <section className="empty-state"><span className="eyebrow">DATA UNAVAILABLE</span><h2>{title}</h2><p>{detail}</p><Link className="text-link" href="/docs">Read integration requirements →</Link></section>;
}
