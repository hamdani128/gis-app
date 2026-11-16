package routes

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"

	acquisitionController "gis-app/backend/internal/controller/acquisition"
	adminController "gis-app/backend/internal/controller/administrator"
	authController "gis-app/backend/internal/controller/auth"
	companyController "gis-app/backend/internal/controller/companies"
	dashboardController "gis-app/backend/internal/controller/dashboard"
	landController "gis-app/backend/internal/controller/lands"
	mappingController "gis-app/backend/internal/controller/mapping"
	reportController "gis-app/backend/internal/controller/report"
	acquisitionModel "gis-app/backend/internal/model/acquisition"
	adminModel "gis-app/backend/internal/model/administrator"
	authModel "gis-app/backend/internal/model/auth"
	companyModel "gis-app/backend/internal/model/companies"
	dashboardModel "gis-app/backend/internal/model/dashboard"
	landModel "gis-app/backend/internal/model/lands"
	mappingModel "gis-app/backend/internal/model/mapping"
	reportModel "gis-app/backend/internal/model/report"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Setup(db *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()

	refController := &adminController.RefValueSettingController{
		Model: &adminModel.RefValueSettingModel{
			DB: db,
		},
	}
	docTypeController := &adminController.DocTypeController{
		Model: &adminModel.DocTypeModel{
			DB: db,
		},
	}
	officerController := &adminController.OfficerController{
		Model: &adminModel.OfficerModel{DB: db},
	}
	businessProcessRoleController := &adminController.BusinessProcessRoleController{
		Model: &adminModel.BusinessProcessRoleModel{DB: db},
	}
	businessProcessSequenceController := &adminController.BusinessProcessSequenceController{
		Model: &adminModel.BusinessProcessSequenceModel{DB: db},
	}
	businessProcessOfficerController := &adminController.BusinessProcessOfficerController{
		Model: &adminModel.BusinessProcessOfficerModel{DB: db},
	}
	moduleController := &adminController.ModuleController{
		Model: &adminModel.ModuleModel{DB: db},
	}
	regionController := &companyController.RegionController{
		Model: &companyModel.RegionModel{DB: db},
	}
	companyInfoController := &companyController.CompanyInfoController{
		Model: &companyModel.CompanyInfoModel{DB: db},
	}
	landPermitController := &companyController.LandPermitController{
		Model: &companyModel.LandPermitModel{DB: db},
	}
	hcvLandController := &landController.HCVLandController{
		Model: &landModel.HCVLandModel{DB: db},
	}
	potentialLandController := &landController.PotentialLandController{
		Model: &landModel.PotentialLandModel{DB: db},
	}
	divisionLandController := &landController.DivisionLandController{
		Model: &landModel.DivisionLandModel{DB: db},
	}
	proposeLandController := &landController.ProposeLandController{Model: &landModel.ProposeLandModel{DB: db}}
	compensatedLandController := &landController.CompensatedLandController{Model: &landModel.CompensatedLandModel{DB: db, Status: 10}}
	newCompensatedLandController := &landController.CompensatedLandController{Model: &landModel.CompensatedLandModel{DB: db, Status: 8}}
	newCompensatedHistoryController := &acquisitionController.CompensatedHistoryController{Model: &acquisitionModel.SubmitMemoLandModel{DB: db}, LandModel: newCompensatedLandController.Model}
	canceledLandController := &landController.CompensatedLandController{Model: &landModel.CompensatedLandModel{DB: db, Status: 99}}
	landOwnerController := &landController.LandOwnerController{Model: &landModel.LandOwnerModel{DB: db}}
	ownerHistoryController := &acquisitionController.CompensatedHistoryController{Model: &acquisitionModel.SubmitMemoLandModel{DB: db}, OwnerModel: landOwnerController.Model}
	memoController := &acquisitionController.MemoController{Model: &acquisitionModel.MemoModel{DB: db}}
	submitMemoLandController := &acquisitionController.SubmitMemoLandController{Model: &acquisitionModel.SubmitMemoLandModel{DB: db}}
	rmApprovalController := &acquisitionController.RMApprovalController{Model: &acquisitionModel.SubmitMemoLandModel{DB: db}}
	gmApprovalController := &acquisitionController.GMApprovalController{Model: &acquisitionModel.SubmitMemoLandModel{DB: db}}
	gisApprovalController := &acquisitionController.ParallelApprovalController{Model: &acquisitionModel.SubmitMemoLandModel{DB: db}, Stage: acquisitionModel.GISApprovalStage}
	legalApprovalController := &acquisitionController.ParallelApprovalController{Model: &acquisitionModel.SubmitMemoLandModel{DB: db}, Stage: acquisitionModel.LegalApprovalStage}
	operationControlApprovalController := &acquisitionController.ParallelApprovalController{Model: &acquisitionModel.SubmitMemoLandModel{DB: db}, Stage: acquisitionModel.OperationControlStage}
	pdApprovalController := &acquisitionController.ParallelApprovalController{Model: &acquisitionModel.SubmitMemoLandModel{DB: db}, Stage: acquisitionModel.PDApprovalStage}
	paymentController := &acquisitionController.PaymentController{Model: &acquisitionModel.SubmitMemoLandModel{DB: db}}
	paymentToOwnerController := &acquisitionController.PaymentToOwnerController{Model: &acquisitionModel.SubmitMemoLandModel{DB: db}}
	reports := &reportController.ReportController{
		Model:     &reportModel.ReportModel{DB: db},
		Documents: &acquisitionModel.SubmitMemoLandModel{DB: db},
		Access:    &authModel.UserModel{DB: db},
	}
	auth := &authController.Controller{
		Model: &authModel.UserModel{DB: db},
	}
	dashboard := &dashboardController.Controller{Model: &dashboardModel.Model{DB: db}}
	mux.HandleFunc("GET /api/dashboard/summary", auth.RequireCompany(dashboard.Summary))
	mux.HandleFunc("GET /api/reports/{code}", auth.RequireCompany(reports.Index))
	mux.HandleFunc("GET /api/reports/{code}/period", auth.RequireCompany(reports.AcquisitionPeriod))
	mux.HandleFunc("GET /api/reports/{code}/summary", auth.RequireCompany(reports.AcquisitionProgressSummary))
	mux.HandleFunc("GET /api/reports/rpt_all_progress/summary", auth.RequireCompany(reports.AllEstateProgressSummary))
	mux.HandleFunc("GET /api/reports/aestategrttreport/summary", auth.RequireCompany(reports.EstateProgressReportSummary))
	mux.HandleFunc("GET /api/reports/plantmaprep/{id}/planting-map", auth.RequireCompany(reports.PlantingMap))
	mux.HandleFunc("GET /api/reports/{code}/{id}/history", auth.RequireCompany(reports.History))
	mux.HandleFunc("GET /api/reports/{code}/{id}/map", auth.RequireCompany(reports.Map))
	mux.HandleFunc("GET /api/reports/{code}/{id}/supported-documents", auth.RequireCompany(reports.SupportedDocuments))
	mux.HandleFunc("GET /api/reports/{code}/{id}/supported-documents/{doc_id}", auth.RequireCompany(reports.SupportedDocumentFile))
	mux.HandleFunc("GET /api/reports/{code}/{id}/supported-documents/{doc_id}/legacy-status", auth.RequireCompany(reports.SupportedDocumentLegacyStatus))
	mux.HandleFunc("GET /api/reports/{code}/{id}/memo", auth.RequireCompany(reports.Memo))
	mux.HandleFunc("GET /api/reports/{code}/{id}/memo/legacy-status", auth.RequireCompany(reports.MemoLegacyStatus))
	mux.HandleFunc("GET /api/reports/{code}/{id}/memo/document", auth.RequireCompany(reports.MemoFile))
	mux.HandleFunc("GET /api/reports/{code}/{id}/local-files", auth.RequireCompany(reports.LocalHistoryFiles))
	mux.HandleFunc("GET /api/reports/{code}/{id}/local-files/{name}", auth.RequireCompany(reports.LocalHistoryFile))

	mux.HandleFunc("GET /api/ref-value-settings", refController.Index)
	mux.HandleFunc("GET /api/ref-value-settings/{id}", refController.Show)
	mux.HandleFunc("POST /api/ref-value-settings", refController.Store)
	mux.HandleFunc("PUT /api/ref-value-settings/{id}", refController.Update)
	mux.HandleFunc("DELETE /api/ref-value-settings/{id}", refController.Destroy)
	mux.HandleFunc("GET /api/doc-types", docTypeController.Index)
	mux.HandleFunc("GET /api/doc-types/{id}", docTypeController.Show)
	mux.HandleFunc("POST /api/doc-types", docTypeController.Store)
	mux.HandleFunc("PUT /api/doc-types/{id}", docTypeController.Update)
	mux.HandleFunc("DELETE /api/doc-types/{id}", docTypeController.Destroy)
	mux.HandleFunc("GET /api/officers", officerController.Index)
	mux.HandleFunc("GET /api/officers/options", officerController.Options)
	mux.HandleFunc("GET /api/officers/{id}", officerController.Show)
	mux.HandleFunc("POST /api/officers", officerController.Store)
	mux.HandleFunc("PUT /api/officers/{id}", officerController.Update)
	mux.HandleFunc("DELETE /api/officers/{id}", officerController.Destroy)
	mux.HandleFunc("PUT /api/officers/{id}/password", officerController.SetPassword)
	mux.HandleFunc("GET /api/officers/{id}/privileges", officerController.Privileges)
	mux.HandleFunc("PUT /api/officers/{id}/privileges", officerController.UpdatePrivileges)
	mux.HandleFunc("GET /api/officers/{id}/logs", officerController.Logs)
	mux.HandleFunc("GET /api/officer-company-relations", officerController.EmployeeCompanies)
	mux.HandleFunc("GET /api/officer-company-relations/{id}", officerController.ShowEmployeeCompany)
	mux.HandleFunc("POST /api/officer-company-relations", officerController.StoreEmployeeCompany)
	mux.HandleFunc("PUT /api/officer-company-relations/{id}", officerController.UpdateEmployeeCompany)
	mux.HandleFunc("DELETE /api/officer-company-relations/{id}", officerController.DestroyEmployeeCompany)
	mux.HandleFunc("GET /api/business-process-roles", businessProcessRoleController.Index)
	mux.HandleFunc("GET /api/business-process-roles/{id}", businessProcessRoleController.Show)
	mux.HandleFunc("POST /api/business-process-roles", businessProcessRoleController.Store)
	mux.HandleFunc("PUT /api/business-process-roles/{id}", businessProcessRoleController.Update)
	mux.HandleFunc("DELETE /api/business-process-roles/{id}", businessProcessRoleController.Destroy)
	mux.HandleFunc("GET /api/business-process-sequences", businessProcessSequenceController.Index)
	mux.HandleFunc("GET /api/business-process-sequences/{id}", businessProcessSequenceController.Show)
	mux.HandleFunc("POST /api/business-process-sequences", businessProcessSequenceController.Store)
	mux.HandleFunc("PUT /api/business-process-sequences/{id}", businessProcessSequenceController.Update)
	mux.HandleFunc("DELETE /api/business-process-sequences/{id}", businessProcessSequenceController.Destroy)
	mux.HandleFunc("GET /api/business-process-officers", businessProcessOfficerController.Index)
	mux.HandleFunc("GET /api/business-process-officers/{id}", businessProcessOfficerController.Show)
	mux.HandleFunc("POST /api/business-process-officers", businessProcessOfficerController.Store)
	mux.HandleFunc("PUT /api/business-process-officers/{id}", businessProcessOfficerController.Update)
	mux.HandleFunc("DELETE /api/business-process-officers/{id}", businessProcessOfficerController.Destroy)
	mux.HandleFunc("GET /api/modules", moduleController.Index)
	mux.HandleFunc("GET /api/modules/options", moduleController.Options)
	mux.HandleFunc("GET /api/modules/{id}", moduleController.Show)
	mux.HandleFunc("POST /api/modules", moduleController.Store)
	mux.HandleFunc("PUT /api/modules/{id}", moduleController.Update)
	mux.HandleFunc("DELETE /api/modules/{id}", moduleController.Destroy)
	mux.HandleFunc("GET /api/regions", regionController.Index)
	mux.HandleFunc("GET /api/regions/employees", regionController.Employees)
	mux.HandleFunc("GET /api/regions/{id}", regionController.Show)
	mux.HandleFunc("POST /api/regions", regionController.Store)
	mux.HandleFunc("PUT /api/regions/{id}", regionController.Update)
	mux.HandleFunc("DELETE /api/regions/{id}", regionController.Destroy)
	mux.HandleFunc("GET /api/company-info", companyInfoController.Index)
	mux.HandleFunc("GET /api/company-info/options", companyInfoController.Options)
	mux.HandleFunc("GET /api/company-info/{id}", companyInfoController.Show)
	mux.HandleFunc("POST /api/company-info", companyInfoController.Store)
	mux.HandleFunc("PUT /api/company-info/{id}", companyInfoController.Update)
	mux.HandleFunc("DELETE /api/company-info/{id}", companyInfoController.Destroy)
	mux.HandleFunc("GET /api/land-permits/options", landPermitController.Options)
	mux.HandleFunc("GET /api/land-permits", landPermitController.Index)
	mux.HandleFunc("POST /api/land-permits", landPermitController.Store)
	mux.HandleFunc("PUT /api/land-permits/{id}", landPermitController.Update)
	mux.HandleFunc("DELETE /api/land-permits/{id}", landPermitController.Destroy)
	mux.HandleFunc("GET /api/hcv-lands/options", hcvLandController.Options)
	mux.HandleFunc("GET /api/hcv-lands", hcvLandController.Index)
	mux.HandleFunc("GET /api/hcv-lands/{id}/map", hcvLandController.Map)
	mux.HandleFunc("GET /api/hcv-lands/{id}/map-upload", hcvLandController.MapUpload)
	mux.HandleFunc("PUT /api/hcv-lands/{id}/map", hcvLandController.SaveMap)
	mux.HandleFunc("POST /api/hcv-lands", hcvLandController.Store)
	mux.HandleFunc("PUT /api/hcv-lands/{id}", hcvLandController.Update)
	mux.HandleFunc("DELETE /api/hcv-lands/{id}", hcvLandController.Destroy)
	mux.HandleFunc("GET /api/potential-lands/options", potentialLandController.Options)
	mux.HandleFunc("GET /api/potential-lands", potentialLandController.Index)
	mux.HandleFunc("GET /api/potential-lands/{id}/map", potentialLandController.Map)
	mux.HandleFunc("GET /api/potential-lands/{id}/map-upload", potentialLandController.MapUpload)
	mux.HandleFunc("PUT /api/potential-lands/{id}/map", potentialLandController.SaveMap)
	mux.HandleFunc("POST /api/potential-lands", potentialLandController.Store)
	mux.HandleFunc("PUT /api/potential-lands/{id}", potentialLandController.Update)
	mux.HandleFunc("DELETE /api/potential-lands/{id}", potentialLandController.Destroy)
	mux.HandleFunc("GET /api/division-lands/options", divisionLandController.Options)
	mux.HandleFunc("GET /api/division-lands", divisionLandController.Index)
	mux.HandleFunc("GET /api/division-lands/{id}/map", divisionLandController.Map)
	mux.HandleFunc("GET /api/division-lands/{id}/map-upload", divisionLandController.MapUpload)
	mux.HandleFunc("PUT /api/division-lands/{id}/map", divisionLandController.SaveMap)
	mux.HandleFunc("POST /api/division-lands", divisionLandController.Store)
	mux.HandleFunc("PUT /api/division-lands/{id}", divisionLandController.Update)
	mux.HandleFunc("DELETE /api/division-lands/{id}", divisionLandController.Destroy)
	mux.HandleFunc("GET /api/propose-lands/permissions", auth.RequireModule("proposelands", "view", auth.ProposeLandPermissions))
	mux.HandleFunc("GET /api/propose-lands/options", auth.RequireModule("proposelands", "view", proposeLandController.Options))
	mux.HandleFunc("GET /api/propose-lands", auth.RequireModule("proposelands", "view", proposeLandController.Index))
	mux.HandleFunc("POST /api/propose-lands", auth.RequireModule("proposelands", "add", proposeLandController.Store))
	mux.HandleFunc("PUT /api/propose-lands/{id}", auth.RequireModule("proposelands", "edit", proposeLandController.Update))
	mux.HandleFunc("DELETE /api/propose-lands/{id}", auth.RequireModule("proposelands", "delete", proposeLandController.Destroy))
	mux.HandleFunc("GET /api/compensated-lands/permissions", auth.RequireModule("compensatedlands", "view", auth.ModulePermissions))
	mux.HandleFunc("GET /api/compensated-lands/options", auth.RequireModule("compensatedlands", "view", compensatedLandController.Options))
	mux.HandleFunc("GET /api/compensated-lands", auth.RequireModule("compensatedlands", "view", compensatedLandController.Index))
	mux.HandleFunc("GET /api/compensated-lands/{id}/processes", auth.RequireModule("compensatedlands", "view", compensatedLandController.Processes))
	mux.HandleFunc("GET /api/compensated-lands/{id}/map", auth.RequireModule("compensatedlands", "view", compensatedLandController.Map))
	mux.HandleFunc("PUT /api/compensated-lands/{id}/map", auth.RequireModule("compensatedlands", "edit", compensatedLandController.SaveMap))
	mux.HandleFunc("PUT /api/compensated-lands/{id}", auth.RequireModule("compensatedlands", "edit", compensatedLandController.Update))
	mux.HandleFunc("GET /api/new-compensated-lands/permissions", auth.RequireModule("newcompensatedlands", "view", auth.ModulePermissions))
	mux.HandleFunc("GET /api/new-compensated-lands/options", auth.RequireModule("newcompensatedlands", "view", newCompensatedLandController.Options))
	mux.HandleFunc("GET /api/new-compensated-lands", auth.RequireModule("newcompensatedlands", "view", newCompensatedLandController.Index))
	mux.HandleFunc("GET /api/new-compensated-lands/{id}/processes", auth.RequireModule("newcompensatedlands", "view", newCompensatedLandController.Processes))
	mux.HandleFunc("GET /api/new-compensated-lands/{id}/map", auth.RequireModule("newcompensatedlands", "view", newCompensatedLandController.Map))
	mux.HandleFunc("GET /api/new-compensated-lands/{id}/supported-documents", auth.RequireModule("newcompensatedlands", "view", newCompensatedHistoryController.SupportedDocuments))
	mux.HandleFunc("GET /api/new-compensated-lands/{id}/supported-documents/{doc_id}", auth.RequireModule("newcompensatedlands", "view", newCompensatedHistoryController.SupportedDocumentFile))
	mux.HandleFunc("GET /api/new-compensated-lands/{id}/supported-documents/{doc_id}/legacy-status", auth.RequireModule("newcompensatedlands", "view", newCompensatedHistoryController.SupportedDocumentLegacyStatus))
	mux.HandleFunc("GET /api/new-compensated-lands/{id}/memo", auth.RequireModule("newcompensatedlands", "view", newCompensatedHistoryController.Memo))
	mux.HandleFunc("GET /api/new-compensated-lands/{id}/memo/document", auth.RequireModule("newcompensatedlands", "view", newCompensatedHistoryController.MemoFile))
	mux.HandleFunc("GET /api/new-compensated-lands/{id}/local-files", auth.RequireModule("newcompensatedlands", "view", newCompensatedHistoryController.LocalHistoryFiles))
	mux.HandleFunc("GET /api/new-compensated-lands/{id}/local-files/{name}", auth.RequireModule("newcompensatedlands", "view", newCompensatedHistoryController.LocalHistoryFile))
	mux.HandleFunc("GET /api/canceled-lands/permissions", auth.RequireModule("canceledland", "view", auth.ModulePermissions))
	mux.HandleFunc("GET /api/canceled-lands/options", auth.RequireModule("canceledland", "view", canceledLandController.Options))
	mux.HandleFunc("GET /api/canceled-lands", auth.RequireModule("canceledland", "view", canceledLandController.Index))
	mux.HandleFunc("GET /api/canceled-lands/{id}/processes", auth.RequireModule("canceledland", "view", canceledLandController.Processes))
	mux.HandleFunc("GET /api/canceled-lands/{id}/map", auth.RequireModule("canceledland", "view", canceledLandController.Map))
	mux.HandleFunc("GET /api/land-owners/permissions", auth.RequireModule("landowner", "view", auth.ModulePermissions))
	mux.HandleFunc("GET /api/land-owners/options", auth.RequireModule("landowner", "view", landOwnerController.Options))
	mux.HandleFunc("GET /api/land-owners", auth.RequireModule("landowner", "view", landOwnerController.Index))
	mux.HandleFunc("GET /api/land-owners/{id}/lands", auth.RequireModule("landowner", "view", landOwnerController.Lands))
	mux.HandleFunc("GET /api/land-owners/{id}/lands/{land_id}", auth.RequireModule("landowner", "view", landOwnerController.LandDetail))
	mux.HandleFunc("GET /api/land-owners/{id}/lands/{land_id}/processes", auth.RequireModule("landowner", "view", landOwnerController.LandProcesses))
	mux.HandleFunc("GET /api/land-owners/{id}/lands/{land_id}/map", auth.RequireModule("landowner", "view", landOwnerController.LandMap))
	mux.HandleFunc("GET /api/land-owners/{id}/lands/{land_id}/supported-documents", auth.RequireModule("landowner", "view", ownerHistoryController.SupportedDocuments))
	mux.HandleFunc("GET /api/land-owners/{id}/lands/{land_id}/supported-documents/{doc_id}", auth.RequireModule("landowner", "view", ownerHistoryController.SupportedDocumentFile))
	mux.HandleFunc("GET /api/land-owners/{id}/lands/{land_id}/supported-documents/{doc_id}/legacy-status", auth.RequireModule("landowner", "view", ownerHistoryController.SupportedDocumentLegacyStatus))
	mux.HandleFunc("GET /api/land-owners/{id}/lands/{land_id}/memo", auth.RequireModule("landowner", "view", ownerHistoryController.Memo))
	mux.HandleFunc("GET /api/land-owners/{id}/lands/{land_id}/memo/document", auth.RequireModule("landowner", "view", ownerHistoryController.MemoFile))
	mux.HandleFunc("GET /api/land-owners/{id}/lands/{land_id}/local-files", auth.RequireModule("landowner", "view", ownerHistoryController.LocalHistoryFiles))
	mux.HandleFunc("GET /api/land-owners/{id}/lands/{land_id}/local-files/{name}", auth.RequireModule("landowner", "view", ownerHistoryController.LocalHistoryFile))
	mux.HandleFunc("POST /api/land-owners", auth.RequireModule("landowner", "add", landOwnerController.Store))
	mux.HandleFunc("PUT /api/land-owners/{id}", auth.RequireModule("landowner", "edit", landOwnerController.Update))
	mux.HandleFunc("DELETE /api/land-owners/{id}", auth.RequireModule("landowner", "delete", landOwnerController.Destroy))
	mux.HandleFunc("GET /api/acquisition-memos/permissions", auth.RequireModule("acquisitionRequest", "view", auth.ModulePermissions))
	mux.HandleFunc("GET /api/acquisition-memos", auth.RequireModule("acquisitionRequest", "view", memoController.Index))
	mux.HandleFunc("GET /api/acquisition-memos/{id}", auth.RequireModule("acquisitionRequest", "view", memoController.Show))
	mux.HandleFunc("GET /api/acquisition-memos/{id}/lands", auth.RequireModule("acquisitionRequest", "view", memoController.Lands))
	mux.HandleFunc("GET /api/acquisition-memos/{id}/document", auth.RequireModule("acquisitionRequest", "view", memoController.Document))
	mux.HandleFunc("POST /api/acquisition-memos", auth.RequireModule("acquisitionRequest", "add", memoController.Store))
	mux.HandleFunc("PUT /api/acquisition-memos/{id}", auth.RequireModule("acquisitionRequest", "edit", memoController.Update))
	mux.HandleFunc("DELETE /api/acquisition-memos/{id}", auth.RequireModule("acquisitionRequest", "delete", memoController.Destroy))
	mux.HandleFunc("GET /api/submit-memo-land", auth.RequireAcquisitionCompany(submitMemoLandController.Index))
	mux.HandleFunc("GET /api/submit-memo-land/{id}", auth.RequireAcquisitionCompany(submitMemoLandController.Show))
	mux.HandleFunc("GET /api/submit-memo-land/{id}/map", auth.RequireAcquisitionCompany(submitMemoLandController.Map))
	mux.HandleFunc("POST /api/submit-memo-land/{id}", auth.RequireAcquisitionCompany(submitMemoLandController.Store))
	mux.HandleFunc("GET /api/rm-approval", auth.RequireAcquisitionCompany(rmApprovalController.Index))
	mux.HandleFunc("GET /api/rm-approval/{id}", auth.RequireAcquisitionCompany(rmApprovalController.Show))
	mux.HandleFunc("GET /api/rm-approval/{id}/map", auth.RequireAcquisitionCompany(rmApprovalController.Map))
	mux.HandleFunc("POST /api/rm-approval/{id}", auth.RequireAcquisitionCompany(rmApprovalController.Store))
	mux.HandleFunc("GET /api/gm-approval", auth.RequireAcquisitionCompany(gmApprovalController.Index))
	mux.HandleFunc("GET /api/gm-approval/{id}", auth.RequireAcquisitionCompany(gmApprovalController.Show))
	mux.HandleFunc("GET /api/gm-approval/{id}/map", auth.RequireAcquisitionCompany(gmApprovalController.Map))
	mux.HandleFunc("POST /api/gm-approval/{id}", auth.RequireAcquisitionCompany(gmApprovalController.Store))
	mux.HandleFunc("GET /api/gis-approval", auth.RequireAcquisitionCompany(gisApprovalController.Index))
	mux.HandleFunc("GET /api/gis-approval/{id}", auth.RequireAcquisitionCompany(gisApprovalController.Show))
	mux.HandleFunc("GET /api/gis-approval/{id}/map", auth.RequireAcquisitionCompany(gisApprovalController.Map))
	mux.HandleFunc("POST /api/gis-approval/{id}", auth.RequireAcquisitionCompany(gisApprovalController.Store))
	mux.HandleFunc("GET /api/legal-approval", auth.RequireAcquisitionCompany(legalApprovalController.Index))
	mux.HandleFunc("GET /api/legal-approval/{id}", auth.RequireAcquisitionCompany(legalApprovalController.Show))
	mux.HandleFunc("GET /api/legal-approval/{id}/map", auth.RequireAcquisitionCompany(legalApprovalController.Map))
	mux.HandleFunc("POST /api/legal-approval/{id}", auth.RequireAcquisitionCompany(legalApprovalController.Store))
	mux.HandleFunc("GET /api/operation-control-approval", auth.RequireAcquisitionCompany(operationControlApprovalController.Index))
	mux.HandleFunc("GET /api/operation-control-approval/{id}", auth.RequireAcquisitionCompany(operationControlApprovalController.Show))
	mux.HandleFunc("GET /api/operation-control-approval/{id}/map", auth.RequireAcquisitionCompany(operationControlApprovalController.Map))
	mux.HandleFunc("POST /api/operation-control-approval/{id}", auth.RequireAcquisitionCompany(operationControlApprovalController.Store))
	mux.HandleFunc("GET /api/pd-approval", auth.RequireAcquisitionCompany(pdApprovalController.Index))
	mux.HandleFunc("GET /api/pd-approval/{id}", auth.RequireAcquisitionCompany(pdApprovalController.Show))
	mux.HandleFunc("GET /api/pd-approval/{id}/map", auth.RequireAcquisitionCompany(pdApprovalController.Map))
	mux.HandleFunc("POST /api/pd-approval/{id}", auth.RequireAcquisitionCompany(pdApprovalController.Store))
	mux.HandleFunc("GET /api/payment", auth.RequireAcquisitionCompany(paymentController.Index))
	mux.HandleFunc("GET /api/payment/document-types", auth.RequireAcquisitionCompany(paymentController.DocumentTypes))
	mux.HandleFunc("GET /api/payment/{id}", auth.RequireAcquisitionCompany(paymentController.Show))
	mux.HandleFunc("GET /api/payment/{id}/map", auth.RequireAcquisitionCompany(paymentController.Map))
	mux.HandleFunc("GET /api/payment/{id}/signature.pdf", auth.RequireAcquisitionCompany(paymentController.SignaturePDF))
	mux.HandleFunc("GET /api/public/payment-approvals/{id}/{token}", paymentController.PublicSignaturePDF)
	mux.HandleFunc("GET /api/payment/{id}/document", auth.RequireAcquisitionCompany(paymentController.Document))
	mux.HandleFunc("GET /api/payment/{id}/documents", auth.RequireAcquisitionCompany(paymentController.Documents))
	mux.HandleFunc("GET /api/payment/{id}/documents/{doc_id}", auth.RequireAcquisitionCompany(paymentController.DocumentFile))
	mux.HandleFunc("GET /api/payment/{id}/supported-documents", auth.RequireAcquisitionCompany(paymentController.SupportedDocuments))
	mux.HandleFunc("GET /api/payment/{id}/supported-documents/{doc_id}", auth.RequireAcquisitionCompany(paymentController.SupportedDocumentFile))
	mux.HandleFunc("GET /api/payment/{id}/supported-documents/{doc_id}/legacy-status", auth.RequireAcquisitionCompany(paymentController.SupportedDocumentLegacyStatus))
	mux.HandleFunc("GET /api/payment/{id}/memo", auth.RequireAcquisitionCompany(paymentController.Memo))
	mux.HandleFunc("GET /api/payment/{id}/memo/document", auth.RequireAcquisitionCompany(paymentController.MemoFile))
	mux.HandleFunc("GET /api/payment/{id}/local-files", auth.RequireAcquisitionCompany(paymentController.LocalHistoryFiles))
	mux.HandleFunc("GET /api/payment/{id}/local-files/{name}", auth.RequireAcquisitionCompany(paymentController.LocalHistoryFile))
	mux.HandleFunc("POST /api/payment/{id}", auth.RequireAcquisitionCompany(paymentController.Store))
	mux.HandleFunc("GET /api/payment-to-owner", auth.RequireAcquisitionCompany(paymentToOwnerController.Index))
	mux.HandleFunc("GET /api/payment-to-owner/document-types", auth.RequireAcquisitionCompany(paymentToOwnerController.DocumentTypes))
	mux.HandleFunc("GET /api/payment-to-owner/{id}", auth.RequireAcquisitionCompany(paymentToOwnerController.Show))
	mux.HandleFunc("GET /api/payment-to-owner/{id}/map", auth.RequireAcquisitionCompany(paymentToOwnerController.Map))
	mux.HandleFunc("GET /api/payment-to-owner/{id}/supported-documents", auth.RequireAcquisitionCompany(paymentController.SupportedDocuments))
	mux.HandleFunc("GET /api/payment-to-owner/{id}/supported-documents/{doc_id}", auth.RequireAcquisitionCompany(paymentController.SupportedDocumentFile))
	mux.HandleFunc("GET /api/payment-to-owner/{id}/supported-documents/{doc_id}/legacy-status", auth.RequireAcquisitionCompany(paymentController.SupportedDocumentLegacyStatus))
	mux.HandleFunc("GET /api/payment-to-owner/{id}/memo", auth.RequireAcquisitionCompany(paymentController.Memo))
	mux.HandleFunc("GET /api/payment-to-owner/{id}/memo/document", auth.RequireAcquisitionCompany(paymentController.MemoFile))
	mux.HandleFunc("GET /api/payment-to-owner/{id}/local-files", auth.RequireAcquisitionCompany(paymentController.LocalHistoryFiles))
	mux.HandleFunc("GET /api/payment-to-owner/{id}/local-files/{name}", auth.RequireAcquisitionCompany(paymentController.LocalHistoryFile))
	mux.HandleFunc("POST /api/payment-to-owner/{id}", auth.RequireAcquisitionCompany(paymentToOwnerController.Store))
	mux.HandleFunc("GET /api/land-permit-files/{name}", companyController.ServeLandPermitFile)
	mux.HandleFunc("POST /api/auth/login", auth.Login)
	mux.HandleFunc("POST /api/auth/forgot-password", auth.ForgotPassword)
	mux.HandleFunc("GET /api/auth/me", auth.Me)
	mux.HandleFunc("PUT /api/auth/me", auth.UpdateAccount)
	mux.HandleFunc("GET /api/auth/menu-access", auth.MenuAccess)
	mux.HandleFunc("GET /api/auth/companies", auth.Companies)
	mux.HandleFunc("POST /api/auth/refresh", auth.Refresh)
	mux.HandleFunc("POST /api/auth/logout", auth.Logout)

	// Archive routes use the same stage checks and company middleware as their
	// corresponding detail pages, while sharing the existing file-serving code.
	registerArchive := func(prefix string, controller *acquisitionController.CompensatedHistoryController, protect func(http.HandlerFunc) http.HandlerFunc) {
		for suffix, handler := range map[string]http.HandlerFunc{
			"supported-documents":                        controller.SupportedDocuments,
			"supported-documents/{doc_id}":               controller.SupportedDocumentFile,
			"supported-documents/{doc_id}/legacy-status": controller.SupportedDocumentLegacyStatus,
			"memo": controller.Memo, "memo/document": controller.MemoFile,
			"local-files": controller.LocalHistoryFiles, "local-files/{name}": controller.LocalHistoryFile,
		} {
			mux.HandleFunc("GET "+prefix+"/{id}/"+suffix, protect(handler))
		}
	}
	archiveModel := submitMemoLandController.Model
	stageChecks := map[string]func(context.Context, string, int64, int64) (acquisitionModel.SubmitMemoLandDetail, error){
		"/api/submit-memo-land": archiveModel.Detail,
		"/api/rm-approval":      archiveModel.RMDetail,
		"/api/gm-approval":      archiveModel.GMDetail,
	}
	for path, stage := range map[string]acquisitionModel.ParallelApprovalStage{
		"/api/gis-approval": acquisitionModel.GISApprovalStage, "/api/legal-approval": acquisitionModel.LegalApprovalStage,
		"/api/operation-control-approval": acquisitionModel.OperationControlStage, "/api/pd-approval": acquisitionModel.PDApprovalStage,
	} {
		stageChecks[path] = func(ctx context.Context, username string, companyID, landID int64) (acquisitionModel.SubmitMemoLandDetail, error) {
			return archiveModel.ParallelDetail(ctx, username, companyID, landID, stage)
		}
	}
	for path, check := range stageChecks {
		controller := &acquisitionController.CompensatedHistoryController{Model: archiveModel, CheckAccess: func(ctx context.Context, r *http.Request, companyID, landID int64) error {
			_, err := check(ctx, authController.Username(r), companyID, landID)
			return err
		}}
		registerArchive(path, controller, auth.RequireAcquisitionCompany)
	}
	memoArchive := &acquisitionController.CompensatedHistoryController{Model: archiveModel, CheckAccess: func(ctx context.Context, r *http.Request, companyID, landID int64) error {
		memoID, err := strconv.ParseInt(r.PathValue("memo_id"), 10, 64)
		if err != nil || memoID <= 0 {
			return pgx.ErrNoRows
		}
		return memoController.Model.CheckLand(ctx, companyID, memoID, landID)
	}}
	registerArchive("/api/acquisition-memos/{memo_id}/lands", memoArchive, func(next http.HandlerFunc) http.HandlerFunc {
		return auth.RequireModule("acquisitionRequest", "view", next)
	})
	companyLandMap := &mappingController.CompanyLandController{Model: &mappingModel.CompanyLandModel{DB: db}}
	mux.HandleFunc("GET /api/mapping/company-lands", auth.RequireAcquisitionCompany(companyLandMap.Index))
	return cors(mux)
}

func cors(next http.Handler) http.Handler {
	origin := strings.TrimRight(os.Getenv("FRONTEND_ORIGIN"), "/")
	if origin == "" {
		origin = "http://localhost:3000"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestOrigin := r.Header.Get("Origin")
		if requestOrigin != "" && requestOrigin == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
