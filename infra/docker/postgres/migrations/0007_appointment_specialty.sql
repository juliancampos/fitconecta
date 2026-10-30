-- ============================================================================
-- Migracao 0007: Tipo de aula (especialidade) no agendamento (ADR 0005)
-- Idempotente: pode ser executada com seguranca mais de uma vez.
-- ============================================================================

-- Guarda a especialidade (tipo de aula) escolhida pelo cliente no agendamento.
-- Nullable para retrocompatibilidade com agendamentos existentes.
ALTER TABLE appointments
    ADD COLUMN IF NOT EXISTS specialty_id INTEGER
        REFERENCES specialties(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_appointments_specialty
    ON appointments(specialty_id);
