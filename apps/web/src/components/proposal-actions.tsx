"use client";

import { useCallback, useEffect, useState } from "react";
import { GovernanceActionPanel } from "./governance-action-panel";
import { api, ApiError } from "@/lib/api";
import { buildApprove, buildCancelProposal, buildExecuteProposal, buildRevokeApproval, defaultRpc, readLiveProposal, type LiveProposal } from "@/lib/governance-tx";
import { LifecycleCard } from "./lifecycle-card";
import { describeLifecycle, type ActionAvailability } from "@/lib/proposal-lifecycle";

type Session = { address: string; network: string };
type Live =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "signed_out"; message: string }
  | { kind: "ready"; address: string; live: LiveProposal };

const reasonOf = (a: ActionAvailability) => (a.allowed ? undefined : a.reason);

/**
 * Governance actions for one proposal. Eligibility comes from the live
 * on-chain proposal state, not from the indexed projection, which may lag.
 */
export function ProposalActions({ proposalId, projectedStatus, onChanged }: { proposalId: number | string; projectedStatus?: string; onChanged?: () => void }) {
  const [live, setLive] = useState<Live>({ kind: "loading" });

  const refresh = useCallback(async () => {
    try {
      const session = await api<Session>("/api/v1/auth/session");
      const value = await readLiveProposal(defaultRpc(), session.address, proposalId);
      setLive({ kind: "ready", address: session.address, live: value });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setLive({ kind: "signed_out", message: "Connect and sign in with Freighter in the sidebar to act on this proposal." });
      } else {
        setLive({ kind: "error", message: error instanceof Error ? error.message : "Live proposal state could not be read." });
      }
    }
  }, [proposalId]);

  useEffect(() => {
    // Fetch-on-mount; state is set only after the awaited reads resolve.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void refresh();
  }, [refresh]);

  const afterConfirmed = () => {
    void refresh();
    onChanged?.();
  };

  if (live.kind === "loading") return <section className="empty-state"><span className="eyebrow">LOADING</span><h2>Reading live proposal state</h2><p>Simulating read-only controller calls on Testnet.</p></section>;
  if (live.kind === "signed_out") return <section className="empty-state"><span className="eyebrow">SIGN IN REQUIRED</span><h2>Wallet session needed</h2><p>{live.message}</p></section>;
  if (live.kind === "error") return <section className="empty-state"><span className="eyebrow">LIVE STATE UNAVAILABLE</span><h2>Governance actions are unavailable</h2><p>{live.message}</p></section>;

  const lifecycle = describeLifecycle(live.live, live.address);
  const id = String(proposalId);
  return (
    <div style={{ display: "grid", gap: "24px" }}>
      <LifecycleCard lifecycle={lifecycle} live={live.live} projectedStatus={projectedStatus} />
      <GovernanceActionPanel
        title="Approve proposal"
        description="Records your approval. When the threshold is reached the policy timelock starts."
        action="approve"
        details={[["Proposal", `#${id}`], ["Approver", live.address]]}
        confirmLabel="Submit approval to Testnet"
        disabledReason={reasonOf(lifecycle.actions.approve)}
        build={(address) => buildApprove(address, id)}
        onConfirmed={afterConfirmed}
      />
      <GovernanceActionPanel
        title="Revoke approval"
        description="Withdraws your approval. If this drops the proposal below the threshold, the timelock resets."
        action="revoke_approval"
        details={[["Proposal", `#${id}`], ["Approver", live.address]]}
        confirmLabel="Submit revocation to Testnet"
        disabledReason={reasonOf(lifecycle.actions.revoke)}
        build={(address) => buildRevokeApproval(address, id)}
        onConfirmed={afterConfirmed}
      />
      <GovernanceActionPanel
        title="Execute proposal"
        description="Applies the approved change on-chain. The contract only accepts this once the threshold is met and the timelock has elapsed, and it rechecks the expected current WASM at execution."
        action="execute_proposal"
        details={[["Proposal", `#${id}`], ["Executor", live.address]]}
        confirmLabel="Submit execution to Testnet"
        disabledReason={reasonOf(lifecycle.actions.execute)}
        build={(address) => buildExecuteProposal(address, id)}
        onConfirmed={afterConfirmed}
      />
      <GovernanceActionPanel
        title="Cancel proposal"
        description="Only the proposer can cancel, and only before the approval threshold is reached."
        action="cancel_proposal"
        details={[["Proposal", `#${id}`], ["Proposer", live.live.proposer]]}
        confirmLabel="Submit cancellation to Testnet"
        disabledReason={reasonOf(lifecycle.actions.cancel)}
        build={(address) => buildCancelProposal(address, id)}
        onConfirmed={afterConfirmed}
      />
    </div>
  );
}
