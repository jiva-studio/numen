import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import {
  MOBILE_POLICY,
  findMobilePageViolations,
  findOptionsReaders,
  findPaletteBypassingCommands,
  findPluginNames,
  findShellSettingsSections,
  verifyHostNamesNoPlugin,
  verifyMobilePagePolicy,
  verifyOptionsAreAppliedByTheCore,
  verifyPaletteRunsCommands,
  verifySettingsPageHasNoShellSection,
} from './check.mjs'

const EDITOR = 'apps/desktop/editor/src'
const at = (found) =>
  found.findings.map((one) => `${one.file.replace(`${EDITOR}/`, '')}:${one.line} ${one.symbol}`)

describe('the editor names no plugin', () => {
  it('fails a plugin id and a reader tab kind, and passes a plain word', () => {
    const found = findPluginNames([
      {
        file: `${EDITOR}/a.ts`,
        text: "const id = 'md.numen.sample'\nconst kind = \"reader:reader-pdf\"\nconst clean = 'note'",
      },
    ])
    assert.deepEqual(at(found), [
      'a.ts:1 spells out md.numen.sample',
      'a.ts:2 spells out "reader:reader-pdf"',
    ])
  })

  it('passes the real editor', () => {
    verifyHostNamesNoPlugin()
  })
})

describe('only the settings api reads optionsBy', () => {
  it('fails a read elsewhere and passes the api', () => {
    const found = findOptionsReaders([
      { file: `${EDITOR}/entities/settings/api/settings.ts`, text: 'x.optionsBy' },
      { file: `${EDITOR}/pages/settings/ui/Row.vue`, text: '\nrow.optionsBy.map(f)' },
    ])
    assert.deepEqual(at(found), [
      'pages/settings/ui/Row.vue:2 reads optionsBy outside entities/settings/api',
    ])
  })

  it('passes the real editor', () => {
    verifyOptionsAreAppliedByTheCore()
  })
})

describe('a palette result opens only through its command', () => {
  const palette = 'features/command-palette'
  const files = [
    {
      file: `${EDITOR}/${palette}/ui/CommandPalette.vue`,
      text: 'runInvocation(x)\nif (item === a) {}',
    },
    {
      file: `${EDITOR}/${palette}/model/usePaletteInput.ts`,
      text: "runInvocation(a)\nrunInvocation(b)\nconst NOTE = 'note'",
    },
    {
      file: `${EDITOR}/features/other/open.ts`,
      text: 'openDestination(x)\nconst MENU_ACTION_PREFIX = 1',
    },
  ]

  it('fails each bypass', () => {
    assert.deepEqual(at(findPaletteBypassingCommands(files)), [
      'features/other/open.ts:1 uses openDestination, a path of a result that is no command',
      'features/other/open.ts:2 uses MENU_ACTION_PREFIX, a path of a result that is no command',
      `${palette}/model/usePaletteInput.ts:3 gives a document an action id that is no command`,
      `${palette}/ui/CommandPalette.vue:2 decides on a result itself (item ===)`,
      `${palette}/ui/CommandPalette.vue:1 runs an invocation in the view; only the field does`,
      `${palette}/model/usePaletteInput.ts:1 calls runInvocation 2 times, once is the choice it is handed`,
    ])
  })

  it('passes a view and a field that follow the rule', () => {
    const clean = [
      { file: `${EDITOR}/${palette}/ui/CommandPalette.vue`, text: 'onChoose(a)' },
      { file: `${EDITOR}/${palette}/model/usePaletteInput.ts`, text: 'runInvocation(a)' },
    ]
    assert.deepEqual(findPaletteBypassingCommands(clean).findings, [])
  })

  it('passes the real editor', () => {
    verifyPaletteRunsCommands()
  })
})

describe('the settings page has no section for the shell', () => {
  it('fails a section named for the window or for naming', () => {
    const found = findShellSettingsSections([
      `${EDITOR}/pages/settings/ui/WindowSettingsSection.vue`,
      `${EDITOR}/pages/settings/ui/NamingSettings.vue`,
      `${EDITOR}/pages/settings/ui/SettingsRow.vue`,
    ])
    assert.equal(found.findings.length, 2)
  })

  it('passes the real page', () => {
    verifySettingsPageHasNoShellSection()
  })
})

describe('the page of the phone', () => {
  const html = (policy) => `<meta http-equiv="Content-Security-Policy" content="${policy}">`
  const policy = Object.entries(MOBILE_POLICY)
    .map(([named, sources]) => `${named} ${sources.join(' ')}`)
    .join('; ')

  it('passes its exact policy and the core as the only host', () => {
    const found = findMobilePageViolations(html(policy), [
      { file: 'apps/mobile/src/a.ts', text: "fetch('http://127.0.0.1:1/')" },
    ])
    assert.deepEqual(found.findings, [])
  })

  it('fails a widened directive, a directive of another kind and another host', () => {
    const found = findMobilePageViolations(
      html(`${policy.replace("form-action 'none'", "form-action 'self'")}; sandbox`),
      [{ file: 'apps/mobile/src/a.ts', text: "\nfetch('https://evil.example/x')" }],
    )
    assert.deepEqual(at(found), [
      "apps/mobile/index.html:1 form-action is not 'none'",
      'apps/mobile/index.html:1 writes sandbox, a directive the policy does not hold',
      'apps/mobile/src/a.ts:2 names the host evil.example, which is not the core',
    ])
  })

  it('fails a page with no policy', () => {
    assert.ok(
      findMobilePageViolations('<html></html>', []).findings.length >=
        Object.keys(MOBILE_POLICY).length,
    )
  })

  it('passes the real page', () => {
    verifyMobilePagePolicy()
  })
})
