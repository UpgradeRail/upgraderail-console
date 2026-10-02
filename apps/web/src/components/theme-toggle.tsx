"use client";

import { useEffect, useSyncExternalStore } from "react";

type Theme = "light" | "dark" | "system";

const labels: Record<Theme, string> = { light: "Light", dark: "Dark", system: "System" };

export function ThemeToggle() {
  const theme = useSyncExternalStore(
    (notify) => {
      window.addEventListener("upgraderail-theme", notify);
      return () => window.removeEventListener("upgraderail-theme", notify);
    },
    () => readTheme(),
    () => "system"
  );

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  function update(next: Theme) {
    window.localStorage.setItem("upgraderail-theme", next);
    window.dispatchEvent(new Event("upgraderail-theme"));
  }

  return <div className="theme-control" aria-label="Color theme">
    {(Object.keys(labels) as Theme[]).map((item) => <button key={item} className={theme === item ? "selected" : ""} onClick={() => update(item)} aria-pressed={theme === item}>{labels[item]}</button>)}
  </div>;
}

function readTheme(): Theme {
  const saved = window.localStorage.getItem("upgraderail-theme");
  return saved === "light" || saved === "dark" || saved === "system" ? saved : "system";
}
