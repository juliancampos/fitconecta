# ADR 0007: Integração com Plataforma de Pagamentos (Pix, Cartão e Split 90/10)

* **Status:** Proposto (pendente de validação comercial e jurídica)
* **Data:** 2026-10-20
* **Decisores:** Time de Arquitetura, Produto, Financeiro e Jurídico
* **Contexto Técnico:** Go (Core API), Next.js (Web), PostgreSQL 16, PSP externo, Pix e cartão de crédito

> **Nota (2026-10-20):** o [ADR 0010](0010-pagamento-obrigatorio-pix-no-agendamento.md) define o MVP de pagamento: Pix obrigatório no agendamento, cobrança dinâmica e confirmação por webhook. Cartão permanece fora do primeiro lançamento.

---

## 1. Contexto e Problema

O **FitConecta+** é um marketplace que intermedeia sessões entre profissionais autônomos de educação física e clientes. O fluxo financeiro é parte central do produto:

1. O cliente escolhe um horário e paga pela plataforma (Pix ou cartão).
2. O agendamento só é **confirmado após a confirmação do pagamento**.
3. A plataforma retém **10%** e repassa **90%** ao profissional.
4. Cancelamentos e reembolsos precisam refletir no agendamento e no extrato.
5. O profissional precisa receber de forma automática, sem repasses manuais da plataforma.

O backend já possui o modelo de `appointments` e `payments` (valor bruto, taxa da plataforma, líquido do profissional, método e status), mas **aindo não existe integração com nenhum PSP**. É necessário decidir:

- Qual provedor usar no MVP.
- Como dividir o valor (split) entre plataforma e profissional.
- Como confirmar pagamentos com segurança (webhook).
- Como cadastrar/onboardingar os profissionais como recebedores.
- Como tratar taxas, reembolsos, expiração e conciliação.

### Restrições do projeto

- Operação **Brasil, em BRL**, com Pix e cartão de crédito.
- Volume inicial baixo (MVP), sem justificar contratos enterprise.
- Time pequeno; integração simples e tempo de mercado curto.
- Necessidade de **split nativo** (não repasse manual).
- Conformidade com **Circular BACEN 3.886/2018** (subadquirente), **LGPD** e **PCI DSS**.
- Evitar que a plataforma custodie dinheiro de terceiros ou vire instituição de pagamento.

---

## 2. Requisitos

### Funcionais

| Requisito | Detalhe |
|---|---|
| Pix | QR Code dinâmico e/ou copia e cola; confirmação em tempo real |
| Cartão | Crédito à vista (e, se possível, parcelado) |
| Split | Divisão automática 90% profissional / 10% plataforma |
| Recebedores | Cadastro/KYC do profissional como recebedor no PSP |
| Webhook | Confirmação assinada, com retry e idempotência |
| Reembolso | Total e, se possível, parcial |
| Expiração | Pré-reserva de 10 minutos; liberar slot se não pagar |
| Conciliação | IDs do PSP, taxas, status e relatórios |
| Segurança | Checkout hospedado ou tokenização para reduzir escopo PCI |

### Não funcionais

- Baixo custo por transação no volume de MVP.
- Sem mensalidade ou com mensalidade irrelevante.
- API/documentação maduras, sandbox e SDK/HTTP simples (Go).
- Disponibilidade e suporte em português.
- Facilidade de onboarding para profissionais pessoa física (CPF).
- Capacidade de trocar de provedor sem reescrever as regras de negócio.

### Regulatórios

- O PSP deve ser instituição autorizada e responsável pela liquidação.
- O split deve ocorrer na origem (subadquirente), não por custódia + TED/Pix manual.
- Dados de cartão não devem trafegar nem ser armazenados pela API.
- KYC/AML e prevenção a fraude delegados ao PSP, com status refletido na plataforma.

---

## 3. Alternativas Consideradas

### 3.1 Resumo comparativo

Tarifas públicas verificadas em **outubro/2026**. Valores podem mudar; condições negociadas por volume não estão refletidas.

| Provedor | Pix | Cartão de crédito (1x) | Split nativo | Mensalidade / mínimo | Observação |
|---|---|---|---|---|---|
| **AbacatePay** | **R$ 0,80 fixo** | 3,5% + R$ 0,60 | **Não** (em desenvolvimento) | Sem mensalidade | Ótimo DX; só CNPJ/MEI; repasse manual via Pix |
| **Adyen** | € 0,11 + contrato/volume | Interchange++ + 0,60% | **Sim** (MarketPay) | Sem mensalidade, mas **fatura mínima** por setor | Enterprise; contrato comercial; entidade local |
| **Mercado Pago** | **0,99%** | 3,99% (D+30) / 4,99% (na hora) | **Sim** (1:1; 1:N sob consulta) | Sem mensalidade | OAuth; contas MP para recebedores; checkout Pro/Bricks |
| **Asaas** | R$ 1,99 (promo R$ 0,99) | **2,99% + R$ 0,49** | **Sim** | Sem mensalidade | Subcontas; split via API; BaaS e NF |
| **Pagar.me (Stone)** | 0,99% | 4,19% | **Sim** (plano Flex, sob consulta) | Sem mensalidade | D+1; split robusto; multiadquirência |
| **Woovi** | **0,80%** (mín. R$ 0,50 / máx. R$ 5) ou R$ 0,85 | Não é o foco | **Sim** (Pix) | Sem mensalidade | Só Pix; subcontas/parceiros; ótimo custo |
| **Stripe** | 1,19% (por convite) | 3,99% + R$ 0,39 | Sim (Connect) | Sem mensalidade + R$ 6/conta ativa + 0,25% + R$ 0,67 por repasse | Global; Connect complexo no Brasil |
| **Pix direto/manual** | R$ 0 (tarifa bancária) | — | **Não** | — | Sem webhook/split; conciliação manual; risco |

### 3.2 AbacatePay

**Prós**
- Pix mais barato do mercado publicado: **R$ 0,80 fixo**, independentemente do valor.
- Sem mensalidade; cartão a 3,5% + R$ 0,60 (1x), 4,0% (2–6x) e 4,5% (7–12x).
- API simples, ótima documentação, SDKs (incluindo Go), webhooks com HMAC e retry.
- Sandbox e checkout hospedado.

**Contras**
- **Não possui split automático** (está "em desenvolvimento" segundo o próprio site).
- Só atende **CNPJ** (MEI é aceito).
- Para repassar 90% ao profissional, seria necessário receber 100% na conta da plataforma e fazer `POST /v2/pix/send` manualmente.
- Esse repasse manual cria custódia de recursos de terceiros, conciliação manual, risco regulatório e risco de erro/calote.
- Cartão fica disponível em **D+30** e exige análise.

**Veredito:** é a opção **mais barata para Pix**, mas **não atende sozinha** ao requisito de split do marketplace. Pode ser reavaliada quando o split for lançado.

### 3.3 Adyen

**Prós**
- Plataforma global, robusta, com adquirência própria e suporte a Pix e cartão.
- **Adyen for Platforms / MarketPay** resolve split, KYC, repasses e disputas em escala.
- Excelente para operação multi-país e grandes volumes.
- Sem mensalidade nem taxa de setup; preço por transação.

**Contras**
- Modelo **Interchange++** (tarifa fixa + custo do meio + 0,60% em cartões), com Pix "€ 0,11 + contrato direto".
- **Fatura mínima** dependendo do setor/modelo de negócio (não publicada).
- Contratação comercial, onboarding e integração mais pesados; normalmente exige volume e entidade local.
- Preço público em euro e pouco transparente para um MVP brasileiro.
- Complexidade desproporcional ao estágio atual.

**Veredito:** tecnicamente excelente e provavelmente competitivo **em escala**, mas **caro e complexo demais para o MVP**. Reavaliar quando houver volume/internacionalização.

### 3.4 Mercado Pago

**Prós**
- Pix a **0,99%** (confirmação imediata) — barato para tickets baixos/médios.
- Cartão a 3,99% (D+30) ou 4,99% (recebimento na hora).
- **Split nativo 1:1** via `marketplace_fee`/`application_fee`, disponível para qualquer conta de desenvolvedor que siga o fluxo OAuth; 1:N sob consulta comercial.
- Checkout Pro, Transparente e Bricks — permite reduzir escopo PCI.
- Marca forte no Brasil, alta taxa de aprovação e familiaridade dos profissionais.
- Sandbox, documentação extensa e webhooks.

**Contras**
- Cada recebedor precisa de conta Mercado Pago e autorização via **OAuth** (token de ~6 meses, precisa renovação).
- Split opera com saldo entre contas MP; transferências de instituições externas não entram no fluxo automático.
- Cartão D+30 (ou antecipação com custo).
- Modelo 1:N (vários recebedores na mesma transação) exige contato comercial.

**Veredito:** **melhor equilíbrio custo × aderência** para um marketplace brasileiro com Pix relevante e split 1:1 (plataforma + profissional).

### 3.5 Asaas

**Prós**
- **Split nativo** para Pix, boleto e cartão, com subcontas e repasses automáticos.
- Cartão à vista entre os mais baratos: **2,99% + R$ 0,49** (2–6x: 3,49%; 7–12x: 3,99%).
- Pix R$ 1,99 (promo R$ 0,99 por 3 meses para novos clientes).
- Sem mensalidade; sandbox; checkout transparente; BaaS; emissão de NF.
- Split calculado sobre o valor líquido, com regras por percentual ou valor fixo.

**Contras**
- Pix fixo de R$ 1,99 fica caro em tickets baixos (ex.: 4% em uma sessão de R$ 50).
- Split é exclusivo via API e **todos os envolvidos precisam ter conta Asaas**.
- Onboarding de subcontas pode ser burocrático para autônomos.

**Veredito:** **melhor opção se o cartão for relevante** ou se a plataforma quiser subcontas/BaaS/NF. É o principal **fallback** do Mercado Pago.

### 3.6 Pagar.me (Stone)

**Prós**
- Pix 0,99%; cartão 4,19% (1x); recebimento D+1.
- Split robusto (multiadquirência, estornos automáticos, antecipação por seller), referência para marketplaces.
- API madura e documentação técnica detalhada.

**Contras**
- Split disponível no plano **Flex**, com **taxas sob consulta** (não públicas).
- Cartão mais caro que Asaas/MP.
- Exige negociação comercial para o recurso central.

**Veredito:** forte candidato **em escala**, quando o volume justificar negociar Flex. Não é a melhor relação custo/esforço no MVP.

### 3.7 Woovi

**Prós**
- Pix muito barato: **0,80%** (mín. R$ 0,50, máx. R$ 5) ou R$ 0,85 fixo.
- **Split Pix nativo** (subcontas/parceiros), com divisão imediata.
- Sem mensalidade no plano padrão.

**Contras**
- **Não é focado em cartão**; não resolve o requisito de cartão sozinho.
- Exigiria um segundo PSP para cartão, aumentando integrações e conciliação.
- BaaS (contas digitais próprias) tem setup de R$ 20.000 — fora do escopo do MVP.

**Veredito:** excelente **otimização futura para Pix** ou trilho secundário; não é a escolha única do MVP por falta de cartão.

### 3.8 Stripe

**Prós**
- API e DX de referência mundial; Connect resolve split/marketplace.
- Cartão 3,99% + R$ 0,39; bom para operação internacional.

**Contras**
- **Pix por convite** no Brasil (1,19%).
- Connect tem custo adicional (R$ 6/conta ativa/mês + 0,25% + R$ 0,67 por repasse, no modelo em que a plataforma define preços).
- Complexidade e limitações de payout local no Brasil.

**Veredito:** não é a melhor relação custo/simplicidade para o MVP brasileiro; faz sentido se houver expansão internacional.

### 3.9 Outras formas de solicitar pagamento (sem PSP de split)

| Forma | Custo | Vantagens | Riscos / limitações | Recomendação |
|---|---|---|---|---|
| **Pix direto (chave/QR estático)** | ~R$ 0 | Sem taxa de gateway; instantâneo | Sem webhook, sem split, sem conciliação automática; cliente pode pagar valor errado; plataforma não retém 10% automaticamente | Apenas contingência de piloto |
| **Pix Cobrança via PSP** | R$ 0,80–1,99 ou 0,80–0,99% | QR dinâmico, webhook, split | Depende do PSP | **Trilho recomendado** |
| **Checkout hospedado / link** | Taxa do PSP | Integração mais rápida; menor escopo PCI | Menos controle de UX | Bom para o MVP |
| **Checkout transparente/Bricks** | Taxa do PSP | Melhor UX; dados tokenizados | Mais integração; exige cuidado PCI | Evolução do MVP |
| **Boleto** | R$ 1,99–3,49 | Baixo custo; populações sem cartão | Compensação D+1/D+2; alto abandono; conflita com confirmação imediata | Não usar como principal |
| **Pix Automático** | Taxa do PSP | Recorrência/assinaturas | Não é o caso de uso inicial | Fase futura |
| **Pix parcelado** | Taxa do PSP | Tickets altos | Suporte varia por PSP | Fase futura |
| **Carteira/escrow próprio** | — | Controle total | Vira PSP de fato; risco regulatório e operacional | **Evitar** |
| **Cripto/stablecoin** | Variável | Liquidação internacional | Volatilidade, regulação, UX | Não recomendado |
| **Dinheiro/na entrega** | — | Sem taxa | Não se aplica a marketplace online | Não se aplica |

> **Conclusão desta seção:** a forma tecnicamente correta de "solicitar pagamento" no MVP é **Pix Cobrança (QR dinâmico) + cartão via checkout hospedado/tokenizado**, com **split nativo** no PSP. Pix direto e repasse manual são apenas contingência.

---

## 4. Análise de Custos

### Premissas

- Ticket de sessão: **R$ 100** (cenário base), com sensibilidade para R$ 50 e R$ 200.
- Cartão à vista (1x); sem antecipação; sem parcelamento.
- Tarifas públicas de outubro/2026; não incluem impostos, antecipação, chargeback ou negociação por volume.
- Adyen não tem preço fechado público para Pix/cartão no Brasil; fica fora da comparação numérica direta.

### Custo por transação — R$ 100

| Provedor | Pix | % efetiva | Cartão 1x | % efetiva |
|---|---:|---:|---:|---:|
| AbacatePay | R$ 0,80 | 0,80% | R$ 4,10 | 4,10% |
| Woovi | R$ 0,80 | 0,80% | — | — |
| Mercado Pago | R$ 0,99 | 0,99% | R$ 3,99 (D+30) | 3,99% |
| Pagar.me | R$ 0,99 | 0,99% | R$ 4,19 | 4,19% |
| Stripe | R$ 1,19 | 1,19% | R$ 4,38 | 4,38% |
| Asaas | R$ 1,99 | 1,99% | **R$ 3,48** | **3,48%** |
| Adyen | € 0,11 + contrato | sob consulta | Interchange++ + 0,60% | sob consulta |

### Sensibilidade por ticket — Pix

| Ticket | AbacatePay (R$ 0,80) | Woovi (0,80%, mín. R$ 0,50) | Mercado Pago (0,99%) | Pagar.me (0,99%) | Asaas (R$ 1,99) |
|---:|---:|---:|---:|---:|---:|
| R$ 50 | R$ 0,80 (1,60%) | R$ 0,50 (1,00%) | R$ 0,50 (0,99%) | R$ 0,50 (0,99%) | R$ 1,99 (3,98%) |
| R$ 100 | R$ 0,80 (0,80%) | R$ 0,80 (0,80%) | R$ 0,99 (0,99%) | R$ 0,99 (0,99%) | R$ 1,99 (1,99%) |
| R$ 200 | R$ 0,80 (0,40%) | R$ 1,60 (0,80%) | R$ 1,98 (0,99%) | R$ 1,98 (0,99%) | R$ 1,99 (1,00%) |
| R$ 300 | R$ 0,80 (0,27%) | R$ 2,40 (0,80%) | R$ 2,97 (0,99%) | R$ 2,97 (0,99%) | **R$ 1,99 (0,66%)** |

**Leitura:** taxas fixas (AbacatePay/Asaas) favorecem tickets altos; percentuais (MP/Pagar.me/Woovi) favorecem tickets baixos. O **ponto de equilíbrio MP × Asaas** no Pix é ~R$ 201 (abaixo disso, MP; acima, Asaas).

### Sensibilidade por ticket — Cartão 1x

| Ticket | AbacatePay | Mercado Pago (D+30) | Pagar.me | Stripe | Asaas |
|---:|---:|---:|---:|---:|---:|
| R$ 50 | R$ 2,35 (4,70%) | R$ 2,00 (3,99%) | R$ 2,10 (4,19%) | R$ 2,39 (4,77%) | **R$ 1,99 (3,97%)** |
| R$ 100 | R$ 4,10 (4,10%) | R$ 3,99 (3,99%) | R$ 4,19 (4,19%) | R$ 4,38 (4,38%) | **R$ 3,48 (3,48%)** |
| R$ 200 | R$ 7,60 (3,80%) | R$ 7,98 (3,99%) | R$ 8,38 (4,19%) | R$ 8,37 (4,19%) | **R$ 6,47 (3,24%)** |

**Leitura:** no cartão, **Asaas é o mais barato** em todos os tickets analisados. AbacatePay só ganha do MP em tickets muito altos (acima de ~R$ 333, onde 3,5% + 0,60 fica abaixo de 3,99%).

### Exemplo mensal — 1.000 sessões de R$ 100 (70% Pix / 30% cartão)

| Provedor | Custo Pix (700) | Custo cartão (300) | **Total/mês** |
|---|---:|---:|---:|
| AbacatePay | R$ 560 | R$ 1.230 | R$ 1.790 *(sem split)* |
| Woovi + Asaas (cartão) | R$ 560 | R$ 1.044 | R$ 1.604 *(duas integrações)* |
| **Mercado Pago** | R$ 693 | R$ 1.197 | **R$ 1.890** |
| Pagar.me | R$ 693 | R$ 1.257 | R$ 1.950 *(Flex sob consulta)* |
| Stripe | R$ 833 | R$ 1.314 | R$ 2.147 *(+ Connect)* |
| Asaas | R$ 1.393 | R$ 1.044 | R$ 2.437 |

**Conclusão:** entre os provedores **com split nativo e cartão**, o **Mercado Pago é o mais barato** no mix Pix-heavy; o **Asaas** vence quando o cartão cresce ou o ticket sobe. AbacatePay é o mais barato no Pix, mas não tem split — o que o desqualifica como escolha única.

---

## 5. Análise Qualitativa

| Critério | AbacatePay | Adyen | Mercado Pago | Asaas | Pagar.me | Woovi | Stripe |
|---|---|---|---|---|---|---|---|
| Split nativo | Não | Sim | Sim (1:1) | Sim | Sim (Flex) | Sim (Pix) | Sim (Connect) |
| Pix | Sim | Sim | Sim | Sim | Sim | Sim | Por convite |
| Cartão | Sim | Sim | Sim | Sim | Sim | Não | Sim |
| Onboarding do profissional | Manual | KYC enterprise | OAuth/conta MP | Subconta Asaas | Recipient/flex | Subconta | Connect |
| Esforço de integração | Baixo | Alto | Médio | Médio | Médio/alto | Baixo | Médio/alto |
| Sandbox | Sim | Sim | Sim | Sim | Sim | Sim | Sim |
| Escopo PCI | Checkout | Checkout | Checkout/Bricks | Checkout | Checkout | Pix | Checkout |
| Custo no MVP | **Baixo (Pix)** | **Alto/incerto** | **Baixo** | **Médio** | Médio | **Baixo (Pix)** | Médio/alto |
| Adequação ao MVP | Parcial | Baixa | **Alta** | **Alta** | Média | Parcial | Baixa |

**Pontos decisivos**

1. **Split é requisito de arquitetura, não diferencial.** Sem split, a plataforma precisaria custodiar o valor e repassar manualmente — o que contraria a recomendação regulatória e adiciona risco operacional.
2. **Pix é o meio mais provável no público brasileiro.** Uma diferença de R$ 1,00 por transação importa no volume, mas menos do que o custo de operar repasses manuais.
3. **Cartão exige prevenção a fraude e PCI.** Checkout hospedado/Bricks reduz escopo e acelera o MVP.
4. **Onboarding do profissional** é tão importante quanto a taxa: OAuth (MP) ou subconta (Asaas) precisa ser simples e rastreável.

---

## 6. Decisão

### Recomendação principal

Adotar o **Mercado Pago** como PSP principal do MVP, com **Asaas** como alternativa/fallback, atrás de uma camada de abstração de provedor.

**Motivos**

- Melhor custo total no cenário mais provável do MVP (Pix-heavy, ticket ~R$ 100).
- **Split nativo 1:1** disponível sem negociação comercial (plataforma + profissional).
- Pix a 0,99%, ampla adoção e familiaridade dos profissionais.
- Checkout Pro/Bricks reduz escopo PCI e acelera a entrega.
- Sandbox e documentação maduros; webhooks e reembolsos suportados.

### Alternativa recomendada

**Asaas** se:

- a participação de cartão crescer (cartão 2,99% + R$ 0,49 é o mais barato);
- o ticket médio subir acima de ~R$ 200 (Pix fixo de R$ 1,99 fica competitivo);
- houver necessidade de subcontas, BaaS, emissão de NF ou um backoffice financeiro mais completo.

### Provedores não selecionados (e gatilhos de reavaliação)

| Provedor | Motivo da não seleção | Quando reavaliar |
|---|---|---|
| **AbacatePay** | Sem split automático (em desenvolvimento); repasse manual traria custódia e risco regulatório | Quando o split for lançado e homologado; potencial economia de R$ 1,19/Pix vs MP em tickets altos |
| **Adyen** | Enterprise, fatura mínima, contrato comercial, preço pouco transparente; complexidade desproporcional | Volume relevante, operação multi-país ou exigência de adquirência própria |
| **Pagar.me** | Split apenas no Flex (sob consulta); cartão 4,19% | Volume que justifique negociar Flex/multiadquirência |
| **Woovi** | Sem cartão; exigiria segundo PSP | Otimização de custo de Pix em alto volume, como trilho secundário |
| **Stripe** | Pix por convite; Connect com custos e complexidade no Brasil | Expansão internacional ou sellers fora do Brasil |
| **Pix direto/manual** | Sem webhook/split; conciliação e custódia manuais | Apenas contingência de piloto com poucos profissionais |

### Se a decisão fosse restrita a AbacatePay × Adyen

- **AbacatePay** vence em **custo e simplicidade** (Pix R$ 0,80, sem mensalidade, ótimo DX), mas **não atende ao split** — exigiria repasse manual, o que não é aceitável como arquitetura definitiva.
- **Adyen** atende ao split e à escala, mas é **caro e complexo** para o estágio atual (fatura mínima, contrato enterprise).
- **Portanto, nenhum dos dois isoladamente é a melhor escolha para o MVP.** A melhor relação custo × requisito é o **Mercado Pago** (com Asaas como fallback); a AbacatePay deve ser reavaliada quando o split estiver disponível.

---

## 7. Política de Taxas e Base de Cálculo do Split

Este ponto precisa ser decidido com Produto/Financeiro antes da implementação, pois afeta a promessa "90% para o profissional".

### Opção A — Profissional recebe 90% do valor bruto (plataforma absorve a taxa)

- Profissional: `round(0,90 × valor_bruto)`.
- Plataforma: `valor_líquido_recebido_pelo_PSP − repasse_ao_profissional`.
- Vantagem: cumpre literalmente a promessa de 90%.
- Desvantagem: em tickets muito baixos, a taxa do PSP pode consumir toda a comissão de 10% (ou gerar margem negativa); exige ticket mínimo ou renegociação.

### Opção B — Profissional recebe 90% do valor líquido (após taxas do PSP)

- Profissional: `round(0,90 × (valor_bruto − taxa_PSP))`.
- Plataforma: `round(0,10 × (valor_bruto − taxa_PSP))`.
- Vantagem: simples, previsível e é o comportamento natural do split sobre o líquido (Asaas/MP).
- Desvantagem: o profissional recebe um pouco menos que 90% do valor anunciado; é preciso comunicar "90% líquido de taxas".

**Recomendação para o MVP:** **Opção B**, com transparência no extrato ("valor bruto − taxa de pagamento = líquido; 90% para você"). Se o produto exigir a Opção A, implementar split por **valor fixo** (quando o PSP permitir) e definir **ticket mínimo** (ex.: R$ 40) para preservar margem. Validar com Jurídico/Financeiro.

> **Atenção:** em ambos os casos, o PSP desconta a taxa antes do split. O `payments.professional_net_cents` deve refletir o valor efetivamente repassado, e `platform_fee_cents` a comissão efetiva da plataforma — não apenas 10% nominais.

---

## 8. Fluxo de Pagamento Proposto

```
Cliente                     Core API                        PSP                     Profissional
  │                            │                              │                          │
  │ 1. POST /appointments      │                              │                          │
  │───────────────────────────>│ cria appointment             │                          │
  │                            │ status=pending_payment       │                          │
  │                            │ expires_at=+10min            │                          │
  │                            │                              │                          │
  │ 2. POST /appointments/{id}/payment                        │                          │
  │    {method: pix|card}      │                              │                          │
  │───────────────────────────>│ cria cobrança (idempotente)  │                          │
  │                            │─────────────────────────────>│                          │
  │                            │<─────────────────────────────│                          │
  │ 3. QR Code / link          │  provider_payment_id         │                          │
  │<───────────────────────────│                              │                          │
  │                            │                              │                          │
  │ 4. paga                    │                              │                          │
  │──────────────────────────────────────────────────────────>│                          │
  │                            │ 5. webhook assinado          │                          │
  │                            │<─────────────────────────────│                          │
  │                            │ valida + idempotência        │                          │
  │                            │ reconsulta status            │                          │
  │                            │─────────────────────────────>│                          │
  │                            │ atualiza payments=paid       │                          │
  │                            │ appointment=confirmed        │                          │
  │                            │ split 90/10 na origem        │                          │
  │                            │─────────────────────────────────────────────────────────>│
  │ 6. confirmação na tela     │                              │                          │
  │<───────────────────────────│                              │                          │
```

### Regras

1. O cliente **nunca** confirma o pagamento diretamente. O endpoint `confirm-payment` deve ser interno (webhook) ou restrito a admin.
2. Webhook: validar assinatura, usar `provider_event_id` único (idempotência) e **reconsultar** o pagamento no PSP antes de confirmar.
3. Expiração: job/cron (ou verificação preguiçosa) cancela `pending_payment` vencido e libera o slot.
4. Reembolso: chamar o PSP e atualizar `payments.status` e `appointments.status` de forma transacional.
5. Split: sempre na origem; nunca custódia + repasse manual.
6. Conciliação: job diário comparando `payments` com o extrato do PSP; divergências geram alerta.

---

## 9. Camada de Abstração (Go)

Para permitir trocar de provedor sem reescrever regras de negócio:

```go
type Provider interface {
    CreatePix(ctx context.Context, in ChargeInput) (Charge, error)
    CreateCard(ctx context.Context, in ChargeInput) (Charge, error)
    Get(ctx context.Context, providerID string) (Charge, error)
    Refund(ctx context.Context, providerID string, amountCents int64) (Refund, error)
    EnsureRecipient(ctx context.Context, in RecipientInput) (Recipient, error)
    ParseWebhook(r *http.Request) (Event, error)
}
```

Implementações: `internal/payments/mercadopago`, `internal/payments/asaas`, e futuramente `abacatepay`/`adyen`. Seleção por `PAYMENT_PROVIDER` (env). O domínio (`appointments`) depende da interface, não do PSP.

### Ajustes de modelo de dados sugeridos

- `payment_recipients`: `user_id`, `provider`, `provider_account_id`, `status`, `kyc_status`.
- `payments`: adicionar `provider`, `provider_payment_id`, `provider_fee_cents`, `net_amount_cents`, `expires_at`, `raw_status`, `metadata` (JSONB).
- `payment_webhook_events`: `provider`, `provider_event_id` (único), `type`, `payload` (JSONB), `processed_at`.
- `payment_refunds`: `payment_id`, `provider_refund_id`, `amount_cents`, `status`.
- Enum `payment_status`: adicionar `processing` e `expired`.

### Endpoints sugeridos

- `POST /api/v1/appointments/{id}/payment` — cria cobrança Pix/cartão.
- `GET /api/v1/payments/{id}` — status e instruções (QR/link).
- `POST /api/v1/payments/webhook/{provider}` — público, validação por assinatura.
- `POST /api/v1/payments/{id}/refund` — admin/profissional conforme regra.
- `POST /api/v1/payments/recipients/onboarding` — inicia vínculo do profissional.
- `GET /api/v1/payments/recipients/me` — status KYC/vínculo.

---

## 10. Consequências

### Positivas

- Split automático na origem, sem custódia e sem repasse manual.
- Menor risco regulatório e operacional.
- Custo competitivo no cenário Pix-heavy do MVP.
- Menor escopo PCI com checkout hospedado/Bricks.
- Base preparada para trocar de PSP (abstração + feature flag).

### Negativas

- Dependência de OAuth e renovação de token (Mercado Pago).
- Onboarding de profissionais como recebedores (KYC) adiciona atrito.
- Cartão com liquidação D+30 (impacto no fluxo de caixa; antecipação tem custo).
- Taxas percentuais podem ficar caras em tickets altos; taxas fixas, em tickets baixos.
- Necessidade de conciliação e monitoramento contínuos.

### Mitigações

- Job de renovação proativa do OAuth (MP) e alerta de expiração.
- Checklist de onboarding e status visível no painel do profissional.
- Dashboard de taxas e repasses; conciliação diária.
- Definir ticket mínimo e política de taxas (Seção 7).
- Feature flag `PAYMENT_PROVIDER` para rollback/fallback.
- Testes de contrato com sandbox e testes E2E de webhook/idempotência.

---

## 11. Plano de Implementação

### Fase 0 — Decisão e preparação

1. Validar este ADR com Produto/Financeiro/Jurídico.
2. Criar conta PJ no Mercado Pago, aplicação no Developers e sandbox.
3. Definir política de taxas (Opção A ou B) e ticket mínimo.
4. Revisar termos de uso, privacidade e consentimento (LGPD).

### Fase 1 — Pix + split (MVP)

1. Implementar interface `Provider` e adapter Mercado Pago.
2. Criar cobrança Pix ao confirmar o agendamento (pré-reserva).
3. Webhook assinado + idempotência + reconsulta.
4. Confirmar agendamento e registrar split/repasses.
5. Expiração de pré-reserva e liberação do slot.
6. Testes: unitários (valores/split), integração (sandbox), E2E (pix simulado).

### Fase 2 — Cartão e operação

1. Checkout Pro/Bricks para cartão (menor escopo PCI).
2. Reembolso total/parcial e cancelamento.
3. Conciliação diária e relatórios de taxas.
4. Painel financeiro do profissional (bruto, taxa, líquido, repasse).

### Fase 3 — Resiliência e escala

1. Adapter Asaas + feature flag para fallback.
2. Métricas e alertas (aprovação, conversão, webhooks, divergências).
3. Reavaliar AbacatePay (split), Pagar.me (Flex) e Adyen (volume/internacional).

---

## 12. Riscos e Mitigação

| Risco | Impacto | Mitigação |
|---|---|---|
| AbacatePay sem split | Alto | Não usar como PSP único; reavaliar quando lançar |
| Token OAuth do MP expirar | Alto | Job de renovação e alerta; fallback Asaas |
| Webhook forjado/replay | Crítico | Validar assinatura, idempotência, reconsulta ao PSP |
| Pagamento duplicado | Alto | Idempotency key + constraint no banco |
| Taxa do PSP > comissão de 10% | Médio | Ticket mínimo; política de taxas; monitorar margem |
| Chargeback/estorno | Médio | Regras de `liable` no split; antifraude do PSP; Política de cancelamento |
| Onboarding/KYC do profissional travar | Médio | Checklist, suporte e status claro; permitir modo pendente |
| D+30 no cartão | Médio | Antecipação opcional (com custo) ou incentivo a Pix |
| Lock-in de PSP | Médio | Interface `Provider` + feature flag + contrato sem exclusividade |
| Divergência de conciliação | Médio | Job diário, logs e alerta; guardar IDs e payloads |
| Regulatório (custódia/split) | Crítico | Split na origem via PSP autorizado; revisão jurídica |
| Indisponibilidade do PSP | Alto | Retry, fallback e monitoramento; comunicação ao usuário |

---

## 13. Métricas de Sucesso

- Taxa de aprovação por método (Pix/cartão).
- Conversão do checkout (cobrança criada → paga).
- Tempo entre pagamento e confirmação do agendamento (< 5 s via webhook).
- Percentual de splits liquidados sem intervenção manual (> 99%).
- Divergências de conciliação por mês (meta: 0).
- Custo do PSP como % do GMV.
- Taxa de chargeback e de reembolso.
- Tempo de onboarding do profissional como recebedor.
- Taxa de expiração de pré-reserva (10 min).

---

## 14. Referências

Verificadas em **2026-10-20**; tarifas e condições podem mudar.

### Provedores

- AbacatePay — Preços e taxas: https://www.abacatepay.com/pricing
- AbacatePay — Marketplaces/split: https://www.abacatepay.com/para/marketplaces
- AbacatePay — Documentação de taxas: https://docs.abacatepay.com/pages/concepts/taxas
- Adyen — Preços: https://www.adyen.com/pt_BR/precos
- Adyen — Pix: https://www.adyen.com/pt_BR/metodos-pagamento/pix
- Adyen — Cartões/Elo: https://www.adyen.com/payment-methods/elo
- Mercado Pago — Split de pagamento: https://www.mercadopago.com.br/blog/split-de-pagamento-checkout-divisao-comissoes
- Mercado Pago — Split para serviços: https://www.mercadopago.com.br/blog/split-pagamento-marketplace-servicos-beleza
- Mercado Pago — Taxas: https://www.mercadopago.com.br/ajuda/tabela-taxas-tarifas_45243
- Mercado Pago — Custos de venda online: https://www.mercadopago.com.br/blog/quanto-custa-vender-on-line-com-mercado-pago
- Asaas — Preços e taxas: https://www.asaas.com/precos-e-taxas
- Asaas — Regras de split: https://central.ajuda.asaas.com/hc/pt-br/articles/55727191439003
- Asaas — Split de pagamentos: https://blog.asaas.com/split-de-pagamento/
- Pagar.me — Ofertas e planos: https://www.pagar.me/ofertas
- Pagar.me — Split/marketplace: https://pagarme.helpjuice.com/p1-funcionalidades/marketplace-criando-uma-transa%C3%A7%C3%A3o-com-split
- Woovi — Planos e preços: https://woovi.com/planos-e-precos/
- Woovi — Split: https://developers.woovi.com/docs/splits/split-introduction
- Stripe — Preços Brasil: https://stripe.com/br/pricing
- Stripe — Pix: https://docs.stripe.com/payments/pix
- Stripe — Connect: https://stripe.com/br/connect/pricing

### Regulação e segurança

- Banco Central — Consulta de normas (Circular 3.886/2018): https://www.bcb.gov.br/estabilidadefinanceira/buscanormas
- Banco Central — Pix: https://www.bcb.gov.br/estabilidadefinanceira/pix
- LGPD — Lei 13.709/2018: https://www.planalto.gov.br/ccivil_03/_ato2015-2018/2018/lei/l13709.htm
- PCI Security Standards Council: https://www.pcisecuritystandards.org/

### Documentos internos

- [ADR 0001 — Modelagem do banco](0001-estrutura-do-banco-de-dados.md)
- [ADR 0005 — Fluxo de agendamento](0005-fluxo-de-agendamento-de-sessao.md)
- [README do projeto](../../readme.md)
