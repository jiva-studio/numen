import type { RequirePermission } from './permission'
import { asFailure } from '@numen/result'
import { READ, Rule, WRITE, type RulesOf, type Value } from './servicePermissions'

type Members = Record<string, Value>

/** What a refusal needs: the check of the permissions, and the path of the service for the message. */
export interface Guard {
  readonly check: RequirePermission
  readonly path?: string
}

function denyPermission(error: unknown) {
  return asFailure({
    error: 'permissionDenied',
    message: error instanceof Error ? error.message : String(error),
  })
}

function requireAll(
  rule: Rule,
  call: { readonly operation: string; readonly args: readonly Value[] },
  check: RequirePermission,
): void {
  const { operation, args } = call
  if (rule.isReadOnly(args)) {
    try {
      check(READ, operation)
    } catch {
      check(WRITE, operation)
    }
    return
  }
  for (const permission of rule.permissions) check(permission, operation)
}

function guardMember(
  service: Members,
  entry: { readonly key: string; readonly rule: Rule },
  guard: { readonly check: RequirePermission; readonly operation: string },
): PropertyDescriptor {
  const { key, rule } = entry
  const { check, operation } = guard
  if (typeof service[key] === 'function') {
    return {
      enumerable: true,
      value: (...args: never[]) => {
        try {
          requireAll(rule, { operation, args }, check)
        } catch (error) {
          if (rule.isResult) return Promise.resolve(denyPermission(error))
          if (rule.isAsync) return Promise.reject(error)
          throw error
        }
        return Reflect.apply(service[key] as () => Value, service, args)
      },
    }
  }
  return {
    enumerable: true,
    get: () => {
      requireAll(rule, { operation, args: [] }, check)
      return service[key]
    },
  }
}

/**
 * The service as the plugin may use it: only the members the rules list, each checked when it is
 * used. A member the service has and the rules do not list is not reachable; a member the service
 * lacks is absent. `guard.path` prefixes the operation named in a refusal.
 */
export function guardService<S extends object>(service: S, rules: RulesOf<S>, guard: Guard): S {
  const source = service as Members
  const guarded: Members = {}
  for (const [key, rule] of Object.entries<Rule | object>(rules)) {
    const member = source[key]
    if (member === undefined) continue
    const operation = guard.path === undefined ? key : `${guard.path}.${key}`
    if (rule instanceof Rule) {
      const descriptor = guardMember(source, { key, rule }, { check: guard.check, operation })
      Object.defineProperty(guarded, key, descriptor)
    } else if (typeof member === 'object' && member !== null) {
      guarded[key] = guardService(member, rule, {
        check: guard.check,
        path: operation,
      })
    }
  }
  return guarded as S
}
