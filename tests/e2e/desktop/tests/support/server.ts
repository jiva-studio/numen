/**
 * A debug server of a run's own, for a scenario that needs plugins the shared server does not have:
 * a vault copy, its own folders, and the plugins the scenario puts in the folder of installed plugins.
 */
import { spawn, type ChildProcess } from 'node:child_process'
import fs from 'node:fs'
import net from 'node:net'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

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
  /** The address the window is loaded from. */
  readonly url: string
  stop(): Promise<void>
}

/** A port the system has free. */
const getFreePort = (): Promise<number> =>
  new Promise((resolve, reject) => {
    const probe = net.createServer()
    probe.once('error', reject)
    probe.listen(0, host, () => {
      const { port } = probe.address() as net.AddressInfo
      probe.close(() => resolve(port))
    })
  })

const isListening = (port: number): Promise<boolean> =>
  new Promise((resolve) => {
    const probe = net.connect({ host, port })
    probe.once('connect', () => (probe.destroy(), resolve(true)))
    probe.once('error', () => resolve(false))
  })

/** Starts the debug server on a copy of the test vault with the plugins installed, and waits until it listens. */
export const startOwnServer = async (installed: readonly InstalledPlugin[]): Promise<OwnServer> => {
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
  for (const plugin of installed) {
    for (const [file, content] of Object.entries(plugin.files)) {
      const target = path.join(settings, 'plugins', plugin.id, file)
      fs.mkdirSync(path.dirname(target), { recursive: true })
      fs.writeFileSync(target, content, { mode: file.startsWith('bin/') ? 0o755 : 0o644 })
    }
  }
  const port = await getFreePort()
  const child: ChildProcess = spawn(
    path.join(repoRoot, 'apps/desktop/numen-debug'),
    [`-debug-http-addr=${host}:${port}`, '-debug-mock-ml', '-no-mcp', `-vault=${vault}`],
    {
      stdio: 'inherit',
      env: {
        ...process.env,
        XDG_CONFIG_HOME: config,
        XDG_CACHE_HOME: path.join(root, 'cache'),
        XDG_DATA_HOME: path.join(root, 'data'),
        XDG_STATE_HOME: path.join(root, 'state'),
        HOME: home,
        NUMEN_MOCK_ML: 'true',
        NUMEN_PLUGINS_PATH: path.join(repoRoot, 'plugins'),
      },
    },
  )
  const stop = async (): Promise<void> => {
    if (child.exitCode === null) {
      const ended = new Promise<void>((resolve) => child.once('exit', () => resolve()))
      child.kill('SIGTERM')
      await ended
    }
    fs.rmSync(root, { recursive: true, force: true })
  }
  const startedAt = Date.now()
  while (!(await isListening(port))) {
    if (child.exitCode !== null || Date.now() - startedAt > 60_000) {
      await stop()
      throw new Error(`the debug server did not start on port ${port}`)
    }
    await new Promise((resolve) => setTimeout(resolve, 100))
  }
  return { url: `https://${host}:${port}`, stop }
}
