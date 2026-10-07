"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { SiteHeader } from "@/components/site-header";
import { decodeRouteParam } from "@/lib/route-param";
import { api, ApiError } from "@/lib/api";

interface Proposal {
  id: string;
  controller_id: string;
  proposal_id: number;
  proposer: string;
  kind?: string | null;
  governance_epoch: number;
  created_ledger: number;
  expires_ledger: number;
  approval_count: number;
  approved_ledger?: number | null;
  execute_after_ledger?: number | null;
  status: string;
  manifest_hash?: string | null;
  created_transaction_hash: string;
}

interface Approval {
  id: string;
  proposal_id: string;
  approver: string;
  approved_ledger: number;
  transaction_hash: string;
  revoked_ledger?: number | null;
}

type State =
  | { kind: "loading" }
  | { kind: "not_found" }
  | { kind: "error"; message: string }
  | { kind: "ready"; proposal: Proposal; approvals: Approval[] };

export default function ExploreProposalPage({
  proposalId: directProposalId,
  params,
}: {
  proposalId?: string;
  params?: Promise<{ proposalId: string }>;
} = {}) {
  const unwrappedParams = params ? use(params) : undefined;
  const proposalId = decodeRouteParam(directProposalId ?? unwrappedParams?.proposalId ?? "");

  const [fetched, setState] = useState<State>({ kind: "loading" });
  const state: State = proposalId ? fetched : { kind: "not_found" };

  useEffect(() => {
    if (!proposalId) return;

    let active = true;

    async function load() {
      try {
        const [proposal, approvals] = await Promise.all([
          api<Proposal>(`/api/v1/proposals/${encodeURIComponent(proposalId)}`),
          api<Approval[]>(`/api/v1/proposals/${encodeURIComponent(proposalId)}/approvals`).catch(() => [] as Approval[]),
        ]);

        if (active) setState({ kind: "ready", proposal, approvals });
      } catch (err) {
        if (active) {
          if (err instanceof ApiError && err.status === 404) {
            setState({ kind: "not_found" });
          } else {
            setState({
              kind: "error",
              message: err instanceof Error ? err.message : "The proposal could not be loaded.",
            });
          }
        }
      }
    }

    void load();
    return () => {
      active = false;
    };
  }, [proposalId]);

  return (
    <>
      <SiteHeader />
      <main className="explore-layout">
        <header>
          <span className="eyebrow">PUBLIC PROPOSAL EXPLORER</span>
          <h1>{state.kind === "ready" ? `Proposal #${state.proposal.proposal_id}` : "Proposal detail"}</h1>
          <p className="data-note" style={{ marginTop: "12px" }}>
            This explorer reads only from the Console indexer. It does not present sample proposals or simulated activity.
          </p>
        </header>

        {state.kind === "loading" && (
          <section className="empty-state">
            <span className="eyebrow">LOADING</span>
            <h2>Reading proposal projection</h2>
            <p>Querying the indexer for proposal state and approvals.</p>
          </section>
        )}

        {state.kind === "not_found" && (
          <section className="empty-state">
            <span className="eyebrow">NOT FOUND</span>
            <h2>Proposal not found</h2>
            <p>No indexed proposal matches the specified identifier: {proposalId || "unspecified"}</p>
            <Link href="/explore" className="text-link">
              Return to Explorer
            </Link>
          </section>
        )}

        {state.kind === "error" && (
          <section className="empty-state">
            <span className="eyebrow">ERROR</span>
            <h2>Failed to load proposal</h2>
            <p>{state.message}</p>
          </section>
        )}

        {state.kind === "ready" && (
          <div style={{ display: "grid", gap: "32px", marginTop: "32px" }}>
            <div className="verification-card" style={{ marginTop: 0 }}>
              <span className="eyebrow">PROPOSAL STATE</span>
              <h2>Lifecycle</h2>
              <dl className="verification-summary">
                <div>
                  <dt>Proposal ID</dt>
                  <dd>#{state.proposal.proposal_id}</dd>
                </div>
                <div>
                  <dt>Status</dt>
                  <dd style={{ textTransform: "capitalize" }}>{state.proposal.status}</dd>
                </div>
                <div>
                  <dt>Kind</dt>
                  <dd>{state.proposal.kind ?? "—"}</dd>
                </div>
                <div>
                  <dt>Proposer</dt>
                  <dd>{state.proposal.proposer}</dd>
                </div>
                <div>
                  <dt>Approvals</dt>
                  <dd>{state.proposal.approval_count}</dd>
                </div>
                <div>
                  <dt>Manifest Hash</dt>
                  <dd>{state.proposal.manifest_hash ?? "—"}</dd>
                </div>
                <div>
                  <dt>Created Ledger</dt>
                  <dd>{state.proposal.created_ledger}</dd>
                </div>
                <div>
                  <dt>Expires Ledger</dt>
                  <dd>{state.proposal.expires_ledger}</dd>
                </div>
                <div>
                  <dt>Timelock</dt>
                  <dd>
                    {state.proposal.execute_after_ledger
                      ? `Executable after ledger ${state.proposal.execute_after_ledger}`
                      : "Awaiting threshold"}
                  </dd>
                </div>
              </dl>
            </div>

            <div>
              <span className="eyebrow">APPROVALS</span>
              <h2 style={{ margin: "4px 0 16px", fontSize: "24px" }}>
                Recorded Approvals ({state.approvals.length})
              </h2>
              {state.approvals.length === 0 ? (
                <section className="empty-state">
                  <span className="eyebrow">NO APPROVALS</span>
                  <h2>No approvals recorded</h2>
                  <p>No approvers have registered approval on-chain yet.</p>
                </section>
              ) : (
                <div className="data-table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Approver</th>
                        <th>Approved Ledger</th>
                        <th>Status</th>
                        <th>Tx Hash</th>
                      </tr>
                    </thead>
                    <tbody>
                      {state.approvals.map((a) => (
                        <tr key={a.id}>
                          <td title={a.approver}>{a.approver}</td>
                          <td>{a.approved_ledger}</td>
                          <td>{a.revoked_ledger ? "Revoked" : "Active"}</td>
                          <td title={a.transaction_hash}>{a.transaction_hash}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        )}
      </main>
    </>
  );
}
