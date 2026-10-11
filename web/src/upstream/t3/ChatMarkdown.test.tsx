// T3 Code ChatMarkdown.test.tsx (PR #16637 bare anchor placeholders), MIT, Copyright (c) 2026 T3 Tools Inc.
// Adapted to this slice's markdown wrapper; upstream-only renderers (details, citations, file links) are excluded.
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { ChatMarkdown, orderedListGutterStyle } from './ChatMarkdown.js';
import { shouldCollapseUserMessage } from './MessagesTimeline.js';

function parse(text: string) {
  return new DOMParser().parseFromString(renderToStaticMarkup(<ChatMarkdown text={text} />), 'text/html');
}

describe('ChatMarkdown bare anchor placeholders', () => {
  it.each(['<A>', '<a>', '<a >', '<a/>', '<A/>', '<a />'])('keeps an unmatched %s from linking the blocks after it', (token) => {
    const text = `- **"From ${token}"** appears in the header.\n\n- **Tests:** cover inheritance.\n\nThe deferred move continues on B.\n\nSee <a href="https://example.com">the link</a>.`;
    const document = parse(text);
    expect(document.querySelector('strong')?.textContent).toBe(`"From ${token}"`);
    expect([...document.querySelectorAll('a')].map((link) => link.textContent)).toEqual(['the link']);
    expect(document.querySelectorAll('li')).toHaveLength(2);
    expect([...document.querySelectorAll('p')].map((paragraph) => paragraph.textContent)).toContain('The deferred move continues on B.');
  });

  it.each(['</a>  ', '<div>more</div>\n</a>'])('keeps a paired anchor whose closing tag sits in a raw block: %j', (closing) => {
    expect(parse(`See <a>label\n\n${closing}\n\nfinish`).querySelector('p')?.textContent).toBe('See label');
  });

  it('keeps a paired anchor after comment-looking raw text', () => {
    expect(parse('See <a>label<script><!-- </script> --></a>').querySelector('p')?.textContent).toBe('See label -->');
  });

  it.each(['<!-- </a> -->', '<div title="</a>">more</div>', '<script>"</a>"</script>'])('ignores a closing anchor that only looks like one inside %s', (html) => {
    const document = parse(`Before <A>.\n\n${html}\n\nAfter.`);
    expect(document.querySelector('p')?.textContent).toBe('Before <A>.');
    expect(document.querySelectorAll('a')).toHaveLength(0);
  });

  it('keeps paired HTML anchors, markdown links and inline code', () => {
    const document = parse('Bare <a>label</a>, <a id="section"></a>, `<A>`, and [docs](https://example.com).');
    expect([...document.querySelectorAll('a')].map((link) => link.textContent)).toEqual(['label', '', 'docs']);
    expect(document.querySelector('code')?.textContent).toBe('<A>');
  });
});

describe('ChatMarkdown links and lists', () => {
  it('marks a link around an image alone so it gets no text underline', () => {
    const document = parse('[![Chart](https://example.com/chart.png)](https://example.com/report) and [the report](https://example.com/report)');
    const [imageLink, textLink] = [...document.querySelectorAll('a')];
    expect(imageLink?.getAttribute('data-markdown-image-link')).toBe('true');
    expect(textLink?.hasAttribute('data-markdown-image-link')).toBe(false);
  });

  it('widens the ordered list gutter to the widest marker, its period and the space', () => {
    expect(orderedListGutterStyle(9, 1)).toBeUndefined();
    expect(orderedListGutterStyle(10, 1)).toEqual({ '--list-gutter': '4ch' });
    expect(orderedListGutterStyle(3, 98)).toEqual({ '--list-gutter': '5ch' });
    expect(orderedListGutterStyle(3, 'not a number')).toBeUndefined();
    const items = Array.from({ length: 12 }, (_, index) => `${index + 1}. item ${index + 1}`).join('\n');
    expect(parse(items).querySelector('ol')?.getAttribute('style')).toContain('--list-gutter:4ch');
    expect(parse('1. one\n2. two').querySelector('ol')?.hasAttribute('style')).toBe(false);
  });
});

describe('collapsing long user messages', () => {
  const link = (index: number) => `[file${index}.ts](/workspace/projects/example/packages/some/deeply/nested/directory/file${index}.ts)`;

  it('measures a link by its label, not its destination', () => {
    const text = `Compare ${Array.from({ length: 8 }, (_, index) => link(index)).join(', ')} with ![chart](https://example.com/${'a'.repeat(200)}.png).`;
    expect(text.length).toBeGreaterThan(600);
    expect(shouldCollapseUserMessage(text)).toBe(false);
  });

  it('still collapses text that is long once rendered', () => {
    expect(shouldCollapseUserMessage(`${link(1)} ${'More text. '.repeat(60)}`)).toBe(true);
    expect(shouldCollapseUserMessage(Array.from({ length: 9 }, () => 'line').join('\n'))).toBe(true);
    expect(shouldCollapseUserMessage('short')).toBe(false);
  });

  it('does not collapse a message that renders no text', () => {
    expect(shouldCollapseUserMessage('[](https://example.com/a/very/long/destination/that/is/not/shown)')).toBe(false);
    expect(shouldCollapseUserMessage('   ')).toBe(false);
  });
});
