import { describe, expect, it } from 'vitest'
import { useSecretDrafts } from './secretDrafts'

describe('the secrets added and not created yet', () => {
  it('get an identifier each, also where they share a name', () => {
    const { getDrafts, addDraft } = useSecretDrafts()
    addDraft('travel', 'Default V1')
    addDraft('travel', 'Default V1')
    const [first, second] = getDrafts('travel')
    expect(first?.id).not.toBe(second?.id)
  })

  it('are listed by group', () => {
    const { getDrafts, addDraft } = useSecretDrafts()
    addDraft('travel', 'Default V1')
    addDraft('work', 'Default V1')
    expect(getDrafts('travel').map((one) => one.group)).toStrictEqual(['travel'])
  })

  it('are discarded one at a time by identifier', () => {
    const { getDrafts, addDraft, discardDraft } = useSecretDrafts()
    addDraft('travel', 'Default V1')
    addDraft('travel', 'Default V1')
    discardDraft(getDrafts('travel')[0]!.id)
    expect(getDrafts('travel')).toHaveLength(1)
  })

  it('keep their identifiers apart after one is discarded', () => {
    const { getDrafts, addDraft, discardDraft } = useSecretDrafts()
    addDraft('travel', 'a')
    const [first] = getDrafts('travel')
    discardDraft(first!.id)
    addDraft('travel', 'a')
    expect(getDrafts('travel')[0]!.id).not.toBe(first!.id)
  })
})

describe('the groups added and holding no secret', () => {
  it('are added, renamed and removed', () => {
    const { emptyGroups, addEmptyGroup, renameEmptyGroup, removeEmptyGroup } = useSecretDrafts()
    addEmptyGroup('New group')
    addEmptyGroup('other')
    renameEmptyGroup('New group', 'Local Llama')
    expect(emptyGroups.value).toStrictEqual(['Local Llama', 'other'])
    removeEmptyGroup('other')
    expect(emptyGroups.value).toStrictEqual(['Local Llama'])
  })
})
