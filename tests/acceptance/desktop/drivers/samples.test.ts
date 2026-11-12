import { describe, expect, it } from 'vitest'
import { createPagedPdf, createSampleWav, readSample } from '../support/samples.js'

describe('readSample', () => {
  it.each([
    ['PDF', 'shore.pdf', '%PDF'],
    ['PDF', 'pumped-storage.pdf', '%PDF'],
    ['EPUB', 'ledger.epub', 'PK'],
    ['EPUB', 'sample.epub', 'PK'],
    ['recording', 'lecture.wav', 'RIFF'],
    ['image', 'llama.png', '\x89PNG'],
  ] as const)('reads the %s "%s" from the repository', (kind, name, start) => {
    expect(readSample(kind, name).subarray(0, start.length).toString('latin1')).toBe(start)
  })

  it('reads the recording talk.mp3 from the repository', () => {
    expect(readSample('recording', 'talk.mp3').length).toBeGreaterThan(512)
  })

  it('makes a PDF of forty pages', () => {
    const pdf = readSample('PDF', 'pages.pdf').toString('latin1')
    expect(pdf.startsWith('%PDF-1.4')).toBe(true)
    expect(pdf).toContain('/Count 40')
  })

  it('makes the files the application must refuse', () => {
    expect(readSample('PDF', 'corrupted.pdf').toString()).toBe('INVALID_PDF_HEADER')
    expect(readSample('EPUB', 'corrupted.epub').toString()).toBe('INVALID_EPUB_ZIP')
  })

  it('makes the empty recordings of 512 bytes', () => {
    expect(readSample('recording', 'sample.mp3')).toHaveLength(512)
    expect(readSample('recording', 'sample.flac')).toHaveLength(512)
  })

  it('names the samples of the kind when the file is not one', () => {
    expect(() => readSample('PDF', 'unknown.pdf')).toThrow('there is no sample PDF "unknown.pdf": the samples are flywheel.pdf')
  })

  it('does not give a PDF for a recording', () => {
    expect(() => readSample('recording', 'shore.pdf')).toThrow('there is no sample recording "shore.pdf"')
  })
})

describe('createSampleWav', () => {
  it('is one second of 16 bit mono samples at 16 kHz', () => {
    const wav = createSampleWav()
    expect(wav.subarray(0, 4).toString()).toBe('RIFF')
    expect(wav.readUInt32LE(24)).toBe(16000)
    expect(wav.readUInt32LE(40)).toBe(32000)
    expect(wav).toHaveLength(44 + 32000)
  })
})

describe('createPagedPdf', () => {
  it('has the number of pages it is given', () => {
    expect(createPagedPdf(3).toString('latin1')).toContain('/Count 3')
  })
})
