import React from "react";

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  hint?: string;
}

export const Input: React.FC<InputProps> = ({
  label,
  error,
  hint,
  id,
  name,
  required,
  className = "",
  ...rest
}) => {
  const inputId = id ?? name;

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={inputId} className="text-sm font-medium text-ink">
        {label}
        {required && <span className="text-brand"> *</span>}
      </label>
      <input
        id={inputId}
        name={name}
        required={required}
        aria-invalid={Boolean(error)}
        className={`w-full rounded-xl border bg-surface px-3.5 py-2.5 text-sm text-ink outline-none transition placeholder:text-slate-400 focus:border-brand focus:ring-2 focus:ring-brand/20 ${
          error ? "border-red-400" : "border-slate-300"
        } ${className}`}
        {...rest}
      />
      {hint && !error && <span className="text-xs text-muted">{hint}</span>}
      {error && (
        <span className="text-xs font-medium text-red-600">{error}</span>
      )}
    </div>
  );
};

export default Input;
