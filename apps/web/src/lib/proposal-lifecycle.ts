import type { LiveProposal, ProposalStateTag } from "./governance-tx";

export type ProposalAction = "approve" | "revoke" | "cancel" | "execute";

export type ActionAvailability = { allowed: true } | { allowed: false; reason: string };

export type Lifecycle = {
  state: ProposalStateTag;
  label: string;
  detail: string;
  terminal: boolean;
  /** Ledgers left until the timelock elapses; null when no timelock is running. */
  timelockRemaining: number | null;
  /** Ledgers left before the proposal expires. */
  expiresIn: number;
  actions: Record<ProposalAction, ActionAvailability>;
};

const labels: Record<ProposalStateTag, string> = {
  AwaitingApprovals: "Awaiting approvals",
  Timelocked: "Timelocked",
  Ready: "Ready to execute",
  Expired: "Expired",
  Stale: "Stale (governance policy changed)",
  Executed: "Executed",
  Cancelled: "Cancelled",
};

const no = (reason: string): ActionAvailability => ({ allowed: false, reason });
const yes: ActionAvailability = { allowed: true };

/**
 * Mirrors the UpgradeController rules so the UI only offers actions the
 * contract can accept. The contract remains the authority: every action is
 * still simulated before signing.
 */
export function describeLifecycle(live: LiveProposal, address: string | null): Lifecycle {
  const { state } = live;
  const terminal = state === "Executed" || state === "Cancelled";
  const timelockRemaining =
    live.executeAfterLedger !== null && state === "Timelocked" ? Math.max(0, live.executeAfterLedger - live.ledger) : null;

  const detailByState: Record<ProposalStateTag, string> = {
    AwaitingApprovals: `${live.approvalCount} of ${live.threshold} required approvals recorded.`,
    Timelocked: `Threshold reached. Executable at ledger ${live.executeAfterLedger} (${timelockRemaining} ledgers remaining at ledger ${live.ledger}).`,
    Ready: `Threshold reached and the timelock elapsed${live.executeAfterLedger !== null ? ` at ledger ${live.executeAfterLedger}` : ""}. It expires at ledger ${live.expiresLedger}.`,
    Expired: `The proposal expired at ledger ${live.expiresLedger}. It can no longer be approved or executed.`,
    Stale: "Governance policy changed after this proposal was created, so it can no longer be approved or executed.",
    Executed: "This proposal was executed on-chain.",
    Cancelled: "This proposal was cancelled by its proposer.",
  };

  const connected = address !== null;
  const base = (): ActionAvailability | null => {
    if (!connected) return no("Connect and sign in with Freighter first.");
    return null;
  };

  let approve: ActionAvailability;
  if (base()) approve = base()!;
  else if (state !== "AwaitingApprovals" && state !== "Timelocked") approve = no(`Proposal is ${labels[state].toLowerCase()}.`);
  else if (!live.isApprover) approve = no("The connected account is not a policy approver.");
  else if (live.hasApproved) approve = no("The connected account has already approved.");
  else approve = yes;

  let revoke: ActionAvailability;
  if (base()) revoke = base()!;
  else if (terminal || state === "Stale") revoke = no(`Proposal is ${labels[state].toLowerCase()}.`);
  else if (!live.hasApproved) revoke = no("The connected account has no active approval to revoke.");
  else revoke = yes;

  let cancel: ActionAvailability;
  if (base()) cancel = base()!;
  else if (terminal) cancel = no(`Proposal is ${labels[state].toLowerCase()}.`);
  else if (live.proposer !== address) cancel = no("Only the proposer can cancel.");
  else if (live.approvalCount >= live.threshold) cancel = no("A proposal cannot be cancelled after the approval threshold is reached.");
  else cancel = yes;

  let execute: ActionAvailability;
  if (base()) execute = base()!;
  else if (state === "Ready") execute = yes;
  else if (state === "Timelocked") execute = no(`Timelock has ${timelockRemaining} ledgers remaining.`);
  else execute = no(`Proposal is ${labels[state].toLowerCase()}.`);

  return {
    state,
    label: labels[state],
    detail: detailByState[state],
    terminal,
    timelockRemaining,
    expiresIn: Math.max(0, live.expiresLedger - live.ledger),
    actions: { approve, revoke, cancel, execute },
  };
}
