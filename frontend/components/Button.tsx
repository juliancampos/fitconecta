import React from "react";

type Variant = "primary" | "secondary" | "ghost";

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  fullWidth?: boolean;
}

const variants: Record<Variant, string> = {
  primary: "bg-brand text-white hover:bg-brand-dark shadow-sm",
  secondary: "border border-slate-300 bg-surface text-ink hover:bg-slate-50",
  ghost: "text-ink hover:bg-slate-100",
};

export const Button: React.FC<ButtonProps> = ({
  variant = "primary",
  fullWidth = false,
  className = "",
  children,
  ...rest
}) => {
  return (
    <button
      className={`inline-flex items-center justify-center gap-2 rounded-xl px-4 py-2.5 text-sm font-semibold transition disabled:cursor-not-allowed disabled:opacity-60 ${variants[variant]} ${fullWidth ? "w-full" : ""} ${className}`}
      {...rest}
    >
      {children}
    </button>
  );
};

export default Button;
