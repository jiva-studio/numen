// Runs Playwright with the network cut: on Linux inside a network namespace that holds the
// loopback interface alone, so a scenario that reaches outside the machine fails.
//
// Each engine runs on its own, against a server and a vault of its own, since a run changes the
// vault. `--project=<name>` runs that one alone.
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

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
  const nix = spawnSync(
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
  const probe = spawnSync('unshare', ['-rn', 'ip', 'link', 'set', 'lo', 'up'], { encoding: 'utf8' })
  if (probe.status !== 0) {
    console.error(
      'run-offline: cannot open a network namespace with `unshare -rn` and `ip`.\n' +
        'Install util-linux and iproute2, and allow unprivileged user namespaces ' +
        '(on Ubuntu: sudo sysctl kernel.apparmor_restrict_unprivileged_userns=0).\n' +
        (probe.stderr ?? probe.error?.message ?? ''),
    )
    process.exit(1)
  }
  return true
}

const runPlaywright = (playwrightArgs, env) => {
  const playwright = ['npx', 'playwright', 'test', ...playwrightArgs]
  if (!offline) {
    return run(playwright[0], playwright.slice(1), env)
  }
  return run(
    'unshare',
    ['-rn', 'sh', '-c', 'ip link set lo up && exec "$@"', 'sh', ...playwright],
    env,
  )
}

const offline = isOffline()
const gioModules = getGioModules()
const env = gioModules ? { ...process.env, GIO_EXTRA_MODULES: gioModules } : process.env

if (args.some((arg) => arg === '--project' || arg.startsWith('--project='))) {
  process.exit(runPlaywright(args, env))
}

let status = 0
for (const engine of ENGINES) {
  const code = runPlaywright(
    [`--project=${engine}`, `--output=${path.join(root, 'test-results', engine)}`, ...args],
    {
      ...env,
      PLAYWRIGHT_HTML_OUTPUT_DIR: path.join(root, 'playwright-report', engine),
    },
  )
  status ||= code
}
process.exit(status)
