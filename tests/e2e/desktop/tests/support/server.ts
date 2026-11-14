/**
 * A debug server of a run's own, for a scenario that needs plugins the shared server does not have:
 * a vault copy, its own folders, and the plugins the scenario puts in the folder of installed plugins.
 */
import { spawn, type ChildProcess } from 'node:child_process'
import fs from 'node:fs'
import net from 'node:net'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { getCertificateFile, getKeyFile, test } from './test.js'

const here = path.dirname(fileURLToPath(import.meta.url))
const e2eRoot = path.resolve(here, '../..')
const repoRoot = path.resolve(e2eRoot, '../../..')
const host = '127.0.0.1'

/** A plugin the scenario installs: the files of its folder, by path. A file under bin/ is executable. */
export interface InstalledPlugin {
  readonly id: string
  readonly files: Readonly<Record<string, string>>
}

export interface OwnServer {
  readonly url: string
  stop(): Promise<void>
}

const getFreePort = (): Promise<number> =>
  new Promise((resolve, reject) => {
    const probe = net.createServer()
    probe.once('error', reject)
    probe.listen(0, host, () => {
      const { port } = probe.address() as net.AddressInfo
      probe.close(() => resolve(port))
    })
  })

const getSharedPairEnv = (): Record<string, string> => {
  const port = new URL(test.info().project.use.baseURL ?? '').port
  return {
    NUMEN_DEBUG_TLS_USE_CERT: getCertificateFile(port),
    NUMEN_DEBUG_TLS_USE_KEY: getKeyFile(port),
  }
}

/** What a scenario sets on the server beyond the plugins it installs. */
interface OwnServerOptions {
  readonly env?: Readonly<Record<string, string>>
  readonly settings?: string
}

const createRunFolders = () => {
  // The core refuses to run a plugin from the temporary folder, so the run's folders are kept under the e2e folder.
  const parent = path.join(e2eRoot, 'test-results')
  fs.mkdirSync(parent, { recursive: true })
  const root = fs.realpathSync(fs.mkdtempSync(path.join(parent, 'own-server-')))
  const vault = path.join(root, 'vault')
  fs.cpSync(path.join(repoRoot, 'tests/vault'), vault, { recursive: true })
  const home = path.join(root, 'home')
  const config = path.join(root, 'config')
  for (const dir of ['cache', 'data', 'state']) fs.mkdirSync(path.join(root, dir))
  fs.mkdirSync(home)
  fs.mkdirSync(config)
  const settings =
    process.platform === 'darwin'
      ? path.join(home, 'Library', 'Application Support', 'numen')
      : path.join(config, 'numen')
  return { root, vault, home, config, settings }
}

const writeServerFiles = (
  settings: string,
  installed: readonly InstalledPlugin[],
  settingsText: string | undefined,
): void => {
  for (const plugin of installed) {
    for (const [file, content] of Object.entries(plugin.files)) {
      const target = path.join(settings, 'plugins', plugin.id, file)
      fs.mkdirSync(path.dirname(target), { recursive: true })
      fs.writeFileSync(target, content, { mode: file.startsWith('bin/') ? 0o755 : 0o644 })
    }
  }
  if (settingsText !== undefined) {
    fs.mkdirSync(settings, { recursive: true })
    fs.writeFileSync(path.join(settings, 'numen.json'), settingsText)
  }
}

/**
 * Starts the debug server on a copy of the test vault with the plugins installed, and waits until it
 * listens. It serves the certificate and key of the shared server of the project, which the browser
 * trusts.
 */
export const startOwnServer = async (
  installed: readonly InstalledPlugin[],
  options: OwnServerOptions = {},
): Promise<OwnServer> => {
  const { root, vault, home, config, settings } = createRunFolders()
  writeServerFiles(settings, installed, options.settings)
  // The server writes it once it is bound: a new connection waits for the handshake from then on.
  const certificate = path.join(root, 'certificate.pem')
  const port = await getFreePort()
  const child: ChildProcess = spawn(
    path.join(repoRoot, 'apps/desktop/numen-debug'),
    [`-debug-http-addr=${host}:${port}`, '-debug-mock-ml', `-vault=${vault}`],
    {
      stdio: ['inherit', 'inherit', 'pipe'],
      env: {
        ...process.env,
        XDG_CONFIG_HOME: config,
        XDG_CACHE_HOME: path.join(root, 'cache'),
        XDG_DATA_HOME: path.join(root, 'data'),
        XDG_STATE_HOME: path.join(root, 'state'),
        HOME: home,
        NUMEN_TEST_MOCK_ML: 'true',
        NUMEN_PLUGINS_PATH: path.join(repoRoot, 'plugins'),
        ...getSharedPairEnv(),
        NUMEN_DEBUG_TLS_CERT: certificate,
        ...options.env,
      },
    },
  )
  let stderr = ''
  child.stderr?.on('data', (chunk: Buffer) => {
    stderr += chunk.toString()
    process.stderr.write(chunk)
  })
  const stop = async (): Promise<void> => {
    if (child.exitCode === null) {
      const ended = new Promise<void>((resolve) => child.once('exit', () => resolve()))
      child.kill('SIGTERM')
      await ended
    }
    // The plugin processes of the server may still write to its folders for a moment after it exits.
    fs.rmSync(root, { recursive: true, force: true, maxRetries: 10, retryDelay: 200 })
  }
  const startedAt = Date.now()
  while (!fs.existsSync(certificate)) {
    if (child.exitCode !== null || Date.now() - startedAt > 60_000) {
      await stop()
      throw new Error(`the debug server did not start on port ${port}: ${stderr.trim()}`)
    }
    await new Promise((resolve) => setTimeout(resolve, 100))
  }
  return { url: `https://${host}:${port}`, stop }
}
