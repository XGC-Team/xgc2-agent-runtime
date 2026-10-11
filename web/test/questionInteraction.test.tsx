import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { carryDisplacedCustomAnswerIntoPrompt } from '../src/upstream/t3/pendingUserInput.js'
import { T3Conversation, T3PendingRequests, type T3PendingRequestsProps } from '../src/T3Conversation.js'

// The Lexical editor is not what is under test: a plain textarea stands in for it, which
// keeps typing deterministic while the cards, the draft and the callbacks stay real.
vi.mock('../src/upstream/t3/ComposerPromptEditor.js', () => ({
  ComposerPromptEditor: ({ value, onChange, placeholder, disabled }: { value: string; onChange: (value: string) => void; placeholder?: string; disabled?: boolean }) =>
    <textarea aria-label={placeholder} value={value} disabled={disabled} onChange={event => onChange(event.target.value)} />,
}))

afterEach(() => { cleanup(); vi.useRealTimers() })

const two = (secret = false) => [
  { id: 'first', header: 'First', question: 'First?', allowCustomAnswer: true, options: [{ value: 'id:a', label: 'Camera' }] },
  { id: 'second', header: 'Second', question: 'Second?', allowCustomAnswer: true, isSecret: secret, options: [{ value: 'id:b', label: 'Telemetry' }] },
]
function props(questions = two(), overrides: Partial<T3PendingRequestsProps> = {}): T3PendingRequestsProps {
  return { model: { sessionKey: 'questions', approvals: [], userInputs: [{ requestId: 'input:1', questions }] }, active: true,
    onApproval: vi.fn().mockResolvedValue(undefined), onUserInput: vi.fn().mockResolvedValue(undefined), onCancelRequest: vi.fn().mockResolvedValue(undefined), ...overrides }
}
const conversation = (input: T3PendingRequestsProps, extra: Record<string, unknown> = {}) =>
  <T3Conversation {...input} model={{ ...input.model, items: [], isRunning: false }} onSend={vi.fn()} onInterrupt={vi.fn()} {...extra} />

describe('question auto-advance', () => {
  it('moves on from a single-choice answer after the delay', async () => {
    vi.useFakeTimers()
    render(<T3PendingRequests {...props()} />)
    fireEvent.click(screen.getByRole('button', { name: /Camera/ }))
    expect(screen.getByText('First?')).toBeTruthy()
    await act(async () => { vi.advanceTimersByTime(250) })
    expect(screen.getByText('Second?')).toBeTruthy()
  })

  it('does not undo a navigation to the previous question with a stale timer', async () => {
    vi.useFakeTimers()
    render(<T3PendingRequests {...props()} />)
    fireEvent.click(screen.getByRole('button', { name: /Camera/ }))
    await act(async () => { vi.advanceTimersByTime(250) })
    // Answering the second question schedules its own advance; going back first must cancel it.
    fireEvent.click(screen.getByRole('button', { name: /Telemetry/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Previous' }))
    expect(screen.getByText('First?')).toBeTruthy()
    await act(async () => { vi.advanceTimersByTime(250) })
    expect(screen.getByText('First?')).toBeTruthy()
    expect(screen.queryByText('Second?')).toBeNull()
  })

  it('does not skip the next question after a manual advance', async () => {
    vi.useFakeTimers()
    const questions = [...two(), { id: 'third', header: 'Third', question: 'Third?', allowCustomAnswer: true, options: [{ value: 'id:c', label: 'Radar' }] }]
    render(<T3PendingRequests {...props(questions)} />)
    fireEvent.click(screen.getByRole('button', { name: /Camera/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Next question' }))
    expect(screen.getByText('Second?')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: /Telemetry/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Previous' }))
    fireEvent.click(screen.getByRole('button', { name: 'Next question' }))
    await act(async () => { vi.advanceTimersByTime(250) })
    expect(screen.getByText('Second?')).toBeTruthy()
    expect(screen.queryByText('Third?')).toBeNull()
  })
})

describe('typed answer text', () => {
  it('moves to the end of the draft behind a blank line', () => {
    expect(carryDisplacedCustomAnswerIntoPrompt('', 'also rename the flag ')).toBe('also rename the flag')
    expect(carryDisplacedCustomAnswerIntoPrompt('first half\n', 'second half')).toBe('first half\n\nsecond half')
    expect(carryDisplacedCustomAnswerIntoPrompt('draft', undefined)).toBe('draft')
    expect(carryDisplacedCustomAnswerIntoPrompt('draft', '   ')).toBe('draft')
  })

  it('goes back to the conversation draft when an option replaces it', () => {
    const onDraftChange = vi.fn()
    render(conversation(props(), { draft: 'Existing note', onDraftChange }))
    const answer = screen.getAllByRole('textbox').find(field => field.getAttribute('aria-label') === 'Your answer') as HTMLTextAreaElement
    fireEvent.change(answer, { target: { value: 'my own answer' } })
    expect(answer.value).toBe('my own answer')
    fireEvent.click(screen.getByRole('button', { name: /Camera/ }))
    expect(onDraftChange).toHaveBeenCalledWith('Existing note\n\nmy own answer')
    expect((screen.getAllByRole('textbox').find(field => field.getAttribute('aria-label') === 'Your answer') as HTMLTextAreaElement).value).toBe('')
    expect(screen.getByRole('button', { name: /Camera/ }).getAttribute('aria-pressed')).toBe('true')
  })

  it('does not touch the draft when nothing was typed', () => {
    const onDraftChange = vi.fn()
    render(conversation(props(), { draft: 'Existing note', onDraftChange }))
    fireEvent.click(screen.getByRole('button', { name: /Camera/ }))
    expect(onDraftChange).not.toHaveBeenCalled()
  })

  it('never copies a secret answer into the visible draft', () => {
    vi.useFakeTimers()
    const onDraftChange = vi.fn()
    render(conversation(props(two(true)), { draft: '', onDraftChange }))
    fireEvent.click(screen.getByRole('button', { name: /Camera/ }))
    act(() => { vi.advanceTimersByTime(250) })
    const secret = screen.getByLabelText('Your answer') as HTMLInputElement
    expect(secret.type).toBe('password')
    fireEvent.change(secret, { target: { value: 'hunter2' } })
    fireEvent.click(screen.getByRole('button', { name: /Telemetry/ }))
    expect(onDraftChange).not.toHaveBeenCalled()
  })

  it('leaves the entry without a composer working: the option simply replaces the text', () => {
    render(<T3PendingRequests {...props()} />)
    const answer = screen.getByLabelText('Your answer') as HTMLTextAreaElement
    fireEvent.change(answer, { target: { value: 'typed' } })
    fireEvent.click(screen.getByRole('button', { name: /Camera/ }))
    expect((screen.getByLabelText('Your answer') as HTMLTextAreaElement).value).toBe('')
    expect(screen.getByRole('button', { name: /Camera/ }).getAttribute('aria-pressed')).toBe('true')
  })
})
