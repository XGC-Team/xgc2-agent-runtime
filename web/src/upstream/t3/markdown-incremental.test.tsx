// T3 Code markdown-incremental.test.tsx (PR #11193), MIT, Copyright (c) 2026 T3 Tools Inc.
// Adapted to this slice's plugin set (remark-gfm/breaks, rehype raw/sanitize) and vitest;
// upstream-only directives/alert/list-recovery plugins and their assertions are excluded.
import type { Root } from 'mdast';
import { renderToStaticMarkup } from 'react-dom/server';
import ReactMarkdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize from 'rehype-sanitize';
import remarkBreaks from 'remark-breaks';
import remarkGfm from 'remark-gfm';
import type { Plugin } from 'unified';
import { describe, expect, it } from 'vitest';
import { createIncrementalMarkdownPlugin } from './markdown-incremental.js';

function render(source: string, incremental?: Plugin<[], Root>, parsedSources?: string[]) {
  let tree: Root | undefined;
  const observeParsing: Plugin<[], Root> = function () {
    const original = this.parser;
    if (original) {
      this.parser = (text, file) => {
        parsedSources?.push(text);
        return original(text, file);
      };
    }
  };
  const capture: Plugin<[], Root> = () => (root) => {
    tree = structuredClone(root);
  };
  const html = renderToStaticMarkup(
    <ReactMarkdown
      remarkPlugins={[observeParsing, capture, remarkGfm, remarkBreaks, ...(incremental ? [incremental] : [])]}
      rehypePlugins={[rehypeRaw, rehypeSanitize]}
    >
      {source}
    </ReactMarkdown>,
  );
  return { html, tree };
}

const prefix = '# Before\n\n```ts\nconst values = [1, 2];\n```\n\n';

describe('incremental Markdown parsing', () => {
  it('reuses the completed fence prefix and parses only the streaming suffix', () => {
    const source = prefix + '- first block\n\n        tail';
    const incremental = createIncrementalMarkdownPlugin();
    // The incremental parser memoizes across the per-render processors and keeps
    // the first processor's underlying parser, so observe from the first render.
    const parsedSources: string[] = [];
    expect(render(source, incremental, parsedSources)).toEqual(render(source));
    parsedSources.length = 0;
    const next = source + ' more';
    expect(render(next, incremental, parsedSources)).toEqual(render(next));
    expect(parsedSources).not.toContain(next);
    expect(parsedSources.some((text) => text.startsWith('- first block'))).toBe(true);
  });

  it.each([
    'a\n===\n\nb\n---\n',
    '- first\n\n  continued\n\n- next\n',
    '> quoted\n>\n> ```js\n> abc\n> ```\n\nend',
    '<div>\nhello\n\n</div>\n\nend',
    '[ref]\n\n[ref]: /later',
    'a[^x]\n\n[^x]: note',
    'a | b\n--|--\na | b\n',
    '```\na\n```\n\nnext\n\n~~~\nb\n~~~\n\nmore',
    '\n\n\tcode\n\nmore',
    'text <https://example.com> *bold*',
    '\uFEFFtext after a byte-order mark',
  ])('preserves the parse tree, positions, and HTML while streaming %j', (tail) => {
    const source = prefix + tail;
    const incremental = createIncrementalMarkdownPlugin();
    for (let end = 0; end <= source.length; end++) {
      const text = source.slice(0, end);
      expect(render(text, incremental), `prefix ${end}`).toEqual(render(text));
    }
  });

  it.each(['\r\n', '\r'])('preserves partial %j line endings', (newline) => {
    const source = (prefix + 'next\n\n```\nlast\n```\n\nend').replaceAll('\n', newline);
    const incremental = createIncrementalMarkdownPlugin();
    for (let end = 0; end <= source.length; end++) {
      const text = source.slice(0, end);
      expect(render(text, incremental)).toEqual(render(text));
    }
  });

  it('updates earlier references when definitions arrive after the cached prefix', () => {
    const before = '[later] and footnote[^note]\n\n' + prefix;
    const incremental = createIncrementalMarkdownPlugin();
    for (const tail of ['text', '[later]: /target', '[later]: /target\n\n[^note]: a note']) {
      expect(render(before + tail, incremental)).toEqual(render(before + tail));
    }
  });

  it('handles edits, replacements, and repeated renders without leaking transformed nodes', () => {
    const incremental = createIncrementalMarkdownPlugin();
    const documents = [
      prefix + '- first\n - second',
      prefix + 'plain text',
      'replacement without fences',
      prefix.replace('Before', 'Edited') + 'edited prefix',
      prefix + 'plain text',
      prefix + 'plain text',
    ];
    for (const document of documents) {
      expect(render(document, incremental)).toEqual(render(document));
    }
  });

  it('does not freeze unclosed, nested, indented, or mismatched fences', () => {
    const prefixes = [
      '```\nopen\n\n',
      '````\n```\n\n',
      '> ```\n> code\n> ```\n\n',
      '- ```\n  code\n  ```\n\n',
      '    ```\n    code\n    ```\n\n',
      '<script>\n```\ncode\n```\n\n',
    ];
    for (const start of prefixes) {
      const incremental = createIncrementalMarkdownPlugin();
      for (const tail of ['', 'text', '\n```\n', '\n```\n\nnext']) {
        expect(render(start + tail, incremental)).toEqual(render(start + tail));
      }
    }
  });
});
