import { describe, expect, it } from "vitest";
import type { LiveProposal, ProposalStateTag } from "./governance-tx";
import { describeLifecycle, projectionLags } from "./proposal-lifecycle";

const me = "GME";
const live = (over: Partial<LiveProposal> = {}): LiveProposal => ({
  ledger: 100, state: "AwaitingApprovals", proposer: "GOTHER", approvalCount: 1, threshold: 2,
  timelockLedgers: 50, executeAfterLedger: null, expiresLedger: 500, isApprover: true, hasApproved: false, ...over,
});
const allowed = (l: ReturnType<typeof describeLifecycle>) =>
  Object.entries(l.actions).filter(([, a]) => a.allowed).map(([k]) => k);

describe("describeLifecycle", () => {
  it("offers approve (not revoke/execute) while awaiting approvals", () => {
    expect(allowed(describeLifecycle(live(), me))).toEqual(["approve"]);
  });

  it("lets the proposer cancel only below the threshold", () => {
    expect(allowed(describeLifecycle(live({ proposer: me }), me))).toEqual(["approve", "cancel"]);
    const reached = describeLifecycle(live({ proposer: me, approvalCount: 2, state: "Timelocked", executeAfterLedger: 150 }), me);
    expect(reached.actions.cancel).toEqual({ allowed: false, reason: expect.stringContaining("after the approval threshold") });
  });

  it("shows the running timelock and blocks execution until it elapses", () => {
    const l = describeLifecycle(live({ state: "Timelocked", approvalCount: 2, executeAfterLedger: 150, hasApproved: true }), me);
    expect(l.timelockRemaining).toBe(50);
    expect(l.detail).toContain("ledger 150");
    expect(l.actions.execute).toEqual({ allowed: false, reason: "Timelock has 50 ledgers remaining." });
    expect(l.actions.revoke.allowed).toBe(true);
  });

  it("allows execution only when Ready", () => {
    const l = describeLifecycle(live({ state: "Ready", approvalCount: 2, executeAfterLedger: 90, hasApproved: true }), me);
    expect(l.actions.execute.allowed).toBe(true);
    expect(l.timelockRemaining).toBeNull();
  });

  it.each<[ProposalStateTag, string[]]>([
    ["Executed", []],
    ["Cancelled", []],
    ["Stale", []],
    ["Expired", []],
  ])("offers nothing for %s that the contract would reject", (state, expected) => {
    const l = describeLifecycle(live({ state, hasApproved: false, proposer: "GOTHER" }), me);
    expect(allowed(l)).toEqual(expected);
  });

  it("still lets an approver revoke an expired proposal, as the contract does", () => {
    expect(describeLifecycle(live({ state: "Expired", hasApproved: true }), me).actions.revoke.allowed).toBe(true);
  });

  it("respects approver membership and prior approval", () => {
    expect(describeLifecycle(live({ isApprover: false }), me).actions.approve).toMatchObject({ allowed: false });
    expect(describeLifecycle(live({ hasApproved: true }), me).actions.approve).toMatchObject({ allowed: false, reason: expect.stringContaining("already approved") });
  });

  it("requires a wallet session for every action", () => {
    const l = describeLifecycle(live({ state: "Ready" }), null);
    expect(allowed(l)).toEqual([]);
  });

  it("flags an indexed projection that is behind the live state", () => {
    expect(projectionLags("active", "Executed")).toBe(true);
    expect(projectionLags("executed", "Executed")).toBe(false);
    expect(projectionLags("executed", "Ready")).toBe(true);
    expect(projectionLags("active", "Timelocked")).toBe(false);
    expect(projectionLags(undefined, "Executed")).toBe(false);
  });
});
