import path from "node:path";
import { expect, test, type Route } from "@playwright/test";
import { mockApi } from "./mock-api";

const fixture = (name: string) => path.join(__dirname, "fixtures", name);

const signedInSession = { address: "GUPLOADER", network: "testnet" };

test.beforeEach(async ({ page }) => {
  await mockApi(page, { "/api/v1/auth/session": { status: 200, body: signedInSession } });
});

test("upload page renders both upload slots when signed in", async ({ page }) => {
  await page.goto("/app/analyses/new");
  await expect(page.getByRole("heading", { name: "New analysis" })).toBeVisible();
  await expect(page.getByText("CURRENT WASM", { exact: true })).toBeVisible();
  await expect(page.getByText("CANDIDATE WASM", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Create analysis job" })).toBeDisabled();
});

test("upload page shows a sign-in state when no session exists", async ({ page }) => {
  await mockApi(page);
  await page.goto("/app/analyses/new");
  await expect(page.getByRole("heading", { name: "Wallet session needed" })).toBeVisible();
});

test("an invalid upload shows a clear validation error", async ({ page }) => {
  await page.route(`**/api/v1/artifacts`, async (route: Route) => {
    await route.fulfill({
      status: 400,
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ error: { code: "invalid_wasm", message: "The uploaded file does not look like a WASM module." } }),
    });
  });
  await page.goto("/app/analyses/new");
  const fileInputs = page.locator('input[type="file"]');
  await fileInputs.first().setInputFiles(fixture("not-wasm.txt"));
  await expect(page.locator('p[role="alert"]')).toHaveText("The uploaded file does not look like a WASM module.");
});

test("uploading real files and creating a job shows honest status transitions through to ready", async ({ page }) => {
  let uploadCount = 0;
  await page.route(`**/api/v1/artifacts`, async (route: Route) => {
    uploadCount += 1;
    const id = `artifact-${uploadCount}`;
    await route.fulfill({
      status: 201,
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        id,
        sha256: "a".repeat(64),
        size_bytes: 8,
        content_type: "application/wasm",
        uploader: signedInSession.address,
        created_at: "2026-01-01T00:00:00Z",
      }),
    });
  });
  await page.route(`**/api/v1/analyses`, async (route: Route) => {
    if (route.request().method() !== "POST") return route.fallback();
    await route.fulfill({
      status: 202,
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ id: "job-upload-1", status: "queued" }),
    });
  });
  let pollCount = 0;
  await page.route(`**/api/v1/analyses/job-upload-1`, async (route: Route) => {
    pollCount += 1;
    const status = pollCount === 1 ? "queued" : pollCount === 2 ? "running" : "ready";
    await route.fulfill({
      status: 200,
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        id: "job-upload-1",
        network: "testnet",
        status,
        engine_version: status === "ready" ? "upgraderail 0.1.0" : "",
        error_message: "",
        current_artifact_id: "artifact-1",
        candidate_artifact_id: "artifact-2",
        created_by: signedInSession.address,
        created_at: "2026-01-01T00:00:00Z",
        started_at: null,
        finished_at: null,
      }),
    });
  });

  await page.goto("/app/analyses/new");
  const fileInputs = page.locator('input[type="file"]');
  await fileInputs.nth(0).setInputFiles(fixture("current.wasm"));
  await fileInputs.nth(1).setInputFiles(fixture("candidate.wasm"));
  await expect(page.getByText("SHA-256 (server-computed):").first()).toBeVisible();

  const createButton = page.getByRole("button", { name: "Create analysis job" });
  await expect(createButton).toBeEnabled();
  await createButton.click();

  // No fake progress bar and no fabricated score: the status shown must be
  // exactly what the API reports at each poll, ending on the real terminal
  // state once the (mocked) worker reports READY.
  await expect(page.getByRole("heading", { name: "Status: QUEUED" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Status: RUNNING" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Status: READY" })).toBeVisible();
  const link = page.getByRole("link", { name: "View the preflight analysis" });
  await expect(link).toBeVisible();
  await expect(link).toHaveAttribute("href", "/app/analyses/job-upload-1");
});

test("a blocked job links to the preflight analysis instead of hiding the result", async ({ page }) => {
  await page.route(`**/api/v1/artifacts`, async (route: Route) => {
    await route.fulfill({
      status: 201,
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ id: "artifact-x", sha256: "b".repeat(64), size_bytes: 8, content_type: "application/wasm", uploader: signedInSession.address, created_at: "2026-01-01T00:00:00Z" }),
    });
  });
  await page.route(`**/api/v1/analyses`, async (route: Route) => {
    if (route.request().method() !== "POST") return route.fallback();
    await route.fulfill({ status: 202, headers: { "content-type": "application/json" }, body: JSON.stringify({ id: "job-blocked-1", status: "queued" }) });
  });
  await page.route(`**/api/v1/analyses/job-blocked-1`, async (route: Route) => {
    await route.fulfill({
      status: 200,
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        id: "job-blocked-1", network: "testnet", status: "blocked", engine_version: "upgraderail 0.1.0", error_message: "",
        current_artifact_id: "artifact-x", candidate_artifact_id: "artifact-x", created_by: signedInSession.address,
        created_at: "2026-01-01T00:00:00Z", started_at: null, finished_at: null,
      }),
    });
  });

  await page.goto("/app/analyses/new");
  const fileInputs = page.locator('input[type="file"]');
  await fileInputs.nth(0).setInputFiles(fixture("current.wasm"));
  await fileInputs.nth(1).setInputFiles(fixture("candidate.wasm"));
  await page.getByRole("button", { name: "Create analysis job" }).click();
  await expect(page.getByRole("heading", { name: "Status: BLOCKED" })).toBeVisible();
  await expect(page.getByRole("link", { name: "View the preflight analysis" })).toHaveAttribute("href", "/app/analyses/job-blocked-1");
});

test("new analysis route is keyboard operable", async ({ page }) => {
  await page.goto("/app/analyses/new");
  await page.getByLabel("Choose WASM file", { exact: false }).first().focus();
  await expect(page.locator(':focus')).toHaveAttribute("type", "file");
});
