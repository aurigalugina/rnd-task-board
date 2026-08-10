-- Seed user untuk pengembangan lokal.
-- Email: spv@rndops.local | Password: password123
-- JANGAN dipakai di luar lingkungan development.

INSERT INTO users (id, display_name, initials, email, password_hash, org_team, theme_preference)
VALUES (
    gen_random_uuid(),
    'SPV Dev Account',
    'SP',
    'spv@rndops.local',
    '$2b$12$SrZrvN7XlWwn2Ql1BdaxhunIs4mJO9Fp.uduH68p4KWt2mcwnxola',
    'R&D',
    'retro-light'
)
RETURNING id;

-- Setelah insert di atas, catat id yang dikembalikan, lalu jalankan manual:
-- INSERT INTO user_roles (user_id, role_id)
-- SELECT '<id-user-di-atas>', id FROM roles WHERE code IN ('spv','dev');
