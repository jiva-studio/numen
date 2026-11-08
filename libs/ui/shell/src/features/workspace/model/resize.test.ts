/**
 * What a branch reports to the workspace about the shares a splitter reports. The
 * negatives matter most: a share that did not change is not reported, and neither
 * is the handle position while it was still being dragged.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { RESIZING, isSameShares, useBranchResize } from './resize'

const create = (shares: readonly number[] = [0.5, 0.5], hasFrame = true) => {
  const resize = vi.fn<(branch: string, shares: readonly number[]) => void>()
  const frame = document.createElement('div')
  const target: { resize?: ReturnType<typeof useBranchResize> } = {}
  const wrapper = mount(
    defineComponent({
      setup() {
        target.resize = useBranchResize({
          getBranchId: () => 'root',
          getShares: () => shares,
          getFrame: () => (hasFrame ? frame : undefined),
          resize,
        })
        return () => h('div')
      },
    }),
  )
  return { wrapper, resize, frame, ...target.resize! }
}

describe('shares compared', () => {
  it('are the same when no share is further than a hair from its other', () => {
    expect(isSameShares([0.5, 0.5], [0.5 + 1e-9, 0.5])).toBe(true)
  })

  it('differ by a share, and by the count of them', () => {
    expect(isSameShares([0.5, 0.5], [0.6, 0.4])).toBe(false)
    expect(isSameShares([0.5, 0.5], [0.5, 0.4])).toBe(false)
    expect(isSameShares([0, 1], [1e-6, 1])).toBe(false)
    expect(isSameShares([0.5, 0.5], [1])).toBe(false)
    expect(isSameShares([0.5], [0.5, 0.5])).toBe(false)
  })
})

describe('a layout reported with no handle retained', () => {
  it('notifies the workspace the shares, as fractions', () => {
    const one = create()

    one.onLayout([70, 30])

    expect(one.resize).toHaveBeenCalledWith('root', [0.7, 0.3])
  })

  it('notifies it nothing where the shares are the ones it holds', () => {
    const one = create()

    one.onLayout([50, 50])

    expect(one.resize).not.toHaveBeenCalled()
  })
})

describe('a handle retained', () => {
  it('marks the branch while it is retained and not after', () => {
    const one = create()

    one.onHold(true)
    expect(one.frame.hasAttribute(RESIZING)).toBe(true)
    one.onHold(false)

    expect(one.frame.hasAttribute(RESIZING)).toBe(false)
  })

  it('notifies the workspace once, where the handle came to rest', () => {
    const one = create()

    one.onHold(true)
    one.onLayout([60, 40])
    one.onLayout([70, 30])
    expect(one.resize).not.toHaveBeenCalled()

    one.onHold(false)

    expect(one.resize).toHaveBeenCalledTimes(1)
    expect(one.resize).toHaveBeenCalledWith('root', [0.7, 0.3])

    one.onLayout([80, 20])
    expect(one.resize).toHaveBeenCalledTimes(2)
    expect(one.resize).toHaveBeenLastCalledWith('root', [0.8, 0.2])
  })

  it('notifies it nothing where the handle came to rest on the shares retained', () => {
    const one = create()

    one.onHold(true)
    one.onLayout([50, 50])
    one.onHold(false)

    expect(one.resize).not.toHaveBeenCalled()
  })

  it('handles holding and releasing when the frame element is not mounted', () => {
    const one = create([0.5, 0.5], false)
    one.onHold(true)
    one.onLayout([60, 40])
    one.onHold(false)
    expect(one.resize).toHaveBeenCalledWith('root', [0.6, 0.4])
  })

  it('notifies it nothing of a handle put down that was never taken up', () => {
    const one = create()
    one.frame.setAttribute(RESIZING, '')

    one.onHold(false)

    expect(one.resize).not.toHaveBeenCalled()
    expect(one.frame.hasAttribute(RESIZING)).toBe(true)
  })

  it('drops where the handle had reached when the branch is removed while it is retained', () => {
    const one = create()

    one.onHold(true)
    one.onLayout([60, 40])
    one.wrapper.unmount()

    expect(one.resize).not.toHaveBeenCalled()
    expect(one.frame.hasAttribute(RESIZING)).toBe(false)
  })
})
