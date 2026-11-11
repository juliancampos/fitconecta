import Image from "next/image";
import Link from "next/link";

const benefits = [
  "Perfis verificados de profissionais autônomos",
  "Disponibilidade real, sem ida e volta no WhatsApp",
  "Pagamento integrado e horário confirmado na hora",
];

export default function AuthLayout({ children }: LayoutProps<"/auth">) {
  return (
    <div className="flex min-h-screen flex-1 bg-background">
      <aside className="relative hidden w-1/2 flex-col justify-between overflow-hidden bg-gradient-to-br from-brand/25 via-surface to-accent/20 p-12 text-white lg:flex">
        <div className="absolute -right-16 -top-16 h-64 w-64 rounded-full bg-brand/30 blur-3xl" />
        <div className="absolute -bottom-24 -left-10 h-72 w-72 rounded-full bg-accent/20 blur-3xl" />

        <Link
          href="/auth/login"
          className="relative flex items-center gap-2 text-xl font-extrabold"
        >
          <Image
            src="/images/logo-fitconecta.png"
            alt=""
            width={40}
            height={40}
            className="h-10 w-10 rounded-xl"
          />
          FitConecta<span className="-ml-2 text-brand">+</span>
        </Link>

        <div className="relative max-w-md">
          <Image
            src="/images/mascote-corrida.png"
            alt=""
            width={900}
            height={897}
            priority
            className="mb-8 w-40 drop-shadow-[0_0_35px_rgba(139,92,246,0.45)]"
          />
          <h1 className="text-3xl font-bold leading-tight">
            Conecte-se ao profissional que vai transformar o seu treino.
          </h1>
          <ul className="mt-8 space-y-3">
            {benefits.map((benefit) => (
              <li key={benefit} className="flex items-start gap-3 text-sm text-white/80">
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
