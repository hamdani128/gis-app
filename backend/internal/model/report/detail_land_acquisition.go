package report

import "context"

func (m *ReportModel) DetailLandAcquisition(ctx context.Context, companyID int64, filter ReportFilter) (ReportPage, error) {
	return m.list(ctx, companyID, filter)
}
