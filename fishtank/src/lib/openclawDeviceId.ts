/**
 * Matches OpenClaw UI device id: SHA-256 hex digest of the raw Ed25519 public key (32 bytes).
 * @see https://github.com/openclaw/openclaw — ui/src/ui/device-identity.ts fingerprintPublicKey
 */
export async function generateOpenclawStyleDeviceId(): Promise<string> {
  const pair = await crypto.subtle.generateKey({ name: 'Ed25519' }, true, ['sign', 'verify']);
  const raw = new Uint8Array(await crypto.subtle.exportKey('raw', pair.publicKey));
  const digest = new Uint8Array(await crypto.subtle.digest('SHA-256', raw));
  return Array.from(digest, (b) => b.toString(16).padStart(2, '0')).join('');
}
