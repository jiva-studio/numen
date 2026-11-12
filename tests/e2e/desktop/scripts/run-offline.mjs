// Runs Playwright with the network cut: on Linux inside a network namespace that holds the
// loopback interface alone, so a scenario that reaches outside the machine fails.
//
// Every engine and shard is a Playwright process of its own, against a server and a vault of its
// own, since a run changes the vault; the processes run at the same time. A shard is a share of the
// spec files (scripts/shards.mjs). The server of the process number n listens on the port
// NUMEN_TEST_PORT (3456) + 10 * n.
//
//   --shards=<n>      shards for each engine; NUMEN_E2E_SHARDS does the same. The default is half
//                     the cores, at most four, and one when the run names a file or a filter.
//   --project=<name>  runs that engine alone.
//   NUMEN_E2E_JSON_DIR=<dir>  writes a JSON report of each process there, for `npm run shards:measure`.
//   -c <config>       runs that config (the flashcards) one engine after the other, on its own.
import { spawn, spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { runTool } from '../../../../tools/shared/tools.mjs'
import { checkShards, getDefaultShardCount } from './shards.mjs'

const ENGINES = ['chromium', 'webkit']

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const args = process.argv.slice(2)

const run = (command, commandArgs, env = process.env) => {
  const result = spawnSync(command, commandArgs, { stdio: 'inherit', env })
  if (result.error) {
    console.error(`run-offline: ${command}: ${result.error.message}`)
    return 1
  }
  return result.status ?? 1
}

// The folders GLib loads its modules from, with its TLS backend among them: WebKit on Linux reaches
// the debug server's https through it. A distribution keeps glib-networking where GLib looks; on
// NixOS it is in the store, and the runner adds it.
const getGioModules = () => {
  const named = process.env.GIO_EXTRA_MODULES
  if (process.platform !== 'linux') {
    return named
  }
  const system = [
    '/usr/lib/x86_64-linux-gnu',
    '/usr/lib/aarch64-linux-gnu',
    '/usr/lib64',
    '/usr/lib',
  ]
  const dirs = [
    ...(named ?? '').split(':').filter(Boolean),
    ...system.map((dir) => path.join(dir, 'gio/modules')),
  ]
  if (dirs.some((dir) => fs.existsSync(path.join(dir, 'libgiognutls.so')))) {
    return named
  }
  const nix = runTool(
    'nix',
    ['build', '--no-link', '--print-out-paths', 'nixpkgs#glib-networking'],
    {
      encoding: 'utf8',
    },
  )
  if (nix.status !== 0) {
    console.error('run-offline: no glib-networking found; WebKit cannot open https.')
    return named
  }
  return [path.join(nix.stdout.trim(), 'lib/gio/modules'), named].filter(Boolean).join(':')
}

const isOffline = () => {
  if (process.platform !== 'linux') {
    return false
  }
  const probe = runTool('unshare', ['-rn', 'ip', 'link', 'set', 'lo', 'up'], { encoding: 'utf8' })
  if (probe.status !== 0) {
    console.error(
      'run-offline: cannot open a network namespace with `unshare -rn` and `ip`.\n' +
        'Install util-linux and iproute2, and allow unprivileged user namespaces ' +
        '(on Ubuntu: sudo sysctl kernel.apparmor_restrict_unprivileged_userns=0).\n' +
        (probe.error ? `${probe.error.message}\n` : '') +
        probe.stderr,
    )
    process.exit(1)
  }
  return true
}

/** The command that runs Playwright with the arguments, inside the network namespace on Linux. */
const getCommand = (playwrightArgs) => {
  const playwright = ['npx', 'playwright', 'test', ...playwrightArgs]
  return offline
    ? ['unshare', ['-rn', 'sh', '-c', 'ip link set lo up && exec "$@"', 'sh', ...playwright]]
    : [playwright[0], playwright.slice(1)]
}

const runPlaywright = (playwrightArgs, env) => {
  const [command, commandArgs] = getCommand(playwrightArgs)
  return run(command, commandArgs, env)
}

/** Runs Playwright and prefixes each line it prints with the label of its process. */
const runLabelledSuite = (label, playwrightArgs, env) =>
  new Promise((resolve) => {
    const [command, commandArgs] = getCommand(playwrightArgs)
    const child = spawn(command, commandArgs, { env, stdio: ['ignore', 'pipe', 'pipe'] })
    for (const [stream, target] of [
      [child.stdout, process.stdout],
      [child.stderr, process.stderr],
    ]) {
      let rest = ''
      stream.setEncoding('utf8')
      stream.on('data', (chunk) => {
        const lines = (rest + chunk).split('\n')
        rest = lines.pop() ?? ''
        for (const line of lines) target.write(`[${label}] ${line}\n`)
      })
      stream.on('end', () => {
        if (rest) target.write(`[${label}] ${rest}\n`)
      })
    }
    const started = Date.now()
    child.on('error', (error) => {
      console.error(`run-offline: ${command}: ${error.message}`)
      resolve({ label, code: 1, seconds: 0 })
    })
    child.on('close', (code, signal) =>
      resolve({ label, code: code ?? (signal ? 1 : 0), seconds: (Date.now() - started) / 1000 }),
    )
  })

const getFlag = (name) => {
  const index = args.findIndex((arg) => arg === name || arg.startsWith(`${name}=`))
  if (index < 0) return undefined
  const inline = args[index].split('=').at(1)
  const value = inline ?? args[index + 1]
  args.splice(index, inline === undefined ? 2 : 1)
  return value
}

const offline = isOffline()
const gioModules = getGioModules()
const env = gioModules ? { ...process.env, GIO_EXTRA_MODULES: gioModules } : process.env

const hasFlag = (...names) =>
  args.some((arg) => names.some((name) => arg === name || arg.startsWith(`${name}=`)))

if (hasFlag('-c', '--config')) {
  // The flashcards config has a server and a port of its own: one engine after the other.
  let status = 0
  for (const engine of ENGINES) {
    const code = runPlaywright(
      [`--project=${engine}`, `--output=${path.join(root, 'test-results', engine)}`, ...args],
      { ...env, PLAYWRIGHT_HTML_OUTPUT_DIR: path.join(root, 'playwright-report', engine) },
    )
    status ||= code
  }
  process.exit(status)
}

const problems = checkShards()
if (problems.length > 0) {
  console.error(`run-offline: ${problems.join('\n')}`)
  process.exit(1)
}

const project = getFlag('--project')
const named = getFlag('--shards') ?? process.env.NUMEN_E2E_SHARDS
// A run that names a file or a filter finds its tests in few shards, and the rest would only start
// a server: it runs on one server unless it names the count.
const hasFilter =
  hasFlag('--grep', '-g', '--grep-invert', '--last-failed', '--only-changed', '--list') ||
  args.some((arg) => !arg.startsWith('-'))
const shards = Number(named ?? (hasFilter ? 1 : getDefaultShardCount()))
if (!Number.isInteger(shards) || shards < 1) {
  console.error(`run-offline: --shards wants a whole number from 1, not ${named}`)
  process.exit(1)
}

const engines = project ? [project] : ENGINES
const basePort = Number(process.env.NUMEN_TEST_PORT ?? '3456')
const jobs = engines.flatMap((engine) =>
  Array.from({ length: shards }, (_, shard) => ({ engine, shard })),
)
/** The environment that has the process called `name` write its JSON report, when a folder is named. */
const getJsonEnv = (name) =>
  process.env.NUMEN_E2E_JSON_DIR
    ? { NUMEN_E2E_JSON: path.resolve(process.env.NUMEN_E2E_JSON_DIR, `${name}.json`) }
    : {}

if (jobs.length === 1) {
  // One process prints to the terminal as it is.
  const [{ engine }] = jobs
  process.exit(
    runPlaywright(
      [`--project=${engine}`, `--output=${path.join(root, 'test-results', engine)}`, ...args],
      {
        ...env,
        PLAYWRIGHT_HTML_OUTPUT_DIR: path.join(root, 'playwright-report', engine),
        ...getJsonEnv(engine),
      },
    ),
  )
}

const results = await Promise.all(
  jobs.map(({ engine, shard }, slot) => {
    const name = shards > 1 ? `${engine}-${shard + 1}` : engine
    return runLabelledSuite(
      name,
      [
        `--project=${engine}`,
        `--output=${path.join(root, 'test-results', name)}`,
        ...(shards > 1 ? ['--pass-with-no-tests'] : []),
        ...args,
      ],
      {
        ...env,
        NUMEN_TEST_PORT: String(basePort + slot * 10),
        ...(shards > 1 ? { NUMEN_E2E_SHARD: `${shard}/${shards}` } : {}),
        PLAYWRIGHT_HTML_OUTPUT_DIR: path.join(root, 'playwright-report', name),
        ...getJsonEnv(name),
      },
    )
  }),
)

for (const { label, code, seconds } of results) {
  const outcome = code === 0 ? 'passed' : `failed (exit ${code})`
  console.log(`run-offline: ${label} ${outcome} in ${seconds.toFixed(0)} s`)
}
process.exit(results.some((one) => one.code !== 0) ? 1 : 0)
