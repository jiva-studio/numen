/**
 * Settings configuration, theme dressing, and palette lists for the window.
 */
import { computed, watch } from 'vue'
import { themes } from '@/entities/settings'
import {
  APPEARANCE,
  APPEARANCE_COMMANDS,
  HANGING,
  INTERFACE_SCALE,
  MODE,
  OFF,
  ON,
  PARTS,
  SYNCING,
  syncSetting,
  TEXT_SCALE,
  useHangingSetting,
  windowAppearance,
} from '@/features/settings-commands'
import { reviewSetting } from '@/entities/settings'
import { settingsStore } from '@/entities/settings'
import { useSettingsTab } from '@/pages/settings'
import { createTextEditorTabKind } from '@/pages/text-editor'
import type { PaletteLists } from '@/features/command-palette'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'
import { WORDS } from '@/shared/words'
import type { useWindowTabs } from '@/entities/tab'

type Words = typeof WORDS

export interface SettingsDeps {
  core: SettingsPort
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
    choose: (item) => void dressed.chooseItem(item),
    syncing: computed({
      get: () => oneName.kept.value,
      set: (on) => void oneName.choose(on ? ON : OFF),
    }),
    isHanging: computed({
      get: () => hungParts.isHanging.value,
      set: (on) => void hungParts.choose(on ? ON : OFF),
    }),
    parts: hungParts.parts,
    partsBounds: hungParts.ends,
    chooseParts: (count) => void hungParts.chooseCount(`${count}`),
    dayStarts: dayBegins.starts,
    latestDayStarts: dayBegins.latest,
    chooseDayStarts: (hour) => void dayBegins.choose(hour),
    setting: (at) => rest.at(at),
    models: (at) => rest.getModelsAt(at),
    write: (written) => void rest.writeSettings(written),
    file: rest.path,
    openFile: () => file.openSettingsFile(),
  })

  watch(dressed.sized, () => onSizeChanged())

  const kept: PaletteLists = {
    getStepGroups: (command, typed) => {
      if (command === APPEARANCE) return dressed.getThemeGroups()
      if (command === MODE) return dressed.modes()
      if (command === INTERFACE_SCALE || command === TEXT_SCALE)
        return dressed.sizes(command, typed)
      if (command === SYNCING) return oneName.getSyncingGroups()
      if (command === HANGING) return hungParts.getHangingGroups()
      if (command === PARTS) return hungParts.getPartsGroups()
      return []
    },
    previewItem: (command, item) => {
      if (APPEARANCE_COMMANDS.includes(command)) dressed.previewItem(item)
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
