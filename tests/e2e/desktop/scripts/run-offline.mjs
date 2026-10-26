// Runs Playwright with the network cut: on Linux inside a network namespace that holds the
// loopback interface alone, so a scenario that reaches outside the machine fails.
import { spawnSync } from 'node:child_process'

const playwright = ['npx', 'playwright', 'test', ...process.argv.slice(2)]

const run = (command, args) => {
  const result = spawnSync(command, args, { stdio: 'inherit' })
  if (result.error) {
    console.error(`run-offline: ${command}: ${result.error.message}`)
    return 1
  }
  return result.status ?? 1
}

if (process.platform !== 'linux') {
  process.exit(run(playwright[0], playwright.slice(1)))
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

process.exit(
  run('unshare', ['-rn', 'sh', '-c', 'ip link set lo up && exec "$@"', 'sh', ...playwright]),
)
