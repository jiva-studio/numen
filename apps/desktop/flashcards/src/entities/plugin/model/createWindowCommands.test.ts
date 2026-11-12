/**
 * The commands the flashcards window gives its plugins: none, since the window registers no command
 * and binds none a plugin registers.
 */
import { describe, expect, it, vi } from 'vitest'
import { createWindowCommands } from './createWindowCommands'

describe('the commands of the flashcards window', () => {
  it('are none, now and after a watcher is added', () => {
    const commands = createWindowCommands()
    const handler = vi.fn()

    commands.watch(handler).dispose()

    expect(commands.getIds()).toStrictEqual([])
    expect(handler).toHaveBeenCalledExactlyOnceWith([])
  })

  it('offer no choices', async () => {
    expect(await createWindowCommands().listItems('app:vault.create')).toStrictEqual([])
  })

  it('refuse a run, naming the command the window does not run', async () => {
    await expect(createWindowCommands().run('app:vault.create')).rejects.toThrowError(
      'The flashcards window runs no command app:vault.create',
    )
  })
})
