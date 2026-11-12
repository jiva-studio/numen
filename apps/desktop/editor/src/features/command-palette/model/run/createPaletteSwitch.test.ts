import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { createPaletteSwitch } from './createPaletteSwitch'
import { createSearchRequests } from './createSearchRequests'

const setup = () => {
  const commands = {
    palette: {
      open: ref(false),
      setOpen: (isOpen: boolean) => (commands.palette.open.value = isOpen),
    },
  }
  const search = { open: ref(false), setOpen: (isOpen: boolean) => (search.open.value = isOpen) }
  return { commands, search, palettes: createPaletteSwitch(commands as never, search as never) }
}

describe('the two palettes of the window', () => {
  it('open the search over the commands, and close it when it is open', () => {
    const { commands, search, palettes } = setup()
    commands.palette.open.value = true

    palettes.toggleSearch()
    expect([commands.palette.open.value, search.open.value]).toStrictEqual([false, true])

    palettes.toggleSearch()
    expect([commands.palette.open.value, search.open.value]).toStrictEqual([false, false])
  })

  it('open the commands over the search, and close them when they are open', () => {
    const { commands, search, palettes } = setup()
    search.open.value = true

    palettes.toggleCommands()
    expect([commands.palette.open.value, search.open.value]).toStrictEqual([true, false])

    palettes.toggleCommands()
    expect([commands.palette.open.value, search.open.value]).toStrictEqual([false, false])
  })

  it('open the search on a request made before they existed', () => {
    const { search, palettes } = setup()
    const requests = createSearchRequests()
    requests.request()

    requests.connect(palettes)
    requests.request()

    expect(search.open.value).toBe(true)
  })
})
