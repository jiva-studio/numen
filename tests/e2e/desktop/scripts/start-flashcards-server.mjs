import { spawn } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { STAND_IN_CONNECTION, STAND_IN_MODEL, startStandIns } from './stand-ins.mjs'

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

// The server writes its certificate here, and Chromium of the run trusts its public key.
// tests/support/certificate.ts names the same file.
const certificateFile = path.join(os.tmpdir(), `numen-e2e-certificate-${port}.pem`)
fs.rmSync(certificateFile, { force: true })
// A run that names the port of another debug server in NUMEN_E2E_SHARED_CERTIFICATE_PORT serves the
// certificate and key of that server, so one browser trusts both.
const sharedPort = process.env.NUMEN_E2E_SHARED_CERTIFICATE_PORT
const certificateEnv = sharedPort
  ? {
      NUMEN_DEBUG_TLS_USE_CERT: path.join(os.tmpdir(), `numen-e2e-certificate-${sharedPort}.pem`),
      NUMEN_DEBUG_TLS_USE_KEY: path.join(os.tmpdir(), `numen-e2e-key-${sharedPort}.pem`),
    }
  : { NUMEN_DEBUG_TLS_CERT: certificateFile }

const standIns = await startStandIns()

// The connection of numen's own agent, which the agent panel runs its tasks on.
const NUMEN_AGENT_CONNECTION = 'c57a4d200'

// The review's agent panel answers from numen's own agent through a connection to the stand-in chat API. The agents
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
      connections: [
        {
          id: STAND_IN_CONNECTION,
          type: 'md.numen.connections.llm/openai-compatible',
          name: 'Stand-in',
          revision: 1,
          fields: { baseUrl: standIns.chatUrl },
        },
        {
          id: NUMEN_AGENT_CONNECTION,
          type: 'md.numen.agents.numen/numen',
          name: 'Numen agent',
          revision: 1,
          fields: {},
        },
      ],
      'md.numen.agents.numen': {
        connections: [{ connection: STAND_IN_CONNECTION, model: STAND_IN_MODEL }],
      },
      'md.numen.views.agent.chat': { connections: [{ connection: NUMEN_AGENT_CONNECTION }] },
      'md.numen.views.flashcards.studio': { agentConnections: [{ connection: NUMEN_AGENT_CONNECTION }] },
      'md.numen.agents.acp': { enabled: false },
    },
    null,
    2,
  ),
)

function cleanup() {
  fs.rmSync(certificateFile, { force: true })
  standIns.close()
  for (const dir of [vault, configDir, homeDir]) {
    try {
      fs.rmSync(dir, { recursive: true, force: true })
    } catch (error) {
      console.warn(`could not remove ${dir}: ${error.message}`)
    }
  }
  // A vault in a folder of its own leaves that folder.
  if (process.env.NUMEN_FLASHCARDS_VAULT_FOLDER) {
    fs.rmSync(path.dirname(vault), { recursive: true, force: true })
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
    // The run has no session bus and no Secret Service, so the keys of its connections are kept in a file.
    XDG_RUNTIME_DIR: path.join(homeDir, 'run'),
    DBUS_SESSION_BUS_ADDRESS: '',
    NUMEN_PLUGINS_PATH: pluginsPath,
    ...certificateEnv,
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
