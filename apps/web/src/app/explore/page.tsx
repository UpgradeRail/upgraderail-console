import { SiteHeader } from "@/components/site-header";
import { DataList } from "@/components/data-list";

export default function ExplorePage() { return <><SiteHeader /><main className="explore-layout"><span className="eyebrow">PUBLIC EXPLORER</span><h1>Controller activity, when indexed.</h1><p className="data-note">This explorer reads only from the Console indexer. It does not present sample fleets or proposals as live activity.</p><DataList endpoint="/api/v1/fleets" empty="No public fleets are indexed" columns={[{ key: "tag", label: "Tag" }, { key: "fleet_hash", label: "Fleet" }, { key: "current_wasm_hash", label: "Current executable" }]} /></main></>; }
