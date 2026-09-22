import type { ComponentPropsWithoutRef, ReactNode } from 'react'

export type DecisionCardState = 'pending' | 'submitted' | 'allowed' | 'denied' | 'canceled' | 'expired' | 'resolved'
export type DecisionCardProps = Omit<ComponentPropsWithoutRef<'article'>, 'title'> & {
  identity: string
  title: ReactNode
  label?: string
  state?: DecisionCardState
  status?: ReactNode
  timestamp?: string
  actions?: ReactNode
  details?: ReactNode
  detailsLabel?: string
  'data-xgc-role'?: string
  'data-xgc-id'?: string
}

/** One review surface. The host owns its decision, authorization and execution. */
export function DecisionCard({ identity, title, label, state = 'pending', status, timestamp, actions, details, detailsLabel = 'Details', children, className = '', ...attributes }: DecisionCardProps) {
  return <article {...attributes} className={`xgc-decision-card ${className}`.trim()}
    aria-label={label ?? (typeof title === 'string' ? title : undefined)}
    data-xgc-role={attributes['data-xgc-role'] ?? 'decision-card'}
    data-xgc-id={attributes['data-xgc-id'] ?? identity}
    data-xgc-decision-state={state} aria-busy={state === 'submitted' || undefined}>
    <header className="xgc-decision-card-heading" data-xgc-role="decision-card-heading" data-xgc-id={identity}>
      <div className="xgc-decision-card-title" data-xgc-role="decision-card-title" data-xgc-id={identity}>{title}</div>
      {timestamp ? <time dateTime={timestamp} data-xgc-role="decision-card-time" data-xgc-id={identity}>
        {new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
      </time> : null}
    </header>
    {children ? <div className="xgc-decision-card-body" data-xgc-role="decision-card-body" data-xgc-id={identity}>{children}</div> : null}
    {details ? <details className="xgc-decision-card-details" data-xgc-role="decision-card-details" data-xgc-id={identity}>
      <summary data-xgc-role="decision-card-details-toggle" data-xgc-id={identity}>{detailsLabel}</summary>
      <div className="xgc-decision-card-detail-content">{details}</div>
    </details> : null}
    {status || actions ? <footer className="xgc-decision-card-footer" data-xgc-role="decision-card-footer" data-xgc-id={identity}>
      {status ? <span className="xgc-decision-card-status" role="status" data-xgc-role="decision-card-status" data-xgc-id={identity}>{status}</span> : null}
      {actions ? <div className="xgc-decision-card-actions" data-xgc-role="decision-card-actions" data-xgc-id={identity}>{actions}</div> : null}
    </footer> : null}
  </article>
}
