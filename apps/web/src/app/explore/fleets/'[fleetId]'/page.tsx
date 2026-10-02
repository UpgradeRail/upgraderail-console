import { SiteHeader } from "@/components/site-header";
import { UnavailableState } from "@/components/empty-state";
export default function ExploreFleetPage() { return <><SiteHeader /><main className="explore-layout"><span className="eyebrow">PUBLIC FLEET</span><h1>Fleet unavailable</h1><UnavailableState /></main></>; }
