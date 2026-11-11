import React from "react";

export const Card: React.FC<{ className?: string; children: React.ReactNode }> = ({
  className = "",
  children,
}) => {
  return (
    <div
      className={`rounded-2xl border border-slate-200 bg-surface p-6 shadow-sm ${className}`}
    >
      {children}
    </div>
  );
};

export default Card;
