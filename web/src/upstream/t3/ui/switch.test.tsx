// T3 Code switch.test.tsx (PR #11580), MIT, Copyright (c) 2026 T3 Tools Inc.
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { Switch } from './switch.js'

afterEach(cleanup)

describe('Switch accessibility', () => {
  it('exposes the checked state to assistive technology', () => {
    const view = render(<Switch aria-label="Enable Claude" checked />)
    expect(screen.getByRole('switch').getAttribute('aria-checked')).toBe('true')
    view.rerender(<Switch aria-label="Enable Claude" checked={false} />)
    expect(screen.getByRole('switch').getAttribute('aria-checked')).toBe('false')
  })

  it('exposes the mixed state', () => {
    render(<Switch aria-label="Enable Claude" checked={false} mixed />)
    expect(screen.getByRole('switch').getAttribute('aria-checked')).toBe('mixed')
  })
})
