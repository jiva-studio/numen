/**
 * Playwright's `test`, with Chromium launched to trust the certificate of the debug server the run
 * started. Chromium aborts the first handshake of each context with a certificate it does not
 * trust, even with ignoreHTTPSErrors, and the server logs a TLS handshake error for it.
 */
import { createHash, X509Certificate } from 'node:crypto'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { test as base } from 'playwright/test'

export * from 'playwright/test'

/**
 * The TLS certificate the debug server listening on port wrote, which scripts/start-server.mjs and
 * scripts/start-flashcards-server.mjs name.
 */
export const getCertificateFile = (port: string): string =>
  path.join(os.tmpdir(), `numen-e2e-certificate-${port}.pem`)

/** The key of that certificate, in PEM, which scripts/start-server.mjs names. */
export const getKeyFile = (port: string): string =>
  path.join(os.tmpdir(), `numen-e2e-key-${port}.pem`)

const getPublicKeyHash = (file: string): string => {
  if (!fs.existsSync(file)) throw new Error(`the debug server wrote no certificate to ${file}`)
  const publicKey = new X509Certificate(fs.readFileSync(file)).publicKey
  return createHash('sha256')
    .update(publicKey.export({ type: 'spki', format: 'der' }))
    .digest('base64')
}

export const test = base.extend({
  // The server writes its certificate after the config is loaded, so the worker reads it.
  launchOptions: [
    async ({ launchOptions, browserName }, use, workerInfo) => {
      if (browserName !== 'chromium') return use(launchOptions)
      const port = new URL(workerInfo.project.use.baseURL ?? '').port
      const publicKeyHash = getPublicKeyHash(getCertificateFile(port))
      await use({
        ...launchOptions,
        args: [
          ...(launchOptions.args ?? []),
          `--ignore-certificate-errors-spki-list=${publicKeyHash}`,
        ],
      })
    },
    { scope: 'worker' },
  ],
})
