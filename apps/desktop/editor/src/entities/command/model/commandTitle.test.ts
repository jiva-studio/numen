import { describe, expect, it } from 'vitest'
import { createCommandTitleReader } from './commandTitle'

describe('the title of a command', () => {
  const getTitle = createCommandTitleReader([
    { id: 'app:settings.open', text: 'Open settings' },
    { id: 'app:vault.create', text: 'Create vault' },
  ])

  it('is the text of the command with the id', () => {
    expect(getTitle('app:vault.create')).toBe('Create vault')
  })

  it('is empty for an id no command has', () => {
    expect(getTitle('app:orchard.prune')).toBe('')
  })
})
