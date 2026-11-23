/**
 * The calls of the note service that read and write connections between notes, over the real client
 * and a router in place of the core.
 */
import { describe, expect, it, vi } from 'vitest'
import { NOTE_ERROR_CODE, NOTE_MARKER, NOTE_ROLE } from '@numen/core-client'
import { createHandledClients } from '@/testing/coreAnswers'
import { createLinkApi } from './notes'
import { createRelations } from './relations'

const request = {
  getConnection: vi.fn(),
  saveConnection: vi.fn(),
  deleteConnectionSide: vi.fn(),
  deleteConnection: vi.fn(),
  createConnectionSide: vi.fn(),
  createRelatedNote: vi.fn(),
  listBacklinks: vi.fn(),
  resolveAddresses: vi.fn(),
}

const clients = createHandledClients(request)
const getRelations = () => createRelations(clients.notes, createLinkApi(clients.notes).resolveLinks)

describe('the connections between notes', () => {
  it('are read with the role the core derived, and the markers of their records', async () => {
    const relations = getRelations()
    request.getConnection.mockResolvedValue({
      connection: {
        first: 'a.md',
        second: 'b.md',
        role: NOTE_ROLE.PARENT,
        firstSide: { label: 'under', note: 'why', marker: NOTE_MARKER.ARROW },
        secondSide: { label: '', note: '', marker: NOTE_MARKER.UNSPECIFIED },
      },
    })

    const read = await relations.getConnection('a.md', 'b.md', 'preset')

    expect(request.getConnection.mock.calls.at(-1)?.[0]).toMatchObject({
      first: 'a.md',
      second: 'b.md',
      type: 'preset',
    })
    expect(read).toEqual({
      ok: true,
      value: {
        first: 'a.md',
        second: 'b.md',
        role: 'parent',
        type: 'preset',
        firstSide: { label: 'under', marker: 'arrow', reason: 'why' },
        secondSide: { label: '', marker: null, reason: '' },
      },
    })

    request.getConnection.mockResolvedValueOnce({ error: NOTE_ERROR_CODE.NOT_FOUND })
    expect(await relations.getConnection('a.md', 'b.md')).toEqual({
      ok: false,
      error: 'notFound',
    })
    expect(request.getConnection.mock.calls.at(-1)?.[0]).toMatchObject({
      first: 'a.md',
      second: 'b.md',
      type: '',
    })

    request.getConnection.mockResolvedValueOnce({})
    expect(await relations.getConnection('a.md', 'b.md')).toEqual({
      ok: false,
      error: 'unreadable',
    })
  })

  it('are saved in one request, with no role on their records', async () => {
    const relations = getRelations()
    request.saveConnection.mockResolvedValue({})

    const saved = await relations.saveConnection({
      first: 'a.md',
      second: 'b.md',
      role: 'child',
      type: '',
      firstSide: { label: '', marker: 'arrow', reason: '' },
      secondSide: null,
    })

    expect(saved).toEqual({ ok: true, value: undefined })
    expect(request.saveConnection.mock.calls.at(-1)?.[0]).toMatchObject({
      connection: {
        first: 'a.md',
        second: 'b.md',
        role: NOTE_ROLE.CHILD,
        firstSide: { type: '', note: '', label: '', marker: NOTE_MARKER.ARROW },
      },
    })
    expect(request.saveConnection.mock.calls.at(-1)?.[0].connection.secondSide).toBeUndefined()
  })

  it('are named by why the core refused them', async () => {
    const relations = getRelations()
    const connection = {
      first: 'a.md',
      second: 'b.md',
      role: null,
      type: '',
      firstSide: null,
      secondSide: null,
    } as const
    request.saveConnection.mockResolvedValueOnce({ isCycle: true })
    expect(await relations.saveConnection(connection)).toEqual({ ok: false, error: 'cycle' })
    request.saveConnection.mockResolvedValueOnce({ isInvalid: true })
    expect(await relations.saveConnection(connection)).toEqual({ ok: false, error: 'invalid' })
    request.saveConnection.mockResolvedValueOnce({ error: NOTE_ERROR_CODE.NOT_FOUND })
    expect(await relations.saveConnection(connection)).toEqual({ ok: false, error: 'notFound' })
  })

  it('have one side, or both, removed under the type of the records', async () => {
    const relations = getRelations()
    request.deleteConnectionSide.mockResolvedValue({})
    request.deleteConnection.mockResolvedValue({})

    expect(await relations.deleteConnectionSide('a.md', 'b.md', 'preset')).toEqual({
      ok: true,
      value: undefined,
    })
    expect(await relations.deleteConnection('a.md', 'b.md')).toEqual({
      ok: true,
      value: undefined,
    })

    expect(request.deleteConnectionSide.mock.calls.at(-1)?.[0]).toMatchObject({
      path: 'a.md',
      to: 'b.md',
      type: 'preset',
    })
    expect(request.deleteConnection.mock.calls.at(-1)?.[0]).toMatchObject({
      first: 'a.md',
      second: 'b.md',
      type: '',
    })
  })

  it('have one record written under the wire role and marker', async () => {
    const relations = getRelations()
    request.createConnectionSide.mockResolvedValue({})

    expect(
      await relations.join('a.md', 'b.md', { role: 'jump', label: 'see', marker: 'arrow' }),
    ).toEqual({ ok: true, value: undefined })

    expect(request.createConnectionSide.mock.calls.at(-1)?.[0]).toMatchObject({
      from: 'a.md',
      to: 'b.md',
      side: { role: NOTE_ROLE.JUMP, type: '', note: '', label: 'see', marker: NOTE_MARKER.ARROW },
    })

    await relations.join('a.md', 'b.md', {
      role: 'child',
      type: 'preset',
      reason: 'because',
      label: 'part',
    })
    expect(request.createConnectionSide.mock.calls.at(-1)?.[0]).toMatchObject({
      from: 'a.md',
      to: 'b.md',
      side: {
        role: NOTE_ROLE.CHILD,
        type: 'preset',
        note: 'because',
        label: 'part',
        marker: NOTE_MARKER.UNSPECIFIED,
      },
    })
  })

  it('have a related note made by the core, which answers with its path', async () => {
    const relations = getRelations()
    request.createRelatedNote.mockResolvedValue({ path: 'physics/Untitled note.md' })

    const made = await relations.createRelated({ from: 'physics/a.md', role: 'child' })

    expect(made).toEqual({ ok: true, value: 'physics/Untitled note.md' })
    expect(request.createRelatedNote.mock.calls.at(-1)?.[0]).toMatchObject({
      from: 'physics/a.md',
      role: NOTE_ROLE.CHILD,
      title: '',
      type: '',
      marker: NOTE_MARKER.UNSPECIFIED,
    })

    await relations.createRelated({
      from: 'physics/a.md',
      role: 'parent',
      title: 'Overview',
      type: 'sample',
      marker: 'arrow',
    })
    expect(request.createRelatedNote.mock.calls.at(-1)?.[0]).toMatchObject({
      from: 'physics/a.md',
      role: NOTE_ROLE.PARENT,
      title: 'Overview',
      type: 'sample',
      marker: NOTE_MARKER.ARROW,
    })

    request.createRelatedNote.mockResolvedValue({ isNameExhausted: true })
    expect(await relations.createRelated({ from: 'a.md', role: 'jump' })).toEqual({
      ok: false,
      error: 'nameExhausted',
    })

    request.createRelatedNote.mockResolvedValue({ error: NOTE_ERROR_CODE.NOT_FOUND })
    expect(await relations.createRelated({ from: 'a.md', role: 'jump' })).toEqual({
      ok: false,
      error: 'notFound',
    })
  })

  it('have the backlinks of a note listed with both roles', async () => {
    const relations = getRelations()
    request.listBacklinks.mockResolvedValue({
      backlinks: [
        {
          from: { path: 'b.md', title: 'B' },
          role: NOTE_ROLE.PARENT,
          sourceRole: NOTE_ROLE.CHILD,
          type: '',
          note: '',
          label: 'under',
          marker: NOTE_MARKER.ARROW,
        },
      ],
    })

    expect(await relations.listBacklinks('a.md')).toEqual([
      {
        from: { path: 'b.md', title: 'B' },
        role: 'parent',
        sourceRole: 'child',
        type: '',
        reason: '',
        label: 'under',
        marker: 'arrow',
      },
    ])
    expect(request.listBacklinks.mock.lastCall?.[0]).toMatchObject({ path: 'a.md' })
  })
})

describe('the wikilinks of a text', () => {
  it('are resolved in the vault named, from no note', async () => {
    const relations = getRelations()
    request.resolveAddresses.mockResolvedValue({
      links: [{ raw: '[[Tides]]', address: 'name://Tides', text: 'Tides' }],
      resolvedAddresses: [{ address: 'name://Tides', path: 'Sea/Tides.md', isCrossed: false }],
    })

    expect(await relations.resolveLinks('See [[Tides]].', 'v-sea')).toEqual([
      { raw: '[[Tides]]', address: 'name://Tides', text: 'Tides', path: 'Sea/Tides.md' },
    ])
    expect(request.resolveAddresses.mock.lastCall?.[0]).toMatchObject({
      from: '',
      text: 'See [[Tides]].',
      vaultId: 'v-sea',
    })
  })
})
