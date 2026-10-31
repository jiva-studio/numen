/**
 * What the link inspector shows of the core, and the one call a save makes.
 */
import { describe, expect, it } from 'vitest'
import type { Connection } from '@numen/plugin-sdk'
import { createNodeIdMap } from '../lib/nodeIdMap'
import { createVault, viewOn } from '../fixtures'
import { setSideLabel, setSidePresent, toggleSideArrow } from './connectionDraft'
import { useLinkInspector } from './useLinkInspector'

/** The connection the core reports between Root.md and Child.md. */
const reported: Connection = {
  first: 'Root.md',
  second: 'Child.md',
  role: 'child',
  type: '',
  firstSide: { label: 'owns', marker: null, reason: 'kept' },
  secondSide: null,
}

/** An inspector over Root.md, on the pair of Root.md and Child.md, over a core that reports the connection given. */
const inspecting = (connection: Connection | null = reported, takes = true) => {
  const plex = viewOn('Root.md', ['Child.md'])
  const vault = createVault(takes, connection)
  const map = createNodeIdMap()
  const messages: string[] = []
  const inspector = useLinkInspector(
    plex.view,
    {
      editor: vault.editor,
      roles: ['parent', 'child', 'jump'],
      showMessage: (text) => messages.push(text),
    },
    map,
    (path) => path.replace(/\.md$/, ''),
  )
  const pair = `${map.getNodeId('Root.md')} ${map.getNodeId('Child.md')}`
  const open = async () => {
    await inspector.openLinkInspector(pair, { x: 1, y: 2 })
    return inspector.linkInspector.value
  }
  return { ...inspector, ...vault, messages, went: plex.went, view: plex.view, pair, open }
}

describe('opening the link inspector', () => {
  it('shows the connection the core reports, with the kinds of link the host offers', async () => {
    const one = inspecting()

    const request = await one.open()

    expect(request?.nodeA).toMatchObject({ title: 'Root', path: 'Root.md' })
    expect(request?.nodeB).toMatchObject({ title: 'Child', path: 'Child.md' })
    expect(request?.connection).toEqual(reported)
    expect(request?.roles).toStrictEqual(['parent', 'child', 'jump'])
  })

  it('shows a pair that no record joins as a connection with no records', async () => {
    const one = inspecting(null)

    const request = await one.open()

    expect(request?.connection).toMatchObject({ firstSide: null, secondSide: null })
  })

  it('opens on nothing where a node was never drawn or the pair is malformed', async () => {
    const one = inspecting()

    await one.openLinkInspector('nowhere elsewhere', { x: 1, y: 2 })
    expect(one.linkInspector.value).toBeNull()

    await one.openLinkInspector('single', { x: 1, y: 2 })
    expect(one.linkInspector.value).toBeNull()
  })

  it('says why and opens on nothing where the core would not report the connection', async () => {
    const one = inspecting()
    const plex = viewOn('Root.md', ['Child.md'])
    const map = createNodeIdMap()
    const messages: string[] = []
    const refusing = useLinkInspector(
      plex.view,
      {
        editor: {
          ...one.editor,
          getConnection: async () => ({ ok: false, error: 'unreadable' }),
        },
        roles: [],
        showMessage: (text) => messages.push(text),
      },
      map,
      (path) => path,
    )

    await refusing.openLinkInspector(`${map.getNodeId('Root.md')} ${map.getNodeId('Child.md')}`, {
      x: 0,
      y: 0,
    })

    expect(refusing.linkInspector.value).toBeNull()
    expect(messages).toHaveLength(1)
  })

  it('closes on dismissal', async () => {
    const one = inspecting()
    await one.open()

    one.dismissLinkInspector()

    expect(one.linkInspector.value).toBeNull()
  })
})

describe('saving the link inspector', () => {
  it('saves the connection in one call and shows the picture again', async () => {
    const one = inspecting()
    const request = await one.open()
    const edited = toggleSideArrow(setSideLabel(request!.connection, 'first', 'holds'), 'first')

    await one.saveLinkInspector(edited)

    expect(one.saved).toEqual([
      {
        ...reported,
        firstSide: { label: 'holds', marker: 'arrow', reason: 'kept' },
      },
    ])
    expect(one.went).toStrictEqual(['Root.md'])
    expect(one.linkInspector.value).toBeNull()
  })

  it('saves a record that carries an arrow and an empty label', async () => {
    const one = inspecting(null)
    const request = await one.open()
    const edited = toggleSideArrow(setSidePresent(request!.connection, 'second', true), 'second')

    await one.saveLinkInspector(edited)

    expect(one.saved[0]?.secondSide).toEqual({ label: '', marker: 'arrow', reason: '' })
  })

  it('asks nothing of the core where the draft is what the core reported', async () => {
    const one = inspecting()
    const request = await one.open()

    await one.saveLinkInspector(request!.connection)

    expect(one.saved).toEqual([])
    expect(one.went).toStrictEqual([])
    expect(one.linkInspector.value).toBeNull()
  })

  it('stays open and says why where the core rejected', async () => {
    const one = inspecting(reported, false)
    const request = await one.open()

    await one.saveLinkInspector(setSideLabel(request!.connection, 'first', 'typed'))

    expect(one.saved).toHaveLength(1)
    expect(one.linkInspector.value).not.toBeNull()
    expect(one.messages).toStrictEqual(['a note cannot sit above itself in the hierarchy'])
    expect(one.went).toStrictEqual(['Root.md'])
  })

  it('does nothing where no inspector is open', async () => {
    const one = inspecting()

    await one.saveLinkInspector(reported)
    await one.removeLinkInspector()

    expect(one.saved).toEqual([])
    expect(one.removed).toEqual([])
  })

  it('travels nowhere while the plex remains nowhere', async () => {
    const one = inspecting()
    const request = await one.open()
    one.view.here.value = ''

    await one.saveLinkInspector(setSideLabel(request!.connection, 'first', 'x'))

    expect(one.went).toStrictEqual([])
    expect(one.linkInspector.value).toBeNull()
  })
})

describe('removing the connection', () => {
  it('removes both records of the pair under the type of the connection and closes', async () => {
    const one = inspecting()
    await one.open()

    await one.removeLinkInspector()

    expect(one.removed).toEqual([['Root.md', 'Child.md']])
    expect(one.went).toStrictEqual(['Root.md'])
    expect(one.linkInspector.value).toBeNull()
  })

  it('stays open where the core would not remove it', async () => {
    const one = inspecting(reported, false)
    await one.open()

    await one.removeLinkInspector()

    expect(one.linkInspector.value).not.toBeNull()
    expect(one.messages).toHaveLength(1)
  })
})
