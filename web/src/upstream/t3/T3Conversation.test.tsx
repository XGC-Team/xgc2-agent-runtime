// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { T3Conversation, T3PendingRequests, type T3PendingRequestsProps } from '../../T3Conversation.js';

const request = { requestId: 'request:1', requestKind: 'command', title: 'Apply experiment patch', detail: 'printf example', options: [{ decision: 'once:only', label: 'Allow once' }] };
function props(overrides: Partial<T3PendingRequestsProps> = {}): T3PendingRequestsProps {
  return { model: { sessionKey: 'target:one/session:1', approvals: [request], userInputs: [] }, active: true,
    onApproval: vi.fn().mockResolvedValue(undefined), onUserInput: vi.fn().mockResolvedValue(undefined), onCancelRequest: vi.fn().mockResolvedValue(undefined), ...overrides };
}
afterEach(() => { cleanup(); vi.useRealTimers(); });
describe('migrated T3 request controls', () => {
  it('uses only native choices and keeps the request title visible', async () => {
    const input = props(); render(<T3PendingRequests {...input} />);
    expect(screen.getByText('Apply experiment patch')).toBeTruthy();
    expect(screen.queryByText('Always allow this session')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Allow once' }));
    await waitFor(() => expect(input.onApproval).toHaveBeenCalledWith('request:1', 'once:only'));
    expect(input.onCancelRequest).not.toHaveBeenCalled();
  });
  it('keeps acknowledged and server-submitted decisions disabled until resolution', async () => {
    const input = props(); const view = render(<T3PendingRequests {...input} />);
    const allow = screen.getByRole('button', { name: 'Allow once' }) as HTMLButtonElement;
    fireEvent.click(allow);
    await waitFor(() => expect(input.onApproval).toHaveBeenCalledTimes(1));
    await act(async () => {});
    expect(allow.disabled).toBe(true);
    fireEvent.click(allow); expect(input.onApproval).toHaveBeenCalledTimes(1);
    view.rerender(<T3PendingRequests {...input} model={{ ...input.model, sessionKey: 'other', approvals: [{ ...request, submitted: true }] }} />);
    expect((screen.getByRole('button', { name: 'Allow once' }) as HTMLButtonElement).disabled).toBe(true);
  });
  it('cancels the request through its separate callback', async () => {
    const input = props(); render(<T3PendingRequests {...input} />);
    fireEvent.click(screen.getByRole('button', { name: 'Cancel request' }));
    await waitFor(() => expect(input.onCancelRequest).toHaveBeenCalledWith('request:1'));
    expect(input.onApproval).not.toHaveBeenCalled();
  });
  it('preserves multi-select option IDs and requires explicit answer submission', async () => {
    const input = props({ model: { sessionKey: 'target:one/session:1', approvals: [], userInputs: [{ requestId: 'input:1', title: 'Choose checks', questions: [{ id: 'checks', header: 'Checks', question: 'Which checks?', multiSelect: true, allowCustomAnswer: false, options: [{ value: 'id:a', label: 'Camera' }, { value: 'id:b', label: 'Telemetry' }] }] }] } });
    render(<T3PendingRequests {...input} />);
    fireEvent.click(screen.getByRole('button', { name: /Camera/ }));
    fireEvent.click(screen.getByRole('button', { name: /Telemetry/ }));
    expect(input.onUserInput).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Submit answer' }));
    await waitFor(() => expect(input.onUserInput).toHaveBeenCalledWith('input:1', { checks: ['id:a', 'id:b'] }));
  });
  it('never submits the last single-choice answer on its auto-advance timer', async () => {
    vi.useFakeTimers();
    const input = props({ model: { sessionKey: 'single', approvals: [], userInputs: [{ requestId: 'input:1', questions: [{ id: 'one', header: 'One', question: 'Which?', allowCustomAnswer: false, options: [{ value: 'id:a', label: 'Camera' }] }] }] } });
    render(<T3PendingRequests {...input} />);
    fireEvent.click(screen.getByRole('button', { name: /Camera/ }));
    await act(async () => { vi.advanceTimersByTime(250); });
    expect(input.onUserInput).not.toHaveBeenCalled();
    expect((screen.getByRole('button', { name: 'Submit answer' }) as HTMLButtonElement).disabled).toBe(false);
  });
  it('does not answer when parked and cancels queued auto-advance', async () => {
    vi.useFakeTimers();
    const input = props({ model: { sessionKey: 'park', approvals: [], userInputs: [{ requestId: 'input:1', questions: [{ id: 'one', header: 'First', question: 'First?', allowCustomAnswer: false, options: [{ value: 'id:a', label: 'Camera' }] }, { id: 'two', header: 'Second', question: 'Second?', allowCustomAnswer: false, options: [{ value: 'id:b', label: 'Telemetry' }] }] }] } });
    const view = render(<T3PendingRequests {...input} />);
    fireEvent.click(screen.getByRole('button', { name: /Camera/ }));
    view.rerender(<T3PendingRequests {...input} active={false} />);
    fireEvent.keyDown(document, { key: '1' });
    await act(async () => { vi.advanceTimersByTime(300); });
    expect(screen.queryByText('Second?')).toBeNull();
    expect(input.onUserInput).not.toHaveBeenCalled();
  });
  it('keeps role/id pairs specific to the surface session scope', () => {
    const input = props(); const { container } = render(<><T3PendingRequests {...input} /><T3PendingRequests {...input} model={{ ...input.model, sessionKey: 'notification:session:1:request:1' }} /></>);
    const keys = Array.from(container.querySelectorAll('[data-xgc-role]')).map(element => {
      expect(element.getAttribute('data-xgc-id')).toBeTruthy();
      return `${element.getAttribute('data-xgc-role')}|${element.getAttribute('data-xgc-id')}`;
    });
    expect(new Set(keys).size).toBe(keys.length);
  });
  it('keeps the same choice actionable after a real rejected response', async () => {
    const input = props({ onApproval: vi.fn().mockRejectedValue(new Error('Request is no longer available')) });
    render(<T3PendingRequests {...input} />);
    fireEvent.click(screen.getByRole('button', { name: 'Allow once' }));
    await waitFor(() => expect(screen.getByRole('alert').textContent).toBe('Request is no longer available'));
    expect((screen.getByRole('button', { name: 'Allow once' }) as HTMLButtonElement).disabled).toBe(false);
  });
});

describe('migrated T3 conversation host boundary', () => {
  it('keeps native approval actionable when message sending is unavailable', () => {
    const input = props();
    render(<T3Conversation {...input} model={{ ...input.model, items: [], isRunning: false }} sendDisabled onSend={vi.fn()} onInterrupt={vi.fn()} />);
    expect((screen.getByRole('button', { name: 'Allow once' }) as HTMLButtonElement).disabled).toBe(false);
    expect((screen.getByRole('button', { name: 'Send message' }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByTestId('composer-editor').getAttribute('contenteditable')).toBe('true');
  });
  it('clears only after successful admission and preserves a newer controlled draft', async () => {
    const input = props({ model: { sessionKey: 'draft', approvals: [], userInputs: [] } });
    const model = { ...input.model, items: [], isRunning: false };
    let resolve!: () => void;
    const onSend = vi.fn(() => new Promise<void>(done => { resolve = done; }));
    const onDraftChange = vi.fn();
    const view = render(<T3Conversation {...input} model={model} draft="first draft" onDraftChange={onDraftChange} onSend={onSend} onInterrupt={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Send message' }));
    expect(onSend).toHaveBeenCalledWith('first draft');
    expect(screen.getByTestId('composer-editor').getAttribute('contenteditable')).toBe('true');
    view.rerender(<T3Conversation {...input} model={model} draft="newer draft" onDraftChange={onDraftChange} onSend={onSend} onInterrupt={vi.fn()} />);
    await act(async () => resolve());
    expect(onDraftChange).not.toHaveBeenCalledWith('');
    expect(screen.getByTestId('composer-editor').getAttribute('contenteditable')).toBe('true');
    fireEvent.click(screen.getByRole('button', { name: 'Send message' }));
    expect(onSend).toHaveBeenCalledWith('newer draft');
    await act(async () => resolve());
    expect(onDraftChange).toHaveBeenCalledWith('');
  });
  it('allows an explicit queue host to accept another message during a running turn without interrupting', async () => {
    const input=props({model:{sessionKey:'queue',approvals:[],userInputs:[]}});
    const send=vi.fn(async()=>undefined),interrupt=vi.fn();
    const view=render(<T3Conversation {...input} model={{...input.model,items:[],isRunning:true}} queueEnabled draft="next" onSend={send} onInterrupt={interrupt}/>);
    fireEvent.click(screen.getByRole('button',{name:'Queue message'}));
    await waitFor(()=>expect(send).toHaveBeenCalledWith('next'));
    expect(interrupt).not.toHaveBeenCalled();
    view.rerender(<T3Conversation {...input} model={{...input.model,items:[],isRunning:true}} queueEnabled draft="after that" onSend={send} onInterrupt={interrupt}/>);
    fireEvent.click(screen.getByRole('button',{name:'Queue message'}));
    await waitFor(()=>expect(send).toHaveBeenCalledWith('after that'));
    expect(screen.getByRole('button',{name:'Stop generation'})).toBeTruthy();
  });
  it('preserves the controlled draft after a rejected send', async () => {
    const input = props({ model: { sessionKey: 'failure', approvals: [], userInputs: [] } });
    const onDraftChange = vi.fn();
    render(<T3Conversation {...input} model={{ ...input.model, items: [], isRunning: false }} draft="keep this" onDraftChange={onDraftChange} onSend={vi.fn().mockRejectedValue(new Error('Connection lost'))} onInterrupt={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Send message' }));
    await waitFor(() => expect(screen.getByRole('alert').textContent).toBe('Connection lost'));
    expect(onDraftChange).not.toHaveBeenCalled();
    expect(screen.getByTestId('composer-editor').getAttribute('contenteditable')).toBe('true');
  });
  it('renders a host dock above the composer inside the same conversation', () => {
    const input = props({ model: { sessionKey: 'dock', approvals: [], userInputs: [] } });
    render(<T3Conversation {...input} model={{ ...input.model, items: [], isRunning: false }}
      dock={<div data-testid="conversation-dock">pad</div>} onSend={vi.fn()} onInterrupt={vi.fn()} />);
    const conversation = document.querySelector('[data-xgc-role="agent-conversation"]');
    const dock = screen.getByTestId('conversation-dock');
    const composer = screen.getByRole('textbox');
    expect(conversation?.contains(dock)).toBe(true);
    expect(conversation?.contains(composer)).toBe(true);
    expect(Boolean(dock.compareDocumentPosition(composer) & Node.DOCUMENT_POSITION_FOLLOWING)).toBe(true);
  });
  it('keeps a dock when sending is unavailable and does not invent a composer', () => {
    const input = props({ model: { sessionKey: 'dock-only', approvals: [], userInputs: [] } });
    render(<T3Conversation {...input} model={{ ...input.model, items: [], isRunning: false }}
      composerEnabled={false} dock={<div>pad</div>} onSend={vi.fn()} onInterrupt={vi.fn()} />);
    expect(screen.getByText('pad')).toBeTruthy();
    expect(screen.queryByRole('textbox')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Send message' })).toBeNull();
  });
  it('limits number shortcuts to the focused question card with multiple pending requests', () => {
    const question = { id: 'check', header: 'Check', question: 'Which?', multiSelect: true, allowCustomAnswer: false, options: [{ value: 'id:a', label: 'Camera' }] };
    const input = props({ model: { sessionKey: 'many', approvals: [], userInputs: [{ requestId: 'first', questions: [question] }, { requestId: 'second', questions: [question] }] } });
    render(<T3PendingRequests {...input} />);
    const options = screen.getAllByRole('button', { name: /Camera/ });
    options[0]?.focus();
    fireEvent.keyDown(options[0]!, { key: '1' });
    expect(options[0]?.getAttribute('aria-pressed')).toBe('true');
    expect(options[1]?.getAttribute('aria-pressed')).toBe('false');
  });
});
