export const acquisitionStages = [
  [1, "Submit Memo Land"],
  [9, "RM Approval"],
  [8, "GM Approval"],
  [2, "GIS Approval"],
  [3, "Legal Approval"],
  [10, "Operation Control Approval"],
  [5, "PD Approval"],
  [6, "Payment"],
  [7, "Payment to Owner"],
] as const;

export default function AcquisitionProcessOverview({ history }: { history: Array<{ sequence_id: number | null }> }) {
  const recorded = new Set(history.map((item) => Number(item.sequence_id)));
  const count = acquisitionStages.filter(([id]) => recorded.has(id)).length;
  const total = acquisitionStages.length;

  return <div className="compensated-process-overview">
    <div className="compensated-process-overview-heading"><div><span>ALUR AKUISISI</span><h3>Process</h3></div><strong>{count}/{total}</strong></div>
    <p>{count} dari {total} tahap memiliki catatan</p>
    <div className="compensated-process-progress" role="progressbar" aria-label="Tahap yang memiliki catatan proses" aria-valuemin={0} aria-valuemax={total} aria-valuenow={count}><span style={{ width: `${count / total * 100}%` }} /></div>
  </div>;
}
