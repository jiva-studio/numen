/**
 * The inventory, the archive reader and the mirror, from fixtures written here and a local server.
 */
import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { createServer } from 'node:http'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { crc32, gzipSync } from 'node:zlib'
import { listAssets, listInventoryDrift, readInventory, readPluginDocuments } from './assets.mjs'
import { getMemberSha256, isMember } from './archive.mjs'
import { fillMirror, listMirrorUrls, verifyMirror } from './mirror.mjs'

const sha = (data) => createHash('sha256').update(data).digest('hex')

const tarEntry = (name, body, type = '0') => {
  const header = Buffer.alloc(512)
  header.write(name)
  header.write(`${body.length.toString(8).padStart(11, '0')}\0`, 124)
  header.write(type, 156)
  header.write('        ', 148)
  const sum = header.reduce((total, byte) => total + byte, 0)
  header.write(`${sum.toString(8).padStart(6, '0')}\0 `, 148)
  return Buffer.concat([header, body, Buffer.alloc((512 - (body.length % 512)) % 512)])
}

const makeTgz = (entries) =>
  gzipSync(
    Buffer.concat([
      ...entries.map(([name, body, type]) => tarEntry(name, Buffer.from(body), type)),
      Buffer.alloc(1024),
    ]),
  )

const makeZip = (name, body) => {
  const data = Buffer.from(body)
  const local = Buffer.alloc(30)
  local.writeUInt32LE(0x04034b50, 0)
  local.writeUInt32LE(crc32(data), 14)
  local.writeUInt32LE(data.length, 18)
  local.writeUInt32LE(data.length, 22)
  local.writeUInt16LE(name.length, 26)
  const entry = Buffer.concat([local, Buffer.from(name), data])
  const central = Buffer.alloc(46)
  central.writeUInt32LE(0x02014b50, 0)
  central.writeUInt32LE(crc32(data), 16)
  central.writeUInt32LE(data.length, 20)
  central.writeUInt32LE(data.length, 24)
  central.writeUInt16LE(name.length, 28)
  const directory = Buffer.concat([central, Buffer.from(name)])
  const end = Buffer.alloc(22)
  end.writeUInt32LE(0x06054b50, 0)
  end.writeUInt16LE(1, 8)
  end.writeUInt16LE(1, 10)
  end.writeUInt32LE(directory.length, 12)
  end.writeUInt32LE(entry.length, 16)
  return Buffer.concat([entry, directory, end])
}

const library = 'library bytes'
const tgz = makeTgz([
  ['pkg/lib/libruntime.so', 'link', '2'],
  ['pkg/lib/libruntime.so.1', library],
])

const dependency = (extra) => ({ id: 'runtime', kind: 'runtime', version: '1', ...extra })
const linuxSource = (urls) => ({
  platform: 'linux/amd64',
  sourceUrls: urls,
  checksumSha256: sha(tgz),
  archiveMember: 'libruntime.so*',
  memberSha256: sha(library),
})

const plugins = [
  {
    name: 'b-plugin',
    document: {
      dependencies: [dependency({ sources: [linuxSource(['https://example.com/rt.tgz'])] })],
    },
  },
  {
    name: 'a-plugin',
    document: {
      dependencies: [dependency({ sources: [linuxSource(['https://mirror.example.com/rt.tgz'])] })],
      contributes: {
        settings: [
          {
            options: {
              x: [
                {
                  dependencies: [
                    {
                      id: 'model',
                      kind: 'model',
                      version: '2',
                      sourceUrl: 'https://example.com/m.bin',
                      checksumSha256: sha('m'),
                    },
                  ],
                },
              ],
            },
          },
        ],
      },
    },
  },
  { name: 'none', document: { dependencies: [{ id: 'system', kind: 'binary' }] } },
]

const withTemp = async (body) => {
  const dir = mkdtempSync(join(tmpdir(), 'mirror-test-'))
  try {
    await body(dir)
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
}

test('listAssets merges consumers and urls, finds nested dependencies and skips sourceless ones', () => {
  const assets = listAssets(plugins)
  assert.deepEqual(
    assets.map((a) => a.path),
    ['model/2/any/m.bin', 'runtime/1/linux-amd64/rt.tgz'],
  )
  assert.deepEqual(assets[1].consumers, ['a-plugin', 'b-plugin'])
  assert.deepEqual(assets[1].urls, [
    'https://example.com/rt.tgz',
    'https://mirror.example.com/rt.tgz',
  ])
})

test('listAssets rejects a source with no checksum', () => {
  const bad = [
    { name: 'p', document: { dependencies: [dependency({ sourceUrl: 'https://example.com/x' })] } },
  ]
  assert.throws(() => listAssets(bad), /no sha256/)
})

test('listAssets rejects two checksums at one path', () => {
  const clash = (sum) => ({
    name: sum.slice(0, 1),
    document: {
      dependencies: [dependency({ sourceUrl: 'https://example.com/x', checksumSha256: sum })],
    },
  })
  assert.throws(() => listAssets([clash('a'.repeat(64)), clash('b'.repeat(64))]), /two assets/)
})

test('listInventoryDrift names a missing, a changed and a stale asset', () => {
  const declared = listAssets(plugins)
  assert.deepEqual(listInventoryDrift(declared, declared), [])
  const recorded = [
    { ...declared[0], sha256: 'x' },
    { ...declared[1], path: 'gone/1/any/g' },
  ]
  assert.equal(listInventoryDrift(declared, recorded).length, 3)
})

test('the inventory committed with the tool is the one the manifests declare', () => {
  assert.deepEqual(listInventoryDrift(listAssets(readPluginDocuments()), readInventory()), [])
})

test('isMember matches a base name pattern and a path pattern', () => {
  assert.ok(isMember('a/b/libx.so.1', 'libx.so*'))
  assert.ok(!isMember('a/b/libx.so.1', 'b/libx.so*'))
  assert.ok(isMember('./a/b/libx.so', 'a/b/libx.so'))
})

test('getMemberSha256 reads the first regular member of a tar.gz and of a zip', () => {
  assert.equal(getMemberSha256(tgz, 'rt.tgz', 'libruntime.so*'), sha(library))
  assert.equal(getMemberSha256(tgz, 'rt.tgz', 'absent*'), null)
  assert.equal(getMemberSha256(makeZip('dir/onnx.dll', 'dll'), 'rt.zip', 'onnx.dll'), sha('dll'))
})

const serve = async (files) => {
  const requests = []
  const server = createServer((req, res) => {
    requests.push(req.url)
    const body = files[req.url]
    res.writeHead(body ? 200 : 404)
    res.end(body)
  })
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve))
  const base = `http://127.0.0.1:${server.address().port}`
  return { base, requests, close: () => server.close() }
}

const runtimeAsset = (urls) =>
  listAssets([
    { name: 'p', document: { dependencies: [dependency({ sources: [linuxSource(urls)] })] } },
  ])

test('fillMirror falls back to a second URL, verifies and lays out the file', async () => {
  const server = await serve({ '/mirror/rt.tgz': tgz, '/bad/rt.tgz': Buffer.from('tampered') })
  await withTemp(async (dir) => {
    const assets = runtimeAsset([
      `${server.base}/missing/rt.tgz`,
      `${server.base}/bad/rt.tgz`,
      `${server.base}/mirror/rt.tgz`,
    ])
    const result = await fillMirror(dir, assets, (url) => fetch(url))
    assert.deepEqual(result.mirrored, ['runtime/1/linux-amd64/rt.tgz'])
    assert.deepEqual(readFileSync(join(dir, assets[0].path)), tgz)
    assert.deepEqual(verifyMirror(dir, assets), { problems: [], extras: [] })
    const again = await fillMirror(dir, assets, () => assert.fail('fetched a verified file'))
    assert.deepEqual(again.kept, [assets[0].path])
  })
  server.close()
})

test('fillMirror reports an asset no URL supplies and stores nothing', async () => {
  const server = await serve({})
  await withTemp(async (dir) => {
    const assets = runtimeAsset([`${server.base}/a.tgz`, `${server.base}/b.tgz`])
    const result = await fillMirror(dir, assets, (url) => fetch(url))
    assert.equal(result.failed.length, 1)
    assert.match(result.failed[0], /a\.tgz: status 404.*b\.tgz: status 404/)
    assert.equal(verifyMirror(dir, assets).problems.length, 1)
  })
  server.close()
})

test('verifyMirror reports a missing file, a changed file, a changed member and an extra file', async () => {
  await withTemp(async (dir) => {
    const assets = runtimeAsset(['https://example.com/rt.tgz'])
    assert.match(verifyMirror(dir, assets).problems[0], /is not in the mirror/)
    mkdirSync(join(dir, 'runtime/1/linux-amd64'), { recursive: true })
    const file = join(dir, assets[0].path)
    writeFileSync(file, 'other')
    writeFileSync(join(dir, 'stray.bin'), 'x')
    const found = verifyMirror(dir, assets)
    assert.match(found.problems[0], /sha256 is/)
    assert.deepEqual(found.extras, ['stray.bin'])
    const wrongMember = [{ ...assets[0], sha256: sha(tgz), memberSha256: sha('else') }]
    writeFileSync(file, tgz)
    assert.match(verifyMirror(dir, wrongMember).problems.join('\n'), /member sha256 is/)
  })
})

test('listMirrorUrls joins the base and the path without doubling the slash', () => {
  const assets = runtimeAsset(['https://example.com/rt.tgz'])
  assert.equal(
    listMirrorUrls('https://m.example.com/pub/', assets)[0].url,
    'https://m.example.com/pub/runtime/1/linux-amd64/rt.tgz',
  )
})
