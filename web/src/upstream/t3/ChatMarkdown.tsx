// T3 Code Markdown pipeline retained; IDE asset/shell/directive integrations removed.
// MIT, Copyright (c) 2026 T3 Tools Inc. See UPSTREAM.md.
// Streaming renders reuse the parsed prefix of completed code blocks (PR #11193).
import { memo, useMemo } from 'react';
import ReactMarkdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize from 'rehype-sanitize';
import remarkBreaks from 'remark-breaks';
import remarkGfm from 'remark-gfm';
import { createIncrementalMarkdownPlugin } from './markdown-incremental.js';
const REHYPE_PLUGINS = [rehypeRaw, rehypeSanitize];
const REMARK_PLUGINS = [remarkGfm, remarkBreaks];
// A closed top-level fence is the only incremental parsing boundary (see
// markdown-incremental.ts), so streaming text without fences gains nothing.
const STREAMING_FENCE_PATTERN = /(?:^|\n) {0,3}(?:`{3}|~{3})/;
export const ChatMarkdown = memo(function ChatMarkdown({ text, isStreaming }: { text: string; isStreaming?: boolean }) {
  const incremental = isStreaming === true && STREAMING_FENCE_PATTERN.test(text);
  const remarkPlugins = useMemo(() => (incremental ? [...REMARK_PLUGINS, createIncrementalMarkdownPlugin()] : REMARK_PLUGINS), [incremental]);
  return <div className="chat-markdown min-w-0 text-sm leading-relaxed">
    <ReactMarkdown remarkPlugins={remarkPlugins} rehypePlugins={REHYPE_PLUGINS}
      components={{ a: ({ children, ...props }) => <a {...props} target="_blank" rel="noreferrer noopener">{children}</a> }}
    >{text}</ReactMarkdown>
  </div>;
});
