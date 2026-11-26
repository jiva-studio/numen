/**
 * How the plugin registry is collected: from a repository written here, and from a GitHub
 * release served by a fake fetch. Nothing goes to the network.
 */
import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { execTool } from '../shared/tools.mjs'
import { countContributors, groupByVersion, isBot } from './changelog.mjs'
import { createCheck } from './checks.mjs'
import { collectRegistry } from './collect.mjs'
import {
  describeManifest,
  getIcon,
  listAddedPermissions,
  pairScreenshots,
  splitReadme,
} from './entry.mjs'
import { listCommitAccounts } from './internal.mjs'
import { collectExternal, createGitHub, readArtifactPlatform } from './external.mjs'
import { getExternalQuality } from './quality.mjs'
import { assertNoCollision, getContentHash, readSources } from './registry.mjs'
import { countTests } from './tests.mjs'

const manifest = (version, extra = {}) => ({
  id: 'org.example.notes',
  name: 'Example Notes',
  version,
  description: 'Notes for the tests',
  targets: ['desktop.editor'],
  permissions: ['vault:read'],
  contributes: { feature: 'notes', category: 'views' },
  ...extra,
})

const README = '# Example Notes\n\nWrites notes.\n\n<!-- more -->\n\n## Settings\n\nNone.\n'

const createRepository = () => {
  const root = mkdtempSync(join(tmpdir(), 'plugin-registry-'))
  const git = (args, date) =>
    execTool('git', ['-c', 'user.name=Test', '-c', 'user.email=test@example.org', ...args], {
      cwd: root,
      encoding: 'utf8',
      env: { ...process.env, GIT_COMMITTER_DATE: date, GIT_AUTHOR_DATE: date },
    })
  const write = (path, text) => {
    mkdirSync(join(root, path, '..'), { recursive: true })
    writeFileSync(join(root, path), text)
  }
  const commit = (subject, date) => {
    git(['add', '-A'], date)
    git(['commit', '--quiet', '-m', subject], date)
  }
  git(['init', '--quiet'], '2026-11-01T00:00:00Z')
  write('LICENSE', 'GNU AFFERO GENERAL PUBLIC LICENSE\nVersion 3, 19 November 2007\n')
  write('plugins/notes/manifest.json', JSON.stringify(manifest('1.0.0')))
  write('plugins/notes/README.md', README)
  write('plugins/notes/screenshots/editor-light.webp', 'light')
  write('plugins/notes/screenshots/editor-dark.webp', 'dark')
  commit('feat(notes): add notes', '2026-11-01T10:00:00Z')
  write('plugins/notes/main.ts', 'export {}\n')
  commit('test(notes): cover the editor', '2026-11-02T10:00:00Z')
  write('plugins/notes/main.ts', 'export const a = 1\n')
  write(
    'plugins/notes/manifest.json',
    JSON.stringify(manifest('1.1.0', { permissions: ['vault:read', 'link:open'] })),
  )
  commit('feat(notes): open links', '2026-11-03T10:00:00Z')
  write('plugins/notes/main.ts', 'export const a = 2\n')
  commit('fix(notes): keep the cursor', '2026-11-04T10:00:00Z')
  write('plugins/registry.json', JSON.stringify({ external: [] }))
  return root
}

const withRepository = async (check) => {
  const root = createRepository()
  try {
    await check(root)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
}

const collectOffline = (root, extra = {}) =>
  collectRegistry({
    root,
    outDir: join(root, 'out'),
    previous: null,
    quality: null,
    sources: join(root, 'plugins', 'registry.json'),
    isOffline: true,
    generatedAt: '2026-11-26T00:00:00.000Z',
    ...extra,
  })

test('an internal plugin takes its manifest, README, screenshots and license', async () => {
  await withRepository(async (root) => {
    const [plugin] = (await collectOffline(root)).plugins

    assert.equal(plugin.id, 'org.example.notes')
    assert.equal(plugin.version, '1.1.0')
    assert.equal(plugin.isOfficial, false)
    assert.equal(plugin.isPaid, false)
    assert.equal(plugin.license, 'AGPL-3.0')
    assert.equal(plugin.readme.intro, 'Writes notes.')
    assert.deepEqual(plugin.screenshots, [
      {
        name: 'editor',
        light: 'plugins/notes/screenshots/editor-light.webp',
        dark: 'plugins/notes/screenshots/editor-dark.webp',
      },
    ])
    assert.deepEqual(
      plugin.permissions.map((one) => one.id),
      ['vault:read', 'link:open'],
    )
    assert.equal(plugin.permissions[1].description, 'Open links in your browser.')
  })
})

test('the changelog groups commits by the version change of the manifest', async () => {
  await withRepository(async (root) => {
    const [plugin] = (await collectOffline(root)).plugins

    assert.deepEqual(plugin.changelog, [
      {
        version: '1.1.0',
        date: '2026-11-03T10:00:00Z',
        changes: ['open links'],
      },
      { version: '1.0.0', date: '2026-11-01T10:00:00Z', changes: ['add notes'] },
    ])
    assert.equal(plugin.updatedAt, '2026-11-04T10:00:00Z')
  })
})

test('the permissions a new version adds are listed until the next version', async () => {
  await withRepository(async (root) => {
    const first = await collectOffline(root)
    const previous = {
      ...first,
      plugins: [{ ...first.plugins[0], version: '1.0.0', permissions: [{ id: 'vault:read' }] }],
    }
    const second = await collectOffline(root, { previous })

    assert.deepEqual(second.plugins[0].permissionsAdded, ['link:open'])
    assert.equal(
      second.plugins[0].checks.find((one) => one.id === 'permissions').status,
      'warn',
    )
    const third = await collectOffline(root, { previous: second })
    assert.deepEqual(third.plugins[0].permissionsAdded, ['link:open'])
  })
})

test('a registry collected again from the same tree has the same content hash', async () => {
  await withRepository(async (root) => {
    const first = await collectOffline(root)
    const second = await collectOffline(root, { generatedAt: '2026-11-27T00:00:00.000Z' })

    assert.equal(second.contentHash, first.contentHash)
  })
})

test('the quality of an internal plugin is read from the quality folder', async () => {
  await withRepository(async (root) => {
    const quality = join(root, 'quality')
    mkdirSync(quality)
    writeFileSync(
      join(quality, 'mutation.json'),
      JSON.stringify({
        targets: [{ target: 'plugins-notes', mutationScore: 81.5, measuredAt: '2026-11-25T00:00:00Z' }],
      }),
    )
    writeFileSync(
      join(quality, 'coverage.json'),
      JSON.stringify({
        targets: [{ target: 'plugins-notes', coverage: 72.3, measuredAt: '2026-11-26T00:00:00Z' }],
      }),
    )
    const [plugin] = (await collectOffline(root, { quality })).plugins

    assert.deepEqual(plugin.quality, {
      reportedBy: 'numen',
      entries: [
        {
          target: 'plugins-notes',
          kind: null,
          mutationScore: 81.5,
          coverage: 72.3,
          measuredAt: '2026-11-26T00:00:00Z',
        },
      ],
      tests: { go: 0, typescript: 0, stories: 0 },
    })
  })
})

const GO_TESTS = `package notes

import "testing"

func TestSave(t *testing.T) {}

func TestMain(m *testing.M) {}

func FuzzParse(f *testing.F) {}

func testHelper(t *testing.T) {}
`

const TS_TESTS = `import { describe, expect, it, test } from 'vitest'

describe('notes', () => {
  it('saves', () => expect(1).toBe(1))
  it.each([1, 2, 3])('keeps %i', () => {})
  test.skip('waits', () => {})
  const unit = 'it'
})
`

const STORIES = `import type { Meta, StoryObj } from '@storybook/vue3'

const meta = {} satisfies Meta
export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {}
export const Filled: Story = {}
export const fixture = {}
`

test('the tests of a plugin folder are counted from its sources', () => {
  const dir = mkdtempSync(join(tmpdir(), 'plugin-tests-'))
  try {
    const write = (path, text) => {
      mkdirSync(join(dir, path, '..'), { recursive: true })
      writeFileSync(join(dir, path), text)
    }
    write('backend/notes_test.go', GO_TESTS)
    write('backend/notes.go', 'package notes\n\nfunc TestLike(t *testing.T) {}\n')
    write('backend/testdata/fixture_test.go', GO_TESTS)
    write('frontend/model/notes.test.ts', TS_TESTS)
    write('frontend/ui/Notes.stories.ts', STORIES)
    write('node_modules/lib/lib.test.ts', TS_TESTS)

    assert.deepEqual(countTests(dir), { go: 2, typescript: 3, stories: 2 })
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})

test('an internal plugin with tests and no measurement lists its test counts', async () => {
  await withRepository(async (root) => {
    writeFileSync(join(root, 'plugins/notes/notes_test.go'), GO_TESTS)
    const [plugin] = (await collectOffline(root)).plugins

    assert.deepEqual(plugin.quality, {
      reportedBy: 'numen',
      entries: [],
      tests: { go: 2, typescript: 0, stories: 0 },
    })
  })
})

test('an internal plugin with no tests and no measurement has no quality', async () => {
  await withRepository(async (root) => {
    const [plugin] = (await collectOffline(root)).plugins

    assert.equal(plugin.quality, null)
  })
})

test('an author who reports only test counts has no measured entry', () => {
  assert.deepEqual(getExternalQuality({ tests: { stories: 4 } }), {
    reportedBy: 'author',
    entries: [],
    tests: { go: 0, typescript: 0, stories: 4 },
  })
  assert.equal(getExternalQuality({ tests: { go: 1.5 } }), null)
})

test('the contributions a person sees are read from the manifest', () => {
  const fields = describeManifest(
    manifest('1.0.0', {
      contributes: {
        category: 'readers',
        tabKinds: [{ kind: 'reader:reader-pdf', title: 'PDF Reader' }],
        readers: [{ id: 'reader-pdf', contentTypes: ['application/pdf'] }],
        textKinds: [{ kind: 'page-text', title: 'Text layer' }],
        themes: [{ id: 'amber', name: 'Amber', file: 'themes/amber.css' }],
        commands: [{ id: 'tree.open', title: 'Open the file tree' }],
        slots: [
          { slot: 'fileTree.context', command: 'app:file.createText' },
          { slot: 'titleBar.commands', command: 'tree.open' },
        ],
        connectionTypes: [{ id: 'openai', title: 'OpenAI', description: 'Models from OpenAI.' }],
        searchFunctions: { version: 1, functions: [{ name: 'hasOpenTask', description: 'open tasks' }] },
      },
    }),
  )

  assert.deepEqual(fields.tabKinds, [{ kind: 'reader:reader-pdf', title: 'PDF Reader' }])
  assert.deepEqual(fields.fileTypes, [{ contentType: 'application/pdf', title: 'PDF Reader' }])
  assert.deepEqual(fields.textKinds, [{ kind: 'page-text', title: 'Text layer' }])
  assert.deepEqual(fields.themes, [{ id: 'amber', name: 'Amber' }])
  assert.deepEqual(fields.slots, [
    { slot: 'fileTree.context', title: '', target: 'app:file.createText' },
    { slot: 'titleBar.commands', title: 'Open the file tree', target: 'tree.open' },
  ])
  assert.deepEqual(fields.connectionTypes, [
    { id: 'openai', title: 'OpenAI', description: 'Models from OpenAI.' },
  ])
  assert.deepEqual(fields.searchFunctions, [{ name: 'hasOpenTask', description: 'open tasks' }])
})

const commitAs = (root, name, email, date) => {
  writeFileSync(join(root, 'plugins/notes/main.ts'), `export const by = ${JSON.stringify(name)}\n`)
  const env = { ...process.env, GIT_COMMITTER_DATE: date, GIT_AUTHOR_DATE: date }
  execTool('git', ['add', '-A'], { cwd: root, env })
  execTool('git', ['-c', `user.name=${name}`, '-c', `user.email=${email}`, 'commit', '--quiet', '-m', 'fix(notes): x'], {
    cwd: root,
    env,
  })
}

test('an internal plugin takes its contributors from the history of its folder', async () => {
  await withRepository(async (root) => {
    const [plugin] = (await collectOffline(root)).plugins

    assert.deepEqual(plugin.contributors, [
      { name: 'Test', commits: 4, login: null, avatarUrl: null },
    ])
  })
})

test('one person with two spellings of their name is one contributor, named as in their newest commit', async () => {
  await withRepository(async (root) => {
    commitAs(root, 'Tést Person', 'TEST@example.org', '2026-11-05T10:00:00Z')
    commitAs(root, 'dependabot[bot]', '49699333+dependabot[bot]@users.noreply.github.com', '2026-11-06T10:00:00Z')
    const registry = await collectOffline(root)

    assert.deepEqual(registry.plugins[0].contributors, [
      { name: 'Tést Person', commits: 5, login: null, avatarUrl: null },
    ])
    assert.equal(JSON.stringify(registry).includes('example.org'), false)
  })
})

test('contributors are ordered by their commits, then by name', () => {
  const authors = [
    ['1', 'Bea', 'bea@example.org'],
    ['2', 'Ann', 'ann@example.org'],
    ['3', 'Cid', 'cid@example.org'],
    ['4', 'Cid', 'cid@example.org'],
    ['5', 'Bea', 'bea@example.org'],
    ['6', 'Cid', 'cid@example.org'],
  ].map(([hash, name, email]) => ({ hash, name, email }))

  assert.deepEqual(
    countContributors(authors).map(({ name, commits }) => ({ name, commits })),
    [
      { name: 'Cid', commits: 3 },
      { name: 'Bea', commits: 2 },
      { name: 'Ann', commits: 1 },
    ],
  )
})

test('a contributor takes the GitHub account of their newest commit that has one', () => {
  const authors = [
    { hash: 'c3', name: 'Ann B', email: 'ann@example.org' },
    { hash: 'c2', name: 'Ann', email: 'Ann@Example.org' },
    { hash: 'c1', name: 'Ann', email: 'ann@example.org' },
  ]
  const accounts = new Map([
    ['c2', { login: 'ann-new', avatarUrl: 'https://avatars.githubusercontent.com/u/2?v=4' }],
    ['c1', { login: 'ann-old', avatarUrl: 'https://avatars.githubusercontent.com/u/1?v=4' }],
  ])

  assert.deepEqual(countContributors(authors, accounts), [
    {
      name: 'Ann B',
      commits: 3,
      login: 'ann-new',
      avatarUrl: 'https://avatars.githubusercontent.com/u/2?v=4',
    },
  ])
})

test('a bot is a GitHub App author, by its name or its noreply address', () => {
  assert.equal(isBot({ name: 'dependabot[bot]', email: 'support@github.com' }), true)
  assert.equal(
    isBot({ name: 'Renovate', email: '29139614+renovate[bot]@users.noreply.github.com' }),
    true,
  )
  assert.equal(isBot({ name: 'Claude', email: 'noreply@anthropic.com' }), false)
  assert.equal(isBot({ name: 'Ann', email: '12+ann@users.noreply.github.com' }), false)
})

test('the icon is the first one the manifest declares on a tab kind, a command, a slot or a setting', () => {
  assert.equal(
    getIcon({
      contributes: {
        category: 'views',
        slots: [{ icon: 'waypoints' }],
        commands: [{ id: 'a' }, { id: 'b', icon: 'message-square' }],
        tabKinds: [{ id: 'chat', icon: 'bot' }],
      },
    }),
    'bot',
  )
  assert.equal(getIcon({ contributes: { slots: [{ icon: 'waypoints' }] } }), 'waypoints')
  assert.equal(
    getIcon({ settings: [{ key: 'model', icon: 'audio-lines' }], contributes: { category: 'engines' } }),
    'audio-lines',
  )
})

test('a plugin that declares no icon takes the icon of its category', async () => {
  await withRepository(async (root) => {
    const [plugin] = (await collectOffline(root)).plugins

    assert.equal(plugin.icon, 'layout-panel-left')
    assert.equal(getIcon({ contributes: { category: 'engines' } }), 'cpu')
    assert.equal(getIcon({}), 'puzzle')
  })
})

test('the registry lists the featured ids of the sources file', async () => {
  await withRepository(async (root) => {
    writeFileSync(
      join(root, 'plugins', 'registry.json'),
      JSON.stringify({ featured: ['org.example.notes'], external: [] }),
    )

    assert.deepEqual((await collectOffline(root)).featured, ['org.example.notes'])
  })
})

test('a featured id that no plugin has is an error', async () => {
  await withRepository(async (root) => {
    writeFileSync(
      join(root, 'plugins', 'registry.json'),
      JSON.stringify({ featured: ['org.example.missing'], external: [] }),
    )

    await assert.rejects(collectOffline(root), /featured org\.example\.missing is not in the registry/)
  })
})

test('an external source with the id of an internal plugin is an error', () => {
  assert.throws(
    () =>
      assertNoCollision(
        [{ id: 'org.example.notes', repo: 'someone/notes-plus' }],
        [{ id: 'org.example.notes', folder: 'notes' }],
      ),
    /id of an internal plugin/,
  )
  assert.throws(
    () =>
      assertNoCollision(
        [{ id: 'org.someone.notes', repo: 'someone/notes' }],
        [{ id: 'org.example.notes', folder: 'notes' }],
      ),
    /internal plugin folder/,
  )
})

test('an external source under the official prefix from another owner is an error', async () => {
  await withRepository(async (root) => {
    const file = join(root, 'plugins', 'registry.json')
    writeFileSync(
      file,
      JSON.stringify({ external: [{ id: 'md.numen.fake', repo: 'someone/fake', author: 'Someone' }] }),
    )

    assert.throws(() => readSources(file), /md\.numen\. from outside jiva-studio/)
  })
})

test('the content hash leaves out generation and measurement times and ETags', () => {
  const registry = (time) => ({
    schema: 1,
    generatedAt: time,
    plugins: [{ id: 'a', source: { etag: time }, quality: { entries: [{ measuredAt: time }] } }],
  })

  assert.equal(getContentHash(registry('2026-11-01')), getContentHash(registry('2026-11-02')))
})

test('a README is split at the more line without its title', () => {
  assert.deepEqual(splitReadme(README), {
    intro: 'Writes notes.',
    text: 'Writes notes.\n\n\n\n## Settings\n\nNone.',
    hasIntro: true,
  })
  assert.equal(splitReadme('# T\n\nNo marker.').hasIntro, false)
  assert.equal(splitReadme(null), null)
})

test('screenshots are paired by name, and other files are left out', () => {
  assert.deepEqual(
    pairScreenshots(['b-dark.webp', 'a-light.webp', 'notes.txt'], (name) => `s/${name}`),
    [
      { name: 'a', light: 's/a-light.webp', dark: null },
      { name: 'b', light: null, dark: 's/b-dark.webp' },
    ],
  )
})

test('commits after the last version change belong to no version', () => {
  const commits = [
    { hash: 'c', date: '3', subject: 'fix: c' },
    { hash: 'b', date: '2', subject: 'feat: b' },
    { hash: 'a', date: '1', subject: 'feat: a' },
  ]

  assert.deepEqual(groupByVersion(commits, new Map([['a', '1.0.0'], ['b', '1.0.0']])), [
    { version: '1.0.0', date: '1', changes: ['a'] },
  ])
})

test('a version lists its features and fixes only', () => {
  const commits = [
    { hash: 'c', date: '3', subject: 'refactor: c' },
    { hash: 'b', date: '2', subject: 'fix(pdf): b' },
    { hash: 'a', date: '1', subject: 'chore: a' },
  ]
  const versions = new Map([['a', '1.0.0'], ['b', '1.1.0'], ['c', '1.2.0']])

  assert.deepEqual(groupByVersion(commits, versions), [
    { version: '1.1.0', date: '2', changes: ['b'] },
  ])
})

test('a README loses the pictures of its screenshots', () => {
  const readme = '# T\n\nWrites.\n\n<!-- more -->\n\n<picture>\n  <img src="screenshots/a-light.webp">\n</picture>\n\n## Use'
  assert.equal(splitReadme(readme).text, 'Writes.\n\n\n\n## Use')
})

test('a version seen before keeps the permissions it added', () => {
  const previous = { version: '2.0.0', permissions: [{ id: 'a' }], permissionsAdded: ['a'] }

  assert.deepEqual(listAddedPermissions({ version: '2.0.0', permissions: [{ id: 'a' }] }, previous), ['a'])
  assert.deepEqual(listAddedPermissions({ version: '2.0.0', permissions: [] }, undefined), [])
})

test('an artifact name states its platform and architecture', () => {
  assert.deepEqual(readArtifactPlatform('plugin-darwin-arm64.tar.gz'), {
    platform: 'macos',
    arch: 'arm64',
  })
  assert.deepEqual(readArtifactPlatform('plugin_windows_x86_64.zip'), {
    platform: 'windows',
    arch: 'amd64',
  })
  assert.equal(readArtifactPlatform('checksums.txt'), null)
})

const ARTIFACT = Buffer.from('artifact')
const ARTIFACT_SHA = createHash('sha256').update(ARTIFACT).digest('hex')
const REPO = 'someone/plugin-word-count'
const ASSET_URL = `https://github.com/${REPO}/releases/download/v2.0.0/word-count-linux-amd64.tar.gz`

const createRelease = () => ({
  [`https://api.github.com/repos/${REPO}/releases/latest`]: {
    etag: '"r2"',
    json: {
      tag_name: 'v2.0.0',
      published_at: '2026-11-20T00:00:00Z',
      assets: [
        {
          name: 'word-count-linux-amd64.tar.gz',
          size: ARTIFACT.length,
          digest: `sha256:${ARTIFACT_SHA}`,
          browser_download_url: ASSET_URL,
        },
      ],
    },
  },
  [`https://raw.githubusercontent.com/${REPO}/v2.0.0/manifest.json`]: {
    text: JSON.stringify({
      ...manifest('2.0.0'),
      id: 'org.someone.word-count',
      name: 'Word Count',
      contributes: { category: 'tools', licenses: [{ id: 'pro', title: 'Pro' }] },
    }),
  },
  [`https://raw.githubusercontent.com/${REPO}/v2.0.0/README.md`]: {
    text: '# Word Count\n\nCounts words.\n\n<!-- more -->\n\nMore.\n',
  },
  [`https://raw.githubusercontent.com/${REPO}/v2.0.0/quality.json`]: {
    text: JSON.stringify({ coverage: 90, mutationScore: 70, tests: { go: 12, typescript: -1 } }),
  },
  [`https://api.github.com/repos/${REPO}`]: { json: { license: { spdx_id: 'MIT' } } },
  [`https://api.github.com/repos/${REPO}/contents/screenshots?ref=v2.0.0`]: {
    json: [
      { type: 'file', name: 'main-light.webp', download_url: 'https://raw.example/main-light.webp' },
      { type: 'file', name: 'main-dark.webp', download_url: 'https://raw.example/main-dark.webp' },
    ],
  },
  'https://raw.example/main-light.webp': { text: 'L' },
  'https://raw.example/main-dark.webp': { text: 'D' },
  [ASSET_URL]: { body: ARTIFACT },
  [`https://api.github.com/repos/${REPO}/releases?per_page=30`]: {
    json: [
      { tag_name: 'v2.0.0', published_at: '2026-11-20T00:00:00Z', body: '- Counts lines', draft: false },
      { tag_name: 'v2.1.0', published_at: '2026-11-21T00:00:00Z', body: 'draft', draft: true },
    ],
  },
})

const createFetcher = (routes, requests) => async (url, init = {}) => {
  requests.push(url)
  const route = routes[url]
  if (!route) return new Response('', { status: 404 })
  if (route.etag && init.headers?.['If-None-Match'] === route.etag) {
    return new Response(null, { status: 304 })
  }
  const headers = route.etag ? { etag: route.etag } : {}
  if (route.json) return new Response(JSON.stringify(route.json), { status: 200, headers })
  return new Response(route.body ?? route.text, { status: 200, headers })
}

const VERIFIERS = {
  canVerifyAttestation: () => true,
  verifyAttestation: async () => createCheck('attestation', 'pass', 'verified'),
  scanVulnerabilities: async () => createCheck('vulnerabilities', 'skipped', 'not installed'),
}

const withOutput = async (check) => {
  const outDir = mkdtempSync(join(tmpdir(), 'plugin-registry-out-'))
  try {
    await check(outDir)
  } finally {
    rmSync(outDir, { recursive: true, force: true })
  }
}

const SOURCE = { id: 'org.someone.word-count', repo: REPO, author: 'Someone' }

test('an external plugin is read from its latest release', async () => {
  await withOutput(async (outDir) => {
    const requests = []
    const fetcher = createFetcher(createRelease(), requests)
    const entry = await collectExternal(SOURCE, {
      github: createGitHub({ fetcher }),
      previousById: new Map(),
      knownSizes: new Map(),
      getSize: async () => null,
      outDir,
      verifiers: VERIFIERS,
    })

    assert.equal(entry.version, '2.0.0')
    assert.equal(entry.isPaid, true)
    assert.equal(entry.license, 'MIT')
    assert.equal(entry.author, 'Someone')
    assert.deepEqual(entry.contributors, [])
    assert.equal(entry.icon, 'wrench')
    assert.deepEqual(entry.source, {
      kind: 'external',
      repo: REPO,
      url: `https://github.com/${REPO}`,
      tag: 'v2.0.0',
      etag: '"r2"',
    })
    assert.deepEqual(entry.platforms, ['linux'])
    assert.equal(entry.artifacts[0].sha256, ARTIFACT_SHA)
    assert.equal(entry.checks.find((one) => one.id === 'attestation').status, 'pass')
    assert.deepEqual(entry.screenshots, [
      {
        name: 'main',
        light: 'plugins/registry/org.someone.word-count/screenshots/main-light.webp',
        dark: 'plugins/registry/org.someone.word-count/screenshots/main-dark.webp',
      },
    ])
    assert.equal(
      existsSync(join(outDir, 'org.someone.word-count', 'screenshots', 'main-dark.webp')),
      true,
    )
    assert.deepEqual(entry.changelog, [
      { version: '2.0.0', date: '2026-11-20T00:00:00Z', notes: '- Counts lines' },
    ])
    assert.deepEqual(entry.quality.entries[0], {
      target: null,
      kind: null,
      mutationScore: 70,
      coverage: 90,
      measuredAt: null,
    })
    assert.equal(entry.quality.reportedBy, 'author')
    assert.deepEqual(entry.quality.tests, { go: 12, typescript: 0, stories: 0 })
  })
})

test('an external plugin whose release has not changed is copied without downloads', async () => {
  await withOutput(async (outDir) => {
    const routes = createRelease()
    const context = {
      github: createGitHub({ fetcher: createFetcher(routes, []) }),
      previousById: new Map(),
      knownSizes: new Map(),
      getSize: async () => null,
      outDir,
      verifiers: VERIFIERS,
    }
    const previous = await collectExternal(SOURCE, context)
    const requests = []
    const entry = await collectExternal(SOURCE, {
      ...context,
      github: createGitHub({ fetcher: createFetcher(routes, requests) }),
      previousById: new Map([[SOURCE.id, previous]]),
    })

    assert.equal(entry, previous)
    assert.deepEqual(requests, [`https://api.github.com/repos/${REPO}/releases/latest`])
  })
})

test('the GitHub accounts of the commits of a path are read page by page', async () => {
  const commitsUrl = (page) =>
    `https://api.github.com/repos/jiva-studio/numen/commits?path=plugins%2Fnotes&per_page=100&page=${page}`
  const fullPage = Array.from({ length: 100 }, (_, index) => ({
    sha: `a${index}`,
    author: index === 0 ? null : { login: 'ann', avatar_url: 'https://avatars.githubusercontent.com/u/1?v=4' },
  }))
  const requests = []
  const fetcher = createFetcher(
    {
      [commitsUrl(1)]: { json: fullPage },
      [commitsUrl(2)]: { json: [{ sha: 'b0', author: { login: 'bea', avatar_url: null } }] },
    },
    requests,
  )

  const accounts = await listCommitAccounts(createGitHub({ fetcher }), 'jiva-studio/numen', 'plugins/notes')

  assert.equal(accounts.size, 100)
  assert.equal(accounts.has('a0'), false)
  assert.deepEqual(accounts.get('b0'), { login: 'bea', avatarUrl: null })
  assert.deepEqual(requests, [commitsUrl(1), commitsUrl(2)])
})
