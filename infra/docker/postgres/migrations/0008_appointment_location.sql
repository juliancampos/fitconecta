-- ============================================================================
-- Migracao 0008: Tipo de local e descricao do agendamento (ADR 0005)
-- Idempotente: pode ser executada com seguranca mais de uma vez.
-- ============================================================================

-- Tipo de local da sessao (select na tela de agendamento).
DO $$ BEGIN
    CREATE TYPE appointment_location AS ENUM ('home', 'studio', 'online');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE appointments
    ADD COLUMN IF NOT EXISTS location_type appointment_location,
    ADD COLUMN IF NOT EXISTS location_notes TEXT;
