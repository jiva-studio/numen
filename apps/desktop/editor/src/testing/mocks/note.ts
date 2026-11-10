/**
 * The note entity's part of the vault mock: reading, writing and finding notes.
 */
import { asValue } from '@numen/core-client'
import { requests } from '../requests'
import { mockVaultState, pendingStream } from '../answers'
import type { VaultCore } from '@/app/vault/core'

export const noteCoreMock = {
  getInitialOpenPath: () =>
    Promise.resolve(mockVaultState.opening ? { path: mockVaultState.opening } : null),
  neighbourhood: (path: string) =>
    Promise.resolve({
      focus: { path, title: path.replace(/\.md$/, '') },
      focusType: 'note' as const,
      related: [],
      connections: [],
    }),
  read: () => Promise.resolve(asValue({ body: 'what is written', at: 'a1' })),
  write: () => Promise.resolve(asValue({ body: '', at: 'a2' })),
  create: ({ title }: { title: string }) => {
    requests.made.push(title)
    return Promise.resolve(asValue({ path: `${title}.md` }))
  },
  rename: (path: string, title: string) => {
    requests.renamed.push(`${path} ${title}`)
    return Promise.resolve(asValue({ path, title, hasFrontmatter: true, moved: null }))
  },
  watchEdits: pendingStream,
  names: () => Promise.resolve(mockVaultState.names),
  search: () => Promise.resolve(mockVaultState.passages),
  headings: () => Promise.resolve(new Map()),
} satisfies Partial<VaultCore>
