import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { JobStatus } from "./create-analysis-from-uploads";
import type { AnalysisJob } from "@/lib/analysis";

const job = (status: string, error = ""): AnalysisJob => ({
  id: "job1",
  network: "testnet",
  status,
  engine_version: "",
  error_message: error,
  current_artifact_id: "a",
  candidate_artifact_id: "b",
  created_by: null,
  created_at: "",
  started_at: null,
  finished_at: null,
});

const render = (status: string, error = "") => renderToStaticMarkup(React.createElement(JobStatus, { id: "job1", job: job(status, error) }));

describe("JobStatus", () => {
  it("shows the exact status the API reported, never a fabricated label", () => {
    for (const status of ["queued", "running", "ready", "blocked", "failed", "cancelled"]) {
      expect(render(status)).toContain(`Status: ${status.toUpperCase()}`);
    }
  });

  it("links a ready job to its preflight analysis", () => {
    const html = render("ready");
    expect(html).toContain('href="/app/analyses/job1"');
    expect(html).toContain("View the preflight analysis");
  });

  it("links a blocked job to its preflight analysis instead of hiding the result", () => {
    const html = render("blocked");
    expect(html).toContain('href="/app/analyses/job1"');
  });

  it("shows the real error message for a failed job, not a generic one", () => {
    const html = render("failed", "engine exited: exit status 1");
    expect(html).toContain("engine exited: exit status 1");
  });

  it("never renders a score, percentage, or progress indicator for a non-terminal job", () => {
    for (const status of ["queued", "running"]) {
      const html = render(status);
      expect(html).not.toMatch(/score|%|progress/i);
      expect(html).not.toContain('href="/app/analyses/job1"');
    }
  });
});
