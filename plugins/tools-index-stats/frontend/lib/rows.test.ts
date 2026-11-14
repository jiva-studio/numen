import { describe, expect, it } from 'vitest'
import type { Stats } from '../api/stats'
import { getRows } from './rows'

const BASE: Stats = {
  vectors: 12480,
  chunks: 13100,
  waiting: 620,
  sources: 84,
  kinds: [
    { kind: 'notes', sources: 70, chunks: 4100 },
    { kind: 'pdf', sources: 9, chunks: 7000 },
    { kind: 'epub', sources: 5, chunks: 2000 },
  ],
  model: 'bge-m3',
  recipe: 'local|bge-m3|1024|512|mean|int8@0.4',
  dimensions: 1024,
  profile: 'balanced',
  threads: 4,
  speed: 18.46,
  changedAt: '2026-11-02T12:00:07Z',
  databaseBytes: 73400320,
  vectorBytes: 52428800,
}

const ids = (stats: Stats) => getRows(stats, 'en-US').map((one) => one.id)
const valueOf = (stats: Stats, id: string) =>
  getRows(stats, 'en-US').find((one) => one.id === id)?.value

describe('getRows', () => {
  it('lists the rows in the order the popover shows them', () => {
    expect(ids(BASE)).toEqual([
      'vectors',
      'chunks',
      'waiting',
      'sources',
      'kind-notes',
      'kind-pdf',
      'kind-epub',
      'model',
      'dimensions',
      'profile',
      'speed',
      'changed-at',
      'database',
      'vector-size',
    ])
  })

  it('writes the counts, the profile and the sizes', () => {
    expect(valueOf(BASE, 'vectors')).toBe('12,480')
    expect(valueOf(BASE, 'chunks')).toBe('13,100')
    expect(valueOf(BASE, 'waiting')).toBe('620')
    expect(valueOf(BASE, 'sources')).toBe('84')
    expect(valueOf(BASE, 'kind-pdf')).toBe('7,000 chunks · 9 files')
    expect(valueOf(BASE, 'dimensions')).toBe('1,024 dimensions')
    expect(valueOf(BASE, 'profile')).toBe('Balanced · 4 threads')
    expect(valueOf(BASE, 'speed')).toBe('18.5 per second')
    expect(valueOf(BASE, 'database')).toBe('70.0 MB')
    expect(valueOf(BASE, 'vector-size')).toBe('50.0 MB')
  })

  it('hides a kind that holds no source and no chunk', () => {
    const stats: Stats = {
      ...BASE,
      kinds: [
        { kind: 'notes', sources: 70, chunks: 4100 },
        { kind: 'pdf', sources: 0, chunks: 0 },
        { kind: 'media', sources: 0, chunks: 0 },
      ],
    }

    expect(ids(stats)).toContain('kind-notes')
    expect(ids(stats)).not.toContain('kind-pdf')
    expect(ids(stats)).not.toContain('kind-media')
  })

  it('shows a kind that holds a source with no chunk', () => {
    const stats: Stats = { ...BASE, kinds: [{ kind: 'other', sources: 2, chunks: 0 }] }

    expect(valueOf(stats, 'kind-other')).toBe('0 chunks · 2 files')
  })

  it('shows the speed only while chunks wait, and a dash before it is measured', () => {
    const embedded: Stats = { ...BASE, vectors: 13100, waiting: 0 }
    expect(ids(embedded)).not.toContain('speed')

    const { speed: _speed, ...unmeasured } = BASE
    expect(valueOf(unmeasured, 'speed')).toBe('–')
  })

  it('shows a dash for the last change before one was seen', () => {
    const { changedAt: _changedAt, ...unchanged } = BASE

    expect(valueOf(unchanged, 'changed-at')).toBe('–')
    expect(valueOf(BASE, 'changed-at')).not.toBe('–')
  })

  it('says thread in the singular for one thread, and names a profile it does not know as given', () => {
    expect(valueOf({ ...BASE, profile: 'eco', threads: 1 }, 'profile')).toBe('Eco · 1 thread')
    expect(valueOf({ ...BASE, profile: 'turbo' }, 'profile')).toBe('turbo · 4 threads')
  })

  it('gives a hint to the rows whose meaning is not obvious only', () => {
    const rows = getRows(BASE, 'en-US')

    expect(rows.find((one) => one.id === 'waiting')?.hint).not.toBe('')
    expect(rows.find((one) => one.id === 'model')?.hint).toBe('')
    expect(rows.find((one) => one.id === 'kind-pdf')?.hint).toBe('')
  })

  it('nests the kinds under the notes and files', () => {
    const rows = getRows(BASE, 'en-US')

    expect(rows.filter((one) => one.isNested).map((one) => one.id)).toEqual([
      'kind-notes',
      'kind-pdf',
      'kind-epub',
    ])
  })
})
