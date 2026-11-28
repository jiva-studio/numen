/** The bytes of a file a scenario names that has no sample: the same bytes for the same path, so both sides can build them again. */
import { createHash } from 'node:crypto'

const MEGABYTE = 1024 * 1024
/** The size of a file a scenario names without a size. */
const DEFAULT_SIZE = 64 * 1024

const fillFrom = (target: Buffer, from: number, seed: Buffer | string): void => {
  const digest = createHash('sha256').update(seed).digest()
  let a = digest.readUInt32LE(0) || 1
  let b = digest.readUInt32LE(4)
  let c = digest.readUInt32LE(8)
  let d = digest.readUInt32LE(12)
  const words = Math.ceil((target.length - from) / 4)
  const output = Buffer.allocUnsafe(words * 4)
  for (let at = 0; at < words; at += 1) {
    let t = d
    const s = a
    d = c
    c = b
    b = s
    t ^= t << 11
    t ^= t >>> 8
    a = (t ^ s ^ (s >>> 19)) >>> 0
    output.writeUInt32LE(a, at * 4)
  }
  output.copy(target, from, 0, target.length - from)
}

/** The bytes of a file of the path, seeded by the path. A longer fixture of the same path starts with the bytes of a shorter one. */
export const buildFixture = (filePath: string, megabytes?: number): Buffer => {
  const size = megabytes === undefined ? DEFAULT_SIZE : megabytes * MEGABYTE
  const content = Buffer.alloc(size)
  fillFrom(content, 0, filePath)
  return content
}

/** The bytes after a change of the file: the old bytes with others appended. */
export const appendToFixture = (content: Buffer, count: number): Buffer => {
  const added = Buffer.alloc(count)
  fillFrom(added, 0, createHash('sha256').update(content).digest())
  return Buffer.concat([content, added])
}
