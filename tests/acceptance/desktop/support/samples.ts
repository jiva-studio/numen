/** The sample files a scenario can put in a vault: a file name, and either where the repository keeps that file or how to make it. */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

export type SampleKind = 'PDF' | 'EPUB' | 'recording' | 'image'

type Source = { readonly folder: string } | { readonly make: () => Buffer }

const here = path.dirname(fileURLToPath(import.meta.url))
const E2E_FIXTURES = path.resolve(here, '../../../e2e/desktop/fixtures')
const VAULT_ASSETS = path.resolve(here, '../../../vault/assets')

/** A mono WAV of one second of silence at 16 kHz. */
export const createSampleWav = (): Buffer => {
  const sampleRate = 16000
  const dataSize = sampleRate * 2
  const buffer = Buffer.alloc(44 + dataSize)
  buffer.write('RIFF', 0)
  buffer.writeUInt32LE(36 + dataSize, 4)
  buffer.write('WAVE', 8)
  buffer.write('fmt ', 12)
  buffer.writeUInt32LE(16, 16)
  buffer.writeUInt16LE(1, 20)
  buffer.writeUInt16LE(1, 22)
  buffer.writeUInt32LE(sampleRate, 24)
  buffer.writeUInt32LE(sampleRate * 2, 28)
  buffer.writeUInt16LE(2, 32)
  buffer.writeUInt16LE(16, 34)
  buffer.write('data', 36)
  buffer.writeUInt32LE(dataSize, 40)
  return buffer
}

/** A US Letter PDF of `pageCount` pages, each with its number printed on it. */
export const createPagedPdf = (pageCount: number): Buffer => {
  const objects = [
    '<< /Type /Catalog /Pages 2 0 R >>',
    '',
    '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>',
  ]
  const kids: string[] = []
  for (let page = 1; page <= pageCount; page++) {
    const content = `BT /F1 96 Tf 220 380 Td (${page}) Tj ET`
    objects.push(`<< /Length ${content.length} >>\nstream\n${content}\nendstream`)
    objects.push(
      `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents ${objects.length} 0 R >>`,
    )
    kids.push(`${objects.length} 0 R`)
  }
  objects[1] = `<< /Type /Pages /Kids [${kids.join(' ')}] /Count ${pageCount} >>`
  let body = '%PDF-1.4\n'
  const offsets: number[] = []
  for (const [index, object] of objects.entries()) {
    offsets.push(body.length)
    body += `${index + 1} 0 obj\n${object}\nendobj\n`
  }
  const xref = body.length
  body += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`
  body += offsets.map((offset) => `${String(offset).padStart(10, '0')} 00000 n \n`).join('')
  body += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF\n`
  return Buffer.from(body, 'latin1')
}

const SAMPLES: Record<SampleKind, ReadonlyMap<string, Source>> = {
  PDF: new Map<string, Source>([
    ['flywheel.pdf', { folder: E2E_FIXTURES }],
    ['paper.pdf', { folder: VAULT_ASSETS }],
    ['pumped-storage.pdf', { folder: E2E_FIXTURES }],
    ['regenerative-braking.pdf', { folder: E2E_FIXTURES }],
    ['sample.pdf', { folder: E2E_FIXTURES }],
    ['shore.pdf', { folder: E2E_FIXTURES }],
    ['pages.pdf', { make: () => createPagedPdf(40) }],
    ['corrupted.pdf', { make: () => Buffer.from('INVALID_PDF_HEADER') }],
  ]),
  EPUB: new Map<string, Source>([
    ['ledger.epub', { folder: E2E_FIXTURES }],
    ['sample.epub', { folder: E2E_FIXTURES }],
    ['corrupted.epub', { make: () => Buffer.from('INVALID_EPUB_ZIP') }],
  ]),
  recording: new Map<string, Source>([
    ['lecture.wav', { folder: E2E_FIXTURES }],
    ['talk.mp3', { folder: E2E_FIXTURES }],
    ['sample.wav', { make: createSampleWav }],
    ['sample.mp3', { make: () => Buffer.alloc(512) }],
    ['sample.flac', { make: () => Buffer.alloc(512) }],
  ]),
  image: new Map<string, Source>([['llama.png', { folder: VAULT_ASSETS }]]),
}

/** The bytes of the sample of a kind, found by its file name. */
export const readSample = (kind: SampleKind, fileName: string): Buffer => {
  const samples = SAMPLES[kind]
  const source = samples.get(fileName)
  if (source === undefined) {
    throw new Error(`there is no sample ${kind} "${fileName}": the samples are ${[...samples.keys()].join(', ')}`)
  }
  if ('make' in source) return source.make()
  const sample = path.join(source.folder, fileName)
  if (!fs.existsSync(sample)) throw new Error(`the sample ${kind} "${fileName}" is missing at ${sample}`)
  return fs.readFileSync(sample)
}
