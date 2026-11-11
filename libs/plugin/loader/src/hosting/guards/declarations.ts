import type { PermissionTarget } from './permission'

/** The capability a manifest names to declare that its plugin registers agent providers. */
const AGENT_PROVIDER_CAPABILITY = 'ai.agent.provider'

/** The parts of a manifest that declare what a plugin registers. */
interface DeclaringManifest {
  readonly permissions?: readonly string[] | undefined
  readonly capabilities?: readonly string[] | undefined
  readonly settingsViews?: readonly { readonly id: string }[] | undefined
  readonly contributes?:
    | {
        readonly readers?: readonly { readonly id: string }[] | undefined
        readonly tabKinds?: readonly { readonly kind: string }[] | undefined
        readonly commands?: readonly { readonly id: string }[] | undefined
        readonly handlers?: readonly { readonly command: string }[] | undefined
        readonly dropHandlers?: readonly { readonly types: readonly string[] }[] | undefined
      }
    | undefined
}

type Registration =
  'drop type' | 'reader' | 'tab kind' | 'command' | 'settings view' | 'agent provider'

interface Undeclared {
  readonly kind: Registration
  readonly id: string
  readonly where: string
}

function fail(target: PermissionTarget, { kind, id, where }: Undeclared): never {
  throw new Error(
    `Plugin "${target.pluginId}" registers the ${kind} "${id}", which its manifest does not declare in ${where}`,
  )
}

function getManifest(target: PermissionTarget): DeclaringManifest | undefined {
  return target.manifest
}

export function requireDeclaredReader(target: PermissionTarget, id: string): void {
  const declared = getManifest(target)?.contributes?.readers ?? []
  if (!declared.some((one) => one.id === id))
    fail(target, { kind: 'reader', id, where: 'contributes.readers' })
}

export function requireDeclaredTabKind(target: PermissionTarget, kind: string): void {
  const declared = getManifest(target)?.contributes?.tabKinds ?? []
  if (!declared.some((one) => one.kind === kind)) {
    fail(target, { kind: 'tab kind', id: kind, where: 'contributes.tabKinds' })
  }
}

/** A shared command is declared by its handler, any other command by its own entry. */
export function requireDeclaredCommand(
  target: PermissionTarget,
  id: string,
  isShared: boolean,
): void {
  const contributes = getManifest(target)?.contributes
  const declared = isShared
    ? (contributes?.handlers ?? []).some((one) => one.command === id)
    : (contributes?.commands ?? []).some((one) => one.id === id)
  if (!declared) {
    fail(target, {
      kind: 'command',
      id,
      where: isShared ? 'contributes.handlers' : 'contributes.commands',
    })
  }
}

export function requireDeclaredSettingsView(target: PermissionTarget, id: string): void {
  const declared = getManifest(target)?.settingsViews ?? []
  if (!declared.some((one) => one.id === id))
    fail(target, { kind: 'settings view', id, where: 'settingsViews' })
}

export function requireDeclaredAgentProvider(target: PermissionTarget, id: string): void {
  if (!(getManifest(target)?.capabilities ?? []).includes(AGENT_PROVIDER_CAPABILITY)) {
    fail(target, { kind: 'agent provider', id, where: 'capabilities' })
  }
}

/** Checks every type of a drop handler against the manifest, and answers the types it read. */
export function requireDeclaredDropTypes(
  target: PermissionTarget,
  types: readonly string[],
): readonly string[] {
  const declared = (getManifest(target)?.contributes?.dropHandlers ?? []).flatMap((one) => [
    ...one.types,
  ])
  const checked = [...types]
  for (const type of checked) {
    if (!declared.includes(type))
      fail(target, { kind: 'drop type', id: type, where: 'contributes.dropHandlers' })
  }
  return checked
}
