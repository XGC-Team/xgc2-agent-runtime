# T3 Code chat presentation migration

This package directly migrates and adapts a chat presentation slice from [T3 Code](https://github.com/pingdotgg/t3code/tree/bf3be75c400cf605dc0c80de9458458854c84131), an application with an existing Codex app-server integration. The current upstream revision is `bf3be75c400cf605dc0c80de9458458854c84131`; the 2026-09-15 resync from the original `6349a0e68a958cc51b7b5198683c1d1db88b8d28` pin is recorded in `UPSTREAM.json.resync`. T3 Code is MIT licensed, copyright 2026 T3 Tools Inc.; the unchanged license is included in `LICENSE.T3Code`.

`UPSTREAM.json` records migrated source files, immutable URLs, original SHA-256 values, extraction boundaries and XGC-owned integration files. `STYLE_UPSTREAM.json` records the stylesheet extraction; additional upstream constants used by the host adapter are recorded in `UPSTREAM.json.styleIntegration`. This is an adapted source migration, not the complete T3 Code frontend, IDE, server or native protocol implementation.

## Source and adaptation boundaries

The upstream `ComposerSurface`, `ComposerBanner` and button/collapsible/scroll-area/spinner components retain their implementations, slots and utility classes with local imports and attribution. Approval and question components retain their original structure and interactions, with explicit adaptations for the shared native request contract, submission state, parked surfaces and annotation identities.

The larger files are presentation slices. Upstream `MessagesTimeline.tsx` is 3,361 lines, `ComposerPromptEditor.tsx` is 2,134 lines and `ChatMarkdown.tsx` is 3,124 lines. The migrated timeline adapts the list geometry, follow configuration, user bubble and long-message disclosure, assistant treatment and `PlainWorkEntryRow` disclosure into the XGC model. It is not the original complete timeline implementation. The plain-text editor uses Lexical and the upstream IME-safe command and Mac Home/End plugins, with the original contenteditable geometry and XGC-controlled text synchronization.

The markdown component is an adapted minimal wrapper using the upstream rendering dependency subset and `chat-markdown` typography entry. It retains react-markdown, GFM, line breaks, raw HTML parsing and sanitization, plus the upstream incremental streaming parser (`markdown-incremental.ts`, PR #11193): while an assistant message streams and contains a fence opener, completed top-level code blocks become parsing boundaries and only the suffix is re-parsed; definitions, footnotes, CR content and BOMs fall back to a full parse. It does not retain the complete upstream renderer or plugin pipeline: ordinary safe web anchors and default sanitization replace the IDE-specific schema and handlers.

`T3Conversation.tsx`, `NativeConversation.tsx`, `nativePresentation.ts`, presentation types/identities and the style-build adapters are XGC-owned integration code. The multi-request layout, bounded pending queue, controlled drafts, callback capability checks, structured native mapping and domain timeline slots are integration choices. They must not be attributed to an unchanged upstream component.

The optional host-provided `emptyState` React node is an XGC adaptation through the conversation wrapper into the timeline, shown only when there are no native or domain items. It lets products describe a real connection action without inventing a composer. Normal turn completion remains in native state and does not create a synthetic activity row; abnormal and cancelled outcomes retain their operator-facing result.

## Shared architecture and removed coupling

Research and GCS consume one shared conversation and request implementation. Native state, transport and request-answer contracts remain owned by `@xgc2/agent-runtime`; session ownership, idempotency and business operations remain with the host domain. Domain cards enter through custom timeline slots without importing workflow or Research business logic into the shared presentation or adding a second composer.

The slice does not import `ChatView`, the T3 router, `@t3tools/contracts`, `@t3tools/client-runtime`, Effect stores, Electron, Clerk, repository/worktree controls, terminal transport or a second Codex app-server client.

Excluded capabilities are explicit: file/skill/citation inline completion; terminal context tokens; attachment upload and IDE asset resolution (including the upstream large-paste-to-text-attachment folding, PR #11442); artifact directives and local-file links; source/diff navigation; Shiki and code-action extensions; image expansion; timeline minimap, work-group/turn folding, checkpoint maps and per-turn scroll snapshots; implementation/worktree menus; composer collapse/focus-handle infrastructure from ChatView/ChatComposer (upstream PRs #10437, #10444, #10463, #11884 live in that excluded layer). Adding one requires a real shared native capability and an intentional presentation adaptation; its existence in T3 Code does not make it available here.

The upstream client-side queued message store (PR #11673) was reviewed against the host-owned `NativePromptQueue` and is not ported (zustand and T3 contract coupling). Semantics this slice leaves to hosts that upstream now has: one queued message leaves per tool-completion boundary (`queuedAfterToolActivityId` re-anchoring on take); a drain generation so Stop invalidates in-flight queued sends and cannot be followed by a queued message starting a new turn; and `holdAtFront` on send failure so the queue keeps its order and nothing behind the failed message overtakes.

## Native interaction contract

- Approval buttons use only the request's actual option IDs and labels. The upstream default `acceptForSession` policy has been removed. Request titles are preserved, and unknown request kinds are not mislabeled as file changes.
- Request cancellation and whole-turn interruption use distinct callbacks. Cancellation is never encoded as a fabricated approval option. Missing send or interrupt capabilities do not produce placeholder operation controls.
- Submitted controls stay locked pending native resolution. A successful callback is not a terminal native event. Question values, descriptions, multiple selection and allowed free text survive the adapter. Single-choice auto-advance moves only between questions; the final answer requires explicit submission.
- Number shortcuts apply only to the focused question card. Parked surfaces cancel queued auto-advance. Stable `data-xgc-role`/`data-xgc-id` pairs include the surface/session and request/item identity.
- Send capability is independent of the ability to answer a pending request. Controlled drafts survive failure and intervening edits; successful sends clear only an unchanged draft. Host domains own any persistence beyond the mounted shared surface.
- Commands, file changes and MCP data use structured native fields rather than guesses from flattened text. Presentation covers a selected subset: command actions and web-search details may remain in native state without a dedicated rich view. This is not complete T3 Code or Codex protocol UI coverage.
- Missing timestamps are not manufactured. Partial payloads are marked without promising an unavailable complete payload. Submission/resolution events do not reconstruct historical free-form answers.

## Styles and host requirements

The style build keeps selectors inside native CSS `@scope (.xgc-native-chat)`. Tailwind preflight is included **inside that scope**, rather than installed as a global page reset. Custom timeline slots are also inside its preflight and utility scope. This is not Shadow DOM isolation: host inheritance and domain controls rendered within the scope remain part of the integration contract.

The host adapter does not establish a stacking context on each native capsule: a controls capsule's existing portal layer must remain clickable over a later conversation sibling. CSS selector containment stays with `@scope`; increasing the popup z-index inside an isolated ancestor cannot fix sibling occlusion. The adapter also applies the requested black background and white arrow only to the enabled send action, retaining the T3 circular geometry/SVG and the separate disabled, busy and interrupt behavior. Send and interrupt keep their original square dimensions without flex shrinking. Model, effort, permission and the primary action share one row; only the model label uses the remaining flexible width. A narrow controls row uses the existing permission-state icon with its full accessible description, tooltip and menu labels. Its size container is on the row, outside the sibling portal nodes, so menu layering stays intact.

CSS `@property` and keyframes registrations remain global and are renamed with `--xgc-t3-` and `xgc-t3-` prefixes. Theme and property references are namespaced to avoid collisions. Tooltip portals receive a container inside the scoped chat root.

Consumers must explicitly import `@xgc2/agent-runtime/styles.css`, provide XGC semantic tokens and fonts, identify light/dark themes with `data-skin`, and give the conversation a bounded viewport. The stylesheet requires native `@scope` support; it does not provide a fallback for browsers without that capability.

The host adapter includes color and font aliases, scrollbar/radius/glass values and container sizing. It preserves the main upstream geometry without promising pixel equivalence to every upstream theme. In particular, it uses upstream default glass values rather than carrying over the upstream dark overrides. These choices and their source constants are recorded separately from the extracted stylesheet.

## Dependencies, exports and rendering limits

Runtime dependencies are React/React DOM 19 peers; Base UI primitives; LegendList; Lexical and `@lexical/react`; Lucide icons; `class-variance-authority`, `clsx` and `tailwind-merge`; and react-markdown with GFM/breaks and rehype raw/sanitize. Versions are recorded in `package.json` and the lockfile. Tailwind and the selector-scoping tool are build dependencies; neither host needs to adopt Tailwind as its application framework. No assistant-ui package is used.

The package exposes ESM entries for the base API, state, client and React presentation, plus the explicit stylesheet export. `typesVersions` supports older TypeScript resolution modes; it does not provide a CommonJS `require` build. Build output is an artifact of the single shared source owner.

The virtualized timeline renders an unmeasured viewport on the server. Message rows appear after browser measurement, so ESM/SSR import support must not be described as full server rendering of message history.

Localization is partial. `locale='zh'` translates selected native mapper labels; upstream Composer placeholders and request-action controls remain in English. `NativeInput` does not currently consume its locale prop. The interface must not be described as fully localized.

This migration does not apply the upstream monorepo's full pnpm patch set. The viewport integration was based on the published, unpatched LegendList 3.3.5 package. The upstream Web patch addresses `anchoredEndSpace` tail contraction and normalization after inline `calc(...)` scroll-padding writes; this slice uses neither feature. Adopting those features requires evaluating the corresponding patch rather than assuming it is present.


## Provider Settings and current-chat controls

The same pin now supplies `ProviderSettingsForm`, `SettingsRow`, the provider list/editor layout, `ProviderModelPicker` trigger and searchable collection, model-row/sidebar markup, `ComposerControl`, and the permission/effort menu slices. New Base UI wrappers and model search ranking retain upstream implementations with local imports. Provider glyphs are the five upstream brand SVGs; the redundant OpenCode rectangular clip definition is removed to prevent repeated DOM IDs.

`NativeProviderSettings` and `NativeComposerControls` are XGC-owned adapters. Both consumers pass the shared decoded settings document or provider rows; neither consumer maintains a duplicate field or capability mapper. Only enabled, CLI path and actual native model/effort/permission defaults are editable. Login status is observed, never inferred from CLI presence; Refresh probes only the selected provider through a real host callback. No credential, shell, environment, billing or digest form is introduced.

Settings drafts capture the document revision on the first edit. A refresh or another provider update does not silently advance that baseline; conflict/error responses retain the draft. The settings form uses immediate controlled text changes for its explicit Save flow, replacing upstream blur-only commits in that variant. Other form and Base UI presentation slices retain their original treatment.

The model picker preserves the upstream trigger, provider rail, scored search and keyboard isolation but uses a bounded Base UI collection without the upstream LegendList virtualization/favorites/legacy layers. Global IDE scroll-lock effects and all environment/continuation stores are excluded. Permission and effort choices come only from native capability metadata: no prompt injection and no fabricated upstream `auto` mode. Picking an option updates the host's current draft only; the host passes exact choices in `Scope.options` or `sendNativePrompt(id, text, key, options)`.

The optional `NativeConversation.composerControls` slot is shown only with a real composer. Hosts disable controls while a turn is running or a selection handoff is in progress. This slice includes no session list, project navigation, Git, branch, diff, commit or worktree product controls. The inspected `ProviderSetupSection` is specific to Antigravity installation/login and does not apply to the five native providers; it is intentionally not exported as a fake setup experience.

New provider Settings and composer controls accept `locale='en' | 'zh'` (English by default). This localizes the shared wrapper/trigger/status labels only. Native model, effort and permission IDs, labels and descriptions are preserved as supplied by the native capability catalog; they are not translated into invented options.

Model, effort and permission availability are projected independently. An unread model catalog keeps the current/native-default model as a passive value with an explicit unread-catalog status; it does not invent selectable models or remove independently advertised permissions. Disabled/unavailable providers retain their visible current choices with callbacks gated. `identityId` scopes control annotation identities when a host renders multiple real control groups for one provider.
