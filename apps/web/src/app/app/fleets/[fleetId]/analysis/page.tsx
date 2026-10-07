import AnalysesPage from "@/app/app/analyses/page";

// Analyses are not yet linked to a fleet in the data model, so this route
// shows the shared analysis list rather than inventing a per-fleet filter.
export default function FleetAnalysisPage() {
  return <AnalysesPage />;
}
