/**
 * The state of the page of licenses and of each license card on it: the declared licenses, their
 * display labels and checkout options, and the activate and deactivate actions of a card.
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { LicenseInfo, LicenseProviderOptionInfo } from '@/entities/license'
import type { Result } from '@numen/result'
import { formatError } from '@/shared/api/errors'
import type { InstallationLicenses } from '../types'
import { WORDS as words } from '../words'

export function useLicensesPage(getLicenses: () => InstallationLicenses) {
  const licenses = computed(() => getLicenses().list.value)
  const watchAbort = new AbortController()

  onMounted(() => {
    runInBackground(getLicenses().load(), () => undefined)
    runInBackground(getLicenses().watch(watchAbort.signal), () => undefined)
  })

  onBeforeUnmount(() => {
    watchAbort.abort()
  })

  return {
    licenses,
    activate: (licenseId: string, key: string) => getLicenses().activate(licenseId, key),
    deactivate: (licenseId: string) => getLicenses().deactivate(licenseId),
  }
}

function getLicenseTitle(license: LicenseInfo): string {
  return license.title !== '' ? license.title : license.id
}

function getStatusText(license: LicenseInfo): string {
  switch (license.status) {
    case 'active':
      return license.instanceName !== ''
        ? `${words.licenseStatusActive} · ${license.instanceName}`
        : words.licenseStatusActive
    case 'expired':
      return words.licenseStatusExpired
    case 'unverified':
      return words.licenseStatusUnverified
    case 'inactive':
      return words.licenseStatusInactive
  }
}

function getBuyOptions(license: LicenseInfo): readonly LicenseProviderOptionInfo[] {
  return license.providers.filter((one) => one.checkoutUrl !== '')
}

export interface LicenseCardInput {
  readonly getLicense: () => LicenseInfo
  readonly activate: (licenseId: string, key: string) => Promise<Result<LicenseInfo, string>>
  readonly deactivate: (licenseId: string) => Promise<Result<LicenseInfo, string>>
}

export interface LicenseContributorRow {
  readonly pluginId: string
  readonly pluginName: string
  readonly description: string
}

function formatContributors(license: LicenseInfo): readonly LicenseContributorRow[] {
  const title = getLicenseTitle(license)
  return license.contributors.map((one) => ({
    pluginId: one.pluginId,
    pluginName: license.contributors.length === 1 && one.pluginName === title ? '' : one.pluginName,
    description: one.description,
  }))
}

export function useLicenseCard(input: LicenseCardInput) {
  const draftKey = ref('')
  const isBusy = ref(false)
  const problem = ref('')

  const setDraftKey = (value: string): void => {
    draftKey.value = value
    if (problem.value !== '') problem.value = ''
  }

  const submitActivate = (): void => {
    const license = input.getLicense()
    const trimmed = draftKey.value.trim()
    if (trimmed === '' || isBusy.value || license.hasKey) return
    isBusy.value = true
    problem.value = ''
    runInBackground(
      input
        .activate(license.id, trimmed)
        .then((answer) => {
          if (!answer.ok) {
            problem.value = answer.error
            return
          }
          draftKey.value = ''
        })
        .finally(() => {
          isBusy.value = false
        }),
      (caughtError) => {
        problem.value = formatError(caughtError)
      },
    )
  }

  const submitDeactivate = (): void => {
    const license = input.getLicense()
    if (isBusy.value || !license.hasKey) return
    isBusy.value = true
    problem.value = ''
    runInBackground(
      input
        .deactivate(license.id)
        .then((answer) => {
          if (!answer.ok) {
            problem.value = answer.error
          }
        })
        .finally(() => {
          isBusy.value = false
        }),
      (caughtError) => {
        problem.value = formatError(caughtError)
      },
    )
  }

  return {
    state: {
      draftKey,
      isBusy,
      problem,
    },
    view: {
      displayTitle: computed(() => getLicenseTitle(input.getLicense())),
      statusText: computed(() => getStatusText(input.getLicense())),
      contributors: computed(() => formatContributors(input.getLicense())),
      hasDraftKey: computed(() => draftKey.value.trim() !== ''),
      buyOptions: computed(() => getBuyOptions(input.getLicense())),
    },
    actions: {
      setDraftKey,
      submitActivate,
      submitDeactivate,
    },
  }
}
