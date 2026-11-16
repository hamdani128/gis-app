# New Compensated Lands: riwayat dokumen dan memo

## Akses melalui Land Owner

Suffix endpoint pada tabel di bawah juga tersedia pada
`/api/land-owners/{owner_id}/lands/{land_id}`. Akses ini membutuhkan sesi aktif,
izin `landowner:view`, dan `company_id`. Sebelum metadata, file, status preview,
atau folder dibaca, backend memeriksa relasi land–owner–company melalui
`LandOwnerModel.LandDetail`. Jalur ini mencakup seluruh status lahan milik owner,
tanpa mensyaratkan izin modul New Compensated Lands. ID tidak valid menghasilkan
400, sedangkan relasi yang tidak ditemukan menghasilkan 404.

Di detail peta Land Owner dan New Compensated Lands, daftar dokumen dan ringkasan
memo ditampilkan pada tab Dokumen / Memo. File baru dimuat saat pengguna menekan
tombol detail, lalu ditampilkan dalam modal. Tab Informasi lahan dan Pengukuran
dan harga tetap tersedia.

## Akses melalui New Compensated Lands

Semua endpoint berikut membutuhkan izin `newcompensatedlands:view`, `company_id`, dan ID lahan berstatus `8` pada company tersebut. Respons metadata memakai envelope `{data: ...}`.

| Method | Path | Data |
| --- | --- | --- |
| GET | `/api/new-compensated-lands/{id}/supported-documents?company_id={company}` | Semua baris `Gis_Compensated_Land_Supported_Docs`, termasuk tipe, grup, catatan, nama file, `available`, dan `legacy_url` bila memakai server GIS lama |
| GET | `/api/new-compensated-lands/{id}/supported-documents/{doc_id}?company_id={company}` | File dokumen yang terkait dengan lahan |
| GET | `/api/new-compensated-lands/{id}/supported-documents/{doc_id}/legacy-status?company_id={company}` | Status URL preview lama: `available` bernilai `true`, `false`, atau `null` bila server lama tidak dapat diperiksa |
| GET | `/api/new-compensated-lands/{id}/local-files?company_id={company}` | Daftar file gambar/PDF pada folder backend `acquisition-land/{id}` |
| GET | `/api/new-compensated-lands/{id}/local-files/{name}?company_id={company}` | Preview satu file dari folder backend tersebut |
| GET | `/api/new-compensated-lands/{id}/memo?company_id={company}` | Memo yang dirujuk lahan dari `Gis_Aquisition_memo`, termasuk `found`, `available`, dan `legacy_url` bila memakai server GIS lama |
| GET | `/api/new-compensated-lands/{id}/memo/document?company_id={company}` | File memo yang terkait dengan lahan |

Dokumen Payment baru dibaca dari `backend/uploads/acquisition-land/{land_id}`. Browser menampilkan preview langsung dari `LEGACY_DOCUMENT_PREVIEW_BASE_URL/{YYYY}/{M}/{doc_file}` (default server GIS lama). Gambar yang gagal dimuat dan PDF dengan status 404 pada server lama beralih ke file backend; jika nama file tidak ditemukan, modal menampilkan pilihan file dari folder lahan. URL unduh lama tetap memakai `LEGACY_DOCUMENT_URL`. File memo historis memakai folder `memo/{Gis_Companies_id}`. Root lokal dapat disetel dengan `ACQUISITION_UPLOADS_DIR`; folder host harus terlihat dari dalam container. Lihat [panduan penyimpanan Docker](docker-document-mounts.md).
