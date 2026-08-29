/**
 * The one thing this module decides on its own: how a file the window carries
 * about is written down and read back.
 *
 * Everything else here hands a message across unchanged, and the schema says
 * what those look like.
 */
import { describe, expect, it, vi } from 'vitest'
import { Role } from '@numen/protocol'

vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })

vi.mock('@connectrpc/connect-web', () => ({ createConnectTransport: () => ({}) }))
vi.mock('@connectrpc/connect', () => ({ createClient: () => asked }))

const asked = {
  read: vi.fn(),
  write: vi.fn(),
  create: vi.fn(),
  join: vi.fn(),
}

const { core } = await import('./vault')

describe('the file a save presents', () => {
  it('comes back as the file it was, through a path that holds spaces', async () => {
    const at = { path: 'mahabharata/The aggressor.md', size: 4096n, mtime: 1_700_000_000_123n }
    asked.read.mockResolvedValue({ body: 'prose', at })
    asked.write.mockResolvedValue({})

    const read = await core.read(at.path)
    await core.write(at.path, 'prose and more', { prose: read.body, at: read.at ?? '' })

    expect(asked.write.mock.calls[0]?.[0]).toEqual({
      path: at.path,
      body: 'prose and more',
      seen: { prose: 'prose', at },
    })
  })

  it('is left out of a save that presents nothing', async () => {
    asked.write.mockResolvedValue({})

    await core.write('Heat.md', 'mine', null)

    expect(asked.write.mock.calls.at(-1)?.[0]).toEqual({ path: 'Heat.md', body: 'mine' })
  })
})

describe('the role a link carries', () => {
  it('goes to the schema under the word the vault uses for it', async () => {
    asked.create.mockResolvedValue({ path: 'Entropy.md' })

    await core.create({
      title: 'Entropy',
      folder: '',
      links: [{ to: 'Ontology.md', role: 'jump', label: 'see also' }],
    })

    expect(asked.create.mock.calls[0]?.[0].links).toEqual([
      { to: 'Ontology.md', role: Role.JUMP, label: 'see also' },
    ])
  })

  it('is every role the vault acts on, on a link written into a note that is there', async () => {
    for (const [role, named] of [
      ['parent', Role.PARENT],
      ['child', Role.CHILD],
      ['jump', Role.JUMP],
      ['ref', Role.REF],
      ['attachment', Role.ATTACHMENT],
    ] as const) {
      asked.join.mockResolvedValue({})

      await core.join('Ontology.md', { to: 'Entropy.md', role })

      expect(asked.join.mock.calls.at(-1)?.[0].link).toEqual({
        to: 'Entropy.md',
        role: named,
        label: '',
      })
    }
  })
})
