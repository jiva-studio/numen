/** Mounting the menu in a test: at the end of the document, and gone after each test. */
import { mount } from '@vue/test-utils'
import { afterEach } from 'vitest'
import { nextTick } from 'vue'
import Menu from '../Menu.vue'
import type { MenuItem } from '../items/item'

type MenuProps = InstanceType<typeof Menu>['$props']

/**
 * The mount of a menu over `items` at (40, 40), opened, and unmounted after each test. A test
 * that unmounts the menu itself calls `forgetMenu`.
 */
export const createMenuMount = (items: MenuItem[]) => {
  let mounted: { unmount: () => void } | null = null

  afterEach(() => {
    mounted?.unmount()
    mounted = null
  })

  const mountMenu = (props: Partial<MenuProps> = {}, slots: Record<string, string> = {}) => {
    const menu = mount(Menu, {
      props: { items, at: { x: 40, y: 40 }, isOpen: true, ...props },
      slots,
      attachTo: document.body,
    })
    mounted = menu
    return menu
  }

  const forgetMenu = () => {
    mounted = null
  }

  return { mountMenu, forgetMenu }
}

/** Twice: the menu measures and takes the keyboard a tick after it is opened. */
export const settle = async () => {
  await nextTick()
  await nextTick()
}
