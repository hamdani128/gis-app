# Dokumen akuisisi pada Docker Ubuntu

Kolom `doc_file` di database hanya menyimpan nama file. Dokumen historis dibuka lewat URL GIS lama; share SMB tidak perlu dipasang di container. Backend membentuk URL dari `date_created` untuk dokumen lahan dan dari `Gis_Companies_id` untuk memo:

| Jenis | Folder URL GIS | File baru lokal |
| --- | --- | --- |
| Upload baru Payment dan Payment to Owner | - | `acquisition-land/{land_id}/{doc_file}` |
| PDF PD dan memo baru | - | `payment-signatures/` dan `acquisition-memos/` |
| Dokumen lahan lama | `folder1={YYYY}&folder2={M}` | - |
| Memo lama | `folder1=memo&folder2={Gis_Companies_id}` | - |

Contoh konfigurasi Compose untuk service backend (sesuaikan nama service dan lokasi host). Volume lokal hanya diperlukan agar upload **baru** tetap ada setelah container dibuat ulang:

```yaml
services:
  backend:
    environment:
      ACQUISITION_UPLOADS_DIR: /data/uploads
      LEGACY_DOCUMENT_URL: http://gis.aepplantations.id/doc/download_doc.php
    volumes:
      - type: bind
        source: /home/vm-ubuntu24/docker/gis-app/backend/uploads
        target: /data/uploads
```

Contoh `source` di atas mengikuti lokasi folder yang ditunjukkan pada server Ubuntu. Sesuaikan jika lokasi deploy berubah. File yang saat ini ada hanya di filesystem container lama perlu disalin ke folder host sebelum bind mount diaktifkan. Bind mount yang dipasang di atas folder berisi file container akan menyembunyikan file lama tersebut. Pastikan user proses backend dapat membaca dan menulis `/data/uploads`.

Periksa dari dalam container setelah deploy:

```sh
docker compose exec backend sh -c 'echo "$ACQUISITION_UPLOADS_DIR"; ls -l /data/uploads/acquisition-land/31708'
```

Backend mengarahkan browser ke URL GIS lama ketika file lokal tidak ada. Penanda `available` untuk dokumen lama berarti URL dapat dibentuk dari metadata; isi file belum diverifikasi oleh backend. Endpoint GIS harus dapat mengirim file pada sesi browser pengguna agar preview/unduhan berhasil.
