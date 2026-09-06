/**
 * The one thing this module decides on its own: how a file the window carries
 * about is written down and read back.
 *
 * Everything else here hands a message across unchanged, and the schema says
 * what those look like.
 */
import { describe, expect, it, vi } from 'vitest'
import { Role } from '@numen/protocol'
import type { Role as WindowRole } from './core'

vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })

// Only the client is stood in for. The rest of the module is what `@numen/wire`
// reads its codes off, and a mock naming one export hides the others from it.
vi.mock('@connectrpc/connect', async (actual) => ({
  ...(await actual<typeof import('@connectrpc/connect')>()),
  createClient: () => asked,
}))

const asked = {
  readNote: vi.fn(),
  writeNote: vi.fn(),
  createNote: vi.fn(),
  writeLink: vi.fn(),
}

const { core } = await import('./vault')

describe('the file a save presents', () => {
  it('comes back as the file it was, through a path that holds spaces', async () => {
    const at = { path: 'mahabharata/The aggressor.md', size: 4096n, mtime: 1_700_000_000_123n }
    asked.readNote.mockResolvedValue({ body: 'prose', at })
    asked.writeNote.mockResolvedValue({})

    const read = await core.read(at.path)
    await core.write(at.path, 'prose and more', { prose: read.body, at: read.at ?? '' })

    expect(asked.writeNote.mock.calls[0]?.[0]).toEqual({
      path: at.path,
      body: 'prose and more',
      seen: { prose: 'prose', at },
    })
  })

  it('is left out of a save that presents nothing', async () => {
    asked.writeNote.mockResolvedValue({})

    await core.write('Heat.md', 'mine', null)

    expect(asked.writeNote.mock.calls.at(-1)?.[0]).toEqual({ path: 'Heat.md', body: 'mine' })
  })
})

describe('the role a link carries', () => {
  it('goes to the schema under the word the vault uses for it', async () => {
    asked.createNote.mockResolvedValue({ path: 'Entropy.md' })

    await core.create({
      title: 'Entropy',
      folder: '',
      links: [{ to: 'Ontology.md', role: 'jump', label: 'see also' }],
    })

    expect(asked.createNote.mock.calls[0]?.[0].links).toEqual([
      { to: 'Ontology.md', role: Role.JUMP, label: 'see also' },
    ])
  })

  // Taken from the schema rather than typed out, so a role added there is one
  // this asks for, and the window is caught having no word for it.
  it('is every role the schema names, on a link written into a note that is there', async () => {
    const carried = Object.keys(Role).filter(
      (name) => Number.isNaN(Number(name)) && name !== 'UNSPECIFIED',
    )
    expect(carried).toContain('ATTACHMENT')

    for (const name of carried) {
      asked.writeLink.mockResolvedValue({})

      await core.join('Ontology.md', {
        to: 'Entropy.md',
        role: name.toLowerCase() as WindowRole,
      })

      expect(asked.writeLink.mock.calls.at(-1)?.[0].link).toEqual({
        to: 'Entropy.md',
        role: Role[name as keyof typeof Role],
        label: '',
      })
    }
  })
})
