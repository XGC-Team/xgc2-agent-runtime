/** Matches Broker.Queue/PromptWithOptions: SHA-256(session + NUL + idempotency key).
 * This is correlation only; the broker remains the admission authority.
 */
export async function agentPromptTurnId(sessionId: string, key: string): Promise<string> {
  const bytes = new TextEncoder().encode(`${sessionId}\0${key}`)
  const digest = await globalThis.crypto.subtle.digest('SHA-256', bytes)
  const hex = Array.from(new Uint8Array(digest), value => value.toString(16).padStart(2, '0')).join('')
  return `t_${hex.slice(0, 32)}`
}
