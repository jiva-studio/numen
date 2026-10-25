import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const sourceVault = path.resolve(__dirname, '../../../vault')
const repoRoot = path.resolve(__dirname, '../../../..')
const binaryPath = path.resolve(repoRoot, 'apps/desktop/numen-flashcards-debug')
const pluginsPath = path.resolve(repoRoot, 'plugins')

const port = process.env.NUMEN_FLASHCARDS_TEST_PORT ?? '3457'
const vault = process.env.NUMEN_FLASHCARDS_VAULT
if (!vault) throw new Error('NUMEN_FLASHCARDS_VAULT names no folder for the vault')

// A vault of its own for the run, and an installation of its own beside it.
fs.rmSync(vault, { recursive: true, force: true })
fs.cpSync(sourceVault, vault, { recursive: true })
const configDir = `${vault}.config`
fs.rmSync(configDir, { recursive: true, force: true })
fs.mkdirSync(configDir, { recursive: true })

function cleanup() {
  for (const dir of [vault, configDir]) {
    try {
      fs.rmSync(dir, { recursive: true, force: true })
    } catch {
      // A busy file is left for the system to clear.
    }
  }
}

const child = spawn(binaryPath, [`-debug-http-addr=127.0.0.1:${port}`, `-debug-vault=${vault}`], {
  stdio: 'inherit',
  env: { ...process.env, XDG_CONFIG_HOME: configDir, NUMEN_PLUGINS_PATH: pluginsPath },
})

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
