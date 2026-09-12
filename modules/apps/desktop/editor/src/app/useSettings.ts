/**
 * Settings configuration, theme dressing, and palette lists for the window.
 */
import { computed, watch } from 'vue'
import { themes } from '@/entities/settings/theme'
import {
  APPEARANCE,
  DRESSING,
  INTERFACE_SCALE,
  MODE,
  TEXT_SCALE,
  windowAppearance,
} from '@/features/settings-commands/appearance'
import { reviewSetting } from '@/entities/settings/review'
import { OFF, ON, SYNCING, syncSetting } from '@/features/settings-commands/sync'
import { HANGING, PARTS, useHangingSetting } from '@/features/settings-commands/hanging'
import { settingsStore } from '@/entities/settings/store'
import { useSettingsTab } from '@/widgets/settings/kind'
import { createTextEditorTabKind } from '@/widgets/text-editor/kind'
import type { PaletteLists } from '@/features/command-palette/lists'
import type { Core } from '@/app/ports/core'
import type { MessageLog } from '@/shared/notices/messages'
import { WORDS } from '@/shared/words'
import type { useWindowTabs } from '@/entities/tab/windowTabs'

type Words = typeof WORDS

export interface SettingsDeps {
  core: Core
  words: Words
  log: MessageLog
  held: ReturnType<typeof useWindowTabs>
  onSizeChanged: () => void
}

export function useSettings({ core, words, log, held, onSizeChanged }: SettingsDeps) {
  const dayBegins = reviewSetting(core, words, log.under('reviewed'))
  const hungParts = useHangingSetting(core, words, log.under('hanging'))
  const dressed = windowAppearance(themes, words, log.under('worn'))
  const oneName = syncSetting(core, words, log.under('named'))
  const rest = settingsStore(core, words, log.under('configured'))

  const file = createTextEditorTabKind(held.handle, core, () => void rest.start())

  const configured = useSettingsTab(held.handle, {
    themes: dressed.list,
    applied: dressed.applied,
    mode: dressed.mode,
    pinned: dressed.pinned,
    sizes: dressed.sized,
    bounds: dressed.bounds,
    chooses: (item) => void dressed.chooses(item),
    syncing: computed({
      get: () => oneName.kept.value,
      set: (on) => void oneName.chooses(on ? ON : OFF),
    }),
    hangs: computed({
      get: () => hungParts.hangs.value,
      set: (on) => void hungParts.chooses(on ? ON : OFF),
    }),
    parts: hungParts.parts,
    partsBounds: hungParts.ends,
    choosesParts: (count) => void hungParts.choosesCount(`${count}`),
    dayStarts: dayBegins.starts,
    latestDayStarts: dayBegins.latest,
    choosesDayStarts: (hour) => void dayBegins.chooses(hour),
    setting: (at) => rest.at(at),
    models: (at) => rest.offers(at),
    writes: (written) => void rest.chooses(written),
    file: rest.path,
    opensFile: () => file.shows(),
  })

  watch(dressed.sized, () => onSizeChanged())

  const kept: PaletteLists = {
    offers: (command, typed) => {
      if (command === APPEARANCE) return dressed.offers()
      if (command === MODE) return dressed.modes()
      if (command === INTERFACE_SCALE || command === TEXT_SCALE)
        return dressed.sizes(command, typed)
      if (command === SYNCING) return oneName.offers()
      if (command === HANGING) return hungParts.offers()
      if (command === PARTS) return hungParts.counts()
      return []
    },
    shows: (command, item) => {
      if (DRESSING.includes(command)) dressed.shows(item)
    },
  }

  const kinds = [configured.kind, file.kind]

  const start = async () => {
    await dayBegins.start()
    void dressed.start()
    void oneName.start()
    void hungParts.start()
    void rest.start()
  }

  const close = () => {
    dressed.close()
  }

  return {
    dayBegins,
    hungParts,
    dressed,
    oneName,
    rest,
    file,
    configured,
    kept,
    kinds,
    start,
    close,
  }
}
