import { UnavailableState } from "@/components/empty-state";
export default function ProposalDetailPage() { return <section className="console-page"><header className="console-heading"><div><span className="eyebrow">PROPOSAL</span><h1>Proposal unavailable</h1><p>Live controller state is read before approval, revocation, or execution.</p></div></header><UnavailableState /></section>; }
