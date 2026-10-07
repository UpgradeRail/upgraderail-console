import { DataList } from "@/components/data-list";

export default function AnalysesPage() {
  return (
    <section className="console-page">
      <header className="console-heading">
        <div>
          <span className="eyebrow">PREFLIGHT</span>
          <h1>Analyses</h1>
          <p>Engine comparison jobs persisted by the worker. Open one to review findings and evidence.</p>
        </div>
      </header>
      <DataList
        endpoint="/api/v1/analyses"
        empty="No analyses exist"
        columns={[
          { key: "id", label: "Analysis" },
          { key: "status", label: "Status" },
          { key: "network", label: "Network" },
          { key: "created_at", label: "Created" },
        ]}
        linkKey="id"
        linkBase="/app/analyses/"
      />
    </section>
  );
}
