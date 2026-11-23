/** The bytes a scenario writes for a file it names: the same bytes for the same path, so both sides can build them again. */

/** The bytes of a file of the path: a PDF header and bytes for ".pdf", a PNG for ".png", other bytes otherwise, all seeded by the path. */
export const buildFixture = (filePath: string, megabytes?: number): Buffer => {
  throw new Error(`not implemented: the bytes of "${filePath}" (${megabytes ?? 'default'} MB)`)
}

/** The bytes after a change of the file: the old bytes with others appended. */
export const appendToFixture = (content: Buffer, count: number): Buffer => {
  throw new Error(`not implemented: ${count} bytes appended to ${content.length} bytes`)
}
