import { ConsoleShell } from "@/components/console-shell";

export default function AppLayout({ children }: Readonly<{ children: React.ReactNode }>) { return <ConsoleShell>{children}</ConsoleShell>; }
