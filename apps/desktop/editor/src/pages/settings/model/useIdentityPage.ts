/**
 * The state of the page of identity and licenses: the licenses and the views plugins registered for the page.
 */
import { computed } from 'vue'
import type { InstallationLicenses, InstallationPlugin } from '../types'
import { useLicensesPage } from './useLicensesPage'

export function useIdentityPage(
  getLicenses: () => InstallationLicenses,
  getPlugin: () => Pick<InstallationPlugin, 'getIdentityViews'>,
) {
  const licenses = useLicensesPage(getLicenses)
  const views = computed(() => getPlugin().getIdentityViews())
  return { ...licenses, views }
}
