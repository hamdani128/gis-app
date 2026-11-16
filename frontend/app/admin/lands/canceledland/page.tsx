import CompensatedLandsView from "../compensatedlands/compensated-lands-view";

export default function CanceledLandsPage() {
  return <CompensatedLandsView title="Canceled Lands" detailName="Canceled Land" apiPath="/api/canceled-lands" readOnly />;
}
