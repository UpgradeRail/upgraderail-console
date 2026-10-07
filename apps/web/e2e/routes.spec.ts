import { expect, test } from "@playwright/test";
import { fixtures, mockApi } from "./mock-api";

test.beforeEach(async ({ page }) => mockApi(page));

const publicRoutes = ["/", "/product", "/how-it-works", "/security", "/docs", "/developers", "/explore"];

for (const route of publicRoutes) {
  test(`public route ${route} renders a heading`, async ({ page }) => {
    const response = await page.goto(route);
    expect(response?.status()).toBe(200);
    await expect(page.locator("h1").first()).toBeVisible();
  });
}

test("overview shows real projected counts from the API", async ({ page }) => {
  await page.goto("/app");
  await expect(page.getByText(fixtures.controllers[0].contract_id).first()).toBeVisible();
  await expect(page.locator("main")).not.toContainText("Loading");
});

test("activity lists indexed events", async ({ page }) => {
  await page.goto("/app/activity");
  await expect(page.getByText("fleet_upgraded").first()).toBeVisible();
});

test("fleet detail shows projected fields and upgrade history", async ({ page }) => {
  await page.goto("/app/fleets/ctrl%3Afleet1");
  await expect(page.getByRole("heading", { name: "Fleet: payments" })).toBeVisible();
  await expect(page.getByText(fixtures.upgrade.transaction_hash).first()).toBeVisible();
});

test("fleet detail for an unknown fleet is an honest 404 state", async ({ page }) => {
  await page.goto("/app/fleets/nope");
  await expect(page.getByRole("heading", { name: "Fleet not found" })).toBeVisible();
});

test("proposal detail shows approvals and signed-out governance state", async ({ page }) => {
  await page.goto("/app/upgrades/ctrl%3A7");
  await expect(page.getByRole("heading", { name: "Proposal #7" })).toBeVisible();
  await expect(page.getByText("GAPPROVER").first()).toBeVisible();
  await expect(page.getByRole("heading", { name: "Wallet session needed" })).toBeVisible();
});

test("proposal detail for an unknown proposal is an honest 404 state", async ({ page }) => {
  await page.goto("/app/upgrades/ctrl%3A999");
  await expect(page.getByRole("heading", { name: "Proposal not found" })).toBeVisible();
});

test("explore fleet and proposal detail routes are reachable", async ({ page }) => {
  await page.goto("/explore/fleets/ctrl%3Afleet1");
  await expect(page.getByText("PUBLIC FLEET EXPLORER")).toBeVisible();
  await page.goto("/explore/upgrades/ctrl%3A7");
  await expect(page.getByText(fixtures.proposal.proposer).first()).toBeVisible();
});

test("upgrade history lists executed upgrades without invented timestamps", async ({ page }) => {
  await page.goto("/app/upgrades/history");
  await expect(page.getByText(fixtures.upgrade.new_wasm_hash)).toBeVisible();
  await expect(page.getByText("Not recorded").first()).toBeVisible();
});

test("preflight analysis shows blocking findings and unproven evidence", async ({ page }) => {
  await page.goto("/app/analyses/an1");
  await expect(page.getByRole("heading", { name: "Engine status: BLOCKED" })).toBeVisible();
  await expect(page.getByText("✖ BLOCKING")).toBeVisible();
  await expect(page.getByText("NOT PROVEN BY STATIC ANALYSIS").first()).toBeVisible();
  await expect(page.getByText("NOT TESTED").first()).toBeVisible();
  await expect(page.getByText("− REMOVED")).toBeVisible();
  await expect(page.getByRole("heading", { name: "This analysis cannot be proposed" })).toBeVisible();
});

test("API failure shows an error state rather than empty data", async ({ page }) => {
  await mockApi(page, { "/api/v1/upgrades": { status: 500, body: { error: { message: "boom" } } } });
  await page.goto("/app/upgrades/history");
  await expect(page.getByRole("heading", { name: "Upgrade history could not be loaded" })).toBeVisible();
});
