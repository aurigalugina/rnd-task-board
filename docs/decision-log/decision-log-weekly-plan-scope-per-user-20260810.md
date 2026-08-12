# Decision Log — Weekly Plan Difilter per User (PIC), Bukan Cross-User

**Tanggal:** 2026-08-10
**Konteks:** weekly-plan-scope-per-user

## Konteks/Masalah

User nanya "ini udah identik sama user id-nya kan??" — dicek, ternyata TIDAK. `weeklyplan.List` join `big_tasks → daily_tasks → day_entries` tanpa filter `pic_user_id` sama sekali, jadi `GET /weekly-plan` mengembalikan baris yang SAMA PERSIS buat SEMUA user yang login, terlepas siapa PIC-nya. Dibuktikan langsung: user `spv` (bukan PIC di daily task manapun minggu berjalan) dapat baris identik dengan user `andi` (PIC beneran).

Ini bug, bukan keputusan sadar — tidak ada decision log/API contract sebelumnya yang menyebut ini cross-user secara sengaja. Halaman-nya sendiri secara eksplisit dinamai **"My Weekly Plan"** (`docs/01-vision-product.md` §4: `- **My Weekly Plan**: rollup mingguan lintas Board...`), dan konteks masalah yang melatarbelakangi fitur ini (`01-vision-product.md` §1 poin 2: "HR meminta laporan mingguan... duplikasi pelaporan") secara implisit tentang laporan mingguan PER ORANG ke HR, bukan rollup tim. Route-nya juga cuma `protected` (semua user login), bukan `spv`-only kayak fitur yang memang dimaksud jadi tampilan manajer/overseer (mis. Review Queue sebelum reviewer-assignment) — konsisten sama "tiap orang punya Weekly Plan sendiri", bukan satu rollup tim yang dilihat bareng-bareng.

## Keputusan

`GET /weekly-plan` dan `POST /weekly-plan/push` DIFILTER ke `daily_tasks.pic_user_id = <user yang login>` — Big Task muncul di Weekly Plan seseorang HANYA kalau dia PIC di minimal satu Daily Task-nya yang punya Day Entry di minggu terpilih, dan `actual_pct`/`expected_pct` yang ditampilkan/di-push CUMA dihitung dari Daily Task DIA (bukan agregat semua PIC di Big Task itu) — konsisten sama premis "laporan pribadi ke HR", bukan progress keseluruhan Big Task.

**Konsekuensi Big Task multi-PIC**: kalau satu Big Task punya beberapa Daily Task dengan PIC beda-beda, Big Task itu akan MUNCUL DI WEEKLY PLAN MASING-MASING PIC, tapi `actual_pct` di baris tiap orang beda-beda (cuma progress kerjaan dia sendiri) — ini WAJAR & disengaja, bukan bug: dashboard (`GET /boards/{id}/summary`, Dashboard page) tetap satu-satunya tempat lihat agregat keseluruhan Big Task lintas semua PIC (BR "Satu sumber data, banyak tampilan" — `01-vision-product.md` §5 poin 3: dashboard buat SPV/manajemen, weekly plan buat HR/personal).

## Alasan

- **Bug murni, bukan trade-off yang perlu didiskusikan lagi**: nama fitur ("My"), konteks masalah di Vision doc, dan pola otorisasi route (bukan spv-only) semuanya konsisten mengarah ke "per user" — tidak ada satupun sumber yang menyiratkan cross-user itu disengaja.
- **Filter di level `daily_tasks.pic_user_id`, bukan di level `big_tasks`**: Big Task bisa punya banyak Daily Task dengan PIC berbeda (skema sudah mengakomodasi ini sejak awal) — filter di Big Task doang gak cukup presisi, harus di Daily Task biar setiap orang cuma lihat/push progress KERJAANNYA SENDIRI.
- **`Push` ikut difilter, bukan cuma `List`**: kalau `List` difilter tapi `Push` tidak, angka yang di-push ke HR bisa beda dari yang ditampilkan di layar (mis. ke-push progress gabungan semua PIC padahal yang kelihatan di UI cuma punya sendiri) — dua-duanya harus konsisten pakai rule filter yang sama.

## Dampak/File Terpengaruh

- `backend/internal/weeklyplan/handler.go` — `List` (tambah param `userID`, filter di JOIN `daily_tasks`) dan `Push` (sama).
- `docs/05-api-contract.md` §8 — catat bahwa `GET /weekly-plan` scope-nya per user (PIC), bukan cross-user.
- Tidak ada perubahan skema DB (kolom `pic_user_id` sudah ada dari awal, tinggal dipakai buat filter).
