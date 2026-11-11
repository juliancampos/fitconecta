# Registros de Decisão de Arquitetura (ADRs)

Este diretório contém os **Architecture Decision Records (ADRs)** do projeto **FitConecta+**. Cada ADR registra uma decisão arquitetural significativa, seu contexto, justificativas e consequências.

## Índice de Decisões

| ID | Título | Status | Data |
|---|---|---|---|
| [ADR 0001](0001-estrutura-do-banco-de-dados.md) | Modelagem e Estrutura do Banco de Dados Relacional (PostgreSQL) | Aceito | 2026-10-10 |
| [ADR 0002](0002-definicao-das-paginas.md) | Definição das Páginas Frontend (Next.js) | Aceito | 2026-10-13 |
| [ADR 0003](0003-armazenamento-de-midias-dos-profissionais.md) | Armazenamento de Mídias dos Profissionais (Avatar e Galeria) | Aceito | 2026-10-14 |
| [ADR 0004](0004-bairros-de-atuacao-dos-profissionais.md) | Bairros de Atuação dos Profissionais | Aceito | 2026-10-14 |
| [ADR 0005](0005-fluxo-de-agendamento-de-sessao.md) | Fluxo de Agendamento de Sessão pelo Cliente | Aceito | 2026-10-15 |
| [ADR 0006](0006-deploy-flyio-separado-backend-frontend.md) | Deploy Separado do Backend e Frontend no Fly.io | Aceito | 2026-10-16 |
| [ADR 0007](0007-integracao-plataforma-de-pagamentos.md) | Integração com Plataforma de Pagamentos (Pix, Cartão e Split 90/10) | Proposto | 2026-10-20 |
| [ADR 0008](0008-painel-administrativo.md) | Painel Administrativo com Acesso Separado, Gestão de Agendamentos, Pagamentos e Status de Usuários | Substituído parcialmente pelo ADR 0009 | 2026-10-20 |
| [ADR 0009](0009-admin-no-mesmo-dominio.md) | Painel Administrativo no Mesmo Domínio em `/admin` | Aceito | 2026-10-20 |
| [ADR 0010](0010-pagamento-obrigatorio-pix-no-agendamento.md) | Pagamento Obrigatório via Pix no Fluxo de Agendamento | Aceito | 2026-10-20 |
| [ADR 0011](0011-servicos-objetivos-publicos-e-precos.md) | Serviços, Objetivos, Públicos, Formatos e Preços por Serviço | Proposto | 2026-10-30 |
| [ADR 0012](0012-dados-pessoais-do-cliente.md) | Dados Pessoais e Físicos do Cliente (Nascimento, Sexo, Altura, Peso, Frequência de Atividades e Objetivo) | Aceito | 2026-11-01 |
| [ADR 0013](0013-validacao-cref-no-cadastro.md) | Validação Automática do CREF no Cadastro do Profissional | Aceito | 2026-11-03 |
| [ADR 0014](0014-cadastro-de-nutricionistas.md) | Ramo do Profissional e Cadastro de Nutricionistas (CFN/CRN) | Aceito | 2026-11-10 |
| [ADR 0015](0015-sessao-iniciada-no-cadastro.md) | Sessão Iniciada no Cadastro (Redirecionamento Logado para a Home) | Aceito | 2026-11-10 |
| [ADR 0016](0016-cadastro-minimo-do-profissional-e-complemento-no-perfil.md) | Cadastro Mínimo do Profissional e Complemento no Perfil | Aceito | 2026-11-11 |
