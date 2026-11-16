# Payment API

Payment menampilkan lahan berstatus `6` yang memiliki proses PD Approval (`sequence_id=5`, `approval=1`) pada company yang diizinkan.

| Method | Path | Hasil |
| --- | --- | --- |
| GET | `/api/payment?company_id={id}` | `{data:{officer,lands}}`, hanya lahan yang menunggu Payment |
| GET | `/api/payment/document-types?company_id={id}` | Jenis dokumen dengan `type_doc_group='Payment'` dan `mandatory_=0` untuk combobox |
| GET | `/api/payment/{id}?company_id={id}` | `{data:{land,history}}` untuk lahan status 6 |
| GET | `/api/payment/{id}/map?company_id={id}` | `{data:{map_data}}` GeoJSON lahan |
| GET | `/api/payment/{id}/signature.pdf?company_id={id}` | PDF persetujuan PD sebagai lampiran untuk diunduh |
| GET | `/api/payment/{id}/signature.pdf?company_id={id}&preview=1` | PDF yang sama, disajikan inline untuk preview modal |
| GET | `/api/public/payment-approvals/{id}/{token}` | Unduh PDF tanpa login melalui tautan QR; token acak 256-bit wajib ada |
| POST | `/api/payment/{id}?company_id={id}` | Multipart `transfer_date` (`YYYY-MM-DD`), `note` (maksimal 10.000 karakter), serta 1â€“10 item paralel `documents`, `document_type_ids`, `document_titles`, `document_notes`; PDF/PNG/JPEG, maksimal 15 MB per file dan 50 MB total |
| GET | `/api/payment/{id}/documents?company_id={id}` | Daftar dokumen Payment yang tersimpan beserta jenis, judul, catatan, dan nama file |
| GET | `/api/payment/{id}/documents/{doc_id}?company_id={id}` | Unduh satu dokumen Payment sesuai ID |
| GET | `/api/payment/{id}/supported-documents?company_id={id}` | Semua riwayat dokumen lahan dari `Gis_Compensated_Land_Supported_Docs`, termasuk tipe, grup, catatan, dan ketersediaan file |
| GET | `/api/payment/{id}/supported-documents/{doc_id}?company_id={id}` | Tampilkan file riwayat dokumen sesuai ID |
| GET | `/api/payment/{id}/memo?company_id={id}` | Metadata memo terkait lahan dari `Gis_Aquisition_memo` |
| GET | `/api/payment/{id}/memo/document?company_id={id}` | Tampilkan file memo bila tersedia |
| GET | `/api/payment/{id}/document?company_id={id}` | Unduh dokumen Payment format lama bila tersedia |

POST hanya dapat dilakukan petugas Payment (`sequence_id=6`) untuk company lahan. Backend mengunci lahan, memeriksa status 6 dan approval PD, mencatat proses Payment (`sequence_id=6`, `approval=1`, tanggal transfer di `request_date`) dan setiap dokumen pada `Gis_Compensated_Land_Supported_Docs` dalam satu transaksi, lalu mengubah status lahan ke `7`. Pengajuan ulang ditolak.

PDF dibuat sekali ketika pertama diakses untuk preview atau unduh dan disimpan di `backend/uploads/payment-signatures/`. Logo AEP disematkan dalam aplikasi. Cap elektronik memuat nama petugas PD, waktu approval, dan ID proses yang dibaca dari riwayat database. Cap ini adalah bukti approval di aplikasi, bukan tanda tangan kriptografis berbasis sertifikat. Dokumen Payment baru disimpan di `backend/uploads/acquisition-land/{land_id}/{file_name}`; root lokal dapat diatur dengan `ACQUISITION_UPLOADS_DIR`. Nama file tetap dicatat pada `Gis_Compensated_Land_Supported_Docs.doc_file`, sedangkan jumlah dokumen per lahan ditentukan oleh `Gis_Compensated_Lands_id`. Pembaca file masih menerima unggahan Payment lama yang berada langsung di `backend/uploads/payment-approvals/`.

PDF versi QR memakai nama `payment-approval-{id}-qr.pdf` dan token disimpan pada `payment-approval-{id}.token` di direktori yang sama. Berkas PDF lama tanpa QR tetap ada, sedangkan akses baru menghasilkan versi QR satu kali. Atur `PUBLIC_API_ORIGIN` ke origin API yang dapat diakses pemindai QR sebelum PDF pertama kali dibuat; default lokal adalah `http://localhost:9000`. Tautan publik memberi akses ke informasi lahan dan harga dalam PDF kepada siapa pun yang memilikinya.

File dokumen historis, termasuk dokumen bergrup Payment yang belum tersimpan lokal, mula-mula dicari pada share `\\10.20.5.22\aepgis\CI\assets\uploads\{YYYY}\{M}\{doc_file}`. Tahun dan bulan berasal dari `Gis_Compensated_Land_Supported_Docs.date_created` masing-masing dokumen. Root share dapat diganti dengan `LEGACY_SUPPORTED_DOC_DIR`. File memo dicari di `\\10.20.5.22\aepgis\CI\assets\uploads\memo\{Gis_Companies_id}\{memo_doc}`; `Gis_Companies_id` dibaca dari record `Gis_Aquisition_memo`, dan root folder `memo` dapat diganti dengan `LEGACY_MEMO_DIR`. API hanya melayani file pada record lahan yang dapat diakses sesuai company dan berstatus Payment (6/7). Bila record atau file lama tidak ada, metadata tetap ditampilkan dengan penanda ketersediaan.

Tanpa mount SMB, backend membentuk `legacy_url` dari metadata dokumen dan mengarahkan browser ke `LEGACY_DOCUMENT_URL` (default `http://gis.aepplantations.id/doc/download_doc.php`). Folder dokumen lahan adalah tahun dan bulan `date_created`; folder memo adalah `memo` dan ID company. `available=true` untuk URL lama berarti tautannya dapat dibentuk, bukan file sudah diverifikasi. Upload baru tetap disimpan pada backend; lihat [panduan penyimpanan Docker](docker-document-mounts.md).
