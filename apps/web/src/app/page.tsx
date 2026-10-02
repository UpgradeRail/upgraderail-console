import Link from "next/link";
import { UpgradeRailDiagram } from "@/components/rail";
import { SiteHeader } from "@/components/site-header";

export default function Home() {
  return <><SiteHeader /><main>
    <section className="home-hero"><span className="eyebrow">SOROBAN UPGRADE OPERATIONS</span><h1>Upgrade contracts without upgrading your risk.</h1><p>Analyze, simulate, approve, and verify Soroban fleet upgrades before they reach production.</p><div className="hero-actions"><Link className="button button-accent" href="/app">Open console</Link><Link className="button" href="/how-it-works">How it works</Link></div></section>
    <UpgradeRailDiagram />
    <section className="feature-band"><div className="feature-grid"><article><span className="eyebrow">01 / EVIDENCE</span><h2>Inspect the executable</h2><p>Use UpgradeRail Engine reports to review compatibility findings and runtime evidence.</p></article><article><span className="eyebrow">02 / GOVERNANCE</span><h2>Keep approval on-chain</h2><p>Proposals, approvals, timelocks, and execution remain with UpgradeController.</p></article><article><span className="eyebrow">03 / HISTORY</span><h2>Verify the record</h2><p>Trace indexed controller events and compare proposal commitments with release manifests.</p></article></div></section>
    <section className="closing"><span className="eyebrow">PUBLIC, REVIEWABLE, WALLET-SIGNED</span><h2>Give upgrades the evidence and process they deserve.</h2><Link className="button button-accent" href="/explore">Explore public activity</Link></section>
  </main><footer className="footer"><span>UpgradeRail Console</span><span>Protocol 28 production target</span></footer></>;
}
