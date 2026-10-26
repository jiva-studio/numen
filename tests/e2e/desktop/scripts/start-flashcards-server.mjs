import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { STAND_IN_MODEL, startStandIns } from './stand-ins.mjs'

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
// The home, cache, data and state folders stay in the run too.
const homeDir = `${vault}.home`
fs.rmSync(homeDir, { recursive: true, force: true })
const cacheDir = path.join(homeDir, '.cache')
const dataDir = path.join(homeDir, '.local', 'share')
const stateDir = path.join(homeDir, '.local', 'state')
for (const dir of [cacheDir, dataDir, stateDir]) fs.mkdirSync(dir, { recursive: true })

const standIns = await startStandIns()

// The review's agent panel answers from numen's own agent over the stand-in chat API. The agents
// that run a program the machine has installed stay off: those programs reach their own services.
const settingsDir =
  process.platform === 'darwin'
    ? path.join(homeDir, 'Library', 'Application Support', 'numen')
    : path.join(configDir, 'numen')
fs.mkdirSync(settingsDir, { recursive: true })
fs.writeFileSync(
  path.join(settingsDir, 'numen.json'),
  JSON.stringify(
    {
      v: 1,
      agent: { use: 'numen' },
      plugins: {
        'md.numen.agents.numen': { baseUrl: standIns.chatUrl, model: STAND_IN_MODEL },
        'md.numen.agents.claude': { enabled: false },
        'md.numen.agents.codex': { enabled: false },
        'md.numen.agents.antigravity': { enabled: false },
        'md.numen.agents.acp': { enabled: false },
      },
    },
    null,
    2,
  ),
)

function cleanup() {
  standIns.close()
  for (const dir of [vault, configDir, homeDir]) {
    try {
      fs.rmSync(dir, { recursive: true, force: true })
    } catch {
      // A busy file is left for the system to clear.
    }
  }
}

const child = spawn(binaryPath, [`-debug-http-addr=127.0.0.1:${port}`, `-debug-vault=${vault}`], {
  stdio: 'inherit',
  env: {
    ...process.env,
    HOME: homeDir,
    XDG_CONFIG_HOME: configDir,
    XDG_CACHE_HOME: cacheDir,
    XDG_DATA_HOME: dataDir,
    XDG_STATE_HOME: stateDir,
    NUMEN_PLUGINS_PATH: pluginsPath,
  },
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
