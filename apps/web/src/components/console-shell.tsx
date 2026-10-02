import Link from "next/link";
import { Mark } from "./site-header";
import { ThemeToggle } from "./theme-toggle";
import { WalletButton } from "./wallet-button";

const links = [["Overview", "/app"], ["Fleets", "/app/fleets"], ["Upgrades", "/app/upgrades"], ["Activity", "/app/activity"], ["Integrations", "/app/integrations"], ["Settings", "/app/settings"]] as const;

export function ConsoleShell({ children }: Readonly<{ children: React.ReactNode }>) {
  return <div className="console-layout"><aside className="console-sidebar"><Link className="brand" href="/"><Mark />UpgradeRail</Link><span className="console-kicker">CONSOLE</span><nav aria-label="Console navigation">{links.map(([name, href]) => <Link key={href} href={href}>{name}</Link>)}</nav><div className="sidebar-bottom"><WalletButton /><span className="network-badge"><i /> Network unavailable</span><ThemeToggle /></div></aside><main className="console-main">{children}</main></div>;
}
