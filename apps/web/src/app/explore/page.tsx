import { SiteHeader } from "@/components/site-header";
import { UnavailableState } from "@/components/empty-state";

export default function ExplorePage() { return <><SiteHeader /><main className="explore-layout"><span className="eyebrow">PUBLIC EXPLORER</span><h1>Controller activity, when indexed.</h1><p className="data-note">This explorer reads from the Console indexer. It currently has no configured public API, so it does not present sample fleets or proposals as live activity.</p><UnavailableState title="Public indexing is not configured" detail="Set an API base URL and run the indexer against an UpgradeController to publish verified public history." /></main></>; }
