import { UnavailableState } from "@/components/empty-state";
export default function FleetDetailPage() { return <section className="console-page"><header className="console-heading"><div><span className="eyebrow">FLEET</span><h1>Fleet unavailable</h1><p>Fleet details require an indexed controller event stream.</p></div></header><UnavailableState /></section>; }
