/**
 * The text versions a plugin's views read and edit: reading needs `vault:read`, an edit
 * `vault:write`, and each plugin is handed texts of its own.
 */
import { describe, expect, it, vi } from 'vitest'
import type { VaultTexts } from '@numen/plugin-sdk'
import { createPluginContext } from './context'
import { createTestContextHost } from '../testing/contextHost'

const createContext = (permissions: readonly string[]) => {
  const texts = {
    readLiveText: vi.fn(() => Promise.resolve(null)),
    createEdit: vi.fn(() => Promise.resolve({ ok: false, error: 'missing' })),
  } as unknown as VaultTexts
  const getTexts = vi.fn(() => texts)
  const host = createTestContextHost({ vault: { getTexts } })
  const context = createPluginContext('md.numen.player', host, {
    id: 'md.numen.player',
    name: 'Player',
    version: '1.0.0',
    permissions,
  })
  return { context, texts, getTexts }
}

describe('the texts of a plugin', () => {
  it('are made for the plugin that asks', () => {
    const { getTexts } = createContext(['vault:read'])

    expect(getTexts).toHaveBeenCalledWith('md.numen.player')
  })

  it('read the live text with vault:read', async () => {
    const { context, texts } = createContext(['vault:read'])

    await context.vault.texts!.readLiveText('talk.mp3')

    expect(texts.readLiveText).toHaveBeenCalledWith('talk.mp3')
  })

  it('refuse a read without vault:read and an edit without vault:write', async () => {
    const { context: noRead, texts: noReadTexts } = createContext([])
    await expect(noRead.vault.texts!.readLiveText('talk.mp3')).rejects.toThrowError(
      'Plugin "md.numen.player" lacks required permission "vault:read" for texts.readLiveText',
    )
    expect(noReadTexts.readLiveText).not.toHaveBeenCalled()

    const { context, texts } = createContext(['vault:read'])

    await expect(context.vault.texts!.createEdit('talk.mp3', 'p/k', [])).rejects.toThrowError(
      'Plugin "md.numen.player" lacks required permission "vault:write" for texts.createEdit',
    )
    expect(texts.createEdit).not.toHaveBeenCalled()
  })

  it('record an edit with vault:write', async () => {
    const { context, texts } = createContext(['vault:read', 'vault:write'])

    await context.vault.texts!.createEdit('talk.mp3', 'p/k', [{ text: 'Hello.\n' }])

    expect(texts.createEdit).toHaveBeenCalledWith('talk.mp3', 'p/k', [{ text: 'Hello.\n' }])
  })
})
