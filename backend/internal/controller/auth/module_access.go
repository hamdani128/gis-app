package auth

import (
	"context"
	"net/http"
	"strconv"
	"time"

	authModel "gis-app/backend/internal/model/auth"
	"gis-app/backend/internal/response"
)

type modulePrivilegeContextKey struct{}
type usernameContextKey struct{}

func Username(r *http.Request) string {
	username, _ := r.Context().Value(usernameContextKey{}).(string)
	return username
}

// RequireCompany verifies a login session and the selected company. The
// acquisition officer model performs the finer employee/sequence check.
func (c *Controller) RequireCompany(next http.HandlerFunc) http.HandlerFunc {
	return c.requireCompany(next, false)
}

// RequireAcquisitionCompany also accepts active acquisition officers assigned
// to company 0, which represents all companies for acquisition processing.
func (c *Controller) RequireAcquisitionCompany(next http.HandlerFunc) http.HandlerFunc {
	return c.requireCompany(next, true)
}

func (c *Controller) requireCompany(next http.HandlerFunc, allowGlobalOfficer bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := secret()
		if !ok {
			response.Error(w, 500, "Konfigurasi sesi belum tersedia")
			return
		}
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			response.Error(w, 401, "Sesi login tidak ditemukan")
			return
		}
		userID, valid := readSession(cookie.Value, key, time.Now())
		if !valid {
			response.Error(w, 401, "Sesi login tidak valid atau sudah berakhir")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		user, err := c.Model.GetActiveByID(ctx, userID)
		if err != nil {
			response.Error(w, 401, "Akun tidak aktif")
			return
		}
		companyID, err := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
		if err != nil || companyID < 0 {
			response.Error(w, 400, "company_id wajib berupa angka nol atau positif")
			return
		}
		if allowGlobalOfficer && companyID == 0 && r.PathValue("id") != "" {
			response.Error(w, 400, "detail lahan wajib memakai company_id lahan")
			return
		}
		if companyID == 0 {
			allowed, e := c.Model.CanManageAllCompanies(ctx, user.Username)
			if e != nil {
				response.Error(w, 500, "Akses company tidak dapat diperiksa")
				return
			}
			if !allowed {
				if allowGlobalOfficer {
					allowed, e = c.Model.HasGlobalAcquisitionOfficer(ctx, user.Username)
					if e != nil {
						response.Error(w, 500, "Akses company tidak dapat diperiksa")
						return
					}
				}
			}
			if !allowed {
				response.Error(w, 403, "Tidak memiliki akses semua company")
				return
			}
		} else {
			companies, e := c.Model.GetAccessibleCompanies(ctx, user.Username)
			if e != nil {
				response.Error(w, 500, "Akses company tidak dapat diperiksa")
				return
			}
			found := false
			for _, company := range companies {
				if int64(company.ID) == companyID {
					found = true
					break
				}
			}
			if !found {
				response.Error(w, 403, "Company tidak tersedia untuk akun ini")
				return
			}
		}
		next(w, r.WithContext(context.WithValue(r.Context(), usernameContextKey{}, user.Username)))
	}
}

// RequireModule protects the new Propose Lands endpoints with the same
// userrole flags used by the legacy CRUD, and checks the selected company.
func (c *Controller) RequireModule(module, action string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := secret()
		if !ok {
			response.Error(w, 500, "Konfigurasi sesi belum tersedia")
			return
		}
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			response.Error(w, 401, "Sesi login tidak ditemukan")
			return
		}
		userID, valid := readSession(cookie.Value, key, time.Now())
		if !valid {
			response.Error(w, 401, "Sesi login tidak valid atau sudah berakhir")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		user, err := c.Model.GetActiveByID(ctx, userID)
		if err != nil {
			response.Error(w, 401, "Akun tidak aktif")
			return
		}
		priv, err := c.Model.GetModulePrivileges(ctx, user.Username, module)
		if err != nil {
			response.Error(w, 500, "Hak akses tidak dapat diperiksa")
			return
		}
		companyID, err := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
		if err != nil || companyID < 0 {
			response.Error(w, 400, "company_id wajib berupa angka nol atau positif")
			return
		}
		if companyID == 0 {
			canManageAll, e := c.Model.CanManageAllCompanies(ctx, user.Username)
			if e != nil {
				response.Error(w, 500, "Akses company tidak dapat diperiksa")
				return
			}
			if !canManageAll {
				response.Error(w, 403, "Tidak memiliki akses semua company")
				return
			}
		} else {
			companies, e := c.Model.GetAccessibleCompanies(ctx, user.Username)
			if e != nil {
				response.Error(w, 500, "Akses company tidak dapat diperiksa")
				return
			}
			found := false
			for _, company := range companies {
				if int64(company.ID) == companyID {
					found = true
					break
				}
			}
			if !found {
				response.Error(w, 403, "Company tidak tersedia untuk akun ini")
				return
			}
		}
		if module == "acquisitionRequest" && companyID > 0 && (action == "view" || action == "add") {
			officer, e := c.Model.HasSubmitMemoOfficer(ctx, user.Username, companyID)
			if e != nil {
				response.Error(w, 500, "Penugasan Submit Memo Land tidak dapat diperiksa")
				return
			}
			priv.Add = priv.Add || officer
		}
		allowed := map[string]bool{"view": priv.View, "add": priv.Add, "edit": priv.Edit, "delete": priv.Del}[action]
		if !allowed {
			response.Error(w, 403, "Tidak memiliki hak akses modul")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), modulePrivilegeContextKey{}, priv)))
	}
}

func (c *Controller) ProposeLandPermissions(w http.ResponseWriter, r *http.Request) {
	priv, _ := r.Context().Value(modulePrivilegeContextKey{}).(authModel.ModulePrivileges)
	response.Success(w, "Hak akses Propose Lands berhasil diambil", priv)
}

func (c *Controller) ModulePermissions(w http.ResponseWriter, r *http.Request) {
	priv, _ := r.Context().Value(modulePrivilegeContextKey{}).(authModel.ModulePrivileges)
	response.Success(w, "Hak akses modul berhasil diambil", priv)
}
