-- Field 'tag' board tidak efektif dipakai -- diganti 'description' (nama board +
-- deskripsi). Lihat decision-log-bigtask-members-refactor-20260811.md.
ALTER TABLE boards ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE boards DROP COLUMN tag;
