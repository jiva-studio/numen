import { windowService } from '@/shared/api/clients'
import { WINDOW } from './window'

/** The id of the vault this window shows. */
export async function getShownVault(): Promise<string> {
  return (await windowService.getCurrentVault({ window: WINDOW })).vault
}
