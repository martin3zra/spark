-- Migration: 20260819190300_seed_admin_user.sql
-- Portable seed (CURRENT_TIMESTAMP + FALSE literal work identically across
-- sqlite/postgres/mysql). No upsert: this only ever runs once against a fresh
-- table, and camel tracks it in camel_migrations so it won't re-run.

-- up
INSERT INTO users (uuid, name, email, password, role, must_change_password, email_verified_at)
VALUES ('53b6d626-20ec-4686-808f-8e55ea577bae', 'Alfredo', 'martin3zra@gmail.com', '$2a$10$EBK8SpME2Yabt2eCnAmRJeqRLj/qJec7E.mTVPA1KciyybNEFvZju', 'admin', false, CURRENT_TIMESTAMP);

-- down
DELETE FROM users WHERE email = 'martin3zra@gmail.com';
