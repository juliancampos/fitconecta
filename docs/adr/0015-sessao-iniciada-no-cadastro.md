# ADR 0015: Sessão Iniciada no Cadastro (Redirecionamento Logado para a Home)

* **Status:** Aceito
* **Data:** 2026-11-10
* **Decisores:** Time de Arquitetura e Engenharia FitConecta+
* **Contexto Técnico:** Next.js (Web), Go (Core API), cookie HttpOnly `fitconecta_jwt`

---

## 1. Contexto e Problema

O backend **já inicia a sessão no cadastro**: `POST /api/v1/auth/register` responde `201` com `{token, user (com role), profile, expires_in}` **e** grava o cookie HttpOnly `fitconecta_jwt` (`setAuthCookie` no handler Register, `backend/core-api/internal/auth/handler.go` — o mesmo cookie do login). O usuário é criado com status `active` (`backend/core-api/internal/auth/repository.go`, `VALUES (..., 'active')`), portanto a sessão é utilizável imediatamente.

O frontend, porém, **descarta** essa sessão: o `handleSubmit` de `frontend/app/auth/register/page.tsx` chama o `apiPost` do registro (linha 290), ignora a resposta e faz `router.push("/auth/login")` (linha 320), forçando o usuário a digitar as credenciais uma segunda vez logo após tê-las criado.

A tela de login já implementa exatamente o redirecionamento desejado (`frontend/app/auth/login/page.tsx`, linhas 57–58): lê `res.data?.user.role` e faz `router.push(role === "professional" ? "/app/professional" : "/app/client")`. A "home" do cliente é a vitrine (`/app/client`, `HomeView`); a do profissional é sua área (`/app/professional`).

O transporte já suporta o fluxo: `apiPost` usa `credentials: "include"` (`frontend/lib/api.ts`), então o cookie do cadastro fica gravado no domínio do frontend via proxy same-origin (`frontend/proxy.ts`) — que também é o guard: sem o cookie, qualquer `/app/*` volta para `/auth/login`.

O cadastro público só aceita roles `client` e `professional` (validação em `backend/core-api/internal/auth/service.go`); admin tem login próprio e não é afetado. Não há verificação de e-mail no produto, então nada impede a sessão imediata.

### Requisito

Ao finalizar o cadastro, o usuário já deve ser redirecionado **logado** para a home — sem passar pela tela de login.

---

## 2. Decisão

**Mudança apenas no frontend, em um único arquivo: `frontend/app/auth/register/page.tsx`.** No `handleSubmit`, capturar a resposta do `apiPost` (tipo `ApiEnvelope` com `user.role`, o mesmo formato já consumido pelo login) e redirecionar espelhando o login: `role === "professional" ? "/app/professional" : "/app/client"`.

Nenhuma mudança de backend, contrato (`backend/core-api/openapi.yaml` já documenta token + cookie no `register`), migração ou proxy. O fluxo de sessão existente permanece intacto: cookie HttpOnly para web, token no corpo para app mobile.

Degradação graciosa: se o cookie não for gravado (navegador bloqueando), o guard do `proxy.ts` devolve o usuário para `/auth/login` — exatamente o comportamento de hoje, sem erro novo.

### 2.1 Alternativas consideradas

| Alternativa | Prós | Contras | Decisão |
|---|---|---|---|
| **A. Chamar `POST /auth/login` logo após o cadastro** | Sessão idêntica à do login | Requisição redundante: o cadastro já grava a sessão (token + cookie) | Rejeitada |
| **B. Redirecionar todos para `/app/client`** | Uma linha, sem leitura de papel | Profissional deve cair na própria área; quebraria a consistência com o login | Rejeitada |
| **C. Mudar o backend** (ex.: sinal de "auto-login" na resposta) | Permitiria lógica nova no servidor | O endpoint já devolve token, cookie e `user.role`; nada a fazer | Rejeitada |
| **D. Exigir confirmação de e-mail antes da sessão** | Reduz contas com e-mail inválido | Não existe verificação de e-mail no produto; fora do escopo | Rejeitada (aditiva, se o produto pedir) |
| **E. Redirect server-side (302 no cadastro)** | Sem JS no redirecionamento | O cadastro é formulário client-side; `router.push` é o padrão do projeto (login) | Rejeitada |

---

## 3. Contrato e Erros

**Nenhuma mudança de contrato.** O frontend passa a consumir o que o backend já entrega:

| Situação | Status HTTP | Comportamento do frontend após este ADR |
|---|---|---|
| Cadastro concluído (`role=client`) | 201 | `router.push("/app/client")` — sessão já ativa pelo cookie |
| Cadastro concluído (`role=professional`) | 201 | `router.push("/app/professional")` — sessão já ativa pelo cookie |
| Cadastro com dados inválidos / e-mail duplicado | 400/409 | Sem redirecionamento; mensagem de erro tratada como hoje (`apiErrorMessage`) |
| Cookie não gravado no navegador | — | Guard do `proxy.ts` devolve para `/auth/login` na primeira rota protegida (degradação graciosa) |

O campo lido (`res.data.user.role`) é o mesmo que o login usa — mesma fonte de confiança, mesma shape de `ApiEnvelope`. Papéis fora de `client`/`professional` não existem no cadastro público (validação server-side em `auth/service.go`), então o fallback `: "/app/client"` só cobre o papel `client`.

---

## 4. Implementação

Arquivo único: **`frontend/app/auth/register/page.tsx`** (~4 linhas no `handleSubmit`):

- Tipar a resposta do `apiPost` de registro como `ApiEnvelope<{ user: { role: string } }>` (mesmo padrão do login).
- Ler `res.data?.user.role`.
- Trocar `router.push("/auth/login")` (linha 320) por `router.push(role === "professional" ? "/app/professional" : "/app/client")`.

Sem mudanças em: backend, `openapi.yaml`, migrações, `proxy.ts`, `frontend/lib/api.ts`.

**Validação:** `npm run lint && npm run build` em `frontend/` (não há infra de teste no frontend — só eslint + build).

---

## 5. Privacidade e Segurança

1. **Papel lido da resposta da API**, nunca do formulário — o servidor continua sendo a fonte de verdade do papel (mesma fonte já usada pela tela de login).
2. **Sessão continua HttpOnly:** o frontend não manipula o token em JS; o cookie é gravado pelo backend e conferido pelo guard do proxy. Nada muda no transporte.
3. **Admin não é afetado:** o cadastro público só cria `client`/`professional`; o admin tem login próprio.
4. **Falha não autentica:** em erro de cadastro (400/409) a tela não redireciona e nenhum cookie foi gravado.

---

## 6. Riscos e Dívidas

- **Cookie não gravado** (navegador bloqueando cookies): o guard do `proxy.ts` devolve para `/auth/login` — degradação graciosa, sem erro novo; o usuário volta ao comportamento de hoje.
- **Papel lido da resposta da API:** mesma fonte de confiança já usada pelo login; sem risco novo.
- **Mensagens de erro do cadastro** continuam tratadas como hoje (`apiErrorMessage`, sem redirecionamento em falha).
- **Sem teste automatizado no frontend:** coberto por lint + build; o fluxo é idêntico ao do login, já exercitado em produção.
- **Verificação de e-mail inexistente:** cadastro já loga o e-mail recém-digitado sem confirmação — decisão de produto vigente, não introduzida por este ADR.

---

## 7. Referências

- `backend/core-api/internal/auth/handler.go` — `setAuthCookie` no Register (sessão já iniciada no cadastro).
- `backend/core-api/internal/auth/repository.go` — criação do usuário com status `active`.
- `backend/core-api/internal/auth/service.go` — cadastro público restrito a `client`/`professional`.
- `backend/core-api/openapi.yaml` — contrato de `POST /auth/register` (token + cookie já documentados).
- `frontend/app/auth/register/page.tsx` — `handleSubmit` atual (resposta descartada, push para `/auth/login`).
- `frontend/app/auth/login/page.tsx` — padrão do redirecionamento por papel a espelhar.
- `frontend/lib/api.ts` — `apiPost` com `credentials: "include"` e shape de `ApiEnvelope`.
- `frontend/proxy.ts` — proxy same-origin e guard de sessão (`/app/*` → `/auth/login` sem cookie).
