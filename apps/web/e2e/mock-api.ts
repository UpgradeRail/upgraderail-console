import type { Page, Route } from "@playwright/test";

export const API_ORIGIN = "http://127.0.0.1:4010";
const h = (c: string) => c.repeat(64);

export const fixtures = {
  controllers: [{ id: "ctrl", network_id: "testnet", contract_id: "CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3", wasm_hash: h("a"), governance_epoch: 1, controller_version: 1 }],
  fleet: { id: "ctrl:fleet1", controller_id: "ctrl", fleet_hash: h("1"), tag: "payments", current_wasm_hash: h("b"), created_ledger: 100, created_transaction_hash: h("c") },
  proposal: { id: "ctrl:7", controller_id: "ctrl", proposal_id: 7, proposer: "GPROPOSER", kind: "UpgradeFleet", governance_epoch: 1, created_ledger: 200, expires_ledger: 900, approval_count: 1, approved_ledger: null, execute_after_ledger: null, status: "active", manifest_hash: h("d"), created_transaction_hash: h("e") },
  approvals: [{ id: "ctrl:7:GAPPROVER", proposal_id: "ctrl:7", approver: "GAPPROVER", approved_ledger: 210, transaction_hash: h("f"), revoked_ledger: null }],
  upgrade: { id: "tx:0", fleet_id: "ctrl:fleet1", proposal_id: "ctrl:6", old_wasm_hash: h("a"), new_wasm_hash: h("b"), manifest_hash: h("d"), ledger_sequence: 150, transaction_hash: h("9") },
  event: { id: "ev1", controller_id: "ctrl", network_id: "testnet", ledger_sequence: 150, transaction_hash: h("9"), event_index: 0, event_type: "fleet_upgraded", topics: [], data: {}, observed_at: "2026-01-01T00:00:00Z" },
  analysis: { id: "an1", network: "testnet", status: "blocked", engine_version: "upgraderail 0.1", error_message: "", current_artifact_id: "a", candidate_artifact_id: "b", created_by: null, created_at: "2026-01-01T00:00:00Z", started_at: null, finished_at: null },
  report: {
    status: "BLOCKED", current_wasm_hash: h("a"), candidate_wasm_hash: h("b"), engine_version: "upgraderail 0.1", created_at: "2026-01-01T00:00:00Z", runtime_evidence: {},
    findings: [
      { code: "FUNC001", severity: "BLOCKING", title: "Public function removed", message: "Function `get_value` is absent from the candidate contract.", evidence: [{ kind: "current_function", reference: "get_value" }] },
      { code: "FUNC004", severity: "INFO", title: "Public function added", message: "Function `read_text` was added by the candidate contract.", evidence: [{ kind: "candidate_function", reference: "read_text" }] },
      { code: "SPEC010", severity: "INFO", title: "Storage compatibility is not statically proven", message: "Run an explicit migration scenario.", evidence: [] },
    ],
    report: { status: "BLOCKED", storage_compatibility: "NOT PROVEN BY STATIC ANALYSIS", authorization_behavior: "NOT TESTED" },
  },
  manifest: { sha256: h("d"), bytes: "e30=", created_at: "2026-01-01T00:00:00Z" },
};

export type Overrides = Record<string, { status: number; body: unknown }>;

function lookup(path: string): unknown | undefined {
  const f = fixtures;
  const table: Record<string, unknown> = {
    "/api/v1/controllers": f.controllers,
    "/api/v1/fleets": [f.fleet],
    "/api/v1/fleets/ctrl:fleet1": f.fleet,
    "/api/v1/fleets/ctrl:fleet1/upgrades": [f.upgrade],
    "/api/v1/proposals": [f.proposal],
    "/api/v1/proposals/ctrl:7": f.proposal,
    "/api/v1/proposals/ctrl:7/approvals": f.approvals,
    "/api/v1/upgrades": [f.upgrade],
    "/api/v1/events": [f.event],
    "/api/v1/analyses": [f.analysis],
    "/api/v1/analyses/an1": f.analysis,
    "/api/v1/analyses/an1/report": f.report,
    "/api/v1/analyses/an1/manifest": f.manifest,
  };
  return table[path];
}

/** Serves deterministic Console API responses; unknown resources are 404. */
export async function mockApi(page: Page, overrides: Overrides = {}) {
  await page.route(`${API_ORIGIN}/**`, async (route: Route) => {
    const url = new URL(route.request().url());
    const path = decodeURIComponent(url.pathname);
    const headers = { "access-control-allow-origin": "http://127.0.0.1:3147", "access-control-allow-credentials": "true", "content-type": "application/json" };
    const override = overrides[path];
    if (override) return route.fulfill({ status: override.status, headers, body: JSON.stringify(override.body) });
    if (path === "/api/v1/auth/session") return route.fulfill({ status: 401, headers, body: JSON.stringify({ error: { message: "No session" } }) });
    const body = lookup(path);
    if (body === undefined) return route.fulfill({ status: 404, headers, body: JSON.stringify({ error: { code: "not_found", message: "resource was not found" } }) });
    return route.fulfill({ status: 200, headers, body: JSON.stringify(body) });
  });
}
