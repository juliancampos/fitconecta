import Link from "next/link";

const benefits = [
  "Perfis verificados de profissionais autônomos",
  "Disponibilidade real, sem ida e volta no WhatsApp",
  "Pagamento integrado e horário confirmado na hora",
];

export default function AuthLayout({ children }: LayoutProps<"/auth">) {
  return (
    <div className="flex min-h-screen flex-1 bg-background">
      <aside className="relative hidden w-1/2 flex-col justify-between overflow-hidden bg-ink p-12 text-white lg:flex">
        <div className="absolute -right-16 -top-16 h-64 w-64 rounded-full bg-brand/30 blur-3xl" />
        <div className="absolute -bottom-24 -left-10 h-72 w-72 rounded-full bg-accent/20 blur-3xl" />

        <Link
          href="/auth/login"
          className="relative flex items-center gap-2 text-xl font-extrabold"
        >
          <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand text-white">
            🦜
          </span>
          FitConecta<span className="-ml-2 text-brand">+</span>
        </Link>

        <div className="relative max-w-md">
          <h1 className="text-3xl font-bold leading-tight">
            Conecte-se ao profissional que vai transformar o seu treino.
          </h1>
          <ul className="mt-8 space-y-3">
            {benefits.map((benefit) => (
              <li key={benefit} className="flex items-start gap-3 text-sm text-slate-200">
                <span
                  className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-accent text-xs font-bold text-white"
                  aria-hidden
                >
                  ✓
                </span>
                {benefit}
              </li>
            ))}
          </ul>
        </div>

        <p className="relative text-xs text-slate-400">
          © FitConecta+. Todos os direitos reservados.
        </p>
      </aside>

      <main className="flex w-full flex-1 items-center justify-center p-6 lg:w-1/2">
        <div className="w-full max-w-md">{children}</div>
      </main>
    </div>
  );
}
