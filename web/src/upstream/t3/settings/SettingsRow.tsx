// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Upstream ref: bf3be75c400cf605dc0c80de9458458854c84131. See UPSTREAM.md.
import type { ComponentPropsWithoutRef,ReactNode } from "react";
import { cn } from "../utils.js";
export function SettingsRow({
  title,
  description,
  status,
  resetAction,
  control,
  children,
  className,
  ...rowProps
}: Omit<ComponentPropsWithoutRef<"div">, "title"> & {
  title: ReactNode;
  description?: ReactNode;
  status?: ReactNode;
  resetAction?: ReactNode;
  control?: ReactNode;
  children?: ReactNode;
}) {
  const renderedReset = resetAction;
  const renderedControl = control;

  return (
    <div
      {...rowProps}
      tabIndex={rowProps.id ? -1 : rowProps.tabIndex}
      data-slot="settings-row"
      className={cn("border-b border-border/50 last:border-b-0 aria-disabled:opacity-50 aria-disabled:[&_*]:text-muted-foreground", children ? "pb-3 pt-3.5" : "py-3.5", className)}
    >
      <div className="flex flex-col gap-3 sm:grid sm:grid-cols-[minmax(0,1fr)_14rem] sm:items-center sm:gap-8">
        <div className="min-w-0 flex-1 space-y-1">
          <div className="flex min-h-5 items-center gap-1.5">
            <h3 className="text-sm font-medium tracking-[-0.005em] text-foreground">{title}</h3>
            <span className="inline-flex h-5 w-5 shrink-0 items-center justify-center">
              {renderedReset}
            </span>
          </div>
          {description ? (
            <p className="max-w-xl text-[13px] leading-[1.45] text-muted-foreground/80">
              {description}
            </p>
          ) : null}
          {status ? <div className="pt-0.5 text-xs text-muted-foreground">{status}</div> : null}
        </div>
        {renderedControl ? (
          <div className="flex w-full shrink-0 items-center gap-2 sm:w-auto sm:justify-end">
            {renderedControl}
          </div>
        ) : null}
      </div>
      {children}
    </div>
  );
}
