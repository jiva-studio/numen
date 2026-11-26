/**
 * How the registry reads the source of a plugin: dependencies and their licenses, known
 * vulnerabilities from a fake osv-scanner report, network use, stories and manual pages, and the
 * checks made of them. Nothing goes to the network and no outside tool runs.
 */
import assert from 'node:assert/strict'
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { execTool } from '../shared/tools.mjs'
import {
  checkHasTests,
  checkManual,
  checkMutationScore,
  checkNetwork,
  checkStories,
  checkVulnerabilities,
} from './checks.mjs'
import { collectRegistry } from './collect.mjs'
import { listNpmDependencies, listNpmPackages, parseGoMod } from './dependencies.mjs'
import { getManualUrl, listManualPages } from './manual.mjs'
import { readNetworkUse } from './network.mjs'
import { listUnstoriedComponents } from './tests.mjs'
import {
  createLockfileScanner,
  parseOsvReport,
  scanPluginVulnerabilities,
} from './vulnerabilities.mjs'

const withTree = (files, check) => {
  const root = mkdtempSync(join(tmpdir(), 'plugin-source-'))
  try {
    for (const [path, text] of Object.entries(files)) {
      mkdirSync(join(root, path, '..'), { recursive: true })
      writeFileSync(join(root, path), text)
    }
    return check(root)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
}

const GO_MOD = `module github.com/jiva-studio/numen/plugins/readers-pdf

go 1.26.11

require github.com/tidwall/gjson v1.18.0

require (
	github.com/jiva-studio/numen/libs/plugin/sdk-go v0.0.0
	github.com/klippa-app/go-pdfium v1.21.3
	golang.org/x/net v0.62.0 // indirect
)
`

const LOCK = {
  lockfileVersion: 3,
  packages: {
    '': { name: 'numen' },
    'node_modules/@lucide/vue': { version: '1.72.0', license: 'ISC', peerDependencies: { vue: '*' } },
    'node_modules/reka-ui': {
      version: '2.9.0',
      license: 'MIT',
      dependencies: { '@floating-ui/vue': '^1.1.0', 'aria-hidden': '^1.2.4' },
    },
    'node_modules/@floating-ui/vue': { version: '1.1.6', license: 'MIT' },
    'node_modules/aria-hidden': { version: '1.2.4', license: 'MIT', dependencies: { tslib: '^2' } },
    'node_modules/aria-hidden/node_modules/tslib': { version: '2.8.1', license: '0BSD' },
    'node_modules/tslib': { version: '1.14.1', license: '0BSD' },
    'node_modules/vue': { version: '3.5.45', license: 'MIT' },
  },
}

const PACKAGE = {
  dependencies: { 'reka-ui': '^2.0.0', '@numen/plugin-sdk': 'workspace:*', 'left-pad': '^1.0.0', '@lucide/vue': '^1.34.0' },
  devDependencies: { vitest: '^5.0.0' },
}

test('the Go requirements of a go.mod leave out the modules of this repository', () => {
  assert.deepEqual(parseGoMod(GO_MOD), [
    { name: 'github.com/tidwall/gjson', version: 'v1.18.0', isIndirect: false },
    { name: 'github.com/klippa-app/go-pdfium', version: 'v1.21.3', isIndirect: false },
    { name: 'golang.org/x/net', version: 'v0.62.0', isIndirect: true },
  ])
})

test('the npm dependencies take the version and license the lockfile resolves', () => {
  assert.deepEqual(listNpmDependencies(PACKAGE, LOCK), [
    { name: '@lucide/vue', version: '1.72.0', license: 'ISC' },
    { name: 'left-pad', version: '^1.0.0', license: null },
    { name: 'reka-ui', version: '2.9.0', license: 'MIT' },
  ])
})

test('the npm packages a plugin installs follow nested resolution and leave out peers', () => {
  assert.deepEqual(listNpmPackages(LOCK, ['reka-ui', '@lucide/vue']), [
    { name: '@floating-ui/vue', version: '1.1.6' },
    { name: '@lucide/vue', version: '1.72.0' },
    { name: 'aria-hidden', version: '1.2.4' },
    { name: 'reka-ui', version: '2.9.0' },
    { name: 'tslib', version: '2.8.1' },
  ])
})

const OSV_REPORT = {
  results: [
    {
      source: { path: '/repo/plugins/pdf/go.mod', type: 'lockfile' },
      packages: [
        {
          package: { name: 'golang.org/x/net', version: '0.62.0', ecosystem: 'Go' },
          vulnerabilities: [
            { id: 'GO-2026-0001', summary: 'Header parsing loops' },
            { id: 'GHSA-aaaa-bbbb-cccc', summary: 'Header parsing loops' },
          ],
          groups: [{ ids: ['GO-2026-0001', 'GHSA-aaaa-bbbb-cccc'] }],
        },
        {
          package: { name: 'stdlib', version: '1.26.11', ecosystem: 'Go' },
          vulnerabilities: [{ id: 'GO-2026-0002', summary: 'In the standard library' }],
          groups: [{ ids: ['GO-2026-0002'] }],
        },
      ],
    },
    {
      source: { path: '/repo/package-lock.json', type: 'lockfile' },
      packages: [
        {
          package: { name: 'aria-hidden', version: '1.2.4', ecosystem: 'npm' },
          vulnerabilities: [{ id: 'GHSA-1111-2222-3333', summary: 'Prototype pollution' }],
        },
        {
          package: { name: 'vitest', version: '5.0.0', ecosystem: 'npm' },
          vulnerabilities: [{ id: 'GHSA-4444-5555-6666', summary: 'Not installed by the plugin' }],
        },
      ],
    },
  ],
}

test('an osv-scanner report is one vulnerability per alias group, without the Go standard library', () => {
  const report = parseOsvReport(OSV_REPORT)

  assert.deepEqual(report.get('/repo/plugins/pdf/go.mod'), [
    {
      id: 'GO-2026-0001',
      url: 'https://osv.dev/vulnerability/GO-2026-0001',
      summary: 'Header parsing loops',
      ecosystem: 'Go',
      name: 'golang.org/x/net',
      version: '0.62.0',
    },
  ])
  assert.equal(report.get('/repo/package-lock.json').length, 2)
})

const createFakeScanner = (calls) =>
  createLockfileScanner({
    isInstalled: () => true,
    run: (name, args) => {
      calls.push([name, ...args])
      const file = args[args.indexOf('--lockfile') + 1]
      const results = OSV_REPORT.results.filter((one) => file.endsWith(one.source.path.split('/').pop()))
      return { status: 1, stdout: JSON.stringify({ results }), stderr: '' }
    },
  })

test('a plugin is scanned in its go.mod and in the npm packages it installs, each file once', async () => {
  const calls = []
  const scan = createFakeScanner(calls)
  const scope = {
    goMod: '/repo/plugins/pdf/go.mod',
    lockfile: '/repo/package-lock.json',
    npmPackages: [{ name: 'aria-hidden', version: '1.2.4' }],
  }

  const first = await scanPluginVulnerabilities(scan, scope)
  await scanPluginVulnerabilities(scan, scope)

  assert.deepEqual(
    first.vulnerabilities.map((one) => one.id),
    ['GHSA-1111-2222-3333', 'GO-2026-0001'],
  )
  assert.equal(calls.length, 2)
  assert.deepEqual(calls[0], [
    'osv-scanner',
    'scan',
    'source',
    '--lockfile',
    '/repo/plugins/pdf/go.mod',
    '--format',
    'json',
  ])
})

test('a scan is skipped where osv-scanner is not installed, and fails when it does not finish', async () => {
  const missing = createLockfileScanner({ isInstalled: () => false, run: () => assert.fail('ran') })
  const broken = createLockfileScanner({
    isInstalled: () => true,
    run: () => ({ status: 127, stdout: '', stderr: 'boom' }),
  })
  const scope = { goMod: '/x/go.mod', lockfile: null, npmPackages: [] }

  assert.deepEqual(await scanPluginVulnerabilities(missing, scope), {
    vulnerabilities: null,
    reason: 'osv-scanner is not installed.',
  })
  assert.deepEqual(await scanPluginVulnerabilities(broken, scope), {
    vulnerabilities: null,
    reason: 'osv-scanner did not finish.',
  })
})

test('the vulnerability check fails with the ids, passes with none and is skipped without a scan', () => {
  assert.deepEqual(
    checkVulnerabilities({
      vulnerabilities: [{ id: 'GO-2026-0001' }, { id: 'GO-2026-0001' }, { id: 'GHSA-x' }],
      reason: '',
    }),
    { id: 'vulnerabilities', status: 'fail', detail: '2 known: GO-2026-0001, GHSA-x.' },
  )
  assert.equal(checkVulnerabilities({ vulnerabilities: [], reason: '' }).status, 'pass')
  assert.deepEqual(checkVulnerabilities({ vulnerabilities: null, reason: 'Not run offline.' }), {
    id: 'vulnerabilities',
    status: 'skipped',
    detail: 'Not run offline.',
  })
})

const NETWORK_TREE = {
  'backend/feed/feed.go':
    'package feed\n\nimport "net/http"\n\nconst base = "https://dl.numen.md/feed"\n\nfunc read() { _, _ = http.Get(base) }\n',
  'backend/service/pages.go':
    'package service\n\nimport "net/http"\n\nfunc serve(w http.ResponseWriter, r *http.Request) {}\n',
  'backend/feed/feed_test.go': 'package feed\n\nconst other = "https://tests.only.org"\n',
  'frontend/api/load.ts': "export const load = () => fetch('https://api.example.org/x')\n",
  'frontend/api/load.test.ts': "fetch('https://stub.numen.md')\n",
  'frontend/ui/Book.vue': '<template><svg xmlns="http://www.w3.org/2000/svg" /></template>\n',
  'frontend/model/cache.ts': 'export const refetch = () => cache.fetch(1)\n',
}

test('network use is read from the client calls and URL literals of the code, tests left out', () => {
  withTree(NETWORK_TREE, (dir) => {
    const use = readNetworkUse(dir, {
      permissions: ['vault:read', 'link:open'],
      contributes: { frameOrigins: ['https://www.youtube-nocookie.com'] },
      dependencies: [
        { id: 'model', sources: [{ platform: 'linux/amd64', sourceUrls: ['https://huggingface.co/m.onnx'] }] },
      ],
    })

    assert.deepEqual(use, {
      isNetworked: true,
      calls: [
        { language: 'go', name: 'http.Get', files: ['backend/feed/feed.go'] },
        { language: 'typescript', name: 'fetch', files: ['frontend/api/load.ts'] },
      ],
      hosts: ['api.example.org', 'dl.numen.md'],
      frameOrigins: ['https://www.youtube-nocookie.com'],
      downloadHosts: ['huggingface.co'],
      canOpenLinks: true,
    })
  })
})

test('a plugin that serves HTTP but makes no request has no network use', () => {
  withTree({ 'backend/service/pages.go': NETWORK_TREE['backend/service/pages.go'] }, (dir) => {
    const use = readNetworkUse(dir, { permissions: [] })

    assert.equal(use.isNetworked, false)
    assert.deepEqual(checkNetwork(use), {
      id: 'network',
      status: 'info',
      detail: 'None found in the code or the manifest.',
    })
  })
})

test('the network check states the answer and then each kind of evidence on a line', () => {
  const detail = checkNetwork({
    isNetworked: true,
    calls: [{ language: 'go', name: 'http.Client', files: ['backend/a.go'] }],
    hosts: ['dl.numen.md'],
    frameOrigins: [],
    downloadHosts: ['huggingface.co'],
    canOpenLinks: true,
  }).detail

  assert.deepEqual(detail.split('\n'), [
    'Yes, read from the code and the manifest.',
    'Go: http.Client in backend/a.go.',
    'Hosts in the code: dl.numen.md.',
    'Downloads from: huggingface.co.',
    'Opens links in the browser (link:open).',
  ])
})

const STORY = (path) => `import Component from '${path}'\nexport default { component: Component }\nexport const Basic = {}\n`

test('a Vue component has a story when a stories file of the plugin imports it', () => {
  withTree(
    {
      'frontend/ui/Tab.vue': '<template />',
      'frontend/ui/Tab.stories.ts': STORY('./Tab.vue'),
      'frontend/ui/parts/Row.vue': '<template />',
      'frontend/ui/parts/Cell.vue': '<template />',
      'frontend/ui/Rows.stories.ts': STORY('./parts/Row.vue'),
    },
    (dir) => {
      const found = listUnstoriedComponents(dir)

      assert.deepEqual(found, {
        components: ['frontend/ui/Tab.vue', 'frontend/ui/parts/Cell.vue', 'frontend/ui/parts/Row.vue'],
        unstoried: ['frontend/ui/parts/Cell.vue'],
      })
      assert.deepEqual(checkStories(found), {
        id: 'stories',
        status: 'warn',
        detail: '1 of 3 components have no story: Cell.vue.',
      })
    },
  )
})

test('the stories check is skipped without a frontend and without components', () => {
  withTree({ 'backend/main.go': 'package main\n' }, (dir) => {
    assert.equal(listUnstoriedComponents(dir), null)
    assert.equal(checkStories(null).detail, 'No frontend.')
  })
  assert.equal(checkStories({ components: [], unstoried: [] }).status, 'skipped')
  assert.equal(checkStories({ components: ['frontend/ui/A.vue'], unstoried: [] }).status, 'pass')
})

const MANUAL = 'apps/docs/src/content/docs'

test('a manual page names the plugin outside the generated blocks', () => {
  withTree(
    {
      [`${MANUAL}/plex.mdx`]: 'The Plex Graph draws your notes.\n',
      [`${MANUAL}/reference.md`]: '<!-- BEGIN AUTOGEN -->\n### `md.numen.readers.pdf`\nPDF Reader\n<!-- END AUTOGEN -->\n',
      [`${MANUAL}/cli.md`]: 'The text layer of a PDF (`md.numen.readers.pdf`).\n',
      [`${MANUAL}/guide/index.md`]: 'Open it in the PDF Reader.\n',
      [`${MANUAL}/other.md`]: 'A PDF Readers list and md.numen.readers.pdfx.\n',
    },
    (root) => {
      assert.deepEqual(listManualPages(root, { id: 'md.numen.readers.pdf', name: 'PDF Reader' }), [
        'https://docs.numen.md/cli/',
        'https://docs.numen.md/guide/',
      ])
      assert.deepEqual(listManualPages(root, { id: 'md.numen.todos', name: 'Todos' }), [])
    },
  )
})

test('a manual page is addressed by its path, an index by its folder', () => {
  assert.equal(getManualUrl('index.mdx'), 'https://docs.numen.md/')
  assert.equal(getManualUrl('plex.mdx'), 'https://docs.numen.md/plex/')
  assert.equal(checkManual([]).status, 'warn')
  assert.equal(checkManual(null).status, 'skipped')
  assert.deepEqual(checkManual(['https://docs.numen.md/plex/']), {
    id: 'manual',
    status: 'pass',
    detail: 'https://docs.numen.md/plex/',
  })
})

test('the tests check warns when no test was found', () => {
  assert.deepEqual(checkHasTests({ go: 0, typescript: 0, stories: 0 }), {
    id: 'tests',
    status: 'warn',
    detail: 'No tests were found.',
  })
  assert.equal(checkHasTests({ go: 2, typescript: 1, stories: 0 }).detail, '3 tests.')
  assert.equal(checkHasTests(undefined).status, 'warn')
})

test('the mutation check passes at 80 percent, warns below and is skipped when not measured', () => {
  const entry = (mutationScore, kind = null) => ({ target: 't', kind, mutationScore, coverage: null })

  assert.deepEqual(checkMutationScore([entry(80, 'go'), entry(91.25, 'ts')]), {
    id: 'mutation',
    status: 'pass',
    detail: '80.0% (go), 91.3% (ts)',
  })
  assert.deepEqual(checkMutationScore([entry(79.9)]), {
    id: 'mutation',
    status: 'warn',
    detail: 'Below 80%: 79.9%.',
  })
  assert.equal(checkMutationScore([entry(null)]).status, 'skipped')
  assert.equal(checkMutationScore(undefined).status, 'skipped')
})

const writeRepository = (root) => {
  const write = (path, text) => {
    mkdirSync(join(root, path, '..'), { recursive: true })
    writeFileSync(join(root, path), text)
  }
  write('plugins/registry.json', JSON.stringify({ external: [] }))
  write(
    'plugins/pdf/manifest.json',
    JSON.stringify({
      id: 'org.example.pdf',
      name: 'Example PDF',
      version: '1.0.0',
      targets: ['desktop.editor'],
      permissions: ['vault:read'],
      contributes: { category: 'readers' },
    }),
  )
  write('plugins/pdf/go.mod', GO_MOD)
  write('plugins/pdf/package.json', JSON.stringify(PACKAGE))
  write('plugins/pdf/backend/feed.go', NETWORK_TREE['backend/feed/feed.go'])
  write('package-lock.json', JSON.stringify(LOCK))
  const git = (args) =>
    execTool('git', ['-c', 'user.name=Test', '-c', 'user.email=test@example.org', ...args], {
      cwd: root,
      encoding: 'utf8',
    })
  git(['init', '--quiet'])
  git(['add', '-A'])
  git(['commit', '--quiet', '-m', 'feat(pdf): add the reader'])
}

const createReaders = ({ isOffline, scanned }) => ({
  isOffline,
  readGoLicenses: async (modules) =>
    new Map(modules.map((one) => [`${one.name}@${one.version}`, one.name.includes('pdfium') ? 'MIT' : null])),
  scanLockfile: async (file) => {
    scanned.push(file)
    const name = file.slice(file.lastIndexOf('/') + 1)
    const results = OSV_REPORT.results.filter((one) => one.source.path.endsWith(name))
    return { status: 'done', vulnerabilities: [...parseOsvReport({ results }).values()].flat() }
  },
})

test('an internal plugin lists its dependencies, vulnerabilities and source checks', async () => {
  const root = mkdtempSync(join(tmpdir(), 'plugin-registry-source-'))
  try {
    writeRepository(root)
    const scanned = []
    const registry = await collectRegistry({
      root,
      outDir: join(root, 'out'),
      previous: null,
      quality: null,
      sources: join(root, 'plugins', 'registry.json'),
      isOffline: false,
      readers: createReaders({ isOffline: false, scanned }),
      generatedAt: '2026-11-26T00:00:00.000Z',
    })
    const [plugin] = registry.plugins

    assert.deepEqual(plugin.dependencies, {
      packages: [
        { ecosystem: 'go', name: 'github.com/tidwall/gjson', version: 'v1.18.0', license: null },
        { ecosystem: 'go', name: 'github.com/klippa-app/go-pdfium', version: 'v1.21.3', license: 'MIT' },
        { ecosystem: 'npm', name: '@lucide/vue', version: '1.72.0', license: 'ISC' },
        { ecosystem: 'npm', name: 'left-pad', version: '^1.0.0', license: null },
        { ecosystem: 'npm', name: 'reka-ui', version: '2.9.0', license: 'MIT' },
      ],
      indirect: { go: 1, npm: 3 },
    })
    assert.deepEqual(
      plugin.vulnerabilities.map((one) => one.id),
      ['GHSA-1111-2222-3333', 'GO-2026-0001'],
    )
    assert.deepEqual(
      plugin.checks.map((one) => [one.id, one.status]),
      [
        ['manifest', 'pass'],
        ['license', 'fail'],
        ['readme', 'fail'],
        ['screenshots', 'warn'],
        ['tests', 'warn'],
        ['mutation', 'skipped'],
        ['stories', 'skipped'],
        ['manual', 'skipped'],
        ['network', 'info'],
        ['vulnerabilities', 'fail'],
      ],
    )
    assert.deepEqual(scanned, [join(root, 'plugins/pdf/go.mod'), join(root, 'package-lock.json')])
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})

test('offline, the vulnerability check of an internal plugin is skipped without a scan', async () => {
  const root = mkdtempSync(join(tmpdir(), 'plugin-registry-source-'))
  try {
    writeRepository(root)
    const scanned = []
    const registry = await collectRegistry({
      root,
      outDir: join(root, 'out'),
      previous: null,
      quality: null,
      sources: join(root, 'plugins', 'registry.json'),
      isOffline: true,
      readers: createReaders({ isOffline: true, scanned }),
      generatedAt: '2026-11-26T00:00:00.000Z',
    })
    const [plugin] = registry.plugins

    assert.deepEqual(plugin.checks.at(-1), {
      id: 'vulnerabilities',
      status: 'skipped',
      detail: 'Not run offline.',
    })
    assert.deepEqual(plugin.vulnerabilities, [])
    assert.deepEqual(scanned, [])
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})
