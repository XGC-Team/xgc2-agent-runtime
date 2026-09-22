import { useT3Identity } from './identity.js';
// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Pinned plain-text Composer slice. Original command/IME and Home/End plugins;
// IDE inline-token, terminal, skill and citation plugins are intentionally excluded.
import { LexicalComposer, type InitialConfigType } from "@lexical/react/LexicalComposer";
import { useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import { LexicalErrorBoundary } from "@lexical/react/LexicalErrorBoundary";
import { HistoryPlugin } from "@lexical/react/LexicalHistoryPlugin";
import { OnChangePlugin } from "@lexical/react/LexicalOnChangePlugin";
import { PlainTextPlugin } from "@lexical/react/LexicalPlainTextPlugin";
import { $getRoot, $createParagraphNode, $createTextNode, $createLineBreakNode,
  $setSelection, $createRangeSelectionFromDom, KEY_ARROW_DOWN_COMMAND,
  KEY_ARROW_UP_COMMAND, KEY_ENTER_COMMAND, KEY_TAB_COMMAND, KEY_DOWN_COMMAND,
  COMMAND_PRIORITY_HIGH, type EditorState, type ElementNode } from "lexical";
import { useCallback, useEffect, useMemo, useRef, type RefObject } from "react";
import { cn, isMacPlatform } from "./utils.js";

function $appendTextWithLineBreaks(parent: ElementNode, text: string): void {
  const lines = text.split("\n");
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index] ?? "";
    if (line.length > 0) {
      parent.append($createTextNode(line));
    }
    if (index < lines.length - 1) {
      parent.append($createLineBreakNode());
    }
  }
}

function ComposerCommandKeyPlugin(props: {
  onCommandKeyDown?: (
    key: "ArrowDown" | "ArrowUp" | "Enter" | "Tab",
    event: KeyboardEvent,
  ) => boolean;
}) {
  const [editor] = useLexicalComposerContext();

  useEffect(() => {
    const handleCommand = (
      key: "ArrowDown" | "ArrowUp" | "Enter" | "Tab",
      event: KeyboardEvent | null,
    ): boolean => {
      if (!props.onCommandKeyDown || !event) {
        return false;
      }

      if (key === "Enter" && (event.isComposing || event.keyCode === 229)) {
        event.stopPropagation();
        return true;
      }

      const handled = props.onCommandKeyDown(key, event);
      if (handled) {
        event.preventDefault();
        event.stopPropagation();
      }
      return handled;
    };

    const unregisterArrowDown = editor.registerCommand(
      KEY_ARROW_DOWN_COMMAND,
      (event) => handleCommand("ArrowDown", event),
      COMMAND_PRIORITY_HIGH,
    );
    const unregisterArrowUp = editor.registerCommand(
      KEY_ARROW_UP_COMMAND,
      (event) => handleCommand("ArrowUp", event),
      COMMAND_PRIORITY_HIGH,
    );
    const unregisterEnter = editor.registerCommand(
      KEY_ENTER_COMMAND,
      (event) => handleCommand("Enter", event),
      COMMAND_PRIORITY_HIGH,
    );
    const unregisterTab = editor.registerCommand(
      KEY_TAB_COMMAND,
      (event) => handleCommand("Tab", event),
      COMMAND_PRIORITY_HIGH,
    );

    return () => {
      unregisterArrowDown();
      unregisterArrowUp();
      unregisterEnter();
      unregisterTab();
    };
  }, [editor, props]);

  return null;
}

function ComposerHomeEndKeyPlugin() {
  const [editor] = useLexicalComposerContext();

  useEffect(() => {
    return editor.registerCommand(
      KEY_DOWN_COMMAND,
      (event) => {
        if (!isMacPlatform(navigator.platform)) {
          return false;
        }
        if (event.key !== "Home" && event.key !== "End") {
          return false;
        }
        if (event.altKey || event.metaKey || event.ctrlKey || event.isComposing) {
          return false;
        }

        const rootElement = editor.getRootElement();
        const selection = window.getSelection();
        const anchorNode = selection?.anchorNode;
        if (!rootElement || !selection || !anchorNode || !rootElement.contains(anchorNode)) {
          return false;
        }
        if (selection.rangeCount === 0 || typeof selection.modify !== "function") {
          return false;
        }

        event.preventDefault();
        event.stopPropagation();

        selection.modify(
          event.shiftKey ? "extend" : "move",
          event.key === "Home" ? "backward" : "forward",
          "lineboundary",
        );
        editor.update(() => {
          $setSelection($createRangeSelectionFromDom(selection, editor));
        });
        return true;
      },
      COMMAND_PRIORITY_HIGH,
    );
  }, [editor]);

  return null;
}


export interface ComposerPromptEditorProps {
  value: string;
  identityId?: string;
  disabled: boolean;
  placeholder: string;
  onChange: (value: string) => void;
  onCommandKeyDown?: (key: "ArrowDown" | "ArrowUp" | "Enter" | "Tab", event: KeyboardEvent) => boolean;
  editorRef?: RefObject<HTMLDivElement | null>;
  className?: string;
}

function replacePlainText(value: string) {
  const root = $getRoot();
  root.clear();
  const paragraph = $createParagraphNode();
  $appendTextWithLineBreaks(paragraph, value);
  root.append(paragraph);
}

/** Host-controlled text synchronization; never rewrites a local keystroke. */
function ControlledTextPlugin({ value, disabled }: Pick<ComposerPromptEditorProps, "value" | "disabled">) {
  const [editor] = useLexicalComposerContext();
  useEffect(() => { editor.setEditable(!disabled); }, [disabled, editor]);
  useEffect(() => {
    const current = editor.getEditorState().read(() => $getRoot().getTextContent());
    if (current === value) return;
    const focused = editor.getRootElement() === document.activeElement;
    editor.update(() => {
      replacePlainText(value);
      if (focused) $getRoot().selectEnd();
      else $setSelection(null);
    });
  }, [editor, value]);
  return null;
}

function ComposerPromptEditorInner(props: ComposerPromptEditorProps) {
  const identity = useT3Identity();
  const handleEditorChange = useCallback((state: EditorState) => {
    state.read(() => props.onChange($getRoot().getTextContent()));
  }, [props.onChange]);
  return (
    <div className="relative [font-family:var(--font-composer,var(--font-sans))] [font-size:var(--font-size-prompt,0.875rem)] [@media(max-width:39.999rem)_and_(pointer:coarse)]:[font-size:max(var(--font-size-prompt,1rem),16px)]"
      data-xgc-role="agent-composer-input" data-xgc-id={`${identity}:${props.identityId ?? "composer"}`}>
      <PlainTextPlugin
        contentEditable={<ContentEditable
          ref={props.editorRef}
          className={cn("block max-h-50 min-h-17.5 w-full overflow-y-auto whitespace-pre-wrap wrap-break-word bg-transparent leading-relaxed text-foreground focus:outline-none", props.className)}
          data-testid="composer-editor"
          data-xgc-role="agent-composer-input-editor" data-xgc-id={`${identity}:${props.identityId ?? "composer"}`}
          aria-label="Message the agent"
          aria-placeholder={props.placeholder}
          placeholder={<span />}
        />}
        placeholder={<div className="pointer-events-none absolute inset-0 leading-relaxed text-placeholder/75">{props.placeholder}</div>}
        ErrorBoundary={LexicalErrorBoundary}
      />
      <ControlledTextPlugin value={props.value} disabled={props.disabled} />
      <OnChangePlugin onChange={handleEditorChange} ignoreSelectionChange />
      <ComposerCommandKeyPlugin {...(props.onCommandKeyDown ? { onCommandKeyDown: props.onCommandKeyDown } : {})} />
      <ComposerHomeEndKeyPlugin />
      <HistoryPlugin />
    </div>
  );
}

export function ComposerPromptEditor(props: ComposerPromptEditorProps) {
  const initialValue = useRef(props.value);
  const config = useMemo<InitialConfigType>(() => ({
    namespace: "t3tools-composer-editor",
    editable: !props.disabled,
    editorState: () => replacePlainText(initialValue.current),
    onError: (error) => { throw error; },
  }), []);
  return <LexicalComposer initialConfig={config}><ComposerPromptEditorInner {...props} /></LexicalComposer>;
}
