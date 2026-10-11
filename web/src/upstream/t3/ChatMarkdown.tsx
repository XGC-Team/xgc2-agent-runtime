// T3 Code Markdown pipeline retained; IDE asset/shell/directive integrations removed.
// MIT, Copyright (c) 2026 T3 Tools Inc. See UPSTREAM.md.
// Streaming renders reuse the parsed prefix of completed code blocks (PR #11193).
// Unmatched `<A>` placeholders stay text (PR #16637); wide ordered-list markers keep their gutter (PR #16523).
import { memo, useMemo } from 'react';
import ReactMarkdown, { type Components } from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize from 'rehype-sanitize';
import remarkBreaks from 'remark-breaks';
import remarkGfm from 'remark-gfm';
import { createIncrementalMarkdownPlugin } from './markdown-incremental.js';

type HastNode = { type?: string; value?: string; tagName?: string; children?: HastNode[] };

/** Keep unmatched inline `<A>` placeholders from opening an HTML link over later blocks. */
function rehypePreserveBareAnchorPlaceholders() {
  return (tree: HastNode) => {
    const anchors: Array<HastNode | null> = [];
    let rawTextTag: string | undefined;
    const visit = (node: HastNode) => {
      if (node.type === 'raw' && typeof node.value === 'string') {
        // Raw blocks can contain several tags. Consume whole tags, quoted attributes,
        // and comments so text resembling a closing anchor cannot pair a placeholder.
        const tags = /<!--[\s\S]*?(?:-->|$)|<\/?[A-Za-z](?:[^"'<>]|"[^"]*"|'[^']*')*>/g;
        let offset = 0;
        while (rawTextTag !== 'plaintext') {
          // Raw text ends at its closing tag even inside comment-looking text.
          const matcher = rawTextTag ? new RegExp(`</${rawTextTag}\\s*>`, 'gi') : tags;
          matcher.lastIndex = offset;
          const match = matcher.exec(node.value);
          if (!match) break;
          const [tag] = match;
          offset = matcher.lastIndex;
          if (rawTextTag) {
            rawTextTag = undefined;
            continue;
          }
          if (tag.startsWith('<!--')) continue;
          const closing = /^<\/([a-z]+)\s*>$/i.exec(tag)?.[1]?.toLowerCase();
          const opening = /^<([a-z]+)(?:\s|\/?>)/i.exec(tag)?.[1]?.toLowerCase();
          if (opening && /^(?:script|style|textarea|title|xmp|iframe|noembed|noframes|plaintext)$/.test(opening)) {
            rawTextTag = opening;
          } else if (opening === 'a') {
            anchors.push(node.value === tag && /^<a\s*\/?>$/i.test(tag) ? node : null);
          } else if (closing === 'a') {
            anchors.pop();
          }
        }
      }
      node.children?.forEach(visit);
    };
    visit(tree);
    for (const anchor of anchors) {
      if (anchor) anchor.type = 'text';
    }
  };
}

/**
 * The default `1.25rem` marker gutter (`.chat-markdown ol`) fits one-character
 * markers. Wider markers can extend past it and get clipped by a collapsed
 * message's overflow. Widen the gutter to fit the widest marker, including a
 * negative marker's minus sign, the period, and the trailing space.
 */
export function orderedListGutterStyle(itemCount: number, start: unknown): { '--list-gutter': string } | undefined {
  const parsedStart = Number.parseInt(String(start ?? 1), 10);
  const firstNumber = Number.isNaN(parsedStart) ? 1 : parsedStart;
  const lastNumber = firstNumber + Math.max(itemCount - 1, 0);
  const markerWidth = Math.max(String(firstNumber).length, String(lastNumber).length);
  if (markerWidth <= 1) return undefined;
  return { '--list-gutter': `${markerWidth + 2}ch` };
}

function hastHasText(node: unknown): boolean {
  if (!node || typeof node !== 'object') return false;
  if ('type' in node && node.type === 'text' && 'value' in node && typeof node.value === 'string' && node.value.trim().length > 0) return true;
  return 'children' in node && Array.isArray(node.children) && node.children.some(hastHasText);
}

const REHYPE_PLUGINS = [rehypePreserveBareAnchorPlaceholders, rehypeRaw, rehypeSanitize];
const REMARK_PLUGINS = [remarkGfm, remarkBreaks];
// A closed top-level fence is the only incremental parsing boundary (see
// markdown-incremental.ts), so streaming text without fences gains nothing.
const STREAMING_FENCE_PATTERN = /(?:^|\n) {0,3}(?:`{3}|~{3})/;
const COMPONENTS = {
  // A link around an image alone has no text to underline (PR #17728).
  a: ({ node, children, ...props }) => <a {...props} data-markdown-image-link={hastHasText(node) ? undefined : true} target="_blank" rel="noreferrer noopener">{children}</a>,
  ol: ({ node, start, style, ...props }) => {
    const itemCount = node?.children?.filter(child => child.type === 'element' && child.tagName === 'li').length ?? 0;
    const gutter = orderedListGutterStyle(itemCount, start);
    return <ol {...props} start={start} style={gutter ? { ...style, ...gutter } : style} />;
  },
} satisfies Components;
export const ChatMarkdown = memo(function ChatMarkdown({ text, isStreaming }: { text: string; isStreaming?: boolean }) {
  const incremental = isStreaming === true && STREAMING_FENCE_PATTERN.test(text);
  const remarkPlugins = useMemo(() => (incremental ? [...REMARK_PLUGINS, createIncrementalMarkdownPlugin()] : REMARK_PLUGINS), [incremental]);
  return <div className="chat-markdown min-w-0 text-sm leading-relaxed">
    <ReactMarkdown remarkPlugins={remarkPlugins} rehypePlugins={REHYPE_PLUGINS} components={COMPONENTS}>{text}</ReactMarkdown>
  </div>;
});
