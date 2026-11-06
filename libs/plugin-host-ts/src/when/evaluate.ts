/** The value of a when clause for the values of its keys. */
import type { WhenKey } from '@numen/plugin-sdk'
import type { WhenExpression } from './parse'

/** The value of every key a condition may name. */
export type WhenValues = Readonly<Record<WhenKey, boolean>>

export function evaluateWhen(expression: WhenExpression, values: WhenValues): boolean {
  switch (expression.operator) {
    case 'key':
      return values[expression.key]
    case '!':
      return !evaluateWhen(expression.operand, values)
    case '&&':
      return evaluateWhen(expression.left, values) && evaluateWhen(expression.right, values)
    case '||':
      return evaluateWhen(expression.left, values) || evaluateWhen(expression.right, values)
  }
}
