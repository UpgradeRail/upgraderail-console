import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { mockApi } from "./mock-api";

test.beforeEach(async ({ page }) => mockApi(page));

const routes = [
  "/", "/product", "/explore", "/app", "/app/activity", "/app/fleets", "/app/fleets/ctrl%3Afleet1",
  "/app/upgrades", "/app/upgrades/ctrl%3A7", "/app/upgrades/history", "/app/analyses", "/app/analyses/an1",
];

async function settle(page: Page, route: string) {
  await page.goto(route);
  await page.waitForLoadState("networkidle");
  await expect(page.getByText(/^(LOADING)$/)).toHaveCount(0);
}

for (const scheme of ["light", "dark"] as const) {
  for (const route of routes) {
    test(`no axe violations on ${route} (${scheme})`, async ({ page }) => {
      await page.emulateMedia({ colorScheme: scheme });
      await page.addInitScript((value) => window.localStorage.setItem("upgraderail-theme", value), scheme);
      await settle(page, route);
      expect(await page.evaluate(() => document.documentElement.dataset.theme)).toBe(scheme);
      const results = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
      expect(results.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(", ")}`)).toEqual([]);
    });
  }
}

test("reduced motion disables transitions and animations", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await settle(page, "/app");
  const offenders = await page.evaluate(() =>
    Array.from(document.querySelectorAll("*")).filter((el) => {
      const s = getComputedStyle(el);
      const t = parseFloat(s.transitionDuration);
      const a = parseFloat(s.animationDuration);
      return (t > 0.001) || (a > 0.001);
    }).map((el) => el.tagName.toLowerCase() + (el.className ? `.${String(el.className).split(" ")[0]}` : "")),
  );
  expect(offenders).toEqual([]);
});

test("theme toggle switches between light and dark", async ({ page }) => {
  await settle(page, "/app");
  await page.getByRole("button", { name: "Dark", exact: true }).click();
  await expect.poll(() => page.evaluate(() => document.documentElement.dataset.theme)).toBe("dark");
  await page.getByRole("button", { name: "Light", exact: true }).click();
  await expect.poll(() => page.evaluate(() => document.documentElement.dataset.theme)).toBe("light");
});

test("console navigation is reachable and operable by keyboard", async ({ page }) => {
  await settle(page, "/app");
  const nav = page.getByRole("navigation", { name: "Console navigation" });
  const link = nav.getByRole("link", { name: "Activity" });
  await link.focus();
  await expect(link).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/\/app\/activity$/);
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
});

test("tab order reaches interactive controls and shows a focus indicator", async ({ page }) => {
  await settle(page, "/app");
  const seen = new Set<string>();
  for (let i = 0; i < 25; i++) {
    await page.keyboard.press("Tab");
    seen.add(await page.evaluate(() => document.activeElement?.tagName ?? ""));
  }
  expect(seen.has("A")).toBe(true);
  expect(seen.has("BUTTON")).toBe(true);
  const outline = await page.evaluate(() => {
    const el = document.activeElement as HTMLElement;
    const s = getComputedStyle(el);
    return { width: s.outlineWidth, style: s.outlineStyle, shadow: s.boxShadow };
  });
  expect(outline.style !== "none" || outline.shadow !== "none").toBe(true);
});
