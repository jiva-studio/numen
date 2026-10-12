import { spawn } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const sourceVault = path.resolve(__dirname, '../../../vault')
const binaryPath = path.resolve(__dirname, '../../../../apps/desktop/numen-debug')

const port = process.env.NUMEN_TEST_PORT ?? '3456'
const host = '127.0.0.1'

// Create isolated temporary vault from template.
const tmpVault = fs.mkdtempSync(path.join(os.tmpdir(), 'numen-e2e-vault-'))
fs.cpSync(sourceVault, tmpVault, { recursive: true })

function cleanup() {
  try {
    fs.rmSync(tmpVault, { recursive: true, force: true })
  } catch {
    // Ignore cleanup errors on busy files.
  }
}

const child = spawn(
  binaryPath,
  [`-debug-http-addr=${host}:${port}`, '-debug-mock-ml', '-no-mcp', `-vault=${tmpVault}`],
  {
    stdio: 'inherit',
    env: {
      ...process.env,
      NUMEN_MOCK_ML: 'true',
    },
  },
)

child.on('exit', (code) => {
  cleanup()
  process.exit(code ?? 0)
})

for (const sig of ['SIGINT', 'SIGTERM']) {
  process.on(sig, () => {
    cleanup()
    child.kill(sig)
  })
}
