# Software Requirement Specification (SRS) — R&D Ops

**Status:** Draft v1
**Referensi:** `01-vision-product.md`, `02-brd.md`

---

## 1. Pendahuluan

### 1.1 Tujuan
Menerjemahkan kebutuhan bisnis (BRD) menjadi spesifikasi fungsional dan non-fungsional yang cukup rinci untuk menjadi dasar desain arsitektur, API, dan basis data.

### 1.2 Definisi & Istilah

| Istilah | Definisi |
|---|---|
| Board | Entitas representasi produk/lini kerja, wadah dari Big Task |
| Big Task | Milestone/kerangka pekerjaan di bawah Board, memiliki tenggat waktu komitmen |
| Daily Task | Unit kerja di bawah Big Task, terdiri atas satu atau lebih baris rencana harian |
| Baris harian (Day Entry) | Rencana dan realisasi kerja untuk satu tanggal kalender dalam suatu Daily Task |
| Sign-off | Aksi eksplisit SPV menandai Big Task selesai secara final |
| Verdict | Status evaluatif Big Task: `on_progress`, `win`, atau `lose` |
| Role | Atribut kapabilitas pengguna (SPV, SA, Dev, QA, Admin), dapat dirangkap |
| Callback ID | Identitas tetap yang dikirim ke sistem HR eksternal untuk keperluan upsert |

### 1.3 Aktor Sistem

| Aktor | Deskripsi |
|---|---|
| Pengguna Terautentikasi | Semua pengguna dengan akun aktif di sistem |
| SPV | Pengguna dengan role `spv`; memiliki hak sign-off dan akses Settings |
| Kontributor | Pengguna dengan role `dev`, `sa`, `qa`, atau `admin`; mencatat dan memperbarui Daily Task |
| Super Admin | Subset SPV dengan akses manajemen pengguna dan tema (pada iterasi ini disatukan dengan role `spv`) |

## 2. Kebutuhan Fungsional

Notasi: `FR-<modul>-<nomor>`. Prioritas: **M**ust have / **S**hould have / **C**ould have (MoSCoW).

### 2.1 Modul Board & Big Task

| ID | Deskripsi | Prioritas |
|---|---|---|
| FR-BRD-01 | Sistem menyediakan operasi create/read untuk Board (nama, tag deskriptif). Board tidak memiliki field tenggat waktu. | M |
| FR-BRD-02 | Sistem menyediakan operasi create/read untuk Big Task di bawah suatu Board (nama, tanggal mulai, tenggat waktu, PIC default). | M |
| FR-BRD-03 | Sistem menghitung `expected_pct` Big Task secara dinamis dari proporsi waktu berjalan terhadap total durasi komitmen (tanggal mulai–tenggat). | M |
| FR-BRD-04 | Sistem menghitung `actual_pct` Big Task sebagai rata-rata `actual_pct` seluruh Daily Task aktif di bawahnya. | M |
| FR-BRD-05 | Sistem menghitung `completion_rate` Board sebagai rata-rata `actual_pct` seluruh Big Task di bawahnya, dihitung ulang setiap kali jumlah Big Task berubah. | M |
| FR-BRD-06 | Sistem menyediakan aksi sign-off Big Task, hanya aktif jika `actual_pct` = 100. | M |
| FR-BRD-07 | Sistem menandai Board berstatus selesai hanya jika seluruh Big Task berstatus signed. | M |
| FR-BRD-08 | Sistem menyediakan aksi undo sign-off. | S |

### 2.2 Modul Daily Task & Perencanaan Harian

| ID | Deskripsi | Prioritas |
|---|---|---|
| FR-DLY-01 | Sistem menyediakan operasi create Daily Task di bawah Big Task, dengan input: judul, PIC, tanggal mulai, tanggal selesai. | M |
| FR-DLY-02 | Saat Daily Task dibuat, sistem menghasilkan satu Day Entry untuk setiap tanggal kalender dalam rentang (inklusif), masing-masing berstatus awal belum selesai. | M |
| FR-DLY-03 | Setiap Day Entry menyimpan: tanggal, teks rencana, status selesai (boolean), teks blocker/catatan lanjutan. | M |
| FR-DLY-04 | Sistem menandai Day Entry yang jatuh pada hari Sabtu/Minggu dengan indikator visual "lembur", tanpa memengaruhi kalkulasi realisasi. | M |
| FR-DLY-05 | Pengguna dapat memperbarui status, teks rencana, dan blocker suatu Day Entry secara independen tanpa membuka form terpisah. | M |
| FR-DLY-06 | Sistem menghitung `actual_pct` Daily Task sebagai (jumlah Day Entry selesai ÷ total Day Entry) × 100. | M |
| FR-DLY-07 | Sistem menyediakan aksi "clone sebagai review", menghasilkan Daily Task baru dengan judul yang menyisipkan tag `[Review SPV]` atau `[Review QA]` di depan judul asal, dan PIC default sesuai role terpilih. | S |

### 2.3 Modul Assignment & Role

| ID | Deskripsi | Prioritas |
|---|---|---|
| FR-ASG-01 | Sistem menyimpan relasi many-to-many antara pengguna dan role. | M |
| FR-ASG-02 | Form assignment PIC menyediakan filter berdasarkan role sebelum memilih pengguna. | M |
| FR-ASG-03 | Assignment dapat menunjuk pengguna yang bukan anggota tim inti (ditandai dengan atribut tim/organisasi berbeda). | M |
| FR-ASG-04 | Sistem menampilkan penanda visual pada Daily Task yang PIC-nya berasal dari luar tim inti. | S |

### 2.4 Modul Kolaborasi (Komentar)

| ID | Deskripsi | Prioritas |
|---|---|---|
| FR-CMT-01 | Sistem menyediakan komentar pada tingkat Big Task, dengan atribut scope opsional yang menunjuk ke Daily Task tertentu (nullable = umum). | M |
| FR-CMT-02 | Sistem menyediakan filter tampilan komentar berdasarkan scope (semua/umum/per Daily Task). | M |
| FR-CMT-03 | Sistem mendukung mention pengguna melalui token `@nama` dengan saran otomatis (autocomplete) saat pengetikan. | S |
| FR-CMT-04 | Sistem menyediakan aksi cepat pada tampilan Daily Task untuk membuka form komentar dengan scope yang telah otomatis terisi. | S |
| FR-CMT-05 | Penulis komentar diambil dari identitas pengguna yang sedang login, tidak dapat dipilih manual. | M |

### 2.5 Modul Referensi/Cheat Sheet

| ID | Deskripsi | Prioritas |
|---|---|---|
| FR-REF-01 | Sistem menyediakan entitas referensi pada tingkat Board dengan tipe: `file`, `url`, atau `note`. | M |
| FR-REF-02 | Entri tipe `url` ditampilkan sebagai tautan yang dapat diklik. | M |
| FR-REF-03 | Entri tipe `note` menerima teks bebas tanpa batasan format. | M |
| FR-REF-04 | Entri tipe `file` menyimpan referensi berkas yang diunggah pengguna. | M |

### 2.6 Modul Pelaporan Mingguan (Weekly Plan)

| ID | Deskripsi | Prioritas |
|---|---|---|
| FR-WKL-01 | Sistem menyediakan tampilan rangkuman mingguan lintas seluruh Board, dengan navigasi minggu sebelumnya/berikutnya. | M |
| FR-WKL-02 | Rangkuman dikelompokkan per Big Task, menampilkan jumlah Day Entry dalam minggu terpilih yang tercakup dan yang berstatus selesai. | M |
| FR-WKL-03 | Sistem menghitung `actual_pct` dan `expected_pct` mingguan berdasarkan proporsi Day Entry dalam minggu yang bersangkutan. | M |
| FR-WKL-04 | Sistem menyediakan aksi push per baris rangkuman ke sistem HR eksternal, mengirimkan payload berisi Big Task, minggu, dan metrik terkait. | M |
| FR-WKL-05 | Sistem menghasilkan dan menyimpan Callback ID unik dan tetap per kombinasi Big Task + minggu pada saat push pertama; push berikutnya menggunakan Callback ID yang sama dan hanya memperbarui waktu push. | M |
| FR-WKL-06 | Sistem menampilkan waktu push terakhir dan Callback ID pada tiap baris rangkuman yang pernah di-push. | S |

### 2.7 Modul Notifikasi & Tinjauan (Review)

| ID | Deskripsi | Prioritas |
|---|---|---|
| FR-NTF-01 | Sistem menyediakan indikator jumlah item yang belum ditinjau, ditampilkan pada elemen navigasi utama. | M |
| FR-NTF-02 | Sistem menyediakan aksi menandai suatu item sebagai telah ditinjau, tanpa memengaruhi status/progress item tersebut. | M |
| FR-NTF-03 | Sistem menyediakan panel/halaman antrean tinjauan yang mendaftar seluruh item belum ditinjau. | M |

### 2.8 Modul Pengguna, Role, & Pengaturan

| ID | Deskripsi | Prioritas |
|---|---|---|
| FR-USR-01 | Pengguna dapat memperbarui nama tampilan dan inisial avatar miliknya sendiri. | M |
| FR-USR-02 | Pengguna dapat memperbarui kredensial (password) miliknya sendiri. | M |
| FR-USR-03 | Peran administratif dapat melihat daftar seluruh pengguna beserta role dan afiliasi tim/organisasi. | M |
| FR-USR-04 | Peran administratif dapat menambah pengguna baru dan mengatur kombinasi role-nya. | S |
| FR-USR-05 | Pengguna dapat memilih preferensi tema tampilan; preferensi tersimpan dan diterapkan pada sesi berikutnya. | S |
| FR-USR-06 | Sistem menyediakan aksi sign out yang mengakhiri sesi aktif. | M |

## 3. Kebutuhan Non-Fungsional

| ID | Kategori | Deskripsi |
|---|---|---|
| NFR-01 | Ketersediaan | Sistem berjalan sebagai layanan internal yang dapat diakses selama jam kerja melalui jaringan privat (mesh); tidak mensyaratkan SLA uptime tingkat produksi publik pada iterasi awal. |
| NFR-02 | Kinerja | Interaksi update harian (toggle status, edit teks) merespons dalam < 300ms pada kondisi jaringan internal normal. |
| NFR-03 | Skalabilitas data | Skema data harus tetap performan hingga skala puluhan Board, ratusan Big Task, dan ribuan Day Entry tanpa perubahan struktural. |
| NFR-04 | Keamanan akses | Autentikasi wajib untuk seluruh operasi tulis; otorisasi berbasis role untuk operasi sensitif (sign-off, manajemen pengguna, push HR). |
| NFR-05 | Auditability | Setiap aksi sign-off dan push HR mencatat aktor dan waktu kejadian. |
| NFR-06 | Portabilitas lingkungan | Aplikasi dapat dijalankan sepenuhnya pada mesin pengembangan lokal melalui satu perintah orkestrasi (docker compose), tanpa dependensi ke layanan cloud pihak ketiga. |
| NFR-07 | Ekstensibilitas skema | Skema basis data menyertakan entitas untuk mekanisme *change request* sejak iterasi awal meskipun antarmukanya belum dibangun, guna menghindari migrasi besar di kemudian hari. |
| NFR-08 | Konsistensi UI | Sistem mendukung penggantian tema visual (minimal 2 varian penuh, 2 varian tambahan sebagai perluasan) tanpa kehilangan fungsi di tema mana pun. |
| NFR-09 | Idempotensi integrasi | Operasi push ke sistem HR eksternal harus idempoten terhadap Callback ID yang sama untuk mencegah duplikasi data pada sisi penerima. |

## 4. Kebutuhan Data (Ringkasan Tingkat Tinggi)

Rincian penuh berada pada `06-db-design.md`. Entitas inti yang teridentifikasi dari kebutuhan fungsional di atas:

`users`, `roles`, `user_roles`, `boards`, `big_tasks`, `big_task_signoffs`, `daily_tasks`, `day_entries`, `comments`, `cheat_sheet_items`, `weekly_push_log`, `change_requests` (disiapkan, belum dipakai UI).

## 5. Batasan (Constraints)

- Backend menggunakan Go; frontend menggunakan Svelte(Kit); basis data PostgreSQL — keputusan stack telah difinalisasi dan menjadi batasan implementasi, bukan opsi terbuka (lihat `04-architecture.md`).
- Iterasi awal berjalan sebagai aplikasi tunggal (monolith) tanpa microservices, mengingat skala tim kecil.

## 6. Traceability (Ringkas)

| Business Requirement (BRD) | Functional Requirement (SRS) |
|---|---|
| BR-1.x (Board & Big Task) | FR-BRD-01 s/d FR-BRD-08 |
| BR-2.x (Perencanaan Harian) | FR-DLY-01 s/d FR-DLY-07 |
| BR-3.x (Progress & Verdict) | FR-BRD-03, FR-BRD-04, FR-DLY-06 |
| BR-4.x (Sign-off) | FR-BRD-06 s/d FR-BRD-08 |
| BR-5.x (Assignment & Handoff) | FR-ASG-01 s/d FR-ASG-04, FR-DLY-07 |
| BR-6.x (Kolaborasi) | FR-CMT-01 s/d FR-CMT-05 |
| BR-7.x (Referensi) | FR-REF-01 s/d FR-REF-04 |
| BR-8.x (Weekly & HR) | FR-WKL-01 s/d FR-WKL-06 |
| BR-9.x (User & Role) | FR-ASG-01, FR-USR-03, FR-USR-04 |
| BR-10.x (Pengaturan) | FR-USR-01, FR-USR-02, FR-USR-05 |
| BR-11.x (Notifikasi) | FR-NTF-01 s/d FR-NTF-03 |
