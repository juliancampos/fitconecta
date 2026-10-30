# ADR 0010: Pagamento Obrigatório via Pix no Fluxo de Agendamento

* **Status:** Aceito
* **Data:** 2026-10-20
* **Decisores:** Produto, Engenharia e Financeiro FitConecta+
* **Contexto Técnico:** Go (Core API), Next.js (Web), PostgreSQL 16, PSP com Pix e split
* **Relacionados:** [ADR 0005](0005-fluxo-de-agendamento-de-sessao.md), [ADR 0007](0007-integracao-plataforma-de-pagamentos.md), [ADR 0008](0008-painel-administrativo.md)

---

## 1. Contexto e Problema

O fluxo de agendamento já cria uma pré-reserva de 10 minutos com `appointments.status = pending_payment` e um registro em `payments` com `status = pending`. Porém, a confirmação atual é **simulada**:

- a tela `/app/client/pagamento/[appointmentId]` oferece **"Confirmar pagamento"** e **"Pagar depois"**;
- `POST /api/v1/appointments/{id}/confirm-payment` pode ser chamado pelo próprio cliente e grava `paid`/`confirmed` sem consultar um provedor;
- não existe cobrança Pix real, QR Code, webhook, expiração efetiva nem reembolso.

A regra de produto passa a ser: **o cliente só tem o horário garantido após pagar**. A tela de pagamentos terá, inicialmente, **apenas Pix** como método disponível.

Este ADR define o fluxo, os estados, o contrato da API, a tela e os critérios de segurança do MVP. A escolha do PSP e as regras comerciais de split continuam no [ADR 0007](0007-integracao-plataforma-de-pagamentos.md), que está em status **Proposto**.

### Requisitos

1. Criar o agendamento somente como **pré-reserva**; não confirmar sem pagamento.
2. Exibir imediatamente a tela de pagamento com **Pix (QR Code + copia e cola)**.
3. Confirmar o agendamento apenas após **webhook válido do PSP**.
4. Liberar o horário automaticamente se o Pix não for pago no prazo.
5. Não permitir que o cliente declare o pagamento como concluído.
6. Manter o split 90/10 e os valores calculados no servidor.
7. Deixar cartão de crédito fora do MVP, preservando a evolução futura.

---

## 2. Decisão

Adotamos um fluxo **Pix-first e payment-first**, com pré-reserva curta:

1. O cliente escolhe dia, horário, tipo de aula e local.
2. Ao confirmar, a API cria o agendamento em `pending_payment` e o pagamento em `pending`, ambos com expiração de **10 minutos**.
3. O cliente é redirecionado para a tela de pagamento.
4. A tela solicita/cria a cobrança Pix no PSP e exibe QR Code, copia e cola, valor e contagem regressiva.
5. O cliente paga fora da plataforma, no app do banco.
6. O PSP envia um **webhook assinado**; a API valida, reconsulta a cobrança e, em transação, grava `payments.status = paid` e `appointments.status = confirmed`.
7. A tela faz polling do status e leva o cliente para **Meus agendamentos** quando confirmado.
8. Se o pagamento não ocorrer até a expiração, o agendamento vira `expired`, o pagamento vira `expired` e o horário é liberado.

> **Importante:** até o webhook de pagamento, o registro é uma **pré-reserva**, não um agendamento confirmado. A UI deve usar esse vocabulário para evitar expectativa incorreta.

### 2.1 Fluxo do usuário

```
Perfil do profissional
  → Escolher dia, horário, tipo de aula e local
  → "Continuar para pagamento"
  → POST /api/v1/appointments
      → 201 pending_payment (10 min)
  → /app/client/pagamento/{appointmentId}
  → POST /api/v1/appointments/{id}/payment (idempotente)
      → QR Code Pix + copia e cola + expires_at
  → Cliente paga no banco
  → PSP envia webhook
  → API confirma pagamento e agendamento
  → Tela detecta "paid" e redireciona para /app/client/agendamentos
```

Se o cliente sair da tela, o item continua em **Meus agendamentos** como `pending_payment` e oferece **"Pagar agora"** enquanto não expirar.

### 2.2 Estados

**Agendamento (`appointments.status`):**

| Estado | Significado | Transições |
|---|---|---|
| `pending_payment` | Pré-reserva ativa, aguardando Pix | → `confirmed` (webhook pago) / `expired` (prazo) / `cancelled_by_*` |
| `confirmed` | Pagamento confirmado; horário garantido | → `completed`, `cancelled_by_*`, `no_show` |
| `expired` | Pré-reserva vencida sem pagamento | Estado final |
| `cancelled_by_client` | Cancelado pelo cliente | Estado final |
| `cancelled_by_professional` | Cancelado pelo profissional | Estado final |
| `cancelled_by_admin` | Cancelado pelo admin | Estado final |
| `completed` | Sessão realizada | Estado final |
| `no_show` | Cliente não compareceu | Estado final |

**Pagamento (`payments.status`):**

| Estado | Significado |
|---|---|
| `pending` | Cobrança Pix criada/aguardando pagamento |
| `paid` | Pagamento confirmado pelo PSP |
| `expired` | QR Code expirou sem pagamento |
| `failed` | Cobrança rejeitada ou inconsistente |
| `refund_pending` | Reembolso necessário/em andamento |
| `refunded` | Reembolso concluído |

`expired` é um novo valor nos enums `appointment_status` e `payment_status`.

### 2.3 Regras de negócio

1. **Pix é o único método no MVP.** A UI não exibe seletor com cartão. A API aceita `payment_method = "pix"`; `credit_card` permanece no domínio, mas desabilitado por feature flag até a fase seguinte.
2. **Valor vem do servidor.** O `amount_cents` é o `session_price_cents` do profissional no momento da criação; o cliente nunca envia valor.
3. **Split 90/10 inalterado.** `platform_fee_cents = 10%` e `professional_net_cents = 90%`, calculados no servidor. O split deve ocorrer na origem pelo PSP; repasse manual/custódia é proibido.
4. **Uma cobrança por agendamento.** `POST /appointments/{id}/payment` é idempotente: se já existir cobrança Pix ativa, retorna a mesma; se estiver expirada, retorna `409`.
5. **Confirmação somente por webhook.** O cliente não pode confirmar pagamento. A rota pública `confirm-payment` deve ser removida. Eventual confirmação manual é operação administrativa auditada.
6. **Prazo de 10 minutos.** `payments.expires_at` é a fonte de verdade; a UI apenas exibe a contagem regressiva com base nesse valor.
7. **Expiração com carência de 2 minutos.** O job marca como expirado apenas após `expires_at + 2 min`, reduzindo a corrida entre pagamento e expiração. Um Pix pago dentro da carência ainda confirma o agendamento.
8. **Pagamento após liberação do horário.** Se o webhook chegar depois da expiração definitiva, o pagamento é registrado como `paid`, o agendamento permanece `expired` e o reembolso é marcado como `refund_pending`.
9. **Cancelamento antes do pagamento** libera o horário imediatamente. Cancelamento após pagamento segue a política de reembolso do ADR 0008/0007.
10. **Sem cadastro/onboarding de recebedor no fluxo do cliente.** Se o profissional não estiver apto a receber no PSP, o agendamento não pode ser criado; a UI mostra indisponibilidade.

### 2.4 Alternativas consideradas

| Alternativa | Prós | Contras | Decisão |
|---|---|---|---|
| **A. Manter pagamento simulado e "Pagar depois"** | Zero integração; entrega rápida | Não garante receita; qualquer cliente confirma sem pagar; contraria a regra de produto | Rejeitada |
| **B. Pagar no local/ao profissional** | Sem taxa de PSP | Plataforma não retém 10%; sem garantia; conciliação manual | Rejeitada |
| **C. Pix estático/chave direta** | Custo zero; simples | Sem webhook, sem valor vinculado, sem split, conciliação manual | Rejeitada |
| **D. Pix dinâmico via PSP + webhook** | Confirmação automática; split; conciliação | Dependência de PSP; exige expiração e reembolso | **Escolhida** |
| **E. Pix + cartão no MVP** | Mais opções ao cliente | Aumenta escopo PCI, integração e testes; decisão comercial pendente | Postergada (ADR 0007) |
| **F. Criar a cobrança Pix junto com o agendamento** | Menos uma chamada | Acopla criação ao PSP; falha do PSP impede a pré-reserva | Rejeitada; cobrança é criada na tela de pagamento |

---

## 3. Detalhamento Técnico

### 3.1 API

| Método | Rota | Autenticação | Comportamento |
|---|---|---|---|
| `POST` | `/api/v1/appointments` | Cliente | Cria pré-reserva `pending_payment` + pagamento `pending`; retorna `appointment` e `payment` |
| `POST` | `/api/v1/appointments/{id}/payment` | Cliente dono | Cria/reutiliza cobrança Pix; retorna QR, copia e cola e `expires_at` |
| `GET` | `/api/v1/appointments/{id}` | Cliente dono/profissional | Mostra status atualizado para polling e tela de pagamento |
| `POST` | `/api/v1/payments/webhook/{provider}` | Pública + assinatura | Valida evento, reconsulta PSP e confirma pagamento/agendamento |
| ~~`POST`~~ | ~~`/api/v1/appointments/{id}/confirm-payment`~~ | — | **Removida**; não deve existir rota de confirmação pelo cliente |

**Resposta de criação da cobrança Pix (exemplo):**

```json
{
  "payment_id": "uuid",
  "appointment_id": "uuid",
  "method": "pix",
  "status": "pending",
  "amount_cents": 15000,
  "expires_at": "2026-10-20T18:42:00Z",
  "pix": {
    "qr_code_image": "data:image/png;base64,...",
    "copy_paste": "00020126...",
    "provider_payment_id": "mp_123"
  }
}
```

- `qr_code_image` pode ser uma imagem base64 ou URL temporária do PSP.
- `copy_paste` é o payload EMV do Pix.
- A resposta nunca inclui segredo do PSP, token de acesso ou dados de cartão.

**Erros relevantes:**

| HTTP | Situação |
|---|---|
| `409` | Agendamento já pago, expirado, cancelado ou cobrança expirada |
| `403` | Agendamento pertence a outro cliente |
| `404` | Agendamento/cobrança inexistente |
| `502` | PSP indisponível ao criar/reconsultar cobrança |
| `422` | Webhook com valor, moeda ou identificador divergente |

### 3.2 Webhook

Ordem obrigatória de processamento:

1. Ler o corpo bruto e validar a assinatura/HMAC do PSP.
2. Registrar o evento em `payment_webhook_events` com `UNIQUE(provider, provider_event_id)`; duplicatas retornam `200` sem reprocessar.
3. Reconsultar a cobrança no PSP por `provider_payment_id`.
4. Validar: `payment_id`, valor, moeda `BRL`, método `pix` e status pago.
5. Em uma transação:
   - travar o agendamento com `SELECT ... FOR UPDATE`;
   - se `pending_payment` e dentro do prazo/carência, atualizar pagamento para `paid` e agendamento para `confirmed`;
   - se já `confirmed`/`paid`, não fazer nada (idempotência);
   - se `expired`, gravar `paid` e marcar `refund_pending`.
6. Responder `200` somente após commit. Em falha transitória, responder `5xx` para o PSP tentar novamente.

O webhook **não** pode confiar apenas no payload: a reconsulta ao PSP é obrigatória.

### 3.3 Expiração e liberação do horário

- Job a cada minuto (ou verificação preguiçosa nas listagens) seleciona agendamentos `pending_payment` com `reserved_until + 2 min < now()`.
- Para cada um, em transação: `appointments.status = expired`, `payments.status = expired`.
- A constraint de exclusão do PostgreSQL passa a permitir novo agendamento no mesmo horário assim que o status sai de `pending_payment`/`confirmed`.
- Um Pix pago após a expiração definitiva não reabre o horário: gera `refund_pending` e alerta operacional.
- O job deve ser idempotente e usar `FOR UPDATE SKIP LOCKED` para evitar processamento duplicado.

### 3.4 Modelo de dados (migration 0010)

Além dos campos já existentes:

```sql
ALTER TYPE appointment_status ADD VALUE IF NOT EXISTS 'expired';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'expired';

ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS provider VARCHAR(40),
    ADD COLUMN IF NOT EXISTS provider_payment_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS provider_fee_cents INTEGER,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS raw_status VARCHAR(80),
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_provider_payment
    ON payments(provider, provider_payment_id)
    WHERE provider_payment_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_payments_status_expires
    ON payments(status, expires_at);

CREATE TABLE IF NOT EXISTS payment_webhook_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider VARCHAR(40) NOT NULL,
    provider_event_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(80) NOT NULL,
    payload JSONB NOT NULL,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_event_id)
);
```

- `payments.metadata` guarda dados não sensíveis do Pix (ex.: `txid`, status bruto), nunca segredos.
- O `provider_payment_id` é a chave de correlação entre plataforma e PSP.
- A migration deve ser conciliada em `init.sql`.

### 3.5 Abstração do PSP e split

O domínio de agendamentos depende de uma interface, não do PSP:

```go
type PixProvider interface {
    CreatePix(ctx context.Context, in PixChargeInput) (PixCharge, error)
    GetPix(ctx context.Context, providerPaymentID string) (PixCharge, error)
    Refund(ctx context.Context, providerPaymentID string, amountCents int64) (Refund, error)
    ParseWebhook(r *http.Request) (WebhookEvent, error)
}
```

- Implementação inicial conforme PSP aprovado no ADR 0007 (Mercado Pago é a recomendação atual).
- `PAYMENT_PROVIDER` seleciona o adapter por ambiente.
- O split 90/10 é configurado na criação da cobrança, com recebedor previamente habilitado.
- Cartão não implementa esta interface no MVP; será adicionado em fase futura.

### 3.6 Tela de pagamento

Rota: `/app/client/pagamento/[appointmentId]`.

Elementos obrigatórios:

- Resumo: profissional, data/hora, tipo de aula, local e valor.
- Método: **Pix** (único; sem seletor de cartão).
- QR Code e botão **"Copiar código Pix"**.
- Contagem regressiva baseada em `expires_at`.
- Status: `pending`, `paid`, `expired`, `failed`.
- Ações: **"Já paguei / Atualizar status"** (força polling) e **"Voltar aos agendamentos"**.
- Estado de sucesso: confirmação e link para **Meus agendamentos**.
- Estado de expiração: mensagem clara de que o horário foi liberado e ação para agendar novamente.

**Remover:**

- botão **"Pagar depois"**;
- botão que chama `confirm-payment`;
- qualquer texto que sugira confirmação instantânea sem pagamento.

### 3.7 Meus agendamentos

- `pending_payment` exibe **"Aguardando pagamento"** e ação **"Pagar agora"** enquanto não expirar.
- `expired` exibe **"Pré-reserva expirada"** e ação **"Agendar novamente"**.
- `confirmed` exibe **"Confirmado"**.
- A listagem deve refletir o status do servidor; não inferir pagamento no cliente.

### 3.8 Segurança e antifraude

- Webhook com validação de assinatura, reconsulta obrigatória e idempotência.
- Rate limit em criação de agendamento e cobrança.
- Validação de posse do agendamento (cliente dono) em todas as rotas.
- Valor e split exclusivamente no servidor; nunca aceitar `amount_cents` do cliente.
- Não logar payload completo com dados pessoais; mascarar chaves e tokens.
- Segredos do PSP apenas em secrets/variáveis de ambiente.
- Alertas para: webhook inválido, divergência de valor, pagamento após expiração e falha de reconsulta.
- LGPD: guardar apenas o necessário para conciliação e obrigações fiscais.

---

## 4. Consequências

### Positivas

- O horário só é garantido após pagamento, alinhando produto e receita.
- Confirmação automática via PSP elimina o "confirmar pagamento" simulado.
- Liberação automática do horário reduz bloqueios indevidos.
- Split 90/10 preservado e conciliável.
- Base pronta para adicionar cartão sem reescrever o domínio.

### Negativas e riscos

| Risco | Impacto | Mitigação |
|---|---|---|
| PSP indisponível ao criar a cobrança | Alto | Retry, mensagem clara, pré-reserva preservada e feature flag de provider |
| Webhook atrasado/perdido | Alto | Reconsulta, retry do PSP, job de conciliação e alerta |
| Cliente paga após expiração | Médio | Regra de reembolso automático/`refund_pending` e comunicação na tela |
| Concorrência entre expiração e webhook | Alto | Transação com `FOR UPDATE`, carência de 2 min e idempotência |
| Divergência de valor | Alto | Validar valor/moeda na reconsulta; não confirmar e gerar alerta |
| Profissional sem recebedor habilitado | Médio | Bloquear criação do agendamento e exibir indisponibilidade |
| Taxa do PSP reduzir a margem de 10% | Médio | Revisar política de taxas no ADR 0007 e monitorar margem por transação |
| Fraude/chargeback Pix | Médio | Antifraude do PSP, limites, monitoramento e política de cancelamento |

---

## 5. Rollout e Migração

1. Aplicar a migration `0010_pix_payment.sql` (idempotente) e reconciliar `init.sql`.
2. Implementar a interface `PixProvider` e o adapter do PSP aprovado.
3. Criar endpoint de cobrança Pix e webhook.
4. Remover a rota pública `confirm-payment` e o botão "Pagar depois".
5. Atualizar a tela de pagamento e **Meus agendamentos**.
6. Expirar pré-reservas antigas sem pagamento na entrada da feature.
7. Feature flag `PIX_PAYMENTS_ENABLED` para rollout gradual e rollback.
8. Validar em sandbox: QR, cópia e cola, webhook duplicado, expiração, pagamento após expiração e reembolso.
9. Só habilitar em produção após revisão jurídica/financeira do ADR 0007.

### Critérios de aceite

- Não é possível confirmar um agendamento sem webhook pago.
- Não existe ação "Pagar depois" nem confirmação manual pelo cliente.
- Pix expirado libera o horário e permite novo agendamento.
- Webhook duplicado não altera o estado duas vezes.
- Pagamento com valor divergente não confirma o agendamento.
- Split 90/10 é registrado corretamente.
- A tela funciona em mobile e exibe QR/copia e cola.

---

## 6. Métricas

- Conversão de `pending_payment` → `confirmed`.
- Tempo médio entre criação da cobrança e pagamento.
- Taxa de expiração de pré-reservas.
- Taxa de erro na criação de cobrança e no webhook.
- Latência do webhook até a confirmação.
- Divergências de conciliação e reembolsos pendentes.
- Margem média após taxas do PSP.

---

## 7. Fora de Escopo

- Cartão de crédito, parcelamento e Pix parcelado.
- Assinaturas/recorrência.
- Carteira própria, custódia ou repasse manual.
- Boleto.
- Cupons, cashback e créditos.
- Checkout transparente com tokenização de cartão.
- App mobile nativo.

---

## 8. Pontos Abertos

- PSP definitivo, taxas e contrato comercial (ADR 0007).
- Política de reembolso em cancelamentos após pagamento.
- Necessidade de antecipação de recebíveis para o profissional.
- Prazo final de expiração e texto jurídico exibido na tela.
- Estratégia de conciliação diária e responsável operacional.

---

## 9. Referências

- [ADR 0005 — Fluxo de Agendamento de Sessão](0005-fluxo-de-agendamento-de-sessao.md)
- [ADR 0007 — Integração com Plataforma de Pagamentos](0007-integracao-plataforma-de-pagamentos.md)
- [ADR 0008 — Painel Administrativo](0008-painel-administrativo.md)
- Banco Central do Brasil — Pix e regulamentação de arranjos de pagamento.
- Documentação do PSP escolhido (cobrança Pix, webhook, split e reembolso).

---

## 10. Status da Implementação (2026-10-20)

- Migração `infra/docker/postgres/migrations/0010_pix_payment.sql` criada e aplicada no banco de desenvolvimento; `init.sql` reconciliado.
- Backend: `POST /api/v1/appointments/{id}/payment`, webhook `POST /api/v1/payments/webhook/{provider}`, job de expiração a cada 30s e remoção da rota `confirm-payment`.
- Provider inicial: `mock` (`PIX_PROVIDER=mock`), com QR Code/copia e cola sintaticamente válidos, webhook assinado por HMAC e sem liquidação financeira real. O adapter do PSP definitivo segue pendente do ADR 0007.
- Frontend: tela Pix com QR Code, copia e cola, contagem regressiva, polling e estados `confirmed`/`expired`; removido o botão "Pagar depois"; "Pagar agora" disponível em Meus agendamentos.
- Pendências: escolher/configurar o PSP real antes de produção e aplicar a migração 0010 no banco de produção.
