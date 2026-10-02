const phases = ["Analyzed", "Simulated", "Approved", "Ready"];

export function UpgradeRailDiagram() {
  return <section className="rail-diagram" aria-label="UpgradeRail process">
    <div className="rail-head"><span>Your contracts</span><span>Candidate executable</span></div>
    <div className="rail-track">
      <div className="contracts"><span>Payment vault</span><span>Settlement engine</span><span>Treasury router</span></div>
      <div className="rail-gate"><span className="gate-mark">↔</span><strong>UpgradeRail</strong><small>governed release evidence</small></div>
      <div className="candidate"><span>candidate WASM</span><small>only after on-chain approval</small></div>
    </div>
    <ol className="phase-list">{phases.map((phase, index) => <li key={phase}><b>{String(index + 1).padStart(2, "0")}</b>{phase}</li>)}</ol>
  </section>;
}
