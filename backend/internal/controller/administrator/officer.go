package administrator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	adminModel "gis-app/backend/internal/model/administrator"
	"gis-app/backend/internal/response"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type OfficerController struct {
	Model *adminModel.OfficerModel
}

func officerError(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "Terjadi kesalahan pada database"
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status, message = http.StatusNotFound, "Data tidak ditemukan"
	case errors.As(err, &pgErr):
		switch pgErr.Code {
		case "23505":
			status, message = http.StatusConflict, "Data dengan nilai yang sama sudah ada"
		case "23503":
			status, message = http.StatusConflict, "Data masih dipakai oleh record lain"
		case "22001", "23502", "23514", "22P02":
			status, message = http.StatusBadRequest, "Data tidak sesuai format atau batas kolom database"
		}
	}
	log.Printf("Officer: %v", err)
	response.Error(w, status, message)
}

func officerID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, http.StatusBadRequest, "ID harus berupa angka positif")
		return 0, false
	}
	return id, true
}

func decodeOfficerJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		response.Error(w, http.StatusBadRequest, "Body JSON tidak valid atau berisi field yang tidak dikenal")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		response.Error(w, http.StatusBadRequest, "Body harus berisi satu objek JSON")
		return false
	}
	return true
}

func readOfficerInput(w http.ResponseWriter, r *http.Request) (adminModel.OfficerInput, bool) {
	var input adminModel.OfficerInput
	if !decodeOfficerJSON(w, r, &input) {
		return input, false
	}
	adminModel.NormalizeOfficerInput(&input)
	if input.Name == "" || input.Username == "" {
		response.Error(w, http.StatusBadRequest, "Name dan username wajib diisi")
		return input, false
	}
	for _, id := range []*int32{input.CompanyID, input.PotentialLandID, input.DeptID} {
		if id != nil && *id <= 0 {
			response.Error(w, http.StatusBadRequest, "ID relasi harus berupa angka positif")
			return input, false
		}
	}
	return input, true
}

func readEmployeeCompanyInput(w http.ResponseWriter, r *http.Request) (adminModel.EmployeeCompanyInput, bool) {
	var input adminModel.EmployeeCompanyInput
	if !decodeOfficerJSON(w, r, &input) {
		return input, false
	}
	adminModel.NormalizeEmployeeCompanyInput(&input)
	if input.EmployeeID == nil || *input.EmployeeID <= 0 || input.CompanyID == nil || *input.CompanyID <= 0 {
		response.Error(w, http.StatusBadRequest, "Employee dan company wajib dipilih")
		return input, false
	}
	return input, true
}

func officerContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 8*time.Second)
}

func (c *OfficerController) Index(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := officerContext(r)
	defer cancel()
	items, err := c.Model.GetOfficers(ctx)
	if err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Data employees berhasil diambil", items)
}

func (c *OfficerController) Options(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := officerContext(r)
	defer cancel()
	options, err := c.Model.GetOptions(ctx)
	if err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Pilihan form berhasil diambil", options)
}

func (c *OfficerController) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := officerID(w, r)
	if !ok {
		return
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	item, err := c.Model.GetOfficer(ctx, id)
	if err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Detail employee berhasil diambil", item)
}

func (c *OfficerController) Store(w http.ResponseWriter, r *http.Request) {
	input, ok := readOfficerInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	item, err := c.Model.CreateOfficer(ctx, input)
	if err != nil {
		officerError(w, err)
		return
	}
	w.Header().Set("Location", "/api/officers/"+strconv.FormatInt(item.ID, 10))
	response.Created(w, "Employee berhasil ditambahkan", item)
}

func (c *OfficerController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := officerID(w, r)
	if !ok {
		return
	}
	input, ok := readOfficerInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	item, err := c.Model.UpdateOfficer(ctx, id, input)
	if err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Employee berhasil diperbarui", item)
}

func (c *OfficerController) Destroy(w http.ResponseWriter, r *http.Request) {
	id, ok := officerID(w, r)
	if !ok {
		return
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	if err := c.Model.DeleteOfficer(ctx, id); err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Employee berhasil dihapus", nil)
}

func (c *OfficerController) EmployeeCompanies(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := officerContext(r)
	defer cancel()
	items, err := c.Model.GetEmployeeCompanies(ctx)
	if err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Data companies employees berhasil diambil", items)
}

func (c *OfficerController) ShowEmployeeCompany(w http.ResponseWriter, r *http.Request) {
	id, ok := officerID(w, r)
	if !ok {
		return
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	item, err := c.Model.GetEmployeeCompany(ctx, id)
	if err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Detail companies employees berhasil diambil", item)
}

func (c *OfficerController) StoreEmployeeCompany(w http.ResponseWriter, r *http.Request) {
	input, ok := readEmployeeCompanyInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	item, err := c.Model.CreateEmployeeCompany(ctx, input)
	if err != nil {
		officerError(w, err)
		return
	}
	w.Header().Set("Location", "/api/officer-company-relations/"+strconv.FormatInt(item.ID, 10))
	response.Created(w, "Relasi company employee berhasil ditambahkan", item)
}

func (c *OfficerController) UpdateEmployeeCompany(w http.ResponseWriter, r *http.Request) {
	id, ok := officerID(w, r)
	if !ok {
		return
	}
	input, ok := readEmployeeCompanyInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	item, err := c.Model.UpdateEmployeeCompany(ctx, id, input)
	if err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Relasi company employee berhasil diperbarui", item)
}

func (c *OfficerController) DestroyEmployeeCompany(w http.ResponseWriter, r *http.Request) {
	id, ok := officerID(w, r)
	if !ok {
		return
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	if err := c.Model.DeleteEmployeeCompany(ctx, id); err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Relasi company employee berhasil dihapus", nil)
}

func (c *OfficerController) SetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := officerID(w, r)
	if !ok {
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if !decodeOfficerJSON(w, r, &input) {
		return
	}
	input.Password = strings.TrimSpace(input.Password)
	if len(input.Password) < 6 {
		response.Error(w, http.StatusBadRequest, "Password minimal 6 karakter")
		return
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	employee, err := c.Model.GetOfficer(ctx, id)
	if err != nil {
		officerError(w, err)
		return
	}
	if employee.Username == "" {
		response.Error(w, http.StatusBadRequest, "Employee ini belum memiliki username")
		return
	}
	count, err := c.Model.SetPassword(ctx, employee.Username, input.Password)
	if err != nil {
		officerError(w, err)
		return
	}
	if count == 0 {
		response.Error(w, http.StatusNotFound, "User dengan username tersebut tidak ditemukan")
		return
	}
	if count > 1 {
		response.Error(w, http.StatusConflict, "Username terdaftar lebih dari satu kali")
		return
	}
	response.Success(w, "Password berhasil diperbarui", nil)
}

func (c *OfficerController) Privileges(w http.ResponseWriter, r *http.Request) {
	id, ok := officerID(w, r)
	if !ok {
		return
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	employee, err := c.Model.GetOfficer(ctx, id)
	if err != nil {
		officerError(w, err)
		return
	}
	if employee.Username == "" {
		response.Error(w, http.StatusBadRequest, "Employee ini belum memiliki username")
		return
	}
	items, err := c.Model.GetPrivileges(ctx, employee.Username)
	if err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Data privilege berhasil diambil", items)
}

func (c *OfficerController) UpdatePrivileges(w http.ResponseWriter, r *http.Request) {
	id, ok := officerID(w, r)
	if !ok {
		return
	}
	var input struct {
		Permissions []adminModel.OfficerPrivilegeInput `json:"permissions"`
	}
	if !decodeOfficerJSON(w, r, &input) {
		return
	}
	if input.Permissions == nil {
		response.Error(w, http.StatusBadRequest, "Daftar permission wajib dikirim")
		return
	}
	seen := make(map[int32]struct{}, len(input.Permissions))
	for _, permission := range input.Permissions {
		if permission.ModuleID <= 0 {
			response.Error(w, http.StatusBadRequest, "Module ID harus berupa angka positif")
			return
		}
		if _, exists := seen[permission.ModuleID]; exists {
			response.Error(w, http.StatusBadRequest, "Module ID tidak boleh dikirim lebih dari satu kali")
			return
		}
		seen[permission.ModuleID] = struct{}{}
		for _, value := range []int32{permission.PAdd, permission.PDel, permission.PView, permission.PEdit} {
			if value != 0 && value != 1 {
				response.Error(w, http.StatusBadRequest, "Nilai permission harus 0 atau 1")
				return
			}
		}
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	employee, err := c.Model.GetOfficer(ctx, id)
	if err != nil {
		officerError(w, err)
		return
	}
	if employee.Username == "" {
		response.Error(w, http.StatusBadRequest, "Employee ini belum memiliki username")
		return
	}
	if err := c.Model.SetPrivileges(ctx, employee.Username, input.Permissions); err != nil {
		officerError(w, err)
		return
	}
	items, err := c.Model.GetPrivileges(ctx, employee.Username)
	if err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Privilege berhasil diperbarui", items)
}

func (c *OfficerController) Logs(w http.ResponseWriter, r *http.Request) {
	id, ok := officerID(w, r)
	if !ok {
		return
	}
	ctx, cancel := officerContext(r)
	defer cancel()
	employee, err := c.Model.GetOfficer(ctx, id)
	if err != nil {
		officerError(w, err)
		return
	}
	if employee.Username == "" {
		response.Success(w, "Tidak ada log untuk employee tanpa username", []adminModel.OfficerLog{})
		return
	}
	items, err := c.Model.GetLogs(ctx, employee.Username)
	if err != nil {
		officerError(w, err)
		return
	}
	response.Success(w, "Data log berhasil diambil", items)
}
