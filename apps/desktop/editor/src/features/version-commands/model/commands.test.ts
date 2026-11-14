/**
 * Delete text, the command of the window the core serves over the text versions of a file, carried
 * out over the real call of the plugin entity. Every path is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { COMMAND_ERROR_CODE } from '@numen/core-client'
import { createHandledApi } from '@/testing/coreAnswers'
import type { CommandTarget } from '@/shared/model/command'
import { invoke } from '@/testing/invocation'
import { createVersionCommands } from './commands'

const DELETE_TEXT = 'app:version.delete'

const createTarget = (path: string, isFolder = false): CommandTarget => ({
  tab: '',
  kind: null,
  path,
  title: path,
  file: path,
  isFolder,
  vault: { id: 'v', name: 'V' },
  isReady: true,
})

const createWindow = (
  answer: { error: number; message: string } = {
    error: COMMAND_ERROR_CODE.UNSPECIFIED,
    message: '',
  },
) => {
  const requests: { pluginId: string; command: string; path: string; itemId?: string }[] = []
  const api = createHandledApi({
    runPluginCommand: (request: {
      pluginId: string
      command: string
      path: string
      itemId?: string
    }) => {
      requests.push(request)
      return { ...answer, openedPath: '' }
    },
  })
  const listItems = vi.fn(async () => [{ id: 'v1', title: 'transcript (live)' }])
  const messages: { text: string; kind: string | undefined }[] = []
  const set = createVersionCommands({
    runDeclaredHandler: api.declaredCommands.runDeclaredHandler,
    listItems,
    writeMessage: (text, kind) => {
      messages.push({ text, kind })
    },
  })
  return { set, listItems, requests, messages }
}

describe('Delete text', () => {
  it('lists the versions the core lists over the file, with no plugin', async () => {
    const { set, listItems } = createWindow()
    const [command] = set.commands

    expect(command?.text).toBe('Delete text')
    expect(await command?.listItems?.(createTarget('talks/a.mp3'))).toStrictEqual([
      { id: 'v1', title: 'transcript (live)' },
    ])
    expect(listItems).toHaveBeenCalledWith({ pluginId: '', id: DELETE_TEXT }, 'talks/a.mp3')
  })

  it('is available over a file and over no folder', () => {
    const [command] = createWindow().set.commands

    expect(command?.isAvailable(createTarget('talks/a.mp3'), {} as never)).toBe(true)
    expect(command?.isAvailable(createTarget('talks', true), {} as never)).toBe(false)
    expect(command?.isAvailable(createTarget(''), {} as never)).toBe(false)
  })

  it('has the core delete the version picked, and writes nothing of a delete that went through', async () => {
    const { set, requests, messages } = createWindow()

    await set.handlers[DELETE_TEXT]?.(invoke(DELETE_TEXT, createTarget('talks/a.mp3'), 'v1'))

    expect(requests).toMatchObject([
      { pluginId: '', command: DELETE_TEXT, path: 'talks/a.mp3', itemId: 'v1' },
    ])
    expect(messages).toStrictEqual([])
  })

  it('writes the error of a delete that failed', async () => {
    const { set, messages } = createWindow({
      error: COMMAND_ERROR_CODE.FAILED,
      message: 'no such version',
    })

    await set.handlers[DELETE_TEXT]?.(invoke(DELETE_TEXT, createTarget('talks/a.mp3'), 'v1'))

    expect(messages).toStrictEqual([{ text: 'no such version', kind: 'error' }])
  })

  it('asks the core for nothing until a version is picked', async () => {
    const { set, requests } = createWindow()

    await set.handlers[DELETE_TEXT]?.(invoke(DELETE_TEXT, createTarget('talks/a.mp3')))

    expect(requests).toStrictEqual([])
  })
})
