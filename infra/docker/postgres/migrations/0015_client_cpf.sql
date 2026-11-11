-- ============================================================================
-- Migracao 0015: CPF do cliente (ADR 0012, revisao de 2026-11-03)
-- Opcional. Guardado só com os 11 digitos; a mascara fica na tela.
-- ============================================================================

ALTER TABLE client_profiles
    ADD COLUMN IF NOT EXISTS cpf VARCHAR(11);

ALTER TABLE client_profiles DROP CONSTRAINT IF EXISTS chk_client_cpf;
ALTER TABLE client_profiles ADD CONSTRAINT chk_client_cpf
    CHECK (cpf IS NULL OR cpf ~ '^[0-9]{11}$');
