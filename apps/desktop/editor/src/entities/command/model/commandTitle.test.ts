import { describe, expect, it } from 'vitest'
import { createCommandTitleReader } from './commandTitle'

describe('the title of a command', () => {
  const createReader = () =>
    createCommandTitleReader([
      { id: 'app:settings.open', text: 'Open settings' },
      { id: 'app:vault.create', text: 'Create vault' },
    ])

  it('is the text of the command with the id', () => {
    expect(createReader()('app:vault.create')).toBe('Create vault')
  })

  it('is empty for an id no command has', () => {
    expect(createReader()('app:orchard.prune')).toBe('')
  })
})
