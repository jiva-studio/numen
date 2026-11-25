/** The dialog "Get a vault from the cloud" chooses the first vault of each list as soon as the lists arrive. */
import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { RemoteVault, Vault } from '../api/vaults'
import CloudVaultDialog from './CloudVaultDialog.vue'

const cloud: RemoteVault[] = [
  { remoteId: 'aa11', name: 'Work', isConnected: true, sizeBytes: 1 },
  { remoteId: 'bb22', name: 'Recipes', isConnected: false, sizeBytes: 2 },
]
const device = (id: string): Vault => ({
  id,
  name: id,
  path: `/home/person/${id}`,
  state: 'notConnected',
  lastSyncedAt: '',
  conflictCount: 0,
  remoteId: '',
})

const getButton = (): HTMLButtonElement => {
  const found = [...document.body.querySelectorAll('button')].find((one) =>
    one.textContent.includes('Get vault'),
  )
  if (found === undefined) throw new Error('no button')
  return found
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('CloudVaultDialog', () => {
  it('lets the person get a vault once the lists arrive after the dialog opened', async () => {
    const wrapper = mount(CloudVaultDialog, {
      attachTo: document.body,
      props: { cloudVaults: [], freeVaults: [], question: '' },
    })
    expect(getButton().disabled).toBe(true)

    await wrapper.setProps({ cloudVaults: cloud, freeVaults: [device('Plain')] })
    getButton().click()
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('choose')).toStrictEqual([['Plain', 'aa11']])
  })

  it('keeps the vault the person chose when the lists are read again', async () => {
    const wrapper = mount(CloudVaultDialog, {
      attachTo: document.body,
      props: { cloudVaults: cloud, freeVaults: [device('Plain')], question: '' },
    })
    document.body.querySelector<HTMLButtonElement>('button[data-slot="select"]')!.click()
    await wrapper.vm.$nextTick()
    const option = [...document.body.querySelectorAll<HTMLElement>('[role="option"]')].find((one) =>
      one.textContent.includes('Recipes'),
    )!
    option.click()
    await wrapper.vm.$nextTick()
    await wrapper.setProps({ cloudVaults: [...cloud] })
    getButton().click()
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('choose')).toStrictEqual([['Plain', 'bb22']])
  })
})
