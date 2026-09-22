// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Upstream ref: bf3be75c400cf605dc0c80de9458458854c84131. See UPSTREAM.md.
"use client";

import { useCommitOnBlur } from "../useCommitOnBlur.js";
import { Input, type InputProps } from "./input.js";

export type DraftInputProps = Omit<InputProps, "value" | "onChange" | "defaultValue"> & {
  readonly value: string;
  readonly onCommit: (next: string) => void;
};

/**
 * Text `<Input>` that buffers keystrokes locally and invokes `onCommit`
 * only when the user finishes editing (blur or Enter). Prevents each
 * keystroke from triggering a settings-wide re-render or a server RPC
 * round-trip, which otherwise makes fields backed by a server-hydrated
 * value feel laggy.
 */
export function DraftInput({ value, onCommit, ...rest }: DraftInputProps) {
  const bag = useCommitOnBlur(value, onCommit);
  return <Input {...rest} {...bag} />;
}
