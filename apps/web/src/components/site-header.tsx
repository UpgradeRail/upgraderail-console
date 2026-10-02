import Link from "next/link";
import { ThemeToggle } from "./theme-toggle";

export function Mark() {
  return <span className="mark" aria-hidden="true"><i /><i /><i /></span>;
}

export function SiteHeader() {
  return <header className="site-header"><div className="header-inner">
    <Link className="brand" href="/"><Mark />UpgradeRail</Link>
    <nav aria-label="Main navigation"><Link href="/product">Product</Link><Link href="/how-it-works">How it works</Link><Link href="/security">Security</Link><Link href="/developers">Developers</Link><Link href="/explore">Explore</Link></nav>
    <div className="header-actions"><ThemeToggle /><Link className="button button-small" href="/app">Open console</Link></div>
  </div></header>;
}
