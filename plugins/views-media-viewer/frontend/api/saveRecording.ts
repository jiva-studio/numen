import { asFailure, asValue, createApi, type HostServices, type Result } from '@numen/plugin-sdk'

const api = createApi('md.numen.views.media.viewer')

export interface RecorderTab {
  readonly save: (audio: Blob, startedAt: string) => Promise<Result<string, string>>
  readonly open: (path: string) => Promise<void>
}

function readBase64(audio: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(reader.error ?? new Error('Could not read the recorded audio'))
    reader.onload = () => resolve(String(reader.result).split(',')[1] ?? '')
    reader.readAsDataURL(audio)
  })
}

export function createRecorderTab(folder: string, host: HostServices): RecorderTab {
  const getVault = () => host.vault.vaults?.getState().current?.id
  const vaultId = getVault()
  return {
    async save(audio, startedAt) {
      if (!vaultId) return asFailure('Open a vault to save the recording')
      const request = {
        vaultId,
        folder,
        startedAt,
        mediaType: audio.type,
        content: await readBase64(audio),
      }
      const result = await api.call<typeof request, { path: string }>('SaveRecording', request)
      return result.ok ? asValue(result.value.path) : asFailure(result.error.message)
    },
    async open(path) {
      if (getVault() === vaultId) await host.tabs.open?.('media', path)
    },
  }
}
