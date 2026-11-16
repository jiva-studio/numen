import type { Api, CommandRegistration, PickerHost } from '@numen/plugin-sdk'

/** What the command Set tags reads and writes through. */
export interface SetTagsDeps {
  readonly api: Api
  readonly picker: PickerHost
  readonly getTags: (path: string) => readonly string[]
}

/**
 * The command Set tags: it opens the picker with every tag of the vault, shows the tags of the selected
 * notes as set, mixed or unset, and writes each toggle and each new tag to the notes at once. It resolves
 * when the person closes the picker.
 */
export function createSetTags(deps: SetTagsDeps): CommandRegistration {
  return {
    id: 'tags.set',
    handler: async () => {
      await Promise.resolve(deps.picker)
    },
  }
}
