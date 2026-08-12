-- access_level: SENGAJA kolom tunggal (bukan lewat roles/user_roles
-- many-to-many yang biasa dipakai project ini) -- super_user/regular_user
-- saling eksklusif per user, beda konsep dari roles (spv/dev/qa dst) yang
-- bisa dirangkap. Lihat docs/decision-log/decision-log-hr-mapping-super-user-20260810.md.
ALTER TABLE users ADD COLUMN access_level TEXT NOT NULL DEFAULT 'regular_user'
    CHECK (access_level IN ('super_user', 'regular_user'));

-- Mapping ke pegawai HR asli -- menggantikan placeholder CRC32 di
-- weeklyplan.pushToMyAgenda begitu user ini di-mapping lewat Manajemen User.
ALTER TABLE users ADD COLUMN hr_user_id INTEGER UNIQUE REFERENCES referensi_user_hr(hr_user_id);
