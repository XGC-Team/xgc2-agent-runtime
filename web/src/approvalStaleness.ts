export const APPROVAL_STALE_AFTER_MS = 5 * 60 * 1000

/** Staleness is advisory. It never authorizes, answers, or invents a terminal event. */
export function isApprovalStale({ open, createdAt, missing = false, now = Date.now() }: {
  open: boolean
  createdAt?: string | null
  missing?: boolean
  now?: number
}): boolean {
  if (missing) return true
  if (!open || !createdAt) return false
  const created = Date.parse(createdAt)
  return Number.isFinite(created) && now - created > APPROVAL_STALE_AFTER_MS
}
