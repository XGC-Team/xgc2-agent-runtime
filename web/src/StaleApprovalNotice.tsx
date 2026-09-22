import { useEffect, useState } from 'react'
import { APPROVAL_STALE_AFTER_MS, isApprovalStale } from './approvalStaleness.js'
import { Button } from './upstream/t3/ui/button.js'

/** An advisory inside the existing card. It never changes its terminal state or skin. */
export function StaleApprovalNotice({ identity, open, createdAt, missing = false, locale = 'en', onRefresh }: {
  identity: string; open: boolean; createdAt?: string | null; missing?: boolean
  locale?: 'en' | 'zh'; onRefresh?: () => Promise<unknown>
}) {
  const [now, setNow] = useState(Date.now)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  useEffect(() => {
    setNow(Date.now())
    const created = createdAt ? Date.parse(createdAt) : NaN
    if (!open || missing || !Number.isFinite(created)) return
    const delay = created + APPROVAL_STALE_AFTER_MS + 1 - Date.now()
    if (delay <= 0) return
    const timer = setTimeout(() => setNow(Date.now()), Math.min(delay, 2_147_483_647))
    return () => clearTimeout(timer)
  }, [open, createdAt, missing])
  if (!isApprovalStale({ open, createdAt, missing, now })) return null
  const refresh = async () => {
    if (!onRefresh || busy) return
    setBusy(true); setError('')
    try { await onRefresh() }
    catch (cause) { setError(cause instanceof Error ? cause.message : String(cause)) }
    finally { setBusy(false) }
  }
  return <div role="status" className="my-1 border-l-2 border-warning pl-2 text-xs text-warning"
    data-xgc-role="native-agent-approval-stale" data-xgc-id={identity} data-xgc-stale="true">
    <span>{locale === 'zh' ? '此授权请求可能已过期' : 'This approval request may have expired'}</span>
    <Button type="button" size="xs" variant="ghost" disabled={busy || !onRefresh}
      data-xgc-role="native-agent-approval-refresh" data-xgc-id={identity}
      onClick={() => void refresh()}>{locale === 'zh' ? '刷新' : 'Refresh'}</Button>
    {error ? <span role="alert">{error}</span> : null}
  </div>
}
