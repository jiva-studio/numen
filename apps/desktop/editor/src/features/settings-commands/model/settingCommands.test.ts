/**
 * The palette commands of the settings, made from the declarations of the owner app and of the
 * plugins: which settings become a command, the choice step each one opens, what choosing a row
 * writes, and how a setting that is previewed is applied while focused and restored on leaving.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { createSettingPalette } from './settingCommands'
import { getWindowConfig } from './windowConfig'
import { createWindowSettingPalette } from './windowPalette'
import { getSettingWhen, type SettingOwner } from '../lib/settingsConfig'
import { getNamedIcon } from '@/shared/ui/icons'
import type { PluginSettingInfo } from '@/entities/settings'

const declare = (overrides: Partial<PluginSettingInfo>): PluginSettingInfo => ({
  key: 'flag',
  type: 'boolean',
  title: 'Flag',
  description: '',
  default: false,
  ...overrides,
})

const THEME = declare({
  key: 'appearance.theme',
  type: 'enum',
  title: 'Theme',
  default: 'preset:numen',
  palette: true,
  icon: 'palette',
  preview: true,
  optionsFrom: 'themes',
  activeOptions: [
    { value: 'preset:numen', title: 'numen', detail: '' },
    { value: 'preset:dracula', title: 'dracula', detail: '' },
    { value: 'mine:sea', title: '', detail: '' },
  ],
  value: 'preset:numen',
})

const MODE = declare({
  key: 'appearance.mode',
  type: 'enum',
  title: 'Light or dark',
  default: 'system',
  palette: true,
  preview: true,
  options: [
    { value: 'system', title: 'System', detail: '' },
    { value: 'light', title: 'Light', detail: '' },
    { value: 'dark', title: 'Dark', detail: '' },
  ],
  value: 'dark',
})

const INTERFACE_SCALE = declare({
  key: 'appearance.interfaceScale',
  type: 'range',
  title: 'Interface size',
  default: 1,
  palette: true,
  preview: true,
  min: 0.8,
  max: 2,
  step: 0.1,
  value: 1,
})

const SYNC = declare({
  key: 'naming.syncTitleAndFilename',
  type: 'boolean',
  title: 'Sync title and filename',
  default: true,
  palette: true,
  value: true,
})

const PROFILE = declare({
  key: 'performance.profile',
  type: 'enum',
  title: 'Performance profile',
  default: 'balanced',
  palette: true,
  options: [
    { value: 'eco', title: 'Eco', detail: 'Minimal power.' },
    { value: 'balanced', title: 'Balanced', detail: 'Standard workload.' },
    { value: 'maximum', title: 'Maximum', detail: 'All cores.' },
  ],
  value: 'balanced',
})

const UNFLAGGED = declare({ key: 'indexing.depth', type: 'range', min: 1, max: 3, step: 1 })

const APP: SettingOwner = {
  id: 'app',
  name: '',
  settings: [THEME, MODE, INTERFACE_SCALE, SYNC, PROFILE, UNFLAGGED],
}

const PLUGIN: SettingOwner = {
  id: 'md.example.scanner',
  name: 'Scanner',
  settings: [
    declare({
      key: 'autoScan',
      title: 'Auto scan',
      palette: true,
      icon: 'scan-text',
      value: false,
    }),
    declare({
      key: 'engine',
      type: 'enum',
      title: 'Engine',
      default: 'plain',
      palette: true,
      options: [
        { value: 'plain', title: 'Plain', detail: '' },
        { value: 'fast', title: 'Fast', detail: '' },
      ],
    }),
    declare({ key: 'label', type: 'string', title: 'Label', default: '', palette: true }),
    declare({ key: 'hidden', title: 'Hidden' }),
  ],
}

const createPalette = (owners: readonly SettingOwner[] = [APP, PLUGIN]) => {
  const applied: string[] = []
  const write = vi.fn(async () => undefined)
  const palette = createSettingPalette({
    getOwners: () => owners,
    words: { current: 'Current', on: 'On', off: 'Off' },
    preview: {
      apply: (path, value) => applied.push(`apply ${path.join('/')} ${JSON.stringify(value)}`),
      restore: (path) => applied.push(`restore ${path.join('/')}`),
    },
    write,
  })
  return { palette, applied, write }
}

const rowsOf = (command: string, owners?: readonly SettingOwner[]) =>
  createPalette(owners).palette.lists.getStepGroups(command, '')[0]?.items ?? []

describe('the commands made from the declarations', () => {
  it('offers one command for each setting flagged for the palette, the owner app first', () => {
    const { palette } = createPalette()

    expect(palette.getCommands().map((one) => one.id)).toStrictEqual([
      'setting:app:appearance.theme',
      'setting:app:appearance.mode',
      'setting:app:appearance.interfaceScale',
      'setting:app:naming.syncTitleAndFilename',
      'setting:app:performance.profile',
      'setting:md.example.scanner:autoScan',
      'setting:md.example.scanner:engine',
    ])
  })

  it('titles a setting of the app by its title and a setting of a plugin by the plugin and its title', () => {
    const texts = createPalette()
      .palette.getCommands()
      .map((one) => one.text)

    expect(texts).toStrictEqual([
      'Theme',
      'Light or dark',
      'Interface size',
      'Sync title and filename',
      'Performance profile',
      'Scanner: Auto scan',
      'Scanner: Engine',
    ])
  })

  it('files every command under the settings and asks for a choice', () => {
    for (const one of createPalette().palette.getCommands()) {
      expect([one.group, one.input]).toStrictEqual(['settings', 'items'])
    }
  })

  it('draws the icon the declaration names and none where it names none', () => {
    const commands = createPalette().palette.getCommands()
    const byId = Object.fromEntries(commands.map((one) => [one.id, one]))

    expect(byId['setting:app:appearance.theme']?.icon).toStrictEqual(getNamedIcon('palette'))
    expect(byId['setting:md.example.scanner:autoScan']?.icon).toStrictEqual(
      getNamedIcon('scan-text'),
    )
    expect(byId['setting:app:appearance.mode']).not.toHaveProperty('icon')
  })

  it('offers no command for a setting that is not flagged or that is not a choice', () => {
    const ids = createPalette()
      .palette.getCommands()
      .map((one) => one.id)

    expect(ids).not.toContain('setting:app:indexing.depth')
    expect(ids).not.toContain('setting:md.example.scanner:hidden')
    expect(ids).not.toContain('setting:md.example.scanner:label')
  })

  it('offers a command only while the condition of its setting holds under the values in force', () => {
    const conditional: SettingOwner = {
      ...APP,
      settings: [
        { ...MODE, when: { setting: 'performance.profile', is: ['eco'] } },
        { ...SYNC, when: { setting: 'performance.profile', is: ['balanced', 'maximum'] } },
        PROFILE,
      ],
    }
    const commands = createPalette([conditional]).palette.getCommands()
    const target = { tab: '', path: '', file: '', vault: { id: 'v' }, isReady: true } as never
    const available = Object.fromEntries(
      commands.map((one) => [one.id, one.isAvailable(target, {} as never)]),
    )

    expect(available).toStrictEqual({
      'setting:app:appearance.mode': false,
      'setting:app:naming.syncTitleAndFilename': true,
      'setting:app:performance.profile': true,
    })
    expect(commands[0]?.when).toBe("string(config.app.performance.profile) in ['eco']")
  })

  it('offers no setting the core hides, as one that names another platform', () => {
    const when = { setting: 'app.platform', is: ['linux'] }
    const owners: readonly SettingOwner[] = [
      {
        id: 'md.example.scanner',
        name: 'Scanner',
        settings: [
          declare({ key: 'native', title: 'Native', palette: true, when, isShown: false }),
          declare({ key: 'any', title: 'Any', palette: true, isShown: true }),
        ],
      },
    ]
    const palette = createSettingPalette({
      getOwners: () => owners,
      words: { current: 'Current', on: 'On', off: 'Off' },
      preview: { apply: vi.fn(), restore: vi.fn() },
      write: vi.fn(async () => undefined),
    })

    expect(palette.getCommands().map((one) => one.text)).toStrictEqual(['Scanner: Any'])
    expect(getSettingWhen(owners[0]!, owners[0]!.settings[0]!)).toBe('')
  })

  it('reads the owners again for each request of the commands', () => {
    let owners: readonly SettingOwner[] = [APP]
    const palette = createSettingPalette({
      getOwners: () => owners,
      words: { current: 'Current', on: 'On', off: 'Off' },
      preview: { apply: vi.fn(), restore: vi.fn() },
      write: vi.fn(async () => undefined),
    })
    const before = palette.getCommands().length

    owners = [APP, PLUGIN]

    expect(palette.getCommands().length).toBe(before + 2)
  })
})

describe('the choice step of a setting', () => {
  it('offers a boolean as On and Off and marks the one in force', () => {
    expect(rowsOf('setting:app:naming.syncTitleAndFilename')).toStrictEqual([
      { id: 'on', title: 'On', detail: 'Current', isCurrent: true },
      { id: 'off', title: 'Off' },
    ])
    expect(rowsOf('setting:md.example.scanner:autoScan')).toStrictEqual([
      { id: 'on', title: 'On' },
      { id: 'off', title: 'Off', detail: 'Current', isCurrent: true },
    ])
  })

  it('offers the default of a boolean the core resolved no value for', () => {
    const owner: SettingOwner = {
      id: 'app',
      name: '',
      settings: [{ ...SYNC, value: undefined, default: false }],
    }

    expect(
      rowsOf('setting:app:naming.syncTitleAndFilename', [owner]).map((one) => one.id),
    ).toStrictEqual(['on', 'off'])
    expect(
      rowsOf('setting:app:naming.syncTitleAndFilename', [owner]).find((one) => one.isCurrent)?.id,
    ).toBe('off')
  })

  it('offers the options in force of an enum, each titled and the one in force marked', () => {
    expect(rowsOf('setting:app:appearance.theme')).toStrictEqual([
      { id: 'preset:numen', title: 'numen', detail: 'Current', isCurrent: true },
      { id: 'preset:dracula', title: 'dracula' },
      { id: 'mine:sea', title: 'mine:sea' },
    ])
  })

  it('offers the declared options of an enum that has none resolved, and keeps what each says', () => {
    expect(rowsOf('setting:app:performance.profile')).toStrictEqual([
      { id: 'eco', title: 'Eco', detail: 'Minimal power.' },
      { id: 'balanced', title: 'Balanced', detail: 'Standard workload.', isCurrent: true },
      { id: 'maximum', title: 'Maximum', detail: 'All cores.' },
    ])
  })

  it('opens an enum on its default where no value is resolved', () => {
    const owner: SettingOwner = {
      id: 'md.example.scanner',
      name: 'Scanner',
      settings: [PLUGIN.settings[1] as PluginSettingInfo],
    }

    expect(
      rowsOf('setting:md.example.scanner:engine', [owner]).find((one) => one.isCurrent)?.id,
    ).toBe('plain')
  })

  it('offers a range as the values from its minimum to its maximum by its step', () => {
    const rows = rowsOf('setting:app:appearance.interfaceScale')

    expect(rows.map((one) => one.id)).toStrictEqual([
      '0.8',
      '0.9',
      '1',
      '1.1',
      '1.2',
      '1.3',
      '1.4',
      '1.5',
      '1.6',
      '1.7',
      '1.8',
      '1.9',
      '2',
    ])
    expect(rows.find((one) => one.id === '1')).toStrictEqual({
      id: '1',
      title: '1',
      detail: 'Current',
      isCurrent: true,
    })
    expect(rows.find((one) => one.id === '0.8')).toStrictEqual({ id: '0.8', title: '0.8' })
  })

  it('offers the maximum of a range whose step does not divide it, and no rows when range bounds or step are invalid', () => {
    const rangeOwner: SettingOwner = {
      id: 'app',
      name: '',
      settings: [
        declare({
          key: 'oddRange',
          type: 'range',
          palette: true,
          min: 0,
          max: 2.5,
          step: 1,
          value: 1,
        }),
        declare({ key: 'noMin', type: 'range', palette: true, max: 2, step: 1 }),
        declare({ key: 'noMax', type: 'range', palette: true, min: 0, step: 1 }),
        declare({ key: 'noStep', type: 'range', palette: true, min: 0, max: 2 }),
        declare({ key: 'zeroStep', type: 'range', palette: true, min: 0, max: 2, step: 0 }),
        declare({ key: 'negStep', type: 'range', palette: true, min: 0, max: 2, step: -1 }),
      ],
    }

    expect(rowsOf('setting:app:oddRange', [rangeOwner]).map((one) => one.id)).toStrictEqual([
      '0',
      '1',
      '2',
      '2.5',
    ])
    expect(rowsOf('setting:app:noMin', [rangeOwner])).toStrictEqual([])
    expect(rowsOf('setting:app:noMax', [rangeOwner])).toStrictEqual([])
    expect(rowsOf('setting:app:noStep', [rangeOwner])).toStrictEqual([])
    expect(rowsOf('setting:app:zeroStep', [rangeOwner])).toStrictEqual([])
    expect(rowsOf('setting:app:negStep', [rangeOwner])).toStrictEqual([])
  })

  it('names the group after the setting', () => {
    const groups = createPalette().palette.lists.getStepGroups(
      'setting:app:performance.profile',
      '',
    )

    expect(groups.map((one) => one.title)).toStrictEqual(['Performance profile'])
  })

  it('offers nothing for a command of no setting', () => {
    const { palette } = createPalette()

    for (const command of [
      'app:note.create',
      'setting:unknown:autoScan',
      'setting:md.example.scanner:unknown',
      'setting:md.example.scanner:label',
      'setting:md.example.scanner:hidden',
    ]) {
      expect(palette.lists.getStepGroups(command, ''), command).toStrictEqual([])
    }
  })
})

describe('choosing a row', () => {
  it('writes a boolean under the path of its owner as true or false', async () => {
    const { palette, write } = createPalette()

    await palette.choose('setting:md.example.scanner:autoScan', 'on')
    await palette.choose('setting:app:naming.syncTitleAndFilename', 'off')

    expect(write.mock.calls).toStrictEqual([
      [[{ at: ['md.example.scanner', 'autoScan'], value: 'true' }]],
      [[{ at: ['app', 'naming', 'syncTitleAndFilename'], value: 'false' }]],
    ])
  })

  it('writes an enum as the text of the option, under the nested path of the app', async () => {
    const { palette, write } = createPalette()

    await palette.choose('setting:app:appearance.theme', 'preset:dracula')
    await palette.choose('setting:md.example.scanner:engine', 'fast')

    expect(write.mock.calls).toStrictEqual([
      [[{ at: ['app', 'appearance', 'theme'], value: '"preset:dracula"' }]],
      [[{ at: ['md.example.scanner', 'engine'], value: '"fast"' }]],
    ])
  })

  it('writes a range as the number of its row', async () => {
    const { palette, write } = createPalette()

    await palette.choose('setting:app:appearance.interfaceScale', '1.3')

    expect(write.mock.calls).toStrictEqual([
      [[{ at: ['app', 'appearance', 'interfaceScale'], value: '1.3' }]],
    ])
  })

  it('writes nothing for a row that is not one of the setting', async () => {
    const { palette, write } = createPalette()

    await palette.choose('setting:app:appearance.theme', 'preset:missing')
    await palette.choose('setting:app:appearance.interfaceScale', '5')
    await palette.choose('setting:app:naming.syncTitleAndFilename', 'maybe')
    await palette.choose('setting:md.example.scanner:label', 'text')
    await palette.choose('setting:unknown:key', 'on')

    expect(write).not.toHaveBeenCalled()
  })

  it('passes on the reason a write is rejected', async () => {
    const { palette, write } = createPalette()
    write.mockRejectedValueOnce(new Error('value out of range'))

    await expect(palette.choose('setting:app:performance.profile', 'eco')).rejects.toThrow(
      'value out of range',
    )
  })
})

describe('a setting that is previewed while focused', () => {
  it('is applied under its path for each row the keyboard moves to', () => {
    const { palette, applied } = createPalette()

    palette.lists.applyItemPreview('setting:app:appearance.theme', 'preset:dracula')
    palette.lists.applyItemPreview('setting:app:appearance.mode', 'light')
    palette.lists.applyItemPreview('setting:app:appearance.interfaceScale', '1.4')

    expect(applied).toStrictEqual([
      'apply app/appearance/theme "preset:dracula"',
      'apply app/appearance/mode "light"',
      'apply app/appearance/interfaceScale 1.4',
    ])
  })

  it('is restored once when the keyboard exits the step without a choice', () => {
    const { palette, applied } = createPalette()

    palette.lists.applyItemPreview('setting:app:appearance.theme', 'preset:dracula')
    palette.lists.applyItemPreview('setting:app:appearance.theme', 'mine:sea')
    palette.lists.applyItemPreview('setting:app:appearance.theme', '')
    palette.lists.applyItemPreview('setting:app:appearance.theme', '')

    expect(applied).toStrictEqual([
      'apply app/appearance/theme "preset:dracula"',
      'apply app/appearance/theme "mine:sea"',
      'restore app/appearance/theme',
    ])
  })

  it('restores nothing when the keyboard never stood on a row', () => {
    const { palette, applied } = createPalette()

    palette.lists.applyItemPreview('setting:app:appearance.theme', '')

    expect(applied).toStrictEqual([])
  })

  it('is not restored after a row was chosen, the write being the value in force', async () => {
    const { palette, applied } = createPalette()

    palette.lists.applyItemPreview('setting:app:appearance.theme', 'preset:dracula')
    await palette.choose('setting:app:appearance.theme', 'preset:dracula')
    palette.lists.applyItemPreview('setting:app:appearance.theme', '')

    expect(applied).toStrictEqual(['apply app/appearance/theme "preset:dracula"'])
  })

  it('is applied only for a row the setting offers', () => {
    const { palette, applied } = createPalette()

    palette.lists.applyItemPreview('setting:app:appearance.theme', 'preset:missing')
    palette.lists.applyItemPreview('setting:app:appearance.interfaceScale', '5')

    expect(applied).toStrictEqual([])
  })

  it('is not applied for a setting that does not carry the flag', () => {
    const { palette, applied } = createPalette()

    palette.lists.applyItemPreview('setting:app:performance.profile', 'eco')
    palette.lists.applyItemPreview('setting:app:naming.syncTitleAndFilename', 'off')
    palette.lists.applyItemPreview('setting:md.example.scanner:engine', 'fast')
    palette.lists.applyItemPreview('setting:app:performance.profile', '')
    palette.lists.applyItemPreview('setting:unknown:missing', 'on')

    expect(applied).toStrictEqual([])
  })

  it('is applied for a plugin that declares the flag', () => {
    const flagged: SettingOwner = {
      ...PLUGIN,
      settings: [{ ...(PLUGIN.settings[1] as PluginSettingInfo), preview: true }],
    }
    const { palette, applied } = createPalette([flagged])

    palette.lists.applyItemPreview('setting:md.example.scanner:engine', 'fast')
    palette.lists.applyItemPreview('setting:md.example.scanner:engine', '')

    expect(applied).toStrictEqual([
      'apply md.example.scanner/engine "fast"',
      'restore md.example.scanner/engine',
    ])
  })
})

describe('windowConfig and windowPalette', () => {
  const TEXT_SCALE_SETTING = declare({
    key: 'appearance.textScale',
    type: 'range',
    title: 'Text size',
    default: 1,
    palette: true,
    preview: true,
    min: 0.8,
    max: 1.4,
    step: 0.2,
    value: 1,
  })
  const CUSTOM_MODE = declare({
    ...MODE,
    options: [...(MODE.options ?? []), { value: 'invalid', title: 'Invalid', detail: '' }],
  })
  const NON_STRING_THEME = declare({
    key: 'appearance.theme',
    type: 'boolean',
    title: 'Theme bool',
    default: true,
    palette: true,
    preview: true,
    value: true,
  })
  const NON_NUMBER_SCALE = declare({
    key: 'appearance.interfaceScale',
    type: 'enum',
    title: 'Scale enum',
    default: 'normal',
    palette: true,
    preview: true,
    options: [{ value: 'normal', title: 'Normal', detail: '' }],
    value: 'normal',
  })
  const OTHER_APPEARANCE_RANGE = declare({
    key: 'appearance.otherScale',
    type: 'range',
    title: 'Other scale',
    default: 1,
    palette: true,
    preview: true,
    min: 1,
    max: 2,
    step: 1,
    value: 1,
  })
  const DEEP_APPEARANCE = declare({
    key: 'appearance.sub.theme',
    type: 'enum',
    title: 'Sub theme',
    default: 'a',
    palette: true,
    preview: true,
    options: [{ value: 'a', title: 'A', detail: '' }],
    value: 'a',
  })
  const OTHER_APP_PREVIEW = declare({
    key: 'editor.tabSize',
    type: 'range',
    title: 'Tab size',
    default: 2,
    palette: true,
    preview: true,
    min: 2,
    max: 4,
    step: 2,
    value: 2,
  })

  it('builds window config from app and plugin settings', () => {
    const rest = {
      state: {
        app: ref([THEME]),
        plugins: ref([{ id: 'md.example.scanner', name: 'Scanner', settings: PLUGIN.settings }]),
      },
    }

    expect(getWindowConfig(rest as never)).toStrictEqual({
      app: { appearance: { theme: 'preset:numen' } },
      md: { example: { scanner: { autoScan: false } } },
    })
  })

  it('previews appearance settings and reloads appearance only when an appearance setting is written', async () => {
    const showThemePreview = vi.fn()
    const showModePreview = vi.fn()
    const showSizePreview = vi.fn()
    const restore = vi.fn()
    const reload = vi.fn(async () => undefined)
    const writeSettings = vi.fn(async () => undefined)
    const appSettings = ref<readonly PluginSettingInfo[]>([
      THEME,
      CUSTOM_MODE,
      INTERFACE_SCALE,
      TEXT_SCALE_SETTING,
      OTHER_APPEARANCE_RANGE,
      DEEP_APPEARANCE,
      OTHER_APP_PREVIEW,
      PROFILE,
    ])
    const rest = {
      state: {
        app: appSettings,
        plugins: ref([
          {
            id: PLUGIN.id,
            name: PLUGIN.name,
            settings: [{ ...(PLUGIN.settings[0] as PluginSettingInfo), preview: true }],
          },
        ]),
      },
      read: { getCommittedSetting: () => true },
      write: { writeSettings },
    }
    const palette = createWindowSettingPalette({
      rest: rest as never,
      appearance: {
        preview: {
          keyboard: { showThemePreview, showModePreview, showSizePreview },
          restore,
        },
        reload,
      } as never,
      words: { current: 'Current', on: 'On', off: 'Off' },
    })

    expect(palette.getCommands().length).toBeGreaterThan(0)

    palette.lists.applyItemPreview('setting:app:appearance.theme', 'preset:dracula')
    expect(showThemePreview).toHaveBeenCalledExactlyOnceWith('preset:dracula')

    palette.lists.applyItemPreview('setting:app:appearance.mode', 'light')
    palette.lists.applyItemPreview('setting:app:appearance.mode', 'invalid')
    expect(showModePreview).toHaveBeenCalledExactlyOnceWith('light')

    palette.lists.applyItemPreview('setting:app:appearance.interfaceScale', '1.2')
    palette.lists.applyItemPreview('setting:app:appearance.textScale', '1.2')
    palette.lists.applyItemPreview('setting:app:appearance.otherScale', '2')
    palette.lists.applyItemPreview('setting:app:appearance.sub.theme', 'a')
    palette.lists.applyItemPreview('setting:app:editor.tabSize', '4')
    palette.lists.applyItemPreview('setting:md.example.scanner:autoScan', 'on')
    expect(showSizePreview).toHaveBeenCalledTimes(2)
    expect(showSizePreview).toHaveBeenNthCalledWith(1, 'interfaceScale', 1.2)
    expect(showSizePreview).toHaveBeenNthCalledWith(2, 'textScale', 1.2)

    appSettings.value = [NON_STRING_THEME, NON_NUMBER_SCALE]
    palette.lists.applyItemPreview('setting:app:appearance.theme', 'on')
    palette.lists.applyItemPreview('setting:app:appearance.interfaceScale', 'normal')
    expect(showThemePreview).toHaveBeenCalledTimes(1)
    expect(showSizePreview).toHaveBeenCalledTimes(2)

    appSettings.value = [THEME, PROFILE]
    await palette.choose('setting:app:performance.profile', 'eco')
    await palette.choose('setting:md.example.scanner:autoScan', 'on')
    expect(reload).not.toHaveBeenCalled()

    await palette.choose('setting:app:appearance.theme', 'preset:dracula')
    expect(reload).toHaveBeenCalledTimes(1)
  })

  it('offers plugin setting commands only while the plugin is enabled', () => {
    let isScannerEnabled = false
    const rest = {
      state: {
        app: ref<readonly PluginSettingInfo[]>([THEME]),
        plugins: ref([
          {
            id: PLUGIN.id,
            name: PLUGIN.name,
            settings: PLUGIN.settings,
          },
        ]),
      },
      read: {
        getCommittedSetting: (path: readonly string[]) =>
          path[0] === PLUGIN.id && path[1] === 'enabled' ? isScannerEnabled : undefined,
      },
      write: { writeSettings: vi.fn(async () => undefined) },
    }
    const palette = createWindowSettingPalette({
      rest: rest as never,
      appearance: {
        preview: {
          keyboard: { showThemePreview: vi.fn(), showModePreview: vi.fn(), showSizePreview: vi.fn() },
          restore: vi.fn(),
        },
        reload: vi.fn(async () => undefined),
      } as never,
      words: { current: 'Current', on: 'On', off: 'Off' },
    })

    expect(palette.getCommands().map((one) => one.id)).toStrictEqual([
      'setting:app:appearance.theme',
    ])

    isScannerEnabled = true
    expect(palette.getCommands().map((one) => one.id)).toStrictEqual([
      'setting:app:appearance.theme',
      'setting:md.example.scanner:autoScan',
      'setting:md.example.scanner:engine',
    ])
  })
})
