/**
 * The slot entries of the plugins in the tree: each names a slot of the window or a submenu a plugin
 * declares, and a command the manifest declares, the editor provides or the application shares; the
 * window and the host name no plugin.
 */
import assert from 'node:assert/strict'
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'
import { parse as parseVue } from '@vue/compiler-sfc'
import ts from 'typescript'
import {
  ManifestCommandSchema,
  ManifestTabKindSchema,
} from '../../libs/core/protocol/src/numen/v1/manifest_pb.ts'
import { parseManifest, readPluginManifests } from './manifests.mjs'

const ROOT = fileURLToPath(new URL('../../', import.meta.url))
const EDITOR_SRC = join(ROOT, 'apps/desktop/editor/src')
const SLOTS = JSON.parse(readFileSync(join(ROOT, 'libs/core/protocol/slots.json'), 'utf8')).slots.map(
  (one) => one.name,
)
const APP_PREFIX = 'app:'
const SHARED_COMMANDS = new Set(
  JSON.parse(readFileSync(join(ROOT, 'libs/core/protocol/commands.json'), 'utf8')).commands.map(
    (one) => one.id,
  ),
)

const isTestFile = (name) => /\.(test|stories|spec)\.[cm]?[jt]s$/.test(name)

/** Every source file of the folder that is neither a test nor a story, as a path. */
const listSources = (folder, extensions) =>
  readdirSync(folder).flatMap((name) => {
    const path = join(folder, name)
    if (name === 'node_modules' || name === 'dist') return []
    if (statSync(path).isDirectory()) return listSources(path, extensions)
    return extensions.some((one) => name.endsWith(one)) && !isTestFile(name) ? [path] : []
  })

const readEditorSources = () =>
  listSources(EDITOR_SRC, ['.ts', '.vue']).filter(
    (file) => !relative(EDITOR_SRC, file).split(sep).includes('testing'),
  )

const unwrapExpression = (node) =>
  node &&
  (ts.isAsExpression(node) || ts.isSatisfiesExpression(node) || ts.isParenthesizedExpression(node))
    ? unwrapExpression(node.expression)
    : node

const getPropertyName = (property) =>
  property.name && (ts.isIdentifier(property.name) || ts.isStringLiteral(property.name))
    ? property.name.text
    : ''

/** The commands the editor provides, by full id, with whether the table gives each an icon name. */
const readEditorCommands = () => {
  const idsPath = join(EDITOR_SRC, 'shared/model/command/ids.ts')
  const idsFile = ts.createSourceFile(
    idsPath,
    readFileSync(idsPath, 'utf8'),
    ts.ScriptTarget.Latest,
    true,
  )
  const byKey = new Map()
  for (const statement of idsFile.statements) {
    if (!ts.isVariableStatement(statement)) continue
    for (const declaration of statement.declarationList.declarations) {
      if (!ts.isIdentifier(declaration.name) || declaration.name.text !== 'COMMAND') continue
      const init = unwrapExpression(declaration.initializer)
      if (!init || !ts.isObjectLiteralExpression(init)) continue
      for (const property of init.properties) {
        if (!ts.isPropertyAssignment(property)) continue
        const call = unwrapExpression(property.initializer)
        if (
          call &&
          ts.isCallExpression(call) &&
          ts.isIdentifier(call.expression) &&
          call.expression.text === 'formatAppCommandId' &&
          call.arguments.length > 0 &&
          ts.isStringLiteral(call.arguments[0])
        ) {
          byKey.set(getPropertyName(property), `${APP_PREFIX}${call.arguments[0].text}`)
        }
      }
    }
  }

  const withIcon = new Set()
  for (const file of listSources(join(EDITOR_SRC, 'features'), ['.ts'])) {
    const sourceFile = ts.createSourceFile(
      file,
      readFileSync(file, 'utf8'),
      ts.ScriptTarget.Latest,
      true,
    )
    const visit = (node) => {
      if (ts.isObjectLiteralExpression(node)) {
        let key = ''
        let hasIcon = false
        for (const property of node.properties) {
          if (!ts.isPropertyAssignment(property)) continue
          const name = getPropertyName(property)
          const value = unwrapExpression(property.initializer)
          if (
            name === 'id' &&
            value &&
            ts.isPropertyAccessExpression(value) &&
            ts.isIdentifier(value.expression) &&
            value.expression.text === 'COMMAND'
          ) {
            key = value.name.text
          }
          if (
            name === 'iconName' &&
            value &&
            ts.isStringLiteral(value) &&
            /^[a-z][a-z0-9-]*$/.test(value.text)
          ) {
            hasIcon = true
          }
        }
        if (key && hasIcon) withIcon.add(key)
      }
      ts.forEachChild(node, visit)
    }
    visit(sourceFile)
  }

  const commands = new Map()
  for (const [key, fullId] of byKey) {
    commands.set(fullId, { hasIconName: withIcon.has(key) })
  }
  return commands
}

/** The string literals of a TypeScript or Vue source file, read from its syntax tree. */
const readStringLiterals = (file) => {
  const text = readFileSync(file, 'utf8')
  const literals = []
  const collectScript = (name, content) => {
    const sourceFile = ts.createSourceFile(name, content, ts.ScriptTarget.Latest, true)
    const visit = (node) => {
      if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
        literals.push(node.text)
      } else if (ts.isTemplateExpression(node)) {
        literals.push(node.head.text)
        for (const span of node.templateSpans) literals.push(span.literal.text)
      }
      ts.forEachChild(node, visit)
    }
    visit(sourceFile)
  }
  if (file.endsWith('.vue')) {
    const { descriptor } = parseVue(text)
    for (const block of [descriptor.script, descriptor.scriptSetup].filter(Boolean)) {
      collectScript(file.replace(/\.vue$/, '.ts'), block.content)
    }
    const visitTemplate = (node) => {
      if (!node) return
      if (typeof node.content === 'string') literals.push(node.content)
      for (const prop of node.props ?? []) {
        if (prop.value && typeof prop.value.content === 'string') literals.push(prop.value.content)
      }
      for (const child of node.children ?? []) visitTemplate(child)
    }
    visitTemplate(descriptor.template?.ast)
    return literals
  }
  collectScript(file, text)
  return literals
}

/** The member names of a TypeScript interface declaration in `file`, with those of the interfaces it extends in the same file. */
const readInterfaceFields = (file, name) => {
  const sourceFile = ts.createSourceFile(
    file,
    readFileSync(file, 'utf8'),
    ts.ScriptTarget.Latest,
    true,
  )
  const interfaces = new Map(
    sourceFile.statements
      .filter((statement) => ts.isInterfaceDeclaration(statement))
      .map((statement) => [statement.name.text, statement]),
  )
  const collect = (interfaceName, seen) => {
    const declaration = interfaces.get(interfaceName)
    if (!declaration || seen.has(interfaceName)) {
      return []
    }
    seen.add(interfaceName)
    const own = declaration.members
      .map((member) => (member.name && ts.isIdentifier(member.name) ? member.name.text : ''))
      .filter(Boolean)
    const inherited = (declaration.heritageClauses ?? [])
      .flatMap((clause) => clause.types)
      .flatMap((type) => collect(type.expression.getText(sourceFile), seen))
    return [...own, ...inherited]
  }
  return collect(name, new Set())
}

test('neither the host module nor the window names a plugin', () => {
  const pluginIds = new Set(readPluginManifests().map(({ document }) => document.id))
  const files = [
    join(ROOT, 'libs/plugin/loader/src/slots/slotRegistry.ts'),
    ...readEditorSources(),
  ]
  for (const file of files) {
    for (const literal of readStringLiterals(file)) {
      assert.ok(
        !literal.includes('md.numen.') && !pluginIds.has(literal),
        `${relative(ROOT, file)} names a plugin: ${literal}`,
      )
    }
  }
})

test('no source records where a note found by its name is shown', () => {
  const schemaFields = ManifestTabKindSchema.fields.flatMap((field) => [field.name, field.jsonName])
  assert.ok(schemaFields.includes('kind'))
  for (const forbidden of [
    'isNoteDestination',
    'noteDestination',
    'is_note_destination',
    'note_destination',
  ]) {
    assert.ok(!schemaFields.includes(forbidden), `ManifestTabKind has field ${forbidden}`)
  }
  const { warnings } = parseManifest(
    JSON.stringify({
      id: 'tools.sample',
      name: 'Sample',
      version: '1.0.0',
      contributes: {
        tabKinds: [{ kind: 'sample', title: 'Sample', isNoteDestination: true }],
      },
    }),
  )
  assert.deepEqual(warnings, ['unknown key "contributes.tabKinds[0].isNoteDestination"'])

  const sdkFields = readInterfaceFields(
    join(ROOT, 'libs/plugin/sdk-ts/src/types.ts'),
    'TabKindRegistration',
  )
  assert.ok(sdkFields.includes('kind'))
  assert.ok(!sdkFields.includes('isNoteDestination') && !sdkFields.includes('noteDestination'))
})

test('a command declaration says nothing of where it is shown', () => {
  const placeFields = ['menu', 'menus', 'location', 'slot', 'slots']
  const protoFields = ManifestCommandSchema.fields.flatMap((field) => [field.name, field.jsonName])
  assert.ok(protoFields.includes('id'))
  for (const field of placeFields) {
    assert.ok(!protoFields.includes(field), `ManifestCommand has field ${field}`)
  }

  const sdkFields = readInterfaceFields(
    join(ROOT, 'libs/plugin/sdk-ts/src/types.ts'),
    'CommandRegistration',
  )
  assert.ok(sdkFields.includes('id'))
  for (const field of placeFields) {
    assert.ok(!sdkFields.includes(field), `CommandRegistration has field ${field}`)
  }
})

test('every slot entry of a manifest of the tree is in a known place and names a command', () => {
  const editorCommands = readEditorCommands()
  const manifests = readPluginManifests()
  const submenus = new Set(
    manifests.flatMap(({ document }) =>
      (document.contributes?.slots ?? []).filter((one) => one.submenu).map((one) => one.submenu),
    ),
  )
  for (const { name, document } of manifests) {
    const declared = new Map((document.contributes?.commands ?? []).map((one) => [one.id, one]))
    for (const entry of document.contributes?.slots ?? []) {
      const where = `${name}: ${entry.slot} ${entry.command ?? entry.view ?? entry.submenu}`
      assert.ok(
        SLOTS.includes(entry.slot) || submenus.has(entry.slot),
        `${where} is neither in a slot of the window nor in a submenu a plugin declares`,
      )
      if (!entry.command) continue
      if (entry.command.startsWith(APP_PREFIX)) {
        assert.ok(
          editorCommands.has(entry.command) || SHARED_COMMANDS.has(entry.command),
          `${where} is neither a command of the editor nor a shared command`,
        )
        if (entry.slot === 'titleBar.commands') {
          assert.ok(
            editorCommands.get(entry.command)?.hasIconName,
            `${where} has no icon name in the editor`,
          )
        }
      } else {
        const command = declared.get(entry.command)
        assert.ok(command, `${where} is not declared by the manifest`)
        if (entry.slot === 'titleBar.commands') {
          assert.match(
            command.icon ?? '',
            /^[a-z][a-z0-9]*(-[a-z0-9]+)*$/,
            `${where} has no icon name`,
          )
        }
      }
    }
  }
})

test('the plugin that edits notes places nothing in the menu of a search result', () => {
  const notes = readPluginManifests().find(({ document }) => document.id === 'md.numen.notes')
  assert.ok(notes, 'the notes plugin is not in the tree')
  const placed = (notes.document.contributes?.slots ?? []).filter(
    (one) => one.slot === 'palette.searchResult',
  )
  assert.deepEqual(placed, [])
})

test('the plugin documentation describes slots', () => {
  const docs = readFileSync(join(ROOT, 'docs/plugins/README.md'), 'utf8')
  const start = docs.search(/^#{2,3} Slots\s*$/m)
  assert.ok(start >= 0, 'the documentation has no Slots section')
  const rest = docs.slice(start + 1)
  const end = rest.search(/\n## /)
  const section = end < 0 ? rest : rest.slice(0, end)
  for (const word of [
    'contributes.slots',
    ...SLOTS,
    'submenu',
    'group',
    'order',
    'alphanumerically',
  ]) {
    assert.ok(section.includes(word), `the Slots section does not mention ${word}`)
  }
  assert.match(
    docs,
    /\|\s*`contributes\.slots`/,
    'the table of contributions has no row for contributes.slots',
  )
})

test('the window frame of the shell names no plugin', () => {
  const pluginIds = new Set(readPluginManifests().map(({ document }) => document.id))
  const folder = join(ROOT, 'libs/ui/workbench/src/features/window-frame')
  for (const file of listSources(folder, ['.ts', '.vue'])) {
    for (const literal of readStringLiterals(file)) {
      assert.ok(
        !literal.includes('md.numen.') && !pluginIds.has(literal),
        `${relative(ROOT, file)} names a plugin: ${literal}`,
      )
    }
  }
})

test('the docs say how the title bar draws an entry', () => {
  const readme = readFileSync(join(ROOT, 'docs/plugins/README.md'), 'utf8')
  const slots = readme.slice(readme.indexOf('\n## Slots'))
  const section = slots.slice(0, slots.indexOf('\n## ', 4))
  const titleBar = section
    .split('\n')
    .filter((line) => line.includes('titleBar.commands'))
    .join('\n')
  assert.match(titleBar, /button/i, 'the section does not say an entry is a button')
  assert.match(titleBar, /vault is open/i, 'the section does not say when the buttons are shown')
  assert.match(titleBar, /hover text.*\bkey\b/i, 'the section does not say the key is in the hover text')
})
