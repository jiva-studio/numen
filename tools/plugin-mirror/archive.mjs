/**
 * Reads the member a manifest names out of a downloaded archive (.tar.gz, .tgz or .zip), the way the
 * host unpacks it: the first regular file whose path, or base name where the pattern has no slash,
 * matches the pattern.
 */
import { createHash } from 'node:crypto'
import { gunzipSync, inflateRawSync } from 'node:zlib'

const BLOCK = 512
const ZIP_EOCD = 0x06054b50
const ZIP_CENTRAL = 0x02014b50
const ZIP_STORED = 0
const ZIP_DEFLATED = 8
const ZIP_UNIX = 3

const escapeRegExp = (text) => text.replace(/[.+^${}()|[\]\\]/g, '\\$&')

export const isMember = (name, pattern) => {
  const target = (pattern.includes('/') ? name : name.slice(name.lastIndexOf('/') + 1)).replace(
    /^\.\//,
    '',
  )
  const source = [...pattern]
    .map((c) => (c === '*' ? '[^/]*' : c === '?' ? '[^/]' : escapeRegExp(c)))
    .join('')
  return new RegExp(`^${source}$`).test(target)
}

const readField = (header, from, length) => {
  const field = header.subarray(from, from + length)
  const end = field.indexOf(0)
  return field.subarray(0, end < 0 ? length : end).toString('utf8')
}

const readPax = (data) => {
  const found = {}
  for (const line of data.toString('utf8').split('\n')) {
    const match = /^\d+ ([^=]+)=(.*)$/.exec(line)
    if (match) found[match[1]] = match[2]
  }
  return found
}

const readTarEntry = (tar, offset, pending) => {
  const header = tar.subarray(offset, offset + BLOCK)
  const size = Number.parseInt(readField(header, 124, 12).trim() || '0', 8)
  const type = String.fromCharCode(header[156] || 48)
  const prefix = readField(header, 345, 155)
  const base = readField(header, 0, 100)
  const name = pending.name ?? (prefix ? `${prefix}/${base}` : base)
  const data = tar.subarray(offset + BLOCK, offset + BLOCK + size)
  return { type, name, data, next: offset + BLOCK + Math.ceil(size / BLOCK) * BLOCK }
}

const findTarMember = (archive, pattern) => {
  const tar = gunzipSync(archive)
  let pending = {}
  for (let offset = 0; offset + BLOCK <= tar.length;) {
    if (tar[offset] === 0) break
    const entry = readTarEntry(tar, offset, pending)
    offset = entry.next
    if (entry.type === 'x') pending = { name: readPax(entry.data).path }
    else if (entry.type === 'L') pending = { name: readField(entry.data, 0, entry.data.length) }
    else if (entry.type === '0' && isMember(entry.name, pattern)) return entry.data
    else pending = {}
  }
  return null
}

const readZipEntry = (zip, offset) => {
  const nameLength = zip.readUInt16LE(offset + 28)
  const extraLength = zip.readUInt16LE(offset + 30)
  const commentLength = zip.readUInt16LE(offset + 32)
  return {
    method: zip.readUInt16LE(offset + 10),
    size: zip.readUInt32LE(offset + 20),
    isRegular:
      zip.readUInt8(offset + 5) !== ZIP_UNIX ||
      ((zip.readUInt32LE(offset + 38) >>> 16) & 0o170000) === 0o100000,
    local: zip.readUInt32LE(offset + 42),
    name: zip.toString('utf8', offset + 46, offset + 46 + nameLength),
    next: offset + 46 + nameLength + extraLength + commentLength,
  }
}

const readZipData = (zip, entry) => {
  const start =
    entry.local + 30 + zip.readUInt16LE(entry.local + 26) + zip.readUInt16LE(entry.local + 28)
  const packed = zip.subarray(start, start + entry.size)
  return entry.method === ZIP_DEFLATED ? inflateRawSync(packed) : packed
}

const findZipMember = (zip, pattern) => {
  const end = zip.lastIndexOf(Buffer.from([0x50, 0x4b, 0x05, 0x06]))
  if (end < 0 || zip.readUInt32LE(end) !== ZIP_EOCD) throw new Error('not a zip archive')
  let offset = zip.readUInt32LE(end + 16)
  for (let left = zip.readUInt16LE(end + 10); left > 0; left -= 1) {
    if (zip.readUInt32LE(offset) !== ZIP_CENTRAL) throw new Error('broken zip directory')
    const entry = readZipEntry(zip, offset)
    offset = entry.next
    if (!entry.name.endsWith('/') && entry.isRegular && isMember(entry.name, pattern)) {
      if (entry.method !== ZIP_STORED && entry.method !== ZIP_DEFLATED)
        throw new Error(`zip method ${entry.method}`)
      return readZipData(zip, entry)
    }
  }
  return null
}

/** Returns the sha256 of the first member of the archive that matches the pattern, or null when none does. */
export const getMemberSha256 = (archive, name, pattern) => {
  const data = name.toLowerCase().endsWith('.zip')
    ? findZipMember(archive, pattern)
    : findTarMember(archive, pattern)
  return data === null ? null : createHash('sha256').update(data).digest('hex')
}
