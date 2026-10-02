import { UnavailableState } from "@/components/empty-state";

export default function ConsolePage() { return <section className="console-page"><header className="console-heading"><div><span className="eyebrow">OVERVIEW</span><h1>Operations</h1><p>Review controller activity and release evidence.</p></div></header><div className="data-note">The indexer is not connected. Fleet counts, proposals, and network status are intentionally unavailable.</div><UnavailableState /></section>; }
