import { sha256 } from '@noble/hashes/sha2.js'

// Keep this operation synchronous with event application: no detached queue
// may outlive the stream owner or reorder the cursor.
export function fingerprintBytes(bytes: Uint8Array): Uint8Array {
  return sha256(bytes)
}
