/** The generated list of the slots against `libs/protocol/slots.json`. */
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { SLOT, isUserArrangedSlot, listUserArrangedSlots } from './slotNames'

interface DeclaredSlot {
  readonly name: string
  readonly arrangement?: string
}

const { slots } = JSON.parse(
  readFileSync(fileURLToPath(new URL('../../../protocol/slots.json', import.meta.url)), 'utf8'),
) as { slots: DeclaredSlot[] }

describe('the slots whose arrangement is user', () => {
  it('are the ones slots.json marks, in its order', () => {
    expect(listUserArrangedSlots()).toStrictEqual(
      slots.filter((one) => one.arrangement === 'user').map((one) => one.name),
    )
    expect(listUserArrangedSlots()).toStrictEqual([
      SLOT.titleBarCommands,
      SLOT.titleBarStatus,
      SLOT.statusBarLeft,
      SLOT.statusBarRight,
    ])
  })

  it('are told from the slots the manifests fix and from names that are no slot', () => {
    for (const one of slots) {
      expect(isUserArrangedSlot(one.name)).toBe(one.arrangement === 'user')
    }
    expect(isUserArrangedSlot('fileTree.context')).toBe(false)
    expect(isUserArrangedSlot('titleBar')).toBe(false)
    expect(isUserArrangedSlot('sideBar.top')).toBe(false)
  })
})
