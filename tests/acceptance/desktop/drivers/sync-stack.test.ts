import { describe, expect, it } from 'vitest'
import { ENV_ROOT_KEY, ENV_URL, getCheckoutId, isStackRequired, readStack } from './sync-stack.js'

const ROOT = 'e0'.repeat(32)

describe('readStack', () => {
  it('reads the address, the root key and the optional addresses of the stand-in and the bucket', () => {
    const env = { [ENV_URL]: 'http://127.0.0.1:8787/', [ENV_ROOT_KEY]: ROOT, NUMEN_SYNC_TEST_PAYMENT_URL: 'http://127.0.0.1:8789/' }

    expect(readStack(env)).toStrictEqual({ url: 'http://127.0.0.1:8787', rootKey: ROOT, paymentUrl: 'http://127.0.0.1:8789', bucketUrl: undefined })
  })

  it('names no stack when the address or the root key is not set', () => {
    expect(readStack({})).toBeUndefined()
    expect(readStack({ [ENV_URL]: 'http://127.0.0.1:8787' })).toBeUndefined()
    expect(readStack({ [ENV_URL]: '', [ENV_ROOT_KEY]: ROOT })).toBeUndefined()
  })

  it('refuses a root key that is not 64 hex digits', () => {
    expect(() => readStack({ [ENV_URL]: 'http://127.0.0.1:8787', [ENV_ROOT_KEY]: 'abc' })).toThrow('64 hex digits')
  })
})

describe('isStackRequired', () => {
  it('is true only when the job says 1', () => {
    expect(isStackRequired({ NUMEN_SYNC_TEST_REQUIRED: '1' })).toBe(true)
    expect(isStackRequired({ NUMEN_SYNC_TEST_REQUIRED: '0' })).toBe(false)
    expect(isStackRequired({})).toBe(false)
  })
})

describe('getCheckoutId', () => {
  it('reads the id from the address of the payment page', () => {
    expect(getCheckoutId('http://127.0.0.1:8788/pay/chk_12')).toBe('chk_12')
    expect(getCheckoutId('http://127.0.0.1:8788/pay/chk_12?x=1')).toBe('chk_12')
  })

  it('refuses an address that is no payment page', () => {
    expect(() => getCheckoutId('http://127.0.0.1:8788/')).toThrow('not the address of a payment page')
  })
})
