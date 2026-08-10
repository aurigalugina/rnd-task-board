# Business Requirement Document (BRD) — R&D Ops

**Status:** Draft v1
**Referensi:** `01-vision-product.md`

---

## 1. Tujuan Dokumen

Menerjemahkan Vision Product menjadi kebutuhan bisnis yang konkret dan dapat diverifikasi, sebagai dasar penyusunan SRS, arsitektur, dan desain data.

## 2. Ruang Lingkup Bisnis

### Termasuk dalam lingkup
- Pencatatan dan pelacakan pekerjaan tim R&D dalam hierarki Board → Big Task → Daily Task.
- Mekanisme kolaborasi lintas peran, termasuk handoff pekerjaan ke pihak QA di luar tim R&D.
- Pelaporan progress ke dua audiens dengan kebutuhan berbeda: manajemen internal (dashboard) dan HR (laporan mingguan).
- Pengelolaan pengguna dan role secara fleksibel (many-to-many).
- Pengaturan aplikasi tingkat pengguna (profil, tema tampilan).

### Di luar lingkup (didokumentasikan, bukan dibangun pada iterasi ini)
- Integrasi otomatis ke sistem HR eksternal (push data terjadi manual melalui aksi pengguna pada iterasi ini; otomasi penuh adalah pengembangan lanjutan).
- Antarmuka percakapan/asisten virtual berbasis LLM.
- Antarmuka untuk menyusun dan mengeksekusi *change request* menjadi kode (skema data disiapkan, alur kerja penuh adalah pengembangan lanjutan).

## 3. Pemangku Kepentingan (Stakeholders)

| Role | Kepentingan |
|---|---|
| SPV & Developer | Pemilik proses; kebutuhan utama pada visibility dan pelaporan |
| System Analyst & Developer | Kontributor utama pencatatan kerja; turut menriase perubahan produk |
| Developer | Kontributor pencatatan kerja harian |
| Project Admin & Technical Writer | Kontributor dokumentasi/referensi |
| QA (eksternal, di luar tim R&D) | Penerima handoff verifikasi; kontributor temuan |
| Manajemen (konsumen dashboard) | Membutuhkan gambaran status project tanpa detail teknis |
| HR (konsumen laporan mingguan) | Membutuhkan laporan progress mingguan yang konsisten dan dapat diverifikasi |

## 4. Proses Bisnis Saat Ini (As-Is)

1. Tim mencatat pekerjaan pada spreadsheet bersama, diperbarui manual oleh masing-masing individu.
2. SPV memantau spreadsheet secara berkala untuk mengetahui status pekerjaan tim.
3. HR meminta laporan mingguan melalui sistem HR terpisah; SPV atau anggota tim menyalin ulang ringkasan dari spreadsheet ke sistem tersebut secara manual.
4. Tidak ada mekanisme formal untuk menandai bahwa SPV telah meninjau suatu pekerjaan.
5. Tidak ada pencatatan konteks proses (rencana harian, kendala, tindak lanjut) — hanya angka persentase akhir.

## 5. Proses Bisnis yang Diusulkan (To-Be)

1. Setiap anggota tim mencatat rencana dan realisasi kerja hariannya langsung di portal, dalam struktur Board → Big Task → Daily Task.
2. Sistem menghitung status berjalan (on progress) secara otomatis; verdict akhir (Win/Lose) hanya ditentukan pada titik keputusan (task selesai atau tenggat terlampaui).
3. SPV meninjau pekerjaan melalui dashboard dan antrean tinjauan (review queue), menandai (flag) yang telah dilihat tanpa memblokir pekerjaan berjalan.
4. SPV melakukan sign-off atas Big Task yang telah selesai; status suatu Board dinyatakan selesai hanya setelah seluruh Big Task di dalamnya di-sign.
5. Pekerjaan yang membutuhkan verifikasi pihak lain (mis. QA) dicatat sebagai Daily Task terpisah dengan assignment eksplisit, sehingga status pekerjaan pihak pertama tidak tergantung pada penyelesaian pihak kedua.
6. Setiap minggu, sistem merangkum data harian menjadi rollup mingguan per Big Task, yang kemudian didorong (push) oleh pengguna berwenang ke sistem HR eksternal. Push dapat dilakukan berulang (upsert) menggunakan identitas callback yang konsisten.

## 6. Kebutuhan Bisnis (Business Requirements)

### 6.1 Manajemen Board & Big Task
- **BR-1.1** Sistem harus memungkinkan pembuatan Board (representasi produk/lini kerja) tanpa batas waktu (tidak wajib memiliki deadline).
- **BR-1.2** Sistem harus memungkinkan pembuatan Big Task di bawah suatu Board, dengan nama, tanggal mulai, dan tenggat waktu komitmen.
- **BR-1.3** Big Task harus dapat ditambahkan kapan saja selama siklus hidup Board berjalan (tidak wajib didefinisikan lengkap di awal).
- **BR-1.4** Penambahan atau pengurangan Big Task harus otomatis memengaruhi perhitungan agregat completion rate Board tanpa intervensi manual.

### 6.2 Perencanaan & Pencatatan Harian
- **BR-2.1** Sistem harus memungkinkan pembuatan Daily Task di bawah suatu Big Task, dengan rentang tanggal mulai dan selesai berbasis hari kalender (bukan hari kerja saja).
- **BR-2.2** Sistem harus memecah rentang tanggal Daily Task menjadi baris rencana per hari, masing-masing dapat diisi rencana kerja, status (selesai/belum), dan catatan kendala.
- **BR-2.3** Sistem harus menandai secara visual baris rencana yang jatuh pada akhir pekan sebagai indikator kerja lembur.
- **BR-2.4** Pengguna harus dapat memperbarui status dan catatan tiap baris harian kapan saja tanpa membuka form terpisah yang menghambat proses (interaksi cepat/inline).

### 6.3 Pelacakan Progress & Penilaian
- **BR-3.1** Sistem harus menghitung persentase realisasi Big Task dari rata-rata realisasi seluruh Daily Task di bawahnya.
- **BR-3.2** Sistem tidak boleh memberikan label evaluatif negatif (mis. "tertinggal") pada pekerjaan yang berjalan sebelum tenggat waktu terlampaui. Status yang ditampilkan selama tenggat belum lewat bersifat netral ("sedang berjalan").
- **BR-3.3** Status akhir "Win" hanya berlaku bila pekerjaan mencapai realisasi penuh dalam batas waktu komitmen.
- **BR-3.4** Status akhir "Lose" hanya berlaku bila tenggat waktu terlampaui tanpa realisasi penuh.

### 6.4 Sign-off & Penyelesaian Project
- **BR-4.1** Sistem harus menyediakan aksi eksplisit bagi peran SPV untuk menandai suatu Big Task sebagai selesai (sign-off).
- **BR-4.2** Aksi sign-off hanya dapat dilakukan apabila realisasi Big Task telah mencapai 100%.
- **BR-4.3** Status keseluruhan suatu Board dinyatakan "selesai" hanya jika seluruh Big Task di dalamnya telah di-sign.
- **BR-4.4** Sign-off harus dapat dibatalkan (undo) oleh peran yang berwenang.

### 6.5 Assignment & Handoff Lintas Peran
- **BR-5.1** Sistem harus mendukung penugasan (assignment) Daily Task kepada pengguna mana pun tanpa batasan role yang kaku, dengan opsi penyaringan berdasarkan role saat memilih.
- **BR-5.2** Sistem harus mendukung penugasan kepada pengguna yang bukan bagian dari tim inti (mis. QA eksternal).
- **BR-5.3** Sistem harus menyediakan mekanisme untuk menduplikasi konteks suatu Daily Task menjadi Daily Task baru dalam rangka handoff (mis. serah-terima ke tahap verifikasi), tanpa menggantung status pekerjaan pihak sebelumnya.

### 6.6 Kolaborasi
- **BR-6.1** Sistem harus menyediakan fasilitas komentar pada tingkat Big Task, dengan opsi mempersempit (scope) komentar ke Daily Task tertentu di bawahnya.
- **BR-6.2** Sistem harus mendukung mention pengguna lain di dalam komentar.
- **BR-6.3** Sistem harus menyediakan akses cepat untuk membuat komentar dari tampilan Daily Task, dengan konteks (scope) yang otomatis terisi sesuai Daily Task tersebut.

### 6.7 Referensi & Bukti Kerja
- **BR-7.1** Sistem harus menyediakan ruang referensi pada tingkat Board untuk menyimpan tiga jenis bukti/rujukan: berkas (file), tautan (URL), atau catatan bebas.
- **BR-7.2** Jenis catatan bebas harus tersedia sebagai alternatif ketika hasil kerja tidak dapat direpresentasikan sebagai berkas atau tautan (mis. keterangan lokasi deployment aplikasi desktop).

### 6.8 Pelaporan Mingguan & Integrasi HR
- **BR-8.1** Sistem harus menyediakan rangkuman mingguan yang mengelompokkan Daily Task berdasarkan Big Task, lintas seluruh Board.
- **BR-8.2** Rangkuman mingguan harus menghitung persentase realisasi dan ekspektasi berdasarkan proporsi hari yang tercakup dalam minggu terpilih.
- **BR-8.3** Sistem harus menyediakan aksi eksplisit untuk mendorong (push) suatu baris rangkuman mingguan ke sistem HR eksternal.
- **BR-8.4** Push berulang atas baris rangkuman yang sama harus bersifat idempoten (upsert) menggunakan identitas callback yang konsisten, dengan pencatatan waktu push terakhir.

### 6.9 Manajemen Pengguna & Role
- **BR-9.1** Sistem harus memungkinkan satu pengguna memiliki lebih dari satu role secara bersamaan.
- **BR-9.2** Role harus dikelola sebagai entitas atomic yang dapat dikombinasikan bebas, bukan kombinasi tetap yang di-hardcode.
- **BR-9.3** Pengelolaan pengguna dan role hanya dapat diakses oleh peran administratif (SPV/Super Admin).

### 6.10 Pengaturan Aplikasi
- **BR-10.1** Pengguna harus dapat memperbarui profil (nama, avatar, kredensial) miliknya sendiri.
- **BR-10.2** Sistem harus menyediakan pilihan tema tampilan, dengan preferensi tersimpan per pengguna.

### 6.11 Notifikasi & Atensi
- **BR-11.1** Sistem harus menyediakan indikator terpusat bagi peran SPV untuk melihat seluruh item yang membutuhkan tinjauan (belum direview).
- **BR-11.2** Tindakan menandai suatu item sebagai telah ditinjau tidak boleh menghalangi progres pekerjaan terkait.

## 7. Aturan Bisnis (Business Rules) — Ringkasan

| Kode | Aturan |
|---|---|
| RULE-01 | Completion rate Board = rata-rata realisasi seluruh Big Task aktif di bawahnya, dihitung ulang otomatis saat jumlah Big Task berubah. |
| RULE-02 | Realisasi Big Task = rata-rata realisasi seluruh Daily Task di bawahnya. |
| RULE-03 | Realisasi Daily Task = proporsi baris harian berstatus selesai terhadap total baris harian. |
| RULE-04 | Verdict "on progress" berlaku selama tenggat waktu belum terlampaui, terlepas dari besar kecilnya gap antara realisasi dan ekspektasi. |
| RULE-05 | Verdict "Win" hanya sah jika status selesai (100%) tercapai sebelum atau tepat pada tenggat waktu. |
| RULE-06 | Verdict "Lose" berlaku otomatis jika tenggat waktu terlampaui tanpa status selesai. |
| RULE-07 | Sign-off Big Task mensyaratkan realisasi 100%. |
| RULE-08 | Status Board "selesai" mensyaratkan seluruh Big Task di bawahnya berstatus sign-off. |
| RULE-09 | Baris harian pada Sabtu/Minggu ditandai sebagai indikator kerja lembur, tanpa memengaruhi perhitungan realisasi. |
| RULE-10 | Identitas callback untuk push mingguan bersifat tetap per kombinasi Big Task dan minggu; push berulang memperbarui waktu, bukan membuat identitas baru. |
| RULE-11 | Satu pengguna dapat memiliki banyak role; satu role dapat dimiliki banyak pengguna (relasi many-to-many). |

## 8. Asumsi

- Jumlah pengguna aktif kecil (skala tim, bukan skala organisasi besar) pada iterasi awal.
- Akses aplikasi berlangsung dalam jaringan internal/mesh privat, bukan terekspos publik.
- Integrasi ke sistem HR pada iterasi ini bersifat *push* satu arah yang dipicu manual oleh pengguna, bukan sinkronisasi dua arah otomatis.
- Struktur organisasi tim (kombinasi role per individu) dapat berubah sewaktu-waktu; sistem tidak boleh mengasumsikan pemetaan role yang tetap.

## 9. Ketergantungan (Dependencies)

- Ketersediaan endpoint/callback pada sistem HR eksternal untuk menerima push data (di luar kendali langsung tim R&D; dikoordinasikan melalui anggota tim yang memiliki akses ke server aplikasi HR).
- Infrastruktur jaringan internal (mesh privat) untuk aksesibilitas lintas anggota tim.

## 10. Di Luar Cakupan (Out of Scope)

- Manajemen payroll, cuti, atau data kepegawaian lain di luar laporan progress kerja.
- Alur persetujuan berlapis (multi-level approval) — sign-off pada iterasi ini bersifat satu langkah oleh peran SPV.
- Pelacakan waktu kerja presisi (time tracking berbasis jam/menit); satuan terkecil adalah hari kalender.

## 11. Kriteria Sukses

- Seluruh anggota tim mencatat pekerjaan harian di portal sebagai satu-satunya sumber pencatatan (spreadsheet lama tidak lagi digunakan paralel).
- Laporan mingguan ke HR dihasilkan dari data yang sama dengan pencatatan harian tim, tanpa entri ulang manual.
- SPV dapat menyelesaikan tinjauan progress tim rutin dalam satu sesi singkat menggunakan dashboard dan antrean tinjauan.

## 12. Risiko Bisnis

| Risiko | Dampak | Mitigasi Awal |
|---|---|---|
| Tim kembali menggunakan spreadsheet paralel karena friksi input dirasa lebih tinggi dari solusi lama | Duplikasi data, portal kehilangan relevansi | Prioritaskan interaksi cepat (inline update) pada Daily Task sejak iterasi pertama |
| Integrasi HR tertunda karena ketergantungan akses server eksternal | Fase 2 roadmap molor | Desain skema data agar rollup mingguan tetap bernilai meski push belum terhubung (dapat diekspor manual) |
| Penambahan role/kombinasi baru di masa depan memerlukan perubahan skema | Biaya migrasi berulang | Terapkan RULE-11 (many-to-many role) sejak awal, bukan sebagai penambahan belakangan |
