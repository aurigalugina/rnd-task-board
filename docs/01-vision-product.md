# Vision Product — R&D Ops

**Status:** Draft v1
**Owner:** SPV & Developer, R&D — PT USSI Pinbuk Prima Software
**Tanggal:** Agustus 2026

---

## 1. Latar Belakang

Tim R&D di PT USSI Pinbuk Prima Software beranggotakan 4 orang (SPV & Developer, System Analyst & Developer, Developer, Project Admin & Technical Writer) dan bertanggung jawab memproduksi serta memelihara lini produk core banking (IBS Gen 2, IBSS Gen 2.5) beserta produk pendukungnya (eStatement, Branchless, CDC, LOS, Onboarding, Self Service, dan seterusnya).

Tim ini adalah tim penerusan dari tim sebelumnya, tanpa pelimpahan SOP/flow kerja yang terdokumentasi rapi. Tooling monitoring pekerjaan (kanban) yang pernah dibangun tim sebelumnya tidak dapat diakses oleh SPV saat ini, sehingga tracking pekerjaan sementara berjalan manual via Google Sheets.

Kondisi ini menimbulkan tiga masalah inti:

1. **Attention bottleneck pada peran SPV.** SPV merangkap sebagai developer aktif dengan beban kerja teknis langsung, sehingga kapasitas untuk memantau progress seluruh tim secara konsisten menjadi terbatas. Mekanisme pelaporan yang ada saat ini bergantung pada inisiatif SPV untuk secara aktif mengecek tiap anggota tim, alih-alih informasi yang terdorong secara terjadwal ke SPV.
2. **Duplikasi pelaporan.** HR meminta laporan mingguan (weekly plan) melalui sistem HR terpisah, sementara tim bekerja dalam granularitas harian di Google Sheets. Tidak ada satu sumber data yang bisa melayani dua kebutuhan ini sekaligus.
3. **Tidak ada jejak proses kerja yang terstruktur.** Progress dicatat sebagai angka persentase tanpa konteks — sulit membedakan pekerjaan yang benar-benar berjalan sesuai target dari yang sekadar diklaim selesai.

## 2. Visi Produk

> Membangun satu portal kerja internal tim R&D yang menjadi sumber kebenaran tunggal (single source of truth) untuk seluruh aktivitas tim — cukup ringan untuk diisi setiap hari tanpa terasa membebani, cukup kaya untuk memberi SPV dan manajemen gambaran akurat tanpa SPV harus terus-menerus menagih, dan cukup fleksibel untuk terus berkembang mengikuti kebutuhan tim dari waktu ke waktu.

Portal ini bukan sekadar pengganti Google Sheets. Ia dirancang di sekitar tiga prinsip:

- **Rendah friksi bagi kontributor.** Update harian harus semudah mengklik status, bukan mengisi form panjang.
- **Atensi terjadwal bagi SPV, bukan atensi konstan.** SPV punya satu titik cek (dashboard, review queue) yang bisa dikunjungi secara ritual, bukan harus memantau terus-menerus.
- **Adil terhadap proses, bukan hanya hasil akhir.** Status "sedang berjalan" tidak dihakimi sebagai "tertinggal" sebelum batas waktu benar-benar terlampaui — penilaian Win/Lose hanya jatuh pada titik keputusan yang wajar (task selesai, atau deadline lewat).

## 3. Target Pengguna

| Peran | Kebutuhan utama |
|---|---|
| SPV & Developer | Visibility harian tanpa harus menagih; approval/atensi ringan; sign-off penyelesaian pekerjaan; ekspor laporan mingguan ke HR |
| System Analyst & Developer | Pencatatan progress harian yang cepat; breakdown Big Task menjadi rencana kerja terukur |
| Developer | Update progress rutin tanpa friksi; handoff pekerjaan ke QA yang jelas |
| Project Admin & Technical Writer | Dokumentasi referensi per board (file, URL, catatan) yang mudah diakses |
| QA eksternal (di luar tim R&D) | Assignment verifikasi yang jelas ruang lingkupnya; komunikasi konteks bug/temuan yang tidak tercecer |
| HR (konsumen tidak langsung) | Laporan mingguan yang konsisten dan dapat diverifikasi progress hariannya |

## 4. Ruang Lingkup Produk

### Hierarki data inti
```
Board (produk, mis. IBS Gen 2)
  └── Big Task (kerangka/milestone, mis. "Tahap Analisis")
        └── Daily Task (unit kerja, bisa multi-hari)
              └── Baris harian (rencana, status, blocker/catatan)
```

### Kemampuan inti (Fase 1 — portal internal, cakupan dokumen ini)
- Manajemen Board, Big Task, dan Daily Task dengan breakdown rencana per hari kalender (termasuk penanda kerja di hari libur).
- Verdict Win / On progress / Lose per Big Task, dihitung dari realisasi vs komitmen waktu, tanpa menghakimi progres yang masih berjalan dalam tenggat.
- Sign-off penyelesaian Big Task oleh SPV; status project pada suatu Board dinyatakan selesai hanya jika seluruh Big Task telah di-sign.
- Assignment fleksibel berbasis role (RBAC ringan: SPV, SA, Dev, QA, Admin) — **satu pengguna dapat memegang lebih dari satu role sekaligus** (mis. SPV merangkap Dev, atau SA merangkap Dev). Role bersifat atomic dan dapat dikombinasikan bebas per pengguna, bukan kombinasi tetap yang di-hardcode (bukan "role Dev & QA" sebagai satu entitas role tersendiri), termasuk untuk pihak QA yang berada di luar tim R&D.
- Alur handoff pekerjaan (mis. Dev → Verifikasi QA → revisi Dev) dicatat sebagai Daily Task terpisah agar akuntabilitas tiap pihak tidak tercampur.
- Komentar berskop (umum per Big Task, atau spesifik per Daily Task) dengan dukungan mention (`@nama`).
- Cheat sheet/referensi per Board: file, URL, atau catatan bebas (untuk kasus bukti kerja non-file, mis. keterangan lokasi deploy aplikasi desktop).
- Dashboard ringkas ala project tracking (status project, hasil Won/Lose, deadline terdekat) untuk kebutuhan pelaporan ke manajemen.
- **My Weekly Plan**: rollup mingguan lintas Board, dikelompokkan per Big Task, dengan mekanisme push (upsert) ke sistem HR eksternal.
- Manajemen user dan preferensi tema aplikasi (pengaturan tampilan, bukan preferensi bisnis).

### Di luar cakupan Fase 1 (didokumentasikan sebagai arah ke depan)
- **Fase 2 — Integrasi MyAgenda-HR.** Push otomatis rollup mingguan ke sistem HR melalui service yang ditanam di server aplikasi HR (akses tersedia melalui System Analyst tim).
- **Fase 3 — Asisten virtual berbasis LLM.** Antarmuka percakapan (kemungkinan besar Telegram, mengikuti kebiasaan komunikasi internal perusahaan) yang membantu anggota tim mengisi rencana mingguan dan progress harian secara natural, lalu mengekstraknya menjadi data terstruktur di portal ini.

## 5. Prinsip Produk yang Tidak Bisa Ditawar

1. **Jangan menghakimi proses yang belum selesai.** Status "on progress" bersifat netral secara visual dan bahasa sampai titik keputusan (deadline atau penyelesaian) benar-benar tercapai.
2. **Approval adalah bentuk atensi, bukan gerbang persetujuan.** Aksi review/approve tidak boleh memblokir pekerjaan berjalan; fungsinya murni menandai bahwa SPV telah melihat dan aware.
3. **Satu sumber data, banyak tampilan.** Data yang sama harus bisa menghasilkan tampilan berbeda untuk SPV/manajemen (dashboard) dan untuk HR (weekly plan), tanpa duplikasi entri.
4. **Sistem harus bisa tumbuh tanpa dirombak ulang.** Skema data (terutama Big Task dan Daily Task) harus mengakomodasi penambahan cakupan pekerjaan di tengah jalan tanpa memerlukan migrasi besar.
5. **Role bersifat atomic dan dapat dirangkap.** Satu pengguna dapat memegang lebih dari satu role secara bersamaan (relasi many-to-many antara pengguna dan role), bukan role gabungan yang bersifat tetap. Ini merefleksikan realita tim yang kecil, di mana perangkapan peran adalah hal biasa, bukan pengecualian.

## 6. Mekanisme Evolusi Produk — Change Request Berbasis Kontribusi Tim

Sebagai prinsip desain jangka panjang (bukan bagian dari fitur Fase 1, namun **dipersiapkan ruang datanya sejak awal**), portal ini dirancang untuk terus relevan dengan menyediakan jalur bagi seluruh anggota tim — bukan hanya SPV — untuk mengusulkan perubahan atau penambahan fitur. Usulan disampaikan melalui percakapan bebas yang kemudian disusun menjadi `change_request.md` terstruktur (dengan bantuan agentic coding assistant yang ditanam di repository project), untuk selanjutnya ditriase oleh SPV dan System Analyst dan dijadwalkan ke siklus pengembangan berikutnya.

Implikasi terhadap desain: skema database menyertakan entitas `change_requests` sejak tahap awal (lihat `06-db-design.md`), meskipun antarmuka dan alur kerjanya baru dibangun pada iterasi lanjutan.

## 7. Indikator Keberhasilan (Awal)

Karena ini adalah tooling internal untuk tim berskala kecil, keberhasilan diukur secara kualitatif pada tahap awal:

- SPV dapat menjawab "apa yang sedang dikerjakan tim hari ini" dalam < 1 menit melalui satu dashboard, tanpa perlu bertanya langsung.
- Tidak ada lagi entri ganda antara pencatatan internal tim dan laporan mingguan ke HR.
- Anggota tim melaporkan progres harian tanpa merasa proses itu sebagai beban administratif tambahan (indikator subjektif, dikonfirmasi lewat diskusi tim setelah masa uji coba).
- Riwayat proses kerja (rencana harian, blocker, catatan lanjutan) tersedia sebagai bahan retrospektif, bukan hanya angka persentase akhir.

## 8. Dokumen Terkait

- `02-brd.md` — Business Requirement Document
- `03-srs.md` — Software Requirement Specification
- `04-architecture.md` — Arsitektur Sistem
- `05-api-contract.md` — Kontrak API
- `06-db-design.md` — Desain Basis Data
