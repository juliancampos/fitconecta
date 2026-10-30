# ADR 0009: Painel Administrativo no Mesmo Domínio em `/admin`

* **Status:** Aceito
* **Data:** 2026-10-20
* **Decisores:** Produto e Engenharia FitConecta+
* **Substitui parcialmente:** [ADR 0008](0008-painel-administrativo.md) (separação de host/app)

---

## 1. Contexto

O ADR 0008 definiu o painel administrativo como um app Next.js separado, publicado em `admin-fit-conecta.fly.dev`/`admin.fitconecta.com.br`. A operação passou a preferir uma única URL pública, com o painel acessível em `/admin`, por exemplo:

- `http://localhost:3000/admin`
- `https://frontend-fit-conecta.fly.dev/admin`

O login administrativo continua precisando ser distinto do login de clientes e profissionais.

## 2. Decisão

- O painel administrativo passa a fazer parte do **mesmo app Next.js do frontend público**, nas rotas `/admin/*`.
- `/admin/login` é uma tela exclusiva do painel, com textos, layout e cookie próprios.
- A sessão administrativa continua isolada:
  - cookie host-only `fitconecta_admin_jwt`;
  - segredo `ADMIN_JWT_SECRET`;
  - audience `fitconecta-admin`;
  - validade padrão de 8 horas.
- O proxy same-origin do frontend:
  - protege `/admin/*`, redirecionando para `/admin/login` sem o cookie administrativo;
  - encaminha `/api/v1/admin/*` para a Core API;
  - mantém as demais rotas `/api/*` com o comportamento existente.
- O backend continua sendo uma única Core API. Não há BFF nem app administrativo separado.
- As páginas administrativas recebem `noindex, nofollow`.

## 3. Consequências

### Positivas

- Um único domínio e um único deploy de frontend.
- Navegação simples: o painel fica em `/admin` sem servidor adicional.
- O cookie administrativo é gravado no mesmo host usado pelo navegador, sem problemas de cookie cross-site.
- Menos configuração de CORS, secrets e observabilidade.

### Negativas

- O host público também expõe os endpoints `/api/v1/admin/*`; a proteção efetiva continua no backend (JWT administrativo, audience, role e status).
- A separação deixa de ser por hostname e passa a ser por rota, cookie e segredo.
- MFA, rate limit distribuído, CSP e rotação de segredos continuam necessários antes de produção com dados reais.

## 4. Implementação

- Telas movidas de `admin/` para `frontend/app/admin/`.
- Componentes e clientes administrativos em `frontend/components/admin/` e `frontend/lib/admin/`.
- `frontend/proxy.ts` passou a proteger `/admin/*` e a encaminhar `/api/v1/admin/*`.
- O app e o deploy `admin-fit-conecta` foram removidos.
- A migration `0009_admin_panel.sql` e o backend administrativo permanecem inalterados.
