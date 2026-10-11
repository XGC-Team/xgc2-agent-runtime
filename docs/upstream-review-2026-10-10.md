# T3 Code upstream review, 2026-10-10

Reviewed repository: [pingdotgg/t3code](https://github.com/pingdotgg/t3code).
Range: `bf3be75c400cf605dc0c80de9458458854c84131..477282263a9115f911380b5aec7d4973267e4edc` (previous pin excluded, review pin included). The table contains all **1,253 commits**, in `git log --reverse` order.

The accepted issue #224 decisions are retained. A full, non-partial local clone supplies the objects; all review/hash reads use `GIT_NO_LAZY_FETCH=1`. The complete per-commit name-status diffs establish scope. The remaining 94 commits touching adopted source/provider areas were checked through their production diffs; excluded refactors/package moves, test fixtures and platform features are classified by their paths and changed hunks. The other 911 commits touch excluded areas and are classified by their path diffs. No lazy fetching or partial-clone dump helper was used.

**Ported** means the applicable part is already implemented on this branch; other parts of the same upstream commit remain excluded. **Already covered** means the native Go driver, broker or shared presenter already supplies the relevant behavior. **Not applicable** means the changed feature/implementation is outside the decided slice. This review adds documentation and provenance; it does not change runtime behavior or APIs.

| Classification | Commits |
|---|---:|
| Ported | 29 |
| Already covered | 27 |
| Not applicable | 1197 |
| Total | 1,253 |

## Adopted scope and extraction boundaries

Native ports cover Codex turn-permissions requests and archived-thread resume; Claude control-channel Stop/MCP registration and skill titles; linear ACP tool output/progress; Grok auto-mode and ACP metadata; and the retained Claude model-manifest data. Stop-hold semantics originate in the earlier upstream client queue store (#11673) and now belong to the Go broker. That earlier commit is outside this review range; the T3 client store is not copied.

Presentation ports cover question-scoped auto-advance and non-secret draft restoration, visible link-label message measurement, literal unmatched `<A>` placeholders, ordered-list gutters, image-link underline exclusion, interrupted tools, switch checked-state accessibility, Stop hover geometry, the copy tooltip, the OpenAI glyph and provider-settings container queries. The Tiptap migration, application shell, server orchestration, unsupported providers and excluded composer/model-picker layers remain excluded.

The [web provenance](../web/UPSTREAM.json) identifies upstream blobs at the review pin and preserves prior extraction hashes in `derivedFrom` when changed. Retained source-section ranges refer to that prior extraction. The [style provenance](../web/STYLE_UPSTREAM.json) retains the base ref/ranges and records the one hand-ported image-link rule separately: `index.css` grew from 2,122 to 2,369 lines without re-extraction. Its old hash was stale from `6349a0e`; it now verifies the declared `bf3be75` source. The [provider provenance](../upstream/t3/PROVENANCE.json) records verified successor sources and retains the removed Cursor ACP contracts at the previous pin because the Cursor SDK migration is excluded.

## Reasons for excluded changes

Repeated exclusion reasons are printed once here and referenced by ID in the commit table. A mixed commit may use the reason for its excluded application layers; its applicable part, if any, takes precedence in the classification.

| Reason | Explanation |
|---|---|
| N01 | Mobile application/navigation, dictation and device UI are excluded. |
| N02 | Web shell, navigation, IDE panels and non-adopted composer/renderer/UI layers are excluded. |
| N03 | Cross-application/client/server integration changes excluded shell, mobile, desktop or host features. |
| N04 | Provider installation, compatibility and updates belong to the host. |
| N05 | Model picker built on instances, favorites, multi-model, update notices, virtualization |
| N06 | T3 server orchestration, services, hosting/auth, observability and application state are not copied. |
| N07 | T3 contracts and client-runtime stores/transport are not imported. |
| N08 | Vendored dependencies, patches and their tooling are outside the adopted package. |
| N09 | Desktop/Electron/window, platform packaging and update behavior are excluded. |
| N10 | Queue or steer and the client queue store (the queue is the broker's) |
| N11 | Source-control, pull-request and repository services are outside the conversation/runtime slice. |
| N12 | Monorepo package infrastructure, lint plugins and tooling are not adopted. |
| N13 | Antigravity, Pi and Muse are unsupported providers. |
| N14 | Claude rewind/fork and transcript turn remapping are not exposed. |
| N15 | Release/configuration, documentation, CI, lint and build tooling are outside the adopted runtime. |
| N16 | Application shell (worktrees, terminal, pull requests, diff, devices, browser, settings pages) |
| N17 | Antigravity installation/sign-in is an unsupported provider surface. |
| N18 | Work-log grouping, timestamps, thinking rows, live rows (excluded slice) |
| N19 | Marketing site/content and its branding are excluded. |
| N20 | Slash commands, skills/goals, webhooks and browser tools are outside the adopted protocol slice. |
| N21 | Provider usage and limits are outside the native conversation contract. |
| N22 | ChatComposer layers the slice excludes (attachments, suggestions, collapse, citations, focus) |
| N23 | Markdown renderer features of the IDE shell |
| N24 | Approval grouping by T3-specific decision names is excluded. |
| N25 | Tiptap composer (skipped by the issue) |
| N26 | Automatic machine/project storage cleanup belongs to the host/storage product. |
| N27 | Shared T3 infrastructure outside the retained search-ranking slice is not imported. |
| N28 | Style and token plumbing, no behavior change, no browser to verify |
| N29 | The prompt contract has no attachments or image paths. |
| N30 | UI primitive API changes |
| N31 | Effect/TypeScript dependencies, lint rules, diagnostics and protocol-generator typing are upstream tooling. |
| N32 | Vendored dependency source, documentation and tests are not adopted. |
| N33 | Hosting/relay, remote access and infrastructure belong to the host. |
| N34 | The ACP client advertises no elicitation capability. |
| N35 | No provider homePath override or T3 continuation-group/cache identity is used. |
| N36 | Native stderr is discarded by design. |
| N37 | Provider sign-in and credential bindings belong to native clients/hosts. |
| N38 | Package moves and tooling; adopted provider glyphs were compared and only the separately ported OpenAI glyph changed. |
| N39 | Unsupported providers |
| N40 | T3 collaboration/browser/device instructions and additionalContext policy are not adopted. |
| N41 | Native credentials/keyrings are not read; Cursor SDK authentication is excluded. |
| N42 | Unused item arrays are internal to the excluded Claude SDK session state. |
| N43 | Pull-request synchronization/watch instructions belong to the application shell. |
| N44 | T3 provider-session directories, settled rows and relay/reaper state are not copied. |
| N45 | OpenCode uses native ACP; no HTTP serve readiness parser is used. |
| N46 | Remote, accounts, permission scopes |
| N47 | Account plan/service-tier labels and Ultrafast mode are not exposed. |
| N48 | Upstream test fixtures and install advisories are not adopted runtime behavior. |
| N49 | Answered-question work rows are excluded. |
| N50 | Fast mode is excluded. |
| N51 | Subagent model/effort lineage and SDK child-thread projections are outside the shared contract. |
| N52 | Command-palette restart and provider-registry workspace caches belong to the application shell. |
| N53 | T3 tool-source/icon enrichment and MCP presentation graph are excluded. |
| N54 | Long-lived SDK queries, background wake/continuation turns and active steering are not used by the per-turn native CLI driver. |
| N55 | Pi and the generic ACP Registry are unsupported provider settings. |
| N56 | Windows SDK executable resolution is outside the Linux native-CLI deployment. |
| N57 | Agent platform features |
| N58 | T3 delegated-task APIs/instructions are not part of this runtime. |
| N59 | Hover tooltip dismissal on scroll is outside this adaptation. |
| N60 | Effect maintenance locks, cached DPoP/maintenance reads and file-cache/id writes are not used. |
| N61 | No T3 settle/unsubscribe or background-terminal termination API is exposed. |
| N62 | Provider packages, module/service names and Effect factories are upstream implementation structure. |
| N63 | T3 MCP tool caller declarations belong to its excluded server tool surface. |
| N64 | Legacy model grouping and Antigravity discovery are outside the local Claude extraction/native catalogs. |
| N65 | Unsupported provider drivers/SDKs and their tests are excluded. |
| N66 | Usage/transcript/account readers are not implemented by this conversation runtime. |
| N67 | T3 Codex shadow-home maintenance/symlink policy is not used. |
| N68 | Excluded banner layout. |
| N69 | The slice does not commit React state from resize observers. |
| N70 | Systemd agent scopes and OOM isolation belong to the host operating-system policy. |
| N71 | Wrapped-gateway cooldown parsing serves automatic usage-limit resume; this runtime requires explicit resume and does not retry model calls. |
| N72 | OpenCode SDK session-title synchronization and SDK subagent reports are excluded; the native ACP path is retained. |
| N73 | Independent Claude subagent Stop is outside the whole-turn Stop API. |
| N74 | TaskCreate/TaskUpdate projection serves the excluded T3 tasks drawer; native tools remain timeline items. |

## Per-commit classification

Subjects retain the English upstream descriptions with implementation version labels omitted; each commit link identifies the original.

| Commit | Subject | Classification | Reason |
|---|---|---|---|
| [e33b710d5f11](https://github.com/pingdotgg/t3code/commit/e33b710d5f1147993e52076a9e033d97ef4106ad) | fix(mobile): match command palette colors to sheets (#11861) | Not applicable | N01 |
| [ae53072af056](https://github.com/pingdotgg/t3code/commit/ae53072af0560c0606202b87b5fdb7e52d48ac5c) | fix(web): keep the composer ready during background worktree setup (#11883) | Not applicable | N02 |
| [c1b2210411da](https://github.com/pingdotgg/t3code/commit/c1b2210411da79350a3f9f5a65a4a64971e4acad) | fix(desktop): keep the sidebar brand and window buttons aligned (#11906) | Not applicable | N03 |
| [9a6b57be2ae6](https://github.com/pingdotgg/t3code/commit/9a6b57be2ae601e6969669a8ad276f974b2a5978) | fix(web): drop the filled well behind the sidebar header buttons (#11660) | Not applicable | N02 |
| [2a264adc6f6c](https://github.com/pingdotgg/t3code/commit/2a264adc6f6c65f98d4273acaa1af567eca83f2a) | fix(server): explain how to configure a missing Codex executable (#11345) | Not applicable | N04 |
| [d1790aa14639](https://github.com/pingdotgg/t3code/commit/d1790aa14639a0361c02fbc5c326d0aa76f95a83) | fix(mobile): add missing thread rename action (#11503) | Not applicable | N01 |
| [caf8b5d79890](https://github.com/pingdotgg/t3code/commit/caf8b5d798909780a7b197cb264e782888b919c1) | fix(mobile): wait for thread deep link hydration (#11502) | Not applicable | N01 |
| [d07ffbe351a1](https://github.com/pingdotgg/t3code/commit/d07ffbe351a10e1001f6d8eafa5b7533b8bd7e23) | fix(mobile): keep iOS chat rows aligned after measurement (#11813) | Not applicable | N01 |
| [438465d6b7dd](https://github.com/pingdotgg/t3code/commit/438465d6b7dd2e50dd38862a0db1252d2d72ecfc) | feat: add customizable soft-tint project monograms (#11845) | Not applicable | N03 |
| [24b711b7f9cc](https://github.com/pingdotgg/t3code/commit/24b711b7f9ccba03518551991e466e2e4f8ace5a) | fix: multiple UI and server bug fixes (#11593) | Not applicable | N05 |
| [7235701de052](https://github.com/pingdotgg/t3code/commit/7235701de052658cd78cb029b0df71af4b8dc21f) | fix(server): release preview hosts after unanswered requests (#11381) | Not applicable | N06 |
| [3efdcc5296f1](https://github.com/pingdotgg/t3code/commit/3efdcc5296f1754e0f3bf7fee5fc2ada510e0438) | Preserve diff tree order and collapsed folders (#11931) | Not applicable | N02 |
| [a5da32750125](https://github.com/pingdotgg/t3code/commit/a5da32750125c3bedb959bf331e5e49e95b95e5d) | fix(client-runtime): preserve cached turns and older-page loading (#8309) | Not applicable | N07 |
| [844203d4fe5f](https://github.com/pingdotgg/t3code/commit/844203d4fe5f82b4bfd24ff31655307bd201893b) | chore(deps): bump Clerk stack to latest stable versions (#11956) | Not applicable | N08 |
| [b18a560bbac7](https://github.com/pingdotgg/t3code/commit/b18a560bbac70c7166677d7f3bb51ecaf42acf08) | fix(mobile): update Reanimated and Worklets (#11957) | Not applicable | N08 |
| [96bddf812580](https://github.com/pingdotgg/t3code/commit/96bddf81258090690bfcf4e6a6414aa0dcf1910a) | fix(desktop): paste as text no longer doubles the pasted text (#11958) | Not applicable | N09 |
| [719a76ca1dbf](https://github.com/pingdotgg/t3code/commit/719a76ca1dbf5490f1aa33ffb9966301e02be9a9) | feat(web): choose queue or steer for follow-up messages (#11964) | Not applicable | N10 |
| [e6ae764f47d6](https://github.com/pingdotgg/t3code/commit/e6ae764f47d678b96f4a1b2c8c76eedf1a342e26) | feat(mobile): add  preview store builds (#11966) | Not applicable | N01 |
| [2c16c1d26417](https://github.com/pingdotgg/t3code/commit/2c16c1d264173d86fc354e9a4c0c78cd42e514b7) | fix(mobile): block incompatible server connections (#11974) | Not applicable | N03 |
| [f0a0ead946ca](https://github.com/pingdotgg/t3code/commit/f0a0ead946ca3f3f310460a91b7dabb0e3fe4834) | fix(web): keep PR controls readable in narrow panels (#11962) | Not applicable | N02 |
| [b84f63bb13f1](https://github.com/pingdotgg/t3code/commit/b84f63bb13f13b1d0ee8537c4d244d6f1efdb987) | fix(server): block updates under legacy service launchers (#11940) | Not applicable | N06 |
| [f4600d77dd7c](https://github.com/pingdotgg/t3code/commit/f4600d77dd7c2fa9f10e8f4500882e427fcc7e26) | fix: reduce GitHub quota use with sharing enabled (#11888) | Not applicable | N11 |
| [87a12b53fdff](https://github.com/pingdotgg/t3code/commit/87a12b53fdff7e2e0318af3edea54005557cea56) | fix(usage): refresh limits when the tab opens (#11928) | Not applicable | N03 |
| [6f7aaffe26f0](https://github.com/pingdotgg/t3code/commit/6f7aaffe26f0e6ae3d47b0ae512b322f890e0f43) | fix(contracts): avoid Intl.Segmenter in monogram validation (Hermes crash) (#11984) | Not applicable | N07 |
| [37a8ab2b29df](https://github.com/pingdotgg/t3code/commit/37a8ab2b29dfa33b4e20ad709a9860f0da7b7eb2) | feat(lint): extend Hermes API bans with a configurable API list (#11982) | Not applicable | N12 |
| [b12c92f695a6](https://github.com/pingdotgg/t3code/commit/b12c92f695a6b12116fb2cda40d610bdbe2a9566) | fix(server): reuse Git index metadata during checkpoint capture (#10792) | Not applicable | N06 |
| [7a368fe7c45a](https://github.com/pingdotgg/t3code/commit/7a368fe7c45a45c284f1b064e440a2d696452b07) | refactor: give project monograms their own icon variant (#11993) | Not applicable | N03 |
| [935c55b3778f](https://github.com/pingdotgg/t3code/commit/935c55b3778fdeae0e25b250ce2a9fa7e79c0327) | fix(clients): disable incompatible environments during discovery (#11990) | Not applicable | N03 |
| [8c18b5bb21a5](https://github.com/pingdotgg/t3code/commit/8c18b5bb21a5349fbadba8c77456ea34c41b7339) | fix(antigravity): stop health checks from filling the disk with _MEI folders (#12008) | Not applicable | N13 |
| [0ec2b08a9a1d](https://github.com/pingdotgg/t3code/commit/0ec2b08a9a1db6fa35fab8d505b5bdc2d3709a8a) | fix(mobile): bare t3code:// links no longer reset navigation to Home (#12002) | Not applicable | N01 |
| [eed974c122e1](https://github.com/pingdotgg/t3code/commit/eed974c122e192de45b54b7dac7ca1a51448c08f) | fix(server): keep Claude rewind when fork history length changes (#11954) | Not applicable | N14 |
| [f8500f112716](https://github.com/pingdotgg/t3code/commit/f8500f11271622ed82df7f6c1d7f73eddb4f43e6) | fix(mobile): use native toolbar search for licenses (#12011) | Not applicable | N01 |
| [0bf2d6b0109f](https://github.com/pingdotgg/t3code/commit/0bf2d6b0109f243add8fb6d727e153e97dd51802) | chore(release): prepare v0.0.41 | Not applicable | N15 |
| [47ef0177dc18](https://github.com/pingdotgg/t3code/commit/47ef0177dc18d328e91c64a1641cfe1f37298160) | Revert "chore(release): prepare v0.0.41" | Not applicable | N15 |
| [3060cc461458](https://github.com/pingdotgg/t3code/commit/3060cc461458e66842dfb656d979c4acca4dfb20) | fix(release): restrict stable channels to main | Not applicable | N15 |
| [53612cc040ad](https://github.com/pingdotgg/t3code/commit/53612cc040ad828566f958a275e3d3bad01c5033) | fix(server): detect unsupported legacy Android command-line tools (#12017) | Not applicable | N06 |
| [fe896e6a90dd](https://github.com/pingdotgg/t3code/commit/fe896e6a90dd2a91e9499c098bb621568a3f341f) | chore(release): prepare v0.0.42 | Not applicable | N15 |
| [0f5a1513e09f](https://github.com/pingdotgg/t3code/commit/0f5a1513e09fb41e0d0f86157aff64cc3657975d) | fix(web): stop the worktree setup card from flashing and shifting (#12015) | Not applicable | N16 |
| [c1b2ed6504bb](https://github.com/pingdotgg/t3code/commit/c1b2ed6504bbd57440d03c6a5af2736f33be7c65) | fix(clients): show unsupported environments as neutral rows with their machine icon (#12026) | Not applicable | N03 |
| [6ee03240bccd](https://github.com/pingdotgg/t3code/commit/6ee03240bccd9287a8c2246e9547697aa5857998) | fix(clients): hold the discovered machine icon across relay refreshes (#12030) | Not applicable | N03 |
| [bf55408a5cdd](https://github.com/pingdotgg/t3code/commit/bf55408a5cddbb0f3f7f50d538398e315b5407d0) | fix(server): resolve Node for standalone helper scripts (#12033) | Not applicable | N17 |
| [b900fc94ebe6](https://github.com/pingdotgg/t3code/commit/b900fc94ebe630155783c07324c58af7e6387a03) | feat(web): reveal timestamps on tool rows and turn folds (#8641) | Not applicable | N18 |
| [ccf220be205f](https://github.com/pingdotgg/t3code/commit/ccf220be205f0e509021dbc8cbda90daa638e20d) | docs: make the standalone installer the primary way to get the CLI (#11696) | Not applicable | N19 |
| [f45c6b4a7559](https://github.com/pingdotgg/t3code/commit/f45c6b4a755905485431872f4a4188de3ed05974) | fix(web): submit PR comments with Cmd/Ctrl+Enter (#11994) | Not applicable | N02 |
| [e8ca3a82806d](https://github.com/pingdotgg/t3code/commit/e8ca3a82806dff84438d5aba66b693468f7ca426) | refactor(web): centralize pull request icon state presentation (#11144) | Not applicable | N16 |
| [0b83045d00d8](https://github.com/pingdotgg/t3code/commit/0b83045d00d88d0a0b0866fe4942a50a41a08471) | feat(providers): expose native slash commands across clients (#11519) | Not applicable | N20 |
| [f4ef5155fb8a](https://github.com/pingdotgg/t3code/commit/f4ef5155fb8aab7776a41fc8ec73d5367f53ba1f) | feat(web): add send shortcut and follow-up controls (#12075) | Not applicable | N10 |
| [052c7ae53e46](https://github.com/pingdotgg/t3code/commit/052c7ae53e46c705d5fd5dfa995275bc3100744e) | feat(chat): show provider thinking traces (#11784) | Not applicable | N18 |
| [eff44be433dc](https://github.com/pingdotgg/t3code/commit/eff44be433dc9683da7764a20334598e47b127b2) | feat(usage): show OpenCode Go, Cursor, and Grok subscription limits (#12115) | Not applicable | N21 |
| [d1a644897119](https://github.com/pingdotgg/t3code/commit/d1a644897119300688cbd8bf26e0b64aeb92c382) | fix(web): dropped folders become path chips on the local environment and are refused on remote ones (#12001) | Not applicable | N22 |
| [384de0858d1f](https://github.com/pingdotgg/t3code/commit/384de0858d1f59ca2ad7c660544a5290fd9138cb) | fix(web): adapt provider settings to available content width (#12138) | Ported | Base provider-settings layout on container width (#12138). |
| [32e8b2584556](https://github.com/pingdotgg/t3code/commit/32e8b2584556c0c55ce0d1d8f72b506cb771d60d) | fix(web): show private repository media in pull request tabs (#11706) | Not applicable | N23 |
| [aae361c4ef64](https://github.com/pingdotgg/t3code/commit/aae361c4ef6478e4dcf970d574f3fd38d908e6b9) | fix(review): show complete counts and load large diffs progressively (#10822) | Not applicable | N03 |
| [52ad70ec43b5](https://github.com/pingdotgg/t3code/commit/52ad70ec43b5dcfcdc72ac79c0964da8e4d163db) | fix(web): prioritize linked pull requests over automatic diffs (#12142) | Not applicable | N02 |
| [f1b497a5552a](https://github.com/pingdotgg/t3code/commit/f1b497a5552abf4616a7477b730b78a1274f3db4) | feat(cli): show installer and update download progress (#12044) | Not applicable | N06 |
| [01e64193d9d2](https://github.com/pingdotgg/t3code/commit/01e64193d9d2abfae8f7b3d20f66824e38397bd6) | fix(web): simplify agent approval prompts (#12082) | Not applicable | N24 |
| [dda44ed33280](https://github.com/pingdotgg/t3code/commit/dda44ed33280106aa46612fd9e1f1475bb5a0712) | fix(web): show tooltips for composer environment and workspace controls (#11787) | Not applicable | N02 |
| [22539f2a4289](https://github.com/pingdotgg/t3code/commit/22539f2a428947ab218589c984b2af44e434012e) | fix(chat): group thoughts into the changing tool activity line (#12147) | Not applicable | N18 |
| [ed123897943e](https://github.com/pingdotgg/t3code/commit/ed123897943e9175d4a0e92a5a8a09d200573336) | fix(web): keep tool timestamps before disclosure chevrons (#12152) | Not applicable | N18 |
| [394af73e0223](https://github.com/pingdotgg/t3code/commit/394af73e02238e551db14f44d57f1469dd2942aa) | fix(web): default diff panel to working tree (#12139) | Not applicable | N02 |
| [e106fe95cc81](https://github.com/pingdotgg/t3code/commit/e106fe95cc814797cfb3dad6d78c062e96d0060f) | design(mobile): unify Android Material layouts and native controls (#11841) | Not applicable | N01 |
| [9686cd9afe20](https://github.com/pingdotgg/t3code/commit/9686cd9afe20d13fcb850199f9b9849a8276737a) | feat(web): choose themes from chat with color previews (#12143) | Not applicable | N02 |
| [e47a11e1e65d](https://github.com/pingdotgg/t3code/commit/e47a11e1e65dcb7dc58630178c3f23fee7c156ec) | fix(web): align follow-up and license settings controls (#12167) | Not applicable | N02 |
| [892b86da124b](https://github.com/pingdotgg/t3code/commit/892b86da124b6f672622e3002faaa93e6a5bed00) | fix(web): align composer task rows (#12165) | Not applicable | N02 |
| [ed88937cf03e](https://github.com/pingdotgg/t3code/commit/ed88937cf03e65fe5d620f170fd14b1fae6bb2e7) | fix(mobile): prevent Android compose FAB animation jitter (#12169) | Not applicable | N01 |
| [c1738f131d6f](https://github.com/pingdotgg/t3code/commit/c1738f131d6f0d5c6dfabb3aad46755a25319663) | fix(server): keep large sparse checkouts on the fast checkpoint path (#12154) | Not applicable | N06 |
| [c12ba7581ac0](https://github.com/pingdotgg/t3code/commit/c12ba7581ac00ba2a623ad23283da5d03d34c2d8) | feat(web): make pull request comments easier to scan (#12150) | Not applicable | N11 |
| [19ee856ec099](https://github.com/pingdotgg/t3code/commit/19ee856ec099dddb94e28a3f933ec7c2575ed5a2) | fix(server): propagate linked pr changes and settle threads immediately (#12161) | Not applicable | N11 |
| [d612d12b8bb2](https://github.com/pingdotgg/t3code/commit/d612d12b8bb278af97b0029f8d48b2403a5f9ee6) | fix(web): reuse cached GitHub PR details across entry points (#12168) | Not applicable | N02 |
| [315b72371153](https://github.com/pingdotgg/t3code/commit/315b7237115356cee49a3a8a983a6d82596e1864) | Remove `new` badge from Fable 5.1 (#12173) | Ported | Reviewed Claude model-manifest data is retained by the local extraction. |
| [962bf62921c0](https://github.com/pingdotgg/t3code/commit/962bf62921c0ecbc9320f9b071e16b1d41484aad) | fix(web): show author avatars in pull request previews (#12125) | Not applicable | N11 |
| [3bee4c286e58](https://github.com/pingdotgg/t3code/commit/3bee4c286e586c2eb7f8d196a65b4b5ea6a95241) | fix(server): settle cancelled worktree setup before rollback (#12176) | Not applicable | N06 |
| [4abb07eac3d4](https://github.com/pingdotgg/t3code/commit/4abb07eac3d47543b5804e8039633ac8ad7f67d2) | feat(mobile): port worktree setup progress and agent handoff (#12177) | Not applicable | N03 |
| [8130a9f13a1e](https://github.com/pingdotgg/t3code/commit/8130a9f13a1ef506f4393423ded21d71f3c9b5e4) | fix(server): flush checkpoint objects and refs before publishing them (#10944) | Not applicable | N06 |
| [88d0c40c8581](https://github.com/pingdotgg/t3code/commit/88d0c40c8581f8886b8c94496c3e4af105851424) | chore(mobile): bump app version to 1.2.1 | Not applicable | N01 |
| [d42a3bd2ae3e](https://github.com/pingdotgg/t3code/commit/d42a3bd2ae3e698141b26f6426aa28d8886e1754) | fix(server): keep ready checkpoints when a later placeholder arrives (#8432) | Not applicable | N06 |
| [901db896691d](https://github.com/pingdotgg/t3code/commit/901db896691da5ab7b7428ef8a5da52883f831dd) | fix(server): keep VCS waits from blocking turn completion (#11970) | Not applicable | N06 |
| [573a8f2f30d5](https://github.com/pingdotgg/t3code/commit/573a8f2f30d59c528997daf1e02bdb58e0d9dc54) | fix(web): keep header spacing stable when sidebar drawer opens (#12162) | Not applicable | N02 |
| [df466c798100](https://github.com/pingdotgg/t3code/commit/df466c79810046d20c0f3ccc4e51709c5703814e) | fix(web): fall back when pull request avatars fail (#11728) | Not applicable | N02 |
| [d359e94cabca](https://github.com/pingdotgg/t3code/commit/d359e94cabca4722029359c77d60cb6d13da1e59) | feat(web): enable rich text composer by default (#12160) | Not applicable | N25 |
| [71d12d8c4a05](https://github.com/pingdotgg/t3code/commit/71d12d8c4a05bc50344d6e3fac1b967c6daaff84) | feat(web): make keybindings searchable from settings search (#12175) | Not applicable | N02 |
| [f17165a76b1a](https://github.com/pingdotgg/t3code/commit/f17165a76b1a782831eb6a05a4383b27283df15d) | fix(web): preserve thread reading positions (#12144) | Already covered | The host timelineState retains reading position. |
| [6d1d549441be](https://github.com/pingdotgg/t3code/commit/6d1d549441be84f19696ab59ed7e2fbf305280d4) | fix(diff): collapse files by default (#12190) | Not applicable | N02 |
| [886c83450797](https://github.com/pingdotgg/t3code/commit/886c83450797c46ee84ae28fc16c2208cafa965e) | fix(web): folder links from chat open the file tree instead of a broken preview (#10909) | Not applicable | N02 |
| [712f8cad32bf](https://github.com/pingdotgg/t3code/commit/712f8cad32bf8961c3c4b53497dd65c32672d59d) | feat(web): command palette search matches thread IDs (#11185) | Not applicable | N02 |
| [01eb4b9a8776](https://github.com/pingdotgg/t3code/commit/01eb4b9a8776e6057dfacf12e7366565e5eab934) | fix(web): align notification icons with titles (#12202) | Not applicable | N02 |
| [2a47fc905a7a](https://github.com/pingdotgg/t3code/commit/2a47fc905a7a1054823919980f625b7cd869f1c1) | fix(skills): support unicode currency symbols as skill aliases (#12098) | Not applicable | N20 |
| [c4ca1b0f94e5](https://github.com/pingdotgg/t3code/commit/c4ca1b0f94e56bd93899704f03015d51948a6b4d) | feat(settings): add automatic storage cleanup per machine and project (#11598) | Not applicable | N26 |
| [c0dded88b1fc](https://github.com/pingdotgg/t3code/commit/c0dded88b1fc8c1c86d65f684fbfa520139e58dc) | feat(web): command palette finds the pull requests and usage pages (#12211) | Not applicable | N02 |
| [0150c6a53b40](https://github.com/pingdotgg/t3code/commit/0150c6a53b409ba3bcb45645709b649cf8708354) | feat(web): start new threads with multiple models in separate worktrees (#12179) | Not applicable | N05 |
| [dd9528a974ba](https://github.com/pingdotgg/t3code/commit/dd9528a974ba534faa8d1588f3f137fa2a281860) | fix(mobile): keep screen awake during dictation (#12227) | Not applicable | N01 |
| [298f8c95cd7c](https://github.com/pingdotgg/t3code/commit/298f8c95cd7c20ea7943a431f22e836fa637457b) | feat(mobile): add favorites to model picker (#12231) | Not applicable | N01 |
| [1ab2dfb5a7bd](https://github.com/pingdotgg/t3code/commit/1ab2dfb5a7bd2996f79407b5d02cae6132a7626c) | fix(desktop): keep preview picking active across subframe navigation (#9741) | Not applicable | N09 |
| [d4d5d12e8ba0](https://github.com/pingdotgg/t3code/commit/d4d5d12e8ba086cfbf79ca3adeb4156b46ead665) | fix(shared): keep the newest shared usage scan (#10315) | Not applicable | N27 |
| [fbc8b960026e](https://github.com/pingdotgg/t3code/commit/fbc8b960026e408c163ea55db8075bbf1a42d628) | fix(web): keep thoughts and failed tool calls in one activity row (#12270) | Not applicable | N18 |
| [2d8378b44048](https://github.com/pingdotgg/t3code/commit/2d8378b4404888f0d5458e8fc9c83b1032e21290) | fix(web): avoid reopening settled threads when adding projects (#11804) | Not applicable | N02 |
| [c25c3e0ee678](https://github.com/pingdotgg/t3code/commit/c25c3e0ee678ef62245cb5eef20eeafb9a86d9d9) | feat(mobile): make Settings easier to navigate and scope (#12272) | Not applicable | N01 |
| [8db3c250f41e](https://github.com/pingdotgg/t3code/commit/8db3c250f41e8f1d0b36da6b5a5093821e72d4f3) | fix(mobile): prevent overlapping text and UI on Android chat messages (#11611) | Not applicable | N01 |
| [03950089ffa5](https://github.com/pingdotgg/t3code/commit/03950089ffa5cecf0ce731227f58551f6437495f) | feat(web): pull request files can be marked as viewed (#7721) | Not applicable | N16 |
| [4749035bda13](https://github.com/pingdotgg/t3code/commit/4749035bda13b4b6260499caedbc0d69a2f60e6f) | fix(web): keep composer banners compact and readable (#12166) | Not applicable | N28 |
| [592021f00801](https://github.com/pingdotgg/t3code/commit/592021f008010ee947c326df0452d093ead8aeb1) | fix(web): collapse thoughts within tool groups (#12302) | Not applicable | N18 |
| [675869d2b0f9](https://github.com/pingdotgg/t3code/commit/675869d2b0f9e059d5950b80390c0cad1cb1590e) | fix(usage): preserve saved totals after transcript cleanup (#12304) | Not applicable | N06 |
| [6deac7a924e6](https://github.com/pingdotgg/t3code/commit/6deac7a924e6d90eef400cd218703cd3ad00a383) | fix(mobile): show Agent behavior icon on Android (#12316) | Not applicable | N01 |
| [e26af33b2287](https://github.com/pingdotgg/t3code/commit/e26af33b228792bb72349ca4e849a96381578a57) | fix(web): keep PR panel actions in the current thread (#12320) | Not applicable | N02 |
| [4cc984fedc33](https://github.com/pingdotgg/t3code/commit/4cc984fedc33184c0a0fb67ea17fb02902ae2224) | fix(web): keep browser pages aligned during panel animations (#12329) | Not applicable | N02 |
| [b17cc2ab5d50](https://github.com/pingdotgg/t3code/commit/b17cc2ab5d5029f898121798ca19db3288cba616) | fix(server): bound provider event log records before serialization (#12305) | Already covered | Event payloads are bounded at 3 MiB and conversations at MaxSessionBytes. |
| [b4620d595554](https://github.com/pingdotgg/t3code/commit/b4620d595554654c259ec7c0afb3d3d5532959a8) | fix(server): reject file rewind in shared workspaces (#12306) | Not applicable | N03 |
| [67993623afb1](https://github.com/pingdotgg/t3code/commit/67993623afb14c017bb0b959478dc892f9c427da) | fix(server): capture checkpoints when baseline lookup fails (#12307) | Not applicable | N06 |
| [d17f46d7631b](https://github.com/pingdotgg/t3code/commit/d17f46d7631b178b6834596264f0f44f210d584b) | fix(server): refresh file search outside checkpoint processing (#12308) | Not applicable | N06 |
| [fcfd9f911a27](https://github.com/pingdotgg/t3code/commit/fcfd9f911a27ee3b813d64582c53d82dfa2e892b) | fix(web): keep chat from jumping when the scroll-to-end pill mounts (#12317) | Not applicable | N02 |
| [1455cb5c35e5](https://github.com/pingdotgg/t3code/commit/1455cb5c35e50211145073be6ca155d56684f0f4) | fix(server): checkpoint workspaces with empty nested repositories (#12181) | Not applicable | N06 |
| [5f225395637c](https://github.com/pingdotgg/t3code/commit/5f225395637cb347cfe416c649825a8c7c210014) | chore(review): keep review bots out of the vendored .repos references (#12333) | Not applicable | N15 |
| [3fd21df62da3](https://github.com/pingdotgg/t3code/commit/3fd21df62da3191f1abf12a0cd74df6de9ed30a2) | fix(server): pass Codex image attachments by path to avoid oversized requests (#11050) | Not applicable | N29 |
| [256e630ef7b8](https://github.com/pingdotgg/t3code/commit/256e630ef7b8372f70f80590e85aada62da821ab) | feat(web): filter sidebar from thread menu (#8719) | Not applicable | N02 |
| [9051ed9c3923](https://github.com/pingdotgg/t3code/commit/9051ed9c3923b7b5fb88d99fe9267380d69baa51) | feat(web): open diff files from a right-click context menu (#11842) | Not applicable | N16 |
| [3fd5d6439d8f](https://github.com/pingdotgg/t3code/commit/3fd5d6439d8fd49d173503ecda96500463a39bd2) | fix(web): keep numbered jumps from stealing browser tabs (#12315) | Not applicable | N02 |
| [53510d44ea7b](https://github.com/pingdotgg/t3code/commit/53510d44ea7b4a44c10c122dd9245194f2f14d09) | fix(mobile): define Clerk colors in every Uniwind theme (#12344) | Not applicable | N01 |
| [8583eac643b8](https://github.com/pingdotgg/t3code/commit/8583eac643b8c248bcd835e8de81c7c0d7182ad3) | refactor(web): reuse searchable picker inputs (#12353) | Not applicable | N30 |
| [5a39013313a7](https://github.com/pingdotgg/t3code/commit/5a39013313a7e6e9d6fbeeb915b8c176ea3dd126) | fix(web): share touch-visible pull request edit actions (#12370) | Not applicable | N02 |
| [05a8c6016a40](https://github.com/pingdotgg/t3code/commit/05a8c6016a4070b92f5d51e7a79b9c76814947df) | fix(mobile): share accessible connection trace controls (#12371) | Not applicable | N01 |
| [a1390ac65592](https://github.com/pingdotgg/t3code/commit/a1390ac655927cda10cfab3e4678d456db7b54b8) | fix(mobile): share settings control row layout (#12356) | Not applicable | N01 |
| [6088c8104cfd](https://github.com/pingdotgg/t3code/commit/6088c8104cfda0c11eafd1e1f636d47106fee136) | refactor(web): share diagnostic process actions (#12358) | Not applicable | N02 |
| [14337401bc80](https://github.com/pingdotgg/t3code/commit/14337401bc801d6239dd10e34997b4481081cf1d) | refactor(mobile): share Android toolbar search fields (#12359) | Not applicable | N01 |
| [4aa841a8d45c](https://github.com/pingdotgg/t3code/commit/4aa841a8d45ccd611a809351db0b693faba6ba13) | refactor(web): share settings group surfaces (#12360) | Not applicable | N16 |
| [8eb1f6e74869](https://github.com/pingdotgg/t3code/commit/8eb1f6e74869130b7602e0c6b0ab56b7805c0889) | refactor(web): reuse inline settings actions (#12362) | Not applicable | N02 |
| [767a5dbd00d0](https://github.com/pingdotgg/t3code/commit/767a5dbd00d0a2050745a2ae2e29050c5282b8c0) | refactor(mobile): share thread list section controls (#12363) | Not applicable | N01 |
| [18dcc43ace0a](https://github.com/pingdotgg/t3code/commit/18dcc43ace0a7b0d93f2fd61907b2a4e27522936) | refactor(mobile): share connection form fields (#12364) | Not applicable | N01 |
| [1a2a03f44b7e](https://github.com/pingdotgg/t3code/commit/1a2a03f44b7ef6e3159a67a262c1586dbd3d3d5b) | refactor(mobile): share local environment lists (#12365) | Not applicable | N01 |
| [58fcae64aece](https://github.com/pingdotgg/t3code/commit/58fcae64aece8198bcbbf051b02554ea893d859e) | refactor(mobile): share file preview feedback (#12368) | Not applicable | N01 |
| [fa2decfce7fa](https://github.com/pingdotgg/t3code/commit/fa2decfce7fa359551634b4d24a2aecd9cef88c9) | refactor(web): share standalone page layout (#12354) | Not applicable | N02 |
| [c7fa1e96943f](https://github.com/pingdotgg/t3code/commit/c7fa1e96943f380a8893e89460d03a362285ab83) | fix(mobile): share settings action row defaults (#12369) | Not applicable | N01 |
| [cc39ec1008ce](https://github.com/pingdotgg/t3code/commit/cc39ec1008ce584e088e79210b69d3359357463b) | fix(mobile): share request action button defaults (#12366) | Not applicable | N01 |
| [994654198d58](https://github.com/pingdotgg/t3code/commit/994654198d5820b6fab9ee09063646a8b6fd8e22) | fix(web): share accessible color picker controls (#12355) | Not applicable | N02 |
| [818add08fcd7](https://github.com/pingdotgg/t3code/commit/818add08fcd73942f1f05edacc23f4019c146eeb) | fix(mobile): use singular label for one settings environment (#12282) | Not applicable | N01 |
| [7a2cce826852](https://github.com/pingdotgg/t3code/commit/7a2cce826852c3a3255d7c7aa1e9c148287eba2c) | feat(mobile): add copy thread ID to thread list actions (#12228) | Not applicable | N01 |
| [243e94470f80](https://github.com/pingdotgg/t3code/commit/243e94470f80a6a5f31a68785ceff30c29ce5d4c) | fix(mobile): remove Android input underline backgrounds (#12394) | Not applicable | N01 |
| [d547e3b12a1a](https://github.com/pingdotgg/t3code/commit/d547e3b12a1af79e9dd81e422255a12d6ecf70d1) | chore(deps): upgrade Effect to rc.115 and Alchemy to beta.78 (#12326) | Not applicable | N31 |
| [e3c85ead63c3](https://github.com/pingdotgg/t3code/commit/e3c85ead63c3a69f9be5dbb873b3d87c9b76b43f) | chore(refs): sync Effect and Alchemy references to rc.115 and beta.78 (#12327) | Not applicable | N32 |
| [747962568689](https://github.com/pingdotgg/t3code/commit/74796256868916da550ecbe9b5e37d359a8cb321) | chore(relay): deploy with the Alchemy CLI and publish client config through an Action (#12401) | Not applicable | N33 |
| [c5bbf3d14952](https://github.com/pingdotgg/t3code/commit/c5bbf3d1495234236e7ad5d7e8d091b0ab081a95) | chore(deps): bump the npm_and_yarn group across 1 directory with 3 updates (#12411) | Not applicable | N15 |
| [56a9bf2bd7d3](https://github.com/pingdotgg/t3code/commit/56a9bf2bd7d3dcdae722a5de84578909dc11aeda) | fix(git): prevent stale branch selections from restoring files (#10574) | Not applicable | N06 |
| [52e4b4429359](https://github.com/pingdotgg/t3code/commit/52e4b4429359904441039a286c61eaafe8474451) | chore(deps): bump parents that carry vulnerable transitive dependencies (#12417) | Not applicable | N15 |
| [9ea9c3d5d2c4](https://github.com/pingdotgg/t3code/commit/9ea9c3d5d2c444133e3ddff40eecf38737951589) | fix(web): keep a file-to-symlink type change from crashing the diff view (#11075) | Not applicable | N02 |
| [ccad9f69aae7](https://github.com/pingdotgg/t3code/commit/ccad9f69aae7465c4ad4ea169d273f61ffac1eb5) | Use T3 Device panel for mobile testing (#12414) | Not applicable | N15 |
| [bd5dc58c65b8](https://github.com/pingdotgg/t3code/commit/bd5dc58c65b87bfd2222e4697c4742b48e639ea4) | fix(web): client spans reach the trace proxy again (#12332) | Not applicable | N02 |
| [eadeaf228267](https://github.com/pingdotgg/t3code/commit/eadeaf228267be215a2e9486f16b876d703dcccc) | fix(bitbucket): preserve rate limits from optional PR reads (#12486) | Not applicable | N11 |
| [9118b6d3e868](https://github.com/pingdotgg/t3code/commit/9118b6d3e868db161ca3f330c0a8011527348dce) | fix(mobile): synchronize native permission registry access (#12482) | Not applicable | N01 |
| [ed6fe339d718](https://github.com/pingdotgg/t3code/commit/ed6fe339d71863f39a1d5405554a3bac892e55b6) | fix(build): retain multiple license notices for one package (#12489) | Not applicable | N15 |
| [dcdf5e6a0136](https://github.com/pingdotgg/t3code/commit/dcdf5e6a013607b42836a270165286181da48296) | fix(build): parse executable imports without matching source strings (#12488) | Not applicable | N06 |
| [4c61e3d7ecc4](https://github.com/pingdotgg/t3code/commit/4c61e3d7ecc4d9957c243677d35dd3313a37c067) | fix(mobile): synchronize native notification delegates (#12483) | Not applicable | N01 |
| [80c1f771b396](https://github.com/pingdotgg/t3code/commit/80c1f771b39674ada4d3724610f08fba855fdd49) | fix(relay): accept delegated thread IDs in activity routes (#12484) | Not applicable | N33 |
| [c03b240265a1](https://github.com/pingdotgg/t3code/commit/c03b240265a1d4954536c5c0758ac2b7fa762ba8) | fix(git): explain fetch failures without exposing remote output (#12485) | Not applicable | N06 |
| [c557bb10aa7c](https://github.com/pingdotgg/t3code/commit/c557bb10aa7cbb78a50df230d1e8e24af79cb662) | fix(web): sidebar search matches message content (#11761) | Not applicable | N02 |
| [6d68677ff909](https://github.com/pingdotgg/t3code/commit/6d68677ff90915c17de06d78bb498d41a1adfe6a) | fix(server): restore secrets when settings persistence fails (#12487) | Not applicable | N06 |
| [8ebb6112377a](https://github.com/pingdotgg/t3code/commit/8ebb6112377a110afb956ac01fb44fee68f5709c) | fix(ci): accept  transfer reports without cross-scenario comparisons (#12492) | Not applicable | N15 |
| [e84f23a97d58](https://github.com/pingdotgg/t3code/commit/e84f23a97d587e0e58215f1fba2eab12717bec28) | fix(web): speed up PR previews with fewer GitHub requests (#11825) | Not applicable | N23 |
| [869347bc2605](https://github.com/pingdotgg/t3code/commit/869347bc26051bba7c3c0274a7dfcae7770d3e3a) | fix(server): retry transient git failures during checkpoint capture (#11665) | Not applicable | N06 |
| [4d36142c022e](https://github.com/pingdotgg/t3code/commit/4d36142c022e0dc797034933d01e200818b4083d) | fix(mobile): keep archived threads visible during iOS search (#12420) | Not applicable | N01 |
| [e0649ed7d8f8](https://github.com/pingdotgg/t3code/commit/e0649ed7d8f8b5f21d114ff2b09ef803184f1d05) | perf(mobile): isolate Material You conversion on Android (#12379) | Not applicable | N01 |
| [53830d413474](https://github.com/pingdotgg/t3code/commit/53830d413474b2a5749c040d4382c96002894b88) | perf(mobile): isolate iOS Live Activity imports (#12380) | Not applicable | N01 |
| [7e7cd32463b5](https://github.com/pingdotgg/t3code/commit/7e7cd32463b528838cb10ef6b89ef6ceac1f7a4c) | refactor(mobile): split home headers by platform (#12381) | Not applicable | N01 |
| [429d66c13666](https://github.com/pingdotgg/t3code/commit/429d66c13666c4e243065bbd02bee48e1eea2307) | refactor(mobile): split native menus by platform (#12382) | Not applicable | N01 |
| [dec0732281f5](https://github.com/pingdotgg/t3code/commit/dec0732281f5b7e787ad3a84e5cbe9a1607ea4ca) | refactor(mobile): isolate thread row appearance by platform (#12383) | Not applicable | N01 |
| [7230f6b1be4d](https://github.com/pingdotgg/t3code/commit/7230f6b1be4d1ad839b9a6ab45d0d70b4abc9a54) | refactor(mobile): split settings selection rows by platform (#12384) | Not applicable | N01 |
| [4fb9dfc8b860](https://github.com/pingdotgg/t3code/commit/4fb9dfc8b8600d830b36da1c15c5f0cd21627e77) | refactor(mobile): centralize platform header rendering (#12388) | Not applicable | N01 |
| [989bde889dd1](https://github.com/pingdotgg/t3code/commit/989bde889dd119f5919ce27c9d56db77efedec4c) | refactor(mobile): configure thread headers through the shared core (#12389) | Not applicable | N01 |
| [54bbed660fc5](https://github.com/pingdotgg/t3code/commit/54bbed660fc5dc99c6bfe335981e6d431275c91e) | refactor(mobile): share file header actions and search configuration (#12390) | Not applicable | N01 |
| [7ee60f4bc964](https://github.com/pingdotgg/t3code/commit/7ee60f4bc964c2d8d8cf182c173faaabcf902b7d) | refactor(mobile): share terminal header and menu configuration (#12391) | Not applicable | N01 |
| [aba1c6352d77](https://github.com/pingdotgg/t3code/commit/aba1c6352d77c33188dc03d2b76bdfe489c95542) | refactor(mobile): share archived thread header configuration (#12399) | Not applicable | N01 |
| [88b9226d229a](https://github.com/pingdotgg/t3code/commit/88b9226d229a7d51283122a8b8b8be8305822a9e) | refactor(mobile): compose review menus through the shared header (#12400) | Not applicable | N01 |
| [93e04160a0c0](https://github.com/pingdotgg/t3code/commit/93e04160a0c0dece8a258384d2118a660415a53d) | feat(mobile): search projects when starting a task (#12496) | Not applicable | N01 |
| [82cd1d1aabe8](https://github.com/pingdotgg/t3code/commit/82cd1d1aabe8baa4a5ae23f3c7b08b3fce092e08) | fix(mobile): preserve multiple model favorites (#12505) | Not applicable | N01 |
| [3bb06ad91038](https://github.com/pingdotgg/t3code/commit/3bb06ad910389fcfc886bd5156f41dc73f307f80) | feat(server): export log records over OTLP (#12493) | Not applicable | N03 |
| [ea15f17afd94](https://github.com/pingdotgg/t3code/commit/ea15f17afd94e7ce7133e52f5b5983910662c9ec) | fix(mobile): use native settings and snooze controls (#12512) | Not applicable | N01 |
| [d99bc5a4d707](https://github.com/pingdotgg/t3code/commit/d99bc5a4d707df2506e60cbb411b8d30022bbf89) | feat(web): sort pull requests by what is blocked on me (#12508) | Not applicable | N02 |
| [8c3b5bef17a6](https://github.com/pingdotgg/t3code/commit/8c3b5bef17a637bf53f297189350359790ef7afb) | fix(mobile): prefer pull-to-refresh on list screens (#12515) | Not applicable | N01 |
| [de6a230db0bb](https://github.com/pingdotgg/t3code/commit/de6a230db0bb71d3fa91d8de0ae7067b68253f82) | fix(acp): accept SDK elicitation requests (#11294) | Not applicable | N34 |
| [bd0b9edac3cf](https://github.com/pingdotgg/t3code/commit/bd0b9edac3cf8fe30f8242dd7943a9240810b2c9) | fix(release): read relay configuration without loading deployment providers (#12518) | Not applicable | N15 |
| [4931d73af3e6](https://github.com/pingdotgg/t3code/commit/4931d73af3e68d8706d7568b9af37c8397caffdb) | fix(ci): reconcile native change labels against pinned commits (#12517) | Not applicable | N15 |
| [e74c668d786f](https://github.com/pingdotgg/t3code/commit/e74c668d786ffc14f40a9540cd3aede02ba5d62e) | fix(release): strip Alchemy progress before parsing relay state (#12519) | Not applicable | N15 |
| [9cb586acd049](https://github.com/pingdotgg/t3code/commit/9cb586acd0494c4f47f60045ee42ef25edb5a07c) | refactor: remove obsolete code (#9917) | Not applicable | N30 |
| [cb3d95c17487](https://github.com/pingdotgg/t3code/commit/cb3d95c17487f3d834a5537f7c3aa0106bc9b5ec) | fix(server): release oversized pull request diff cache entries (#12523) | Not applicable | N11 |
| [823119350ee2](https://github.com/pingdotgg/t3code/commit/823119350ee29cca434b1295aa5c560f4c9ef98a) | feat(mobile): view and control agent devices (#12531) | Not applicable | N03 |
| [5378f87f9917](https://github.com/pingdotgg/t3code/commit/5378f87f99175bded2b9241876319567875aca03) | fix(preview): recover host registration after request timeouts (#12535) | Not applicable | N06 |
| [0f1b572b9865](https://github.com/pingdotgg/t3code/commit/0f1b572b9865256dd61e245b4dd09f6a7b15f0e1) | fix(mobile): align built-in theme colors with desktop (#12534) | Not applicable | N03 |
| [82059df1523d](https://github.com/pingdotgg/t3code/commit/82059df1523d070b00ab6f45159115f626a4977e) | feat(desktop): export main process telemetry over OTLP (#12520) | Not applicable | N09 |
| [efb96939fbb1](https://github.com/pingdotgg/t3code/commit/efb96939fbb135b9f4de3bff19447200b178b23a) | fix(codex): surface app permission requests as approvable (#7861) | Ported | Codex permissions approval request, granted for the turn only. |
| [a8693eb4ed3a](https://github.com/pingdotgg/t3code/commit/a8693eb4ed3a66d72e6b814117e7beeaf0fb5f8b) | chore(desktop): leave main process metrics export off until a metric exists (#12540) | Not applicable | N09 |
| [803f94e787fa](https://github.com/pingdotgg/t3code/commit/803f94e787faef498edecba5dbd07493ac8bc817) | fix(release): drop placeholder allowBuilds entry that broke desktop builds (#12544) | Not applicable | N15 |
| [29a23fdf1481](https://github.com/pingdotgg/t3code/commit/29a23fdf1481485593d7d90e195b7c8bf2a705a8) | fix(mobile): adapt workspace navigation and expand controls (#12551) | Not applicable | N01 |
| [7a326c244f72](https://github.com/pingdotgg/t3code/commit/7a326c244f72d6421c2f1cbf004d55806c631526) | chore(mobile): add dev client script with preview environment (#12558) | Not applicable | N15 |
| [408ff8ae9bd7](https://github.com/pingdotgg/t3code/commit/408ff8ae9bd7eb2e7e90cbfd8b3fcfe63641bf23) | fix: detect installed editors outside PATH (#12439) | Not applicable | N03 |
| [dfbb11bdd7c3](https://github.com/pingdotgg/t3code/commit/dfbb11bdd7c3f1a5575cb55d3e3abb12be025727) | fix(web): show plain text in collapsed thought previews (#12377) | Not applicable | N18 |
| [9accc567670d](https://github.com/pingdotgg/t3code/commit/9accc567670da5096e9a694c1adb0d35adb51d9d) | fix(web): wrap long titles in confirmation dialogs (#12571) | Not applicable | N02 |
| [b44c1ce5d25e](https://github.com/pingdotgg/t3code/commit/b44c1ce5d25ee0d5a5be82e380618a886c19ea96) | fix(mobile): keep the Android composer placeholder on one line (#12605) | Not applicable | N01 |
| [8dd02470b1e7](https://github.com/pingdotgg/t3code/commit/8dd02470b1e7603b8d36a21ae27c510480351f18) | fix(web): restore providers settings heading (#12552) | Not applicable | N16 |
| [7810fb263f5a](https://github.com/pingdotgg/t3code/commit/7810fb263f5a92b9d438d1eabf408d2ac705e944) | Add new GitHub user 'yordis' to VOUCHED.td (#12546) | Not applicable | N15 |
| [f9e8f578fa0f](https://github.com/pingdotgg/t3code/commit/f9e8f578fa0fb1ea25e458334a28bbb4fe22302d) | chore: vouch cestercian (#12638) | Not applicable | N15 |
| [dcf8942304cf](https://github.com/pingdotgg/t3code/commit/dcf8942304cfef382af55c0068cb2f04c86757fc) | fix(web): keep desktop annotation screenshots under CSP (#12636) | Not applicable | N02 |
| [e36725682bd2](https://github.com/pingdotgg/t3code/commit/e36725682bd2935f1bcfdf81660cb569b5f3df29) | fix(web): keep typed text when a question option is clicked (#12577) | Ported | Restore replaced non-secret typed answers to the conversation draft (#12577). |
| [63ff33756c47](https://github.com/pingdotgg/t3code/commit/63ff33756c47f127391962458d8388a647e8ed44) | fix(server): empty Claude homePath shares continuation with ~/.claude (#12624) | Not applicable | N35 |
| [52d08a14b9e4](https://github.com/pingdotgg/t3code/commit/52d08a14b9e475371658fabbb12c056c3d3dad67) | fix(desktop): include SnapShot app text for Flatpak and GTK4 (#12635) | Not applicable | N09 |
| [d6f291303ddc](https://github.com/pingdotgg/t3code/commit/d6f291303ddc0c9a14f570266a4d9eff6d431593) | fix(server): surface ACP stderr when cursor-agent exits at session start (#12625) | Not applicable | N36 |
| [599c9776eb88](https://github.com/pingdotgg/t3code/commit/599c9776eb887d4fff42da7d2daf8aa07ee9a9e2) | fix(web): align pull request state glyph to top of row (#11268) | Not applicable | N02 |
| [7445aa733ada](https://github.com/pingdotgg/t3code/commit/7445aa733ada33e45289e5aa5055f79142556513) | fix(web): align menu item icons in pull request detail panel (#11263) | Not applicable | N02 |
| [4a560b4e4ebb](https://github.com/pingdotgg/t3code/commit/4a560b4e4ebb37efb7f57805ba79e37f5500bdca) | fix(web): honor whitespace settings in pull request diffs (#12438) | Not applicable | N02 |
| [0ff87f251daf](https://github.com/pingdotgg/t3code/commit/0ff87f251dafb32703d5f531e7b248ad0a581ee2) | fix(web): keep citation comment when popover is dismissed (#10831) | Not applicable | N02 |
| [c14f6015bfe4](https://github.com/pingdotgg/t3code/commit/c14f6015bfe479d313355cb234af1a5c16dbb15f) | fix(web): keep narrow chat headers readable and aligned (#12453) | Not applicable | N16 |
| [fa1e1715538f](https://github.com/pingdotgg/t3code/commit/fa1e1715538f6f8ac076761f7821f68479f616ab) | refactor(observability): hold OTLP export settings per signal (#12657) | Not applicable | N06 |
| [ead1dee22916](https://github.com/pingdotgg/t3code/commit/ead1dee22916b414906eafdae5f5a1293f16e637) | fix(web): explain what enabling network access means in its confirmation (#10098) | Not applicable | N02 |
| [9f0c9f725d45](https://github.com/pingdotgg/t3code/commit/9f0c9f725d4553d34e3c91c13d67ebcc58be5b0f) | fix(web): reuse current PR status in the sidebar (#12545) | Not applicable | N11 |
| [6a699f0f2fbd](https://github.com/pingdotgg/t3code/commit/6a699f0f2fbd8847d7ec2df8d9245ce8c2eb8707) | fix(web): stabilize pull request loading layout (#12721) | Not applicable | N02 |
| [55c24273a025](https://github.com/pingdotgg/t3code/commit/55c24273a025874cdb8a96a25ee0c59eeb4e3051) | fix(desktop): align preview recording cursors and show input feedback (#12779) | Not applicable | N03 |
| [7ade2d2c6b37](https://github.com/pingdotgg/t3code/commit/7ade2d2c6b37df6f1bc879bfcbf64658b7af2554) | fix(web): the Run on / Workspace menu closes after a pick (#12685) | Not applicable | N02 |
| [3cfebf4aa3ce](https://github.com/pingdotgg/t3code/commit/3cfebf4aa3ce7410806c0ba4cc4d991cc1ce8dbb) | fix(web): keep portaled menus clickable over Electron drag regions (#12527) | Not applicable | N16 |
| [5d9e27a59688](https://github.com/pingdotgg/t3code/commit/5d9e27a596884d30499c7401296916fd729c817f) | fix(web): render citations in queued messages (#12403) | Not applicable | N10 |
| [2d8f9a8f55d2](https://github.com/pingdotgg/t3code/commit/2d8f9a8f55d2195ac820c91f0f1e8f66a055f841) | fix(web): keep the timeline still when the resting composer expands (#12771) | Not applicable | N02 |
| [45e06f48a033](https://github.com/pingdotgg/t3code/commit/45e06f48a03354b600782259010ae6222241aff0) | fix: composer hero reads project name to screen readers (#12397) | Not applicable | N03 |
| [f6cc6bc7e9cb](https://github.com/pingdotgg/t3code/commit/f6cc6bc7e9cbe1feeb7811ea92d519ad77c76596) | fix(mobile): respect word wrap in diffs (#12590) | Not applicable | N01 |
| [adcd90858c3b](https://github.com/pingdotgg/t3code/commit/adcd90858c3bfece59095465d2f49860ba3dcea9) | fix(web): allow full contrast in assistant replies (#12405) | Not applicable | N28 |
| [933492470085](https://github.com/pingdotgg/t3code/commit/93349247008517663aa16a3ed145d5b0f3f044ff) | fix(web): pull request chips share the link hover preview (#12719) | Not applicable | N23 |
| [584450a1f198](https://github.com/pingdotgg/t3code/commit/584450a1f198c8592d28947a0a7a060d28013298) | fix(web): compact the worktree setup glass popover (#12802) | Not applicable | N16 |
| [67285e4b8719](https://github.com/pingdotgg/t3code/commit/67285e4b8719000eecb9de2a424609ca5c6f0274) | fix(web): route keyboard submit through the primary worktree action (#12526) | Not applicable | N02 |
| [c789cd174ae5](https://github.com/pingdotgg/t3code/commit/c789cd174ae50f50ecb0a34b96a81e00b0f5c823) | fix(web): skip image inline chip when composer is empty (#12528) | Not applicable | N22 |
| [f4d979a6c613](https://github.com/pingdotgg/t3code/commit/f4d979a6c6139ab10aa1013335bd94a313a5b56e) | fix(web): only show notice details when text is clipped (#12760) | Not applicable | N02 |
| [9f73ca3677fa](https://github.com/pingdotgg/t3code/commit/9f73ca3677faa975be00f0ac55460279671df726) | fix(devices): recover simulator streams after failures (#12639) | Not applicable | N03 |
| [1ba471a37cd6](https://github.com/pingdotgg/t3code/commit/1ba471a37cd6b0f18820795f4505206f4723a0e3) | chore(server): bump device tooling versions (#12809) | Not applicable | N06 |
| [2b178231ebfa](https://github.com/pingdotgg/t3code/commit/2b178231ebfa597947efb4ec31bf60e0bb9e3128) | fix: allow more attachments without raising the image payload budget (#12620) | Not applicable | N03 |
| [051a4057a67c](https://github.com/pingdotgg/t3code/commit/051a4057a67ceebc7a36b13678f404b7ee871a46) | fix(web): device Reconnect starts one stream instead of two (#12808) | Not applicable | N02 |
| [58f89818e3ad](https://github.com/pingdotgg/t3code/commit/58f89818e3ada7f680bede396d41ed27eb618203) | fix(server): tolerate shutting down an iOS simulator that is already off (#12807) | Not applicable | N06 |
| [411da2a7356d](https://github.com/pingdotgg/t3code/commit/411da2a7356dd795b720228e9c834ea9bad0415f) | feat(web): use the linked pull request row layout on the pull requests page (#12536) | Not applicable | N11 |
| [2efb8178d83f](https://github.com/pingdotgg/t3code/commit/2efb8178d83f4f7c4ccc3ae1165f2a2ded0b2742) | fix(clients): keep backslashes in copied Codex citations (#12243) | Not applicable | N07 |
| [f7efb5354a3e](https://github.com/pingdotgg/t3code/commit/f7efb5354a3e6fb5e6b13a658ba57020e2c554ad) | feat(web): truncate branch names and paths in the middle (#12805) | Not applicable | N02 |
| [dca84efb53b0](https://github.com/pingdotgg/t3code/commit/dca84efb53b071691a88c6712751fb7de9dcbfdb) | fix(web): paste markdown with inline code inside bold, italic, or strikethrough (#12290) | Not applicable | N02 |
| [a9ab9049bbba](https://github.com/pingdotgg/t3code/commit/a9ab9049bbbab89fe23dcfb26c3169ef70a0597e) | feat(web): show the pull request refresh spinning in the detail header (#12833) | Not applicable | N02 |
| [a4bc7deb992c](https://github.com/pingdotgg/t3code/commit/a4bc7deb992cc3d48dd4a40ee1f84254f190f318) | fix(web): dismiss composer suggestions with Escape (#12836) | Not applicable | N22 |
| [a6cb1dd2054c](https://github.com/pingdotgg/t3code/commit/a6cb1dd2054c96c5be41980e63594ea51554a552) | fix(mobile): keep the source worktree when starting a thread on a branch (#12623) | Not applicable | N01 |
| [6cc7f7006e24](https://github.com/pingdotgg/t3code/commit/6cc7f7006e24283072cc3e0b3ed8ced80c6d71a5) | fix(web): keep composer controls visible while they fit (#12837) | Not applicable | N22 |
| [5c5fa5ffb6ab](https://github.com/pingdotgg/t3code/commit/5c5fa5ffb6ab45cadb6f0d873af0242f4f6d2c1c) | feat(devices): show installed and running tool versions per host (#12816) | Not applicable | N03 |
| [3b836f994214](https://github.com/pingdotgg/t3code/commit/3b836f9942143cdcf1ad7f121e78b59476585781) | feat(devices): show automatic update progress and host retry (#12817) | Not applicable | N03 |
| [61b25b88d792](https://github.com/pingdotgg/t3code/commit/61b25b88d7922a480790066341867f3f8ae42fcb) | feat(devices): add read-only update discovery and remote ownership (#12818) | Not applicable | N03 |
| [790be6d75502](https://github.com/pingdotgg/t3code/commit/790be6d755026d61bd13a970dd77176c12ca0425) | fix(devices): safely reclaim obsolete managed tool versions (#12819) | Not applicable | N06 |
| [30e3649c3281](https://github.com/pingdotgg/t3code/commit/30e3649c3281fe88b3d8d125293f1bb58132a3c2) | fix(web): match thread notification icons to sidebar status (#12806) | Not applicable | N02 |
| [f391b88c3754](https://github.com/pingdotgg/t3code/commit/f391b88c3754047c3c9434b76328699c083723b0) | fix(web): move sidebar shelves as one block (#11772) | Not applicable | N02 |
| [6b0a04ade021](https://github.com/pingdotgg/t3code/commit/6b0a04ade021588dd9f1afbfa0e75cccdb7b2da4) | fix(web): offer undo after unpinning a thread (#10744) | Not applicable | N02 |
| [5781b5240bd5](https://github.com/pingdotgg/t3code/commit/5781b5240bd5d2e21c651f6b228975ac40cbd67b) | feat(web): undo settle, snooze and archive, with a mod+z shortcut (#12848) | Not applicable | N02 |
| [1eeeabb26f65](https://github.com/pingdotgg/t3code/commit/1eeeabb26f654c679efe265cd03edf69c3df3a6f) | fix(mobile): use a proper pull request icon on iOS (#12855) | Not applicable | N01 |
| [b379b5b1407b](https://github.com/pingdotgg/t3code/commit/b379b5b1407b4c718095cd13395173b6cc005218) | test(web): remove redundant favicon test (#12856) | Not applicable | N02 |
| [1de563c1491c](https://github.com/pingdotgg/t3code/commit/1de563c1491c7d82563e4553bf5bf689ce6adbb9) | feat(devices): offer manual updates in tool version details (#12877) | Not applicable | N03 |
| [371b52d9dad7](https://github.com/pingdotgg/t3code/commit/371b52d9dad76876f84609a4cd0f80eaa757b69c) | feat(web): answer pull request actions on the row at once (#12843) | Not applicable | N02 |
| [33cb911ceb89](https://github.com/pingdotgg/t3code/commit/33cb911ceb89d55b812f486d4e001d0f86d77680) | fix(mobile): stop iOS autocorrect from rewriting search queries (#12949) | Not applicable | N08 |
| [2c2fa8cd7ded](https://github.com/pingdotgg/t3code/commit/2c2fa8cd7ded97be76587ae1cfce54ee0a65f544) | fix(web): pull request embed chip shows the state icon (#12951) | Not applicable | N02 |
| [2d377bfb1c8c](https://github.com/pingdotgg/t3code/commit/2d377bfb1c8c85a61679fa6cdb6e8181cfb13626) | fix(web): dismiss selection actions when pressing buttons (#12950) | Not applicable | N02 |
| [d2a90b921a58](https://github.com/pingdotgg/t3code/commit/d2a90b921a5843654a58448d0bc59828e1e3ccc4) | fix(web): name message copy actions accurately (#12865) | Ported | Name the action in the copy tooltip (#12865). |
| [80d9c181da41](https://github.com/pingdotgg/t3code/commit/80d9c181da41ee3890791c9b65018ce093e7bc9e) | fix(contracts): old message-sent events without turnId no longer stop the server from starting (#12763) | Not applicable | N07 |
| [e1cbb705273a](https://github.com/pingdotgg/t3code/commit/e1cbb705273a261d26b489f35a7ddca6567d9663) | fix(web): the custom snooze calendar starts the week where the locale does (#12745) | Not applicable | N02 |
| [76cc9b08f19d](https://github.com/pingdotgg/t3code/commit/76cc9b08f19d89012f16d18f47478c4f7b9b0a6f) | chore(mobile): bump app version to 1.3.0 | Not applicable | N01 |
| [1262d2f3ab29](https://github.com/pingdotgg/t3code/commit/1262d2f3ab2940c1d47ca4f5c4ca42e5c7259765) | feat(server): let t3.json limit or disable submodule init in new worktrees (#12953) | Not applicable | N06 |
| [742173a13250](https://github.com/pingdotgg/t3code/commit/742173a13250bcba880b700aaead34e1b0810dff) | feat(settings): resolve t3.json inside the project settings resolver (#12954) | Not applicable | N16 |
| [0141bc2bf5fc](https://github.com/pingdotgg/t3code/commit/0141bc2bf5fcf52a563240a6bce4b58050496db5) | feat(settings): choose how new worktrees initialize submodules (#12955) | Not applicable | N11 |
| [9a609a4e444b](https://github.com/pingdotgg/t3code/commit/9a609a4e444ba739d6fcd607769d68b679b0bcc5) | fix(web): show thread undo notice in the sidebar (#12972) | Not applicable | N02 |
| [5423ba0fd084](https://github.com/pingdotgg/t3code/commit/5423ba0fd08447240282c08362ee8e382bf16903) | feat(web): merge the comment and review buttons into one composer (#12945) | Not applicable | N16 |
| [e65bc1c73e5e](https://github.com/pingdotgg/t3code/commit/e65bc1c73e5e922216415b8e553bc74682d6810d) | fix(web): allow text selection when renaming threads (#12935) | Not applicable | N02 |
| [5a61f50cce12](https://github.com/pingdotgg/t3code/commit/5a61f50cce12685f70d530aa5fdf83d9c9b065d7) | chore(lint): report className restyling of components/ui exports (#12982) | Not applicable | N31 |
| [c26119ada356](https://github.com/pingdotgg/t3code/commit/c26119ada3565dda55bc3e86fdf47c7f9088e4b5) | refactor(web): drop className overrides that repeat the base styles (#12984) | Not applicable | N28 |
| [438bf466fd7e](https://github.com/pingdotgg/t3code/commit/438bf466fd7e97bb80f0d1f936b2c857830249b7) | fix(web): close menus when clicking into the browser tab (#11148) | Not applicable | N02 |
| [b5a0f810108d](https://github.com/pingdotgg/t3code/commit/b5a0f810108d42ca8635b5a3d75a6e885bb3a254) | refactor(web): give Spinner and RefreshIcon a size prop (#12985) | Not applicable | N28 |
| [aff9318bf46b](https://github.com/pingdotgg/t3code/commit/aff9318bf46beaf05cc7155b428d3f0b8711efd2) | fix(web): retry failed attachment uploads after reconnect (#10338) | Not applicable | N03 |
| [da6a85b13659](https://github.com/pingdotgg/t3code/commit/da6a85b1365993d0ff2a79698cd82498de247d8a) | fix(web): respect panel motion in composer transitions (#11064) | Not applicable | N22 |
| [7c2702d68ae2](https://github.com/pingdotgg/t3code/commit/7c2702d68ae2e10fa9d3a08c996ed47c393dcffd) | fix(web): read panel animation settings in the composer (#13098) | Not applicable | N22 |
| [f25a8e4b72c1](https://github.com/pingdotgg/t3code/commit/f25a8e4b72c1164f2ab7b1c8481496f168d17f55) | feat(models): add opus 5.5 without changing existing aliases (#13094) | Ported | Reviewed Claude model-manifest data is retained by the local extraction. |
| [17e34773b148](https://github.com/pingdotgg/t3code/commit/17e34773b148fd7351a44ab39e9616b4073a1fa8) | Update model manifest with new timestamps and models | Ported | Reviewed Claude model-manifest data is retained by the local extraction. |
| [c0490e9d0eb0](https://github.com/pingdotgg/t3code/commit/c0490e9d0eb057218bfa5df3f25a7c26c37145a5) | refactor(web): use ghost-muted where ghost buttons restyled to muted (#13020) | Not applicable | N28 |
| [1a81ea8c1532](https://github.com/pingdotgg/t3code/commit/1a81ea8c153214ddc74b7046062a8957c2612e8d) | refactor(web): fold repeated overrides into ui defaults (#13021) | Not applicable | N28 |
| [ce98d4107758](https://github.com/pingdotgg/t3code/commit/ce98d4107758ed7a492f94a93b3685a828caeba1) | refactor(web): mark the current menu value with MenuRadioGroup (#13022) | Not applicable | N02 |
| [ed856946c63e](https://github.com/pingdotgg/t3code/commit/ed856946c63e51861a8153d264856db0dbb677a4) | refactor(web): add an active prop to CommandItem (#13023) | Not applicable | N02 |
| [bbd5cc03231b](https://github.com/pingdotgg/t3code/commit/bbd5cc03231b8b831b44bdbd6e62bf736391cb1f) | chore(lint): exempt CollapsibleTrigger from no-restyle (#13024) | Not applicable | N15 |
| [9515016166f9](https://github.com/pingdotgg/t3code/commit/9515016166f926968dacaefc69e2ea0af63942ae) | refactor(web): use icon-xs where icon buttons were forced to size-6 (#13025) | Not applicable | N28 |
| [b4f1f18b7356](https://github.com/pingdotgg/t3code/commit/b4f1f18b735679aedc4369190fc3f07ecaf8966c) | refactor(web): add radius="none" to ScrollArea (#13026) | Not applicable | N28 |
| [7b54af90f79a](https://github.com/pingdotgg/t3code/commit/7b54af90f79a6858f75ff86d42a2f083d4edfa2a) | refactor(web): add font="mono" to Input (#13027) | Not applicable | N28 |
| [be80e369394f](https://github.com/pingdotgg/t3code/commit/be80e369394f5d0cb771b51669686f2bc916e543) | refactor(web): add SidebarInput (#13028) | Not applicable | N02 |
| [8d02447c36df](https://github.com/pingdotgg/t3code/commit/8d02447c36df2cf6cbfe1501738314685a22a1fc) | refactor(web): add a label variant to Badge (#13029) | Not applicable | N28 |
| [7d658c24dbf8](https://github.com/pingdotgg/t3code/commit/7d658c24dbf81d7cfc594b89d9cd5b8bab0e1046) | refactor(web): give Skeleton three shapes (#13030) | Not applicable | N02 |
| [aa2b25d29b4b](https://github.com/pingdotgg/t3code/commit/aa2b25d29b4b13ef1c2ee80e1afade1635de9ca8) | refactor(web): one wrap width for tooltips, plus a code variant (#13031) | Not applicable | N28 |
| [f3d3fe456927](https://github.com/pingdotgg/t3code/commit/f3d3fe4569274c5ee6f3c5435809330bef494a2d) | refactor(web): one vertical rhythm for dialog bodies (#13032) | Not applicable | N02 |
| [18aa7036732c](https://github.com/pingdotgg/t3code/commit/18aa7036732c0fc2e2ab79e54d5d8f59a160843d) | refactor(web): ghost-muted icons follow the text; add ghost-destructive (#13033) | Not applicable | N28 |
| [4c9fbf1f647c](https://github.com/pingdotgg/t3code/commit/4c9fbf1f647c83179af1c2acd78eded04ef07fe8) | refactor(web): InlineButton underlines on hover and takes a tone (#13034) | Not applicable | N28 |
| [d7736e06b6a4](https://github.com/pingdotgg/t3code/commit/d7736e06b6a4f8211f452908963fd23c4a04b2c4) | refactor(web): one minimum width for menus, three widths for popovers (#13035) | Not applicable | N28 |
| [da89e102f80b](https://github.com/pingdotgg/t3code/commit/da89e102f80b7ef6c8166d41cc202035f6ad75a4) | refactor(web): every textarea caps its growth; the diff comment box is a Textarea (#13036) | Not applicable | N28 |
| [016cdb962235](https://github.com/pingdotgg/t3code/commit/016cdb9622359967afe80a27c538056ef1e7b857) | refactor(web): stacked sidebar groups share one inset (#13037) | Not applicable | N02 |
| [e420ef98333c](https://github.com/pingdotgg/t3code/commit/e420ef98333c97205f836201cfeb94682a41893c) | refactor(web): Collapsible stays a plain container (#13038) | Not applicable | N28 |
| [0fa4a859f699](https://github.com/pingdotgg/t3code/commit/0fa4a859f6992be80302e2a7508e28b9b64e4991) | refactor(web): show more / show less are ordinary sidebar sub-rows (#13039) | Not applicable | N02 |
| [0ee02ebf2287](https://github.com/pingdotgg/t3code/commit/0ee02ebf228769beec4b240db6254ba2bad565af) | refactor(web): Empty has three sizes (#13040) | Not applicable | N28 |
| [4321aa24fd21](https://github.com/pingdotgg/t3code/commit/4321aa24fd21c2b5360cc58e84de4ccc09c801f9) | refactor(web): one row height for select, combobox and radio items (#13041) | Not applicable | N28 |
| [1c2f933892cc](https://github.com/pingdotgg/t3code/commit/1c2f933892cce833823fcfbf3099a51813ba67fb) | refactor(web): render menu and popover triggers through Button (#13042) | Not applicable | N02 |
| [83bbfa7e87e1](https://github.com/pingdotgg/t3code/commit/83bbfa7e87e16a3c21413fe55524c1d05ce88dab) | refactor(web): sidebar alerts use the standard variants; one keycap (#13043) | Not applicable | N28 |
| [f193a6863c49](https://github.com/pingdotgg/t3code/commit/f193a6863c494fdfa36a1ef3132e42a7d7b42926) | fix(server): bypass owned caches on explicit provider refresh (#13109) | Already covered | RefreshSettings re-inspects the native client. |
| [d7819c18813f](https://github.com/pingdotgg/t3code/commit/d7819c18813fa03b033cc1c9472c9acc0ffc0618) | chore(devices): bump agent-device to 0.21.12 (#13124) | Not applicable | N06 |
| [bed0b4cdad41](https://github.com/pingdotgg/t3code/commit/bed0b4cdad41408bb09b445a0a8f8ea0e5b62d1b) | chore(mobile): drop dead nitro-markdown tgz override and @expo/metro-runtime (#13148) | Not applicable | N01 |
| [242816af805b](https://github.com/pingdotgg/t3code/commit/242816af805b048aec653d6cc359ecc784ef98c5) | feat(web): show settings scope as a sentence at the top of the page (#13139) | Not applicable | N16 |
| [db9a0671bb89](https://github.com/pingdotgg/t3code/commit/db9a0671bb8921ebf5f9da16cd54f7e87f5ef3b0) | refactor(web): move settings scope pickers into breadcrumbs (#13165) | Not applicable | N16 |
| [7e65b226e7a7](https://github.com/pingdotgg/t3code/commit/7e65b226e7a74ecd13bf8b4738ea6bdd096cb1ba) | feat(auth): share provider sign-in flows and credential bindings (#12983) | Not applicable | N37 |
| [ec28eefa0d01](https://github.com/pingdotgg/t3code/commit/ec28eefa0d017e2b0cb65ba4269cee1adddb0e98) | refactor(mobile): git sheets use uniwind platform variants instead of className ternaries (#13161) | Not applicable | N01 |
| [340965191210](https://github.com/pingdotgg/t3code/commit/34096519121033ccf72cd43ea5350f9482df188a) | chore(mobile): name the two project favicon caches by their job (#13160) | Not applicable | N01 |
| [6dcde218ad6c](https://github.com/pingdotgg/t3code/commit/6dcde218ad6c07af4dde2a69517c9316adaf732f) | revert(mobile): git sheets back to Platform.OS ternaries (un-guarded uniwind variants broke both platforms) (#13169) | Not applicable | N01 |
| [5d493d67cedc](https://github.com/pingdotgg/t3code/commit/5d493d67cedc3401af42e77d6a238bc49204a09b) | docs(mobile): document the two mobile routes that intentionally skip deep links (#13164) | Not applicable | N01 |
| [5821b778fd29](https://github.com/pingdotgg/t3code/commit/5821b778fd29e9c5b6945f4e7150f92d19a0a3da) | refactor(mobile): break module cycles with focused extractions (#13151) | Not applicable | N01 |
| [f22331240ed1](https://github.com/pingdotgg/t3code/commit/f22331240ed1a97fae560e314bffb75fe39db8ca) | fix(server): generate PR diffs from branch changes (#13170) | Not applicable | N11 |
| [68607c5a9914](https://github.com/pingdotgg/t3code/commit/68607c5a99147710b92228e0642f6c0a642e32a6) | fix(web): preserve nested scroll behavior in chat timeline (#13167) | Already covered | Nested scroll ports are ignored. |
| [b954af60c402](https://github.com/pingdotgg/t3code/commit/b954af60c402ff0698f9f2d9a214695e817f07fe) | test(web): cover usage model ordering without static markup (#13104) | Not applicable | N02 |
| [e4422eec752e](https://github.com/pingdotgg/t3code/commit/e4422eec752e8cce46af2865d8fbd3d5b7468a2a) | fix(desktop): find linuxbrew node for the WSL backend (#7827) | Not applicable | N12 |
| [ca864a25b148](https://github.com/pingdotgg/t3code/commit/ca864a25b14813e390cbcaeb334b97cd61717ce3) | chore(models): use GPT-6 Luna for text generation (#13115) | Not applicable | N07 |
| [a493946bb42a](https://github.com/pingdotgg/t3code/commit/a493946bb42ab16e1d18285dac3e9e4603651330) | fix(mobile): keep ordinary offline outbox failures out of console.warn (#13144) | Not applicable | N01 |
| [96c4bfa0a27f](https://github.com/pingdotgg/t3code/commit/96c4bfa0a27f776f7cd17df7c7115c3de79ee399) | feat(providers): check remote compatibility ranges (#13130) | Not applicable | N04 |
| [2eaff08241a5](https://github.com/pingdotgg/t3code/commit/2eaff08241a51dadfd9ef5ad53cba74e6d0fceb4) | chore(lint): keep mobile theme escape-hatch allowlist honest (#13146) | Not applicable | N01 |
| [6975efd3dd52](https://github.com/pingdotgg/t3code/commit/6975efd3dd52c95dd978ff26f591139d8c5c2503) | fix(web): the pull request badge reads at the meta size again (#13175) | Not applicable | N02 |
| [b9199617d571](https://github.com/pingdotgg/t3code/commit/b9199617d57178b136ece0b8d649565cd463a0ff) | fix(mobile): uniwind platform variants stay guarded on both platforms (#13172) | Not applicable | N01 |
| [21be723addfc](https://github.com/pingdotgg/t3code/commit/21be723addfc01c797acd0b7a1e6547c9be6bff0) | refactor(mobile): git sheets use uniwind platform variants instead of className ternaries (#13185) | Not applicable | N01 |
| [db898a30629a](https://github.com/pingdotgg/t3code/commit/db898a30629a9d87ad0a9bc4bde1b13e672fa2d3) | refactor(mobile): remaining className platform ternaries become class variants (#13188) | Not applicable | N01 |
| [219c1d265d91](https://github.com/pingdotgg/t3code/commit/219c1d265d917f0e7083b9033f4014c0c2940463) | fix(web): align provider emails without clipping (#13174) | Not applicable | N16 |
| [151324b2c0bf](https://github.com/pingdotgg/t3code/commit/151324b2c0bf0ace789820b8ea4ec8ceb55753d9) | perf(mobile): recycle the default  home list and scope the snooze minute tick (#13149) | Not applicable | N01 |
| [0c91f687de60](https://github.com/pingdotgg/t3code/commit/0c91f687de606f62e157209f34e6aa74744b555a) | refactor(mobile): retire the legacy grouped thread list (#13183) | Not applicable | N01 |
| [5975ec78b7ea](https://github.com/pingdotgg/t3code/commit/5975ec78b7eaff654e6183b99f8de7a05591e99d) | fix(server): background PR checks spend less GitHub quota (#13189) | Not applicable | N11 |
| [18de6bb32805](https://github.com/pingdotgg/t3code/commit/18de6bb328059703b5d1b0661610076e3c0fa2d4) | fix(server): background PR sync reads summaries in batches (#13198) | Not applicable | N11 |
| [eafb4a9340bd](https://github.com/pingdotgg/t3code/commit/eafb4a9340bd2c2271b7c8fac7eddd55beea3972) | fix(server): GitHub PR lookups stop probing owner-qualified heads (#13200) | Not applicable | N11 |
| [aca3c87cdbfb](https://github.com/pingdotgg/t3code/commit/aca3c87cdbfb59ea5ed52fe2df82df508d57eb31) | chore(mobile): clear the legacy-list deletion fallout (#13203) | Not applicable | N01 |
| [29931dd3a9a4](https://github.com/pingdotgg/t3code/commit/29931dd3a9a4996a5282f59564a64e6be496d5c6) | hatch/variant functions (#13191) | Not applicable | N28 |
| [266d70cc4d88](https://github.com/pingdotgg/t3code/commit/266d70cc4d887b7459ec6557c430ec7bb92bd3fc) | refactor(web): context chips render through one ContextChip component (#13192) | Not applicable | N28 |
| [361b047f5f55](https://github.com/pingdotgg/t3code/commit/361b047f5f559abccacc802cd5ab9992b01473ea) | refactor(web): ui components drop their secondary className props (#13193) | Not applicable | N28 |
| [866fc907592c](https://github.com/pingdotgg/t3code/commit/866fc907592c9746b407484087d66c409e8034fa) | refactor(web): menu triggers and items stop restyling ui/menu (#13205) | Not applicable | N02 |
| [21079391c72d](https://github.com/pingdotgg/t3code/commit/21079391c72d1eeaa3c40f1cc5eb3e78abf0a846) | refactor(web): field controls stop restyling Input, Select, Combobox and Command (#13206) | Not applicable | N28 |
| [f46522777372](https://github.com/pingdotgg/t3code/commit/f46522777372be1705e3b7a1ab08756f236db5d2) | refactor(web): app code stops restyling sidebar, popover, table and misc ui exports (#13207) | Not applicable | N28 |
| [71c9b693137c](https://github.com/pingdotgg/t3code/commit/71c9b693137c070b4887b365c68659f1a2fb7aed) | refactor(web): Button consumers outside the composer stop restyling it (#13208) | Not applicable | N28 |
| [0e512db34d78](https://github.com/pingdotgg/t3code/commit/0e512db34d787b21b827f07c6a62fba7fb00369e) | refactor(web): composer controls own their look instead of restyling ui components (#13209) | Not applicable | N28 |
| [10882bea739a](https://github.com/pingdotgg/t3code/commit/10882bea739a98f11b942c20c6570eb6126dd010) | chore(web): no-restyle fails lint, and the ceiling gate goes (#13210) | Not applicable | N15 |
| [23c7ab901afa](https://github.com/pingdotgg/t3code/commit/23c7ab901afa66dc8e791a3873b600726a6ba8c7) | fix(mobile): recover from screen render errors (#13197) | Not applicable | N01 |
| [829af7b73a96](https://github.com/pingdotgg/t3code/commit/829af7b73a960d390a0b32becd37ad6483627aa4) | feat(web): navigate back and forward with mod+[ and mod+] (#13212) | Not applicable | N02 |
| [b21c5456548f](https://github.com/pingdotgg/t3code/commit/b21c5456548f17190e7f3688f9883b98cbf5221d) | fix(web): sort title matches by recent activity (#13219) | Not applicable | N02 |
| [eb6c170c42ea](https://github.com/pingdotgg/t3code/commit/eb6c170c42eaec20d063ce80ff810e97d4dddb37) | test(desktop): remove redundant keyring module-load test (#13220) | Not applicable | N09 |
| [f5ef0ddb90a8](https://github.com/pingdotgg/t3code/commit/f5ef0ddb90a8c36584e181b1913e7b8a5df30ffc) | chore(mobile): bump app version to 1.3.1 | Not applicable | N01 |
| [11e91f126405](https://github.com/pingdotgg/t3code/commit/11e91f126405883c0194da5b1e3ad39d81c48966) | feat(mobile): capture Live Activities and agent notifications in showcase screenshots (#13316) | Not applicable | N01 |
| [68fb7f4b8509](https://github.com/pingdotgg/t3code/commit/68fb7f4b850991ba0ca5bd0d7c4aa13d4e5d6bb7) | feat(mobile): manage environment and provider updates (#13302) | Not applicable | N01 |
| [e407f9bb090d](https://github.com/pingdotgg/t3code/commit/e407f9bb090d0cec231df21ad30e9e13186d9518) | fix(ci): shard release tests like pull request CI (#13321) | Not applicable | N15 |
| [effaab94e369](https://github.com/pingdotgg/t3code/commit/effaab94e36992ea9712c2a56d5ff602a3cead04) | fix(web): show previous worktree branch on second line (#13314) | Not applicable | N02 |
| [d4cd7d5c3312](https://github.com/pingdotgg/t3code/commit/d4cd7d5c33122473da22ae118c93477ef7e38310) | fix(providers): restore compatibility ranges for every harness (#13328) | Not applicable | N04 |
| [894d33419dce](https://github.com/pingdotgg/t3code/commit/894d33419dceb366eecda8e5e27c6fc043c3959e) | fix(preview): use the visible browser for new agent sessions (#13064) | Not applicable | N03 |
| [e4eb9977f0b0](https://github.com/pingdotgg/t3code/commit/e4eb9977f0b02b29cb2ee361b0c433c685e87615) | fix(server): stop replaying old agent alerts on restart (#13340) | Not applicable | N06 |
| [c0912debcb14](https://github.com/pingdotgg/t3code/commit/c0912debcb14895c1d907738094d4e997eb51572) | fix(web): use a brain icon for the effort dropdown (#13309) | Not applicable | N28 |
| [6b4b19096a7e](https://github.com/pingdotgg/t3code/commit/6b4b19096a7ea22fe0850972df4bce5a5a6c8263) | fix(desktop): SnapShot shortcut helper no longer adds a Dock icon on macOS (#13286) | Not applicable | N09 |
| [9030a60eafde](https://github.com/pingdotgg/t3code/commit/9030a60eafde905100a052f86beeb8be7642b5a9) | fix(web): composer chip rings no longer clip at the editor edge (#13301) | Not applicable | N22 |
| [21e2b7de0062](https://github.com/pingdotgg/t3code/commit/21e2b7de0062a3c1dca7209ca48b736dd9a36c84) | fix(web): switches announce their real state to screen readers (#11580) | Ported | Expose switch checked state to screen readers (#11580). |
| [f1add18ae156](https://github.com/pingdotgg/t3code/commit/f1add18ae1560f509405a5eece07f3cfb1cc850f) | fix(shared): preserve final quoted empty CSV records (#11425) | Not applicable | N27 |
| [78af372cf46a](https://github.com/pingdotgg/t3code/commit/78af372cf46a80147f05183b69d7a8339259ca37) | feat(web): add an interactive 3D device workspace (#12787) | Not applicable | N03 |
| [9383f4ad7128](https://github.com/pingdotgg/t3code/commit/9383f4ad71285b9f9a6a9b65f2f961ea2cf4308f) | chore: add scratchyone to vouched list (#13353) | Not applicable | N15 |
| [e67abcf798f8](https://github.com/pingdotgg/t3code/commit/e67abcf798f8c4d8458755e3b4dde02c2c1f628b) | feat(observability): honor the OpenTelemetry kill switch (#13355) | Not applicable | N03 |
| [b2b43bef7344](https://github.com/pingdotgg/t3code/commit/b2b43bef73447c483ceae486890cb79f01c369cb) | fix(server): preserve racy edits in review diff previews (#12613) | Not applicable | N06 |
| [cb1a3f34670b](https://github.com/pingdotgg/t3code/commit/cb1a3f34670b06148d88806a9534772879c30ecd) | fix(web): show repository names on linked pull requests (#13061) | Not applicable | N02 |
| [80fc23ac32cd](https://github.com/pingdotgg/t3code/commit/80fc23ac32cd8fe5011a94420cff1f469367f1fc) | lint/unknown and static (#13366) | Not applicable | N38 |
| [315fcca10a82](https://github.com/pingdotgg/t3code/commit/315fcca10a826d8aa1cbdffc3ef5b5eb5a1e2f9d) | fix(web): web colors come from theme tokens (#13371) | Not applicable | N28 |
| [f26ee083fe41](https://github.com/pingdotgg/t3code/commit/f26ee083fe41720207fb10616359500dcecc19d3) | fix(web): appearance classes use theme tokens and scale values (#13397) | Not applicable | N28 |
| [ffb5fcce7336](https://github.com/pingdotgg/t3code/commit/ffb5fcce7336f9b3d5bde3a5f496a816907663af) | fix(server): keep Codex's reset answer when the re-probe fails (#13363) | Not applicable | N21 |
| [84c436bcf714](https://github.com/pingdotgg/t3code/commit/84c436bcf714076572804d005eb7123ca1588188) | fix(mobile): branch search finds remote and space-typed branches (#13454) | Not applicable | N11 |
| [7cfb4987fe27](https://github.com/pingdotgg/t3code/commit/7cfb4987fe27da3e1d4e86abacae3a1fc12af5bb) | chore(ci): use GPT 6 Sol Max for check agents (#13473) | Not applicable | N15 |
| [3412097bff7f](https://github.com/pingdotgg/t3code/commit/3412097bff7ffa62ef79e735c65d1e5cff71c356) | feat(server): show and redeem Claude banked resets (#13118) | Not applicable | N21 |
| [87d842801921](https://github.com/pingdotgg/t3code/commit/87d84280192195b338e27acf2333771a13d04e61) | fix(observability): a malformed OTEL_RESOURCE_ATTRIBUTES no longer stops startup (#13469) | Not applicable | N03 |
| [cdb26fe63549](https://github.com/pingdotgg/t3code/commit/cdb26fe63549df633751ab0eed145b9357b30a21) | fix(antigravity): let Stop end commands that outlived their turn (#13388) | Not applicable | N13 |
| [f61f979c7387](https://github.com/pingdotgg/t3code/commit/f61f979c7387d9f20d7609dbf8322ca570f3f63b) | fix(web,mobile): drop the baked-in tile from the Antigravity icon (#13373) | Not applicable | N39 |
| [53456bc0129f](https://github.com/pingdotgg/t3code/commit/53456bc0129f61325dbaa498e2399c33b4360cb8) | fix(marketing): use the official OpenCode and Antigravity logos (#13365) | Not applicable | N19 |
| [e759847f932c](https://github.com/pingdotgg/t3code/commit/e759847f932c0de08261ec98a5fac3bae037120f) | fix(acp): keep one answer when a running tool reports progress (#13386) | Ported | Shown ACP tool progress does not split an answer. |
| [66129c6fd518](https://github.com/pingdotgg/t3code/commit/66129c6fd518aa01d873ebb47060b41c17312d60) | feat(web): run shell commands from chat in the thread terminal (#13060) | Not applicable | N16 |
| [f3cb2a1fee65](https://github.com/pingdotgg/t3code/commit/f3cb2a1fee65a55a4083c4cfe9317ca0d1e7c1d5) | fix(antigravity): keep Windows runtime unpacking under MAX_PATH (#13389) | Not applicable | N13 |
| [567783ecdc65](https://github.com/pingdotgg/t3code/commit/567783ecdc658c8bf86c4c4980df5df8ebdd9037) | fix(codex): the protocol generator runs again on Effect rc.115 (#13480) | Not applicable | N31 |
| [d5d48742c9ab](https://github.com/pingdotgg/t3code/commit/d5d48742c9ab00dff82fe8bcbc8da0ba68748959) | feat(codex): require Codex 0.156 and regenerate its protocol (#13481) | Already covered | Used Codex protocol methods are unchanged; the added permissions request was separately ported. |
| [0109670411d1](https://github.com/pingdotgg/t3code/commit/0109670411d13b9e380766e98b386b2f756ecf5b) | feat(threads): add per-thread auto-settle switch (#11846) | Not applicable | N03 |
| [8251c8de78a0](https://github.com/pingdotgg/t3code/commit/8251c8de78a021014b83ffe986dc52a480d71e41) | fix(web): working and monitoring threads fade in the sidebar again (#13506) | Not applicable | N02 |
| [a36af0637a63](https://github.com/pingdotgg/t3code/commit/a36af0637a63ebf316beb5db5670be226df01cbc) | fix(server): streamed section titles wait for the text under them (#13504) | Not applicable | N06 |
| [720490adc9b8](https://github.com/pingdotgg/t3code/commit/720490adc9b8f3676c614f3611d8c7fd9f4dd368) | fix(web): normalize disabled control opacity (#11441) | Not applicable | N28 |
| [9a91177c3eca](https://github.com/pingdotgg/t3code/commit/9a91177c3ecadaaa088d00e467491de4be9d7ce3) | fix(web): sidebar Back always returns to the main app (#13516) | Not applicable | N02 |
| [d4a33457cb0d](https://github.com/pingdotgg/t3code/commit/d4a33457cb0da797728f4846a9da8592d7d81d36) | fix(desktop): desktop updates reconnect in seconds, not minutes (#12006) | Not applicable | N03 |
| [8d7b5e998c72](https://github.com/pingdotgg/t3code/commit/8d7b5e998c72b4674326cbac7d23bafe9b222668) | fix(connect): remove tunnels after hosts go offline (#9386) | Not applicable | N06 |
| [9c524d57718e](https://github.com/pingdotgg/t3code/commit/9c524d57718e639a6abe0a5fa3cbf6e9b086df86) | fix(mobile): capture a lit 6.9-inch lock screen in the agent-activity showcase (#13522) | Not applicable | N01 |
| [72447f23d9d9](https://github.com/pingdotgg/t3code/commit/72447f23d9d9102f647f95f548aadf92edd5b400) | feat(grok): offer one-click updates through `grok update` (#13523) | Not applicable | N04 |
| [59abcd67ac2c](https://github.com/pingdotgg/t3code/commit/59abcd67ac2c54bd63a422e92fa2e31b2c667b6d) | fix(mobile): make Android subscription usage widgets scrollable (#13474) | Not applicable | N01 |
| [fd4651005810](https://github.com/pingdotgg/t3code/commit/fd465100581055383126aab922a8da2c5d7952fd) | fix(web): keep sidebar terminal pulses in sync (#12962) | Not applicable | N02 |
| [46f3c2ca2c99](https://github.com/pingdotgg/t3code/commit/46f3c2ca2c99ceab0547452ac863232944621fce) | feat(web): add iPhone Duo 3D controls (#12813) | Not applicable | N03 |
| [d23eab13d9d9](https://github.com/pingdotgg/t3code/commit/d23eab13d9d91697c5d4e7a458213b3f6473bc96) | fix(relay): export tunnel cleanup counters to Axiom (#13528) | Not applicable | N33 |
| [0cab7d5ab978](https://github.com/pingdotgg/t3code/commit/0cab7d5ab978a8adaee6827f81693249d0dfeee5) | fix(server): Grok accounts with no usage yet no longer vanish from Limits (#12799) | Not applicable | N21 |
| [91e53e501c9e](https://github.com/pingdotgg/t3code/commit/91e53e501c9e7eb03f0ea4a5b72221d08eff9c5c) | fix(server): report the Grok account email so usage limits merge across environments (#12588) | Not applicable | N21 |
| [3e2370fbbf1e](https://github.com/pingdotgg/t3code/commit/3e2370fbbf1efacb341a68e6f28efb06b695e148) | feat(web): add usage page keybinding (#9434) | Not applicable | N02 |
| [568c9bc4d0ef](https://github.com/pingdotgg/t3code/commit/568c9bc4d0ef0dddf74d813619c9877bd35624d7) | chore: clear Effect language service suggestions (#13536) | Not applicable | N31 |
| [fc46b8c3dd69](https://github.com/pingdotgg/t3code/commit/fc46b8c3dd691ed2e39da28a2407e1b1597f2832) | ci(relay): add a forced manual relay deploy (#13550) | Not applicable | N15 |
| [ebdcda135e31](https://github.com/pingdotgg/t3code/commit/ebdcda135e31693c5283f8ac409019bd69e07f8e) | fix(web): selected text stays visible on a revealed file line (#13548) | Not applicable | N02 |
| [29abbf9b424b](https://github.com/pingdotgg/t3code/commit/29abbf9b424b122c52bb20dccffac68bee367c09) | fix(web): collapsed composer bar stops flipping its labels while you scroll (#13555) | Not applicable | N02 |
| [ab70c8943f48](https://github.com/pingdotgg/t3code/commit/ab70c8943f48c08e67e3597f62941c169bc58c16) | fix(clients): sync status no longer flickers when opening running threads (#13551) | Not applicable | N16 |
| [6391be272956](https://github.com/pingdotgg/t3code/commit/6391be2729566a1fe1045ff8617f003580b37bfd) | fix(server): newer Codex models get T3 Code's instructions again (#13547) | Not applicable | N40 |
| [9957349c0a06](https://github.com/pingdotgg/t3code/commit/9957349c0a066a9efc657600fcb901c9b9cca34b) | feat(web): control Android foldables in the Device panel (#13534) | Not applicable | N03 |
| [4f27a8463105](https://github.com/pingdotgg/t3code/commit/4f27a8463105662ddfc834e01bca375d5124be79) | fix(mcp): preview snapshots fit in the agent's tool output again (#13558) | Not applicable | N06 |
| [0c84b4289cc9](https://github.com/pingdotgg/t3code/commit/0c84b4289cc9dee9af770a823f812ea1e8bea05d) | fix(web): paste after clicking away from the composer lands in it again (#13553) | Not applicable | N02 |
| [99641fd09a50](https://github.com/pingdotgg/t3code/commit/99641fd09a509feb644b3c8ef718c12c60cf436c) | feat(desktop): keep running threads synced in the background (#13554) | Not applicable | N02 |
| [1e192b25583d](https://github.com/pingdotgg/t3code/commit/1e192b25583dd682324acd52f396be3d2ec9ee23) | fix(mcp): preview errors tell agents what to do instead (#13559) | Not applicable | N06 |
| [df9826f0806c](https://github.com/pingdotgg/t3code/commit/df9826f0806cc4874cd79f24a42a1e9e506030c2) | feat(web): agents working banner links to the Agents panel (#13572) | Not applicable | N02 |
| [7b84431161a6](https://github.com/pingdotgg/t3code/commit/7b84431161a64c9f9e9b57637dec23474a1b4236) | fix(web): size the Android fold model from the inner display (#13574) | Not applicable | N07 |
| [a107f8a0792c](https://github.com/pingdotgg/t3code/commit/a107f8a0792c3add90d398602998c55c3ebaef27) | fix(clients): a preview app no longer knocks the desktop's own server offline (#13577) | Not applicable | N07 |
| [20f0ff1781fb](https://github.com/pingdotgg/t3code/commit/20f0ff1781fb02898a6fa5f97f76f36079cff0b8) | fix(web): keep nested task states out of parent bullets (#11477) | Not applicable | N02 |
| [c13f7d93fff0](https://github.com/pingdotgg/t3code/commit/c13f7d93fff0f578886985a3f673b4b21b7f6ac8) | feat(release): ship a Linux .deb that updates itself (#13575) | Not applicable | N03 |
| [d10bd1360ace](https://github.com/pingdotgg/t3code/commit/d10bd1360ace3ab1dbccc1ee706c9169982d98ae) | perf(desktop): cache compiled JavaScript between launches (#13501) | Not applicable | N03 |
| [b3de243d5ed4](https://github.com/pingdotgg/t3code/commit/b3de243d5ed44783c4e7990d8e63cb519d390a03) | fix(dev): one t3.json setup action that works on every OS (#13589) | Not applicable | N15 |
| [13d6b30515e9](https://github.com/pingdotgg/t3code/commit/13d6b30515e9d44bb4cc2d223b8b3179c2ba4c79) | fix(web): new worktree threads no longer say "checkout" during setup (#13590) | Not applicable | N02 |
| [86054b6df2c5](https://github.com/pingdotgg/t3code/commit/86054b6df2c5e38b6e27b980eae2716bfe1da202) | fix(desktop): `t3 app` keeps working after a second desktop app quits (#13585) | Not applicable | N09 |
| [e3e7cc3fc2e0](https://github.com/pingdotgg/t3code/commit/e3e7cc3fc2e02966efe4efc45216dadcd046bfbb) | fix(usage): price Claude fast-mode requests at the fast rate (#13599) | Not applicable | N06 |
| [e55e7315e4a4](https://github.com/pingdotgg/t3code/commit/e55e7315e4a4c3870711acd6825ea5347c2d4812) | fix: update OpenAI logo to current brand asset (#13611) | Ported | Update the OpenAI brand glyph (#13611). |
| [1c1270663880](https://github.com/pingdotgg/t3code/commit/1c1270663880903b16568594c4bf5f322407903a) | fix(mobile): render assigned project icons in chat list (#12810) | Not applicable | N01 |
| [e5a46d6c5d00](https://github.com/pingdotgg/t3code/commit/e5a46d6c5d00b89afba5274a94d42428c8d79763) | feat(usage): read cursor, opencode, and antigravity history (#10409) | Not applicable | N21 |
| [d06f0ff1048d](https://github.com/pingdotgg/t3code/commit/d06f0ff1048d42a156824765316c5a2dc54f5bca) | fix(sqlite): retry failed statement preparations (#10584) | Not applicable | N27 |
| [7a12aff471ff](https://github.com/pingdotgg/t3code/commit/7a12aff471ffe2b22b9fee495b04b32c43f45a37) | fix(mobile): scale Android controls with appearance text size (#13356) | Not applicable | N01 |
| [a4f6078bebcb](https://github.com/pingdotgg/t3code/commit/a4f6078bebcb3283e53e1cea247257136ce82cc8) | fix(web): return focus to the composer after saving a citation note (#13450) | Not applicable | N02 |
| [fd996d15e220](https://github.com/pingdotgg/t3code/commit/fd996d15e220144ac6e66670169aa87d074ce846) | feat(observability): honor the standard OTLP endpoint, headers, and protocol variables (#13492) | Not applicable | N06 |
| [1a0c915c4102](https://github.com/pingdotgg/t3code/commit/1a0c915c41024ddb47e16befd9fc730c1173af02) | fix(terminal): settling a thread closes its idle shells (#13673) | Not applicable | N06 |
| [f5bd2fddbf00](https://github.com/pingdotgg/t3code/commit/f5bd2fddbf00021b29c1dbe32210240886806718) | fix(server): load Cursor keyring with createRequire (#13678) | Not applicable | N41 |
| [4293433eccbe](https://github.com/pingdotgg/t3code/commit/4293433eccbe6c6661acd020e584ae5d2234bf1e) | perf(server): avoid rereading unchanged files in review previews (#13395) | Not applicable | N06 |
| [2598ce7a5ab0](https://github.com/pingdotgg/t3code/commit/2598ce7a5ab0b25e32e0b67b084a9ec23cd8144f) | feat(desktop): honor the standard OTLP variables in the main process and WSL backends (#13641) | Not applicable | N09 |
| [ed809f7ad274](https://github.com/pingdotgg/t3code/commit/ed809f7ad274ebccdfbf029ccfd7ff2582fd486a) | fix(web): restore compact provider instance badges (#13700) | Not applicable | N05 |
| [525af2d1ad4b](https://github.com/pingdotgg/t3code/commit/525af2d1ad4b6de847ea658c37a081fd7afe8ef6) | perf(mobile): lighter Home rows away from the viewport (#13702) | Not applicable | N01 |
| [aade3a69a853](https://github.com/pingdotgg/t3code/commit/aade3a69a8536be98d2fe9b0fb9193289b70e90b) | fix(usage): price Cursor cache savings by base model (#13731) | Not applicable | N06 |
| [595a1e1f537f](https://github.com/pingdotgg/t3code/commit/595a1e1f537f403eba408817c2aba35e61c2ac29) | perf(mobile): render Home rows further ahead while scrolling (#13705) | Not applicable | N01 |
| [2a9832b80190](https://github.com/pingdotgg/t3code/commit/2a9832b8019039f812706639c11ccc24ce4c4e05) | refactor(observability): name each service after its application (#13699) | Not applicable | N03 |
| [c216ba4dade6](https://github.com/pingdotgg/t3code/commit/c216ba4dade683bf3a7c96bf63b33a85afc1a16c) | perf(server): stop remapping every thread on each thread event (#13720) | Not applicable | N06 |
| [5660ab5cb556](https://github.com/pingdotgg/t3code/commit/5660ab5cb55633a1657e715b0af9bd1e449c8a63) | Remove unused items tracking from Claude adapter state (#13718) | Not applicable | N42 |
| [20885cecf77e](https://github.com/pingdotgg/t3code/commit/20885cecf77e205b8931ca1b29f0a2832836a157) | feat(observability): name the command on subprocess spans (#13701) | Not applicable | N06 |
| [7c901a37c4e8](https://github.com/pingdotgg/t3code/commit/7c901a37c4e8f35c408a27dbd1bd2efea06a74f5) | fix(cli): t3 triage points agents at log files that exist (#13685) | Not applicable | N06 |
| [8aa5be2f02c3](https://github.com/pingdotgg/t3code/commit/8aa5be2f02c3b4b9b5b3b86acbf5f2b0467438f4) | fix(server): the SQLite WAL file shrinks back after large writes (#13684) | Not applicable | N06 |
| [75bf92e652f7](https://github.com/pingdotgg/t3code/commit/75bf92e652f72cb0062c102236aeb130d1d714b3) | feat(cli): summarize the server trace file from the command line (#13698) | Not applicable | N06 |
| [6530de0339d2](https://github.com/pingdotgg/t3code/commit/6530de0339d2ca49957d0039133c49e3a08557f7) | perf(server): pull request sync reads only threads with linked pull requests (#13704) | Not applicable | N43 |
| [1dc8cbe6d14a](https://github.com/pingdotgg/t3code/commit/1dc8cbe6d14a7b4e131a2806f43bde8b60d6aaa7) | feat(observability): write a server heap snapshot on SIGUSR2 (#13694) | Not applicable | N06 |
| [574b18090281](https://github.com/pingdotgg/t3code/commit/574b18090281225de3449816c8366f0ee9ab886c) | perf(server): shutdown no longer rewrites every stopped session row (#13688) | Not applicable | N44 |
| [b6eefc926ae3](https://github.com/pingdotgg/t3code/commit/b6eefc926ae305f4cb85cca797668b010cee1136) | perf(server): build the thread list snapshot without decoding it twice (#13693) | Not applicable | N06 |
| [b2577d6eface](https://github.com/pingdotgg/t3code/commit/b2577d6eface9589d120bdc53e4ec0431b1e9331) | fix(client): slow servers finish loading the thread list instead of loading it twice (#13683) | Not applicable | N07 |
| [6989856aa630](https://github.com/pingdotgg/t3code/commit/6989856aa630611dd7d6dcc8cfe9f97456816284) | perf(web): hidden terminal drawers no longer keep full thread history in memory (#13686) | Not applicable | N02 |
| [3b0a495b0e05](https://github.com/pingdotgg/t3code/commit/3b0a495b0e05d11dbf88ff2148e7c21b49508172) | perf(server): per-thread settlement and PR checks no longer rebuild the whole thread list (#13691) | Not applicable | N06 |
| [ecd3237b184e](https://github.com/pingdotgg/t3code/commit/ecd3237b184e87af5d744a9dc28422cc1eb68e1a) | fix(mobile): running threads open at the latest message (#13530) | Not applicable | N01 |
| [ee18e56f902f](https://github.com/pingdotgg/t3code/commit/ee18e56f902f2ccd74dffdc50363051be02ae0ab) | feat(observability): record event loop stalls in the server trace (#13697) | Not applicable | N06 |
| [999161ef844a](https://github.com/pingdotgg/t3code/commit/999161ef844a7a38c8a6d08195c1240759169832) | perf(server): stop re-running git for every project each minute (#13689) | Not applicable | N06 |
| [94c42162b443](https://github.com/pingdotgg/t3code/commit/94c42162b443072ca59a0733c02ee54229a61fef) | fix(usage): hide the Cursor keychain prompt when Cursor isn't set up (#13714) | Not applicable | N03 |
| [04c15f34b756](https://github.com/pingdotgg/t3code/commit/04c15f34b756aadbff4c93ff07a217fa2398e5d5) | feat(web): add chat width setting for wide screens (#11594) | Not applicable | N16 |
| [4408bf5a4fb0](https://github.com/pingdotgg/t3code/commit/4408bf5a4fb0c97373d57abfdf6f541e315b1e67) | fix(opencode): accept  serve ready line when spawning server (#13651) | Not applicable | N45 |
| [1e844a3d0647](https://github.com/pingdotgg/t3code/commit/1e844a3d0647e9d2ca87d91b2fab5833bba19d08) | fix(editors): stop treating the agy CLI as the Antigravity IDE (#7079) | Not applicable | N06 |
| [eeea71a88108](https://github.com/pingdotgg/t3code/commit/eeea71a88108c375c04853f58c298a518a54acaf) | fix(web): make the empty workspace draggable on desktop (#13713) | Not applicable | N02 |
| [dd8332da5735](https://github.com/pingdotgg/t3code/commit/dd8332da57355bccd7e666289267f2c94597debd) | fix(server): installed editors no longer vanish when discovery is slow (#13669) | Not applicable | N06 |
| [29a32660d3cb](https://github.com/pingdotgg/t3code/commit/29a32660d3cb2ade1f13283f889b519f73b24fbd) | fix(git): exclude SSH ports from provider URLs (#12537) | Not applicable | N11 |
| [5d707bb71de1](https://github.com/pingdotgg/t3code/commit/5d707bb71de19eb0eb6fe4688602ee853aec2e3c) | fix(web): Mod+B bolds on non-Latin layouts (#13409) | Not applicable | N02 |
| [a21b42cec478](https://github.com/pingdotgg/t3code/commit/a21b42cec478b093cdc50cc2105b5368ea8b3546) | fix(server): prune expired replay-protection files from the secrets directory (#13695) | Not applicable | N06 |
| [8bc9b78f8270](https://github.com/pingdotgg/t3code/commit/8bc9b78f827072a89707f6e4b577710e27263697) | fix(web): terminal links drop a trailing colon (#13408) | Not applicable | N02 |
| [4923ff417f15](https://github.com/pingdotgg/t3code/commit/4923ff417f15fe1930fbb679e548706b55f7cde8) | fix(server): bump node-pty to 1.2.0-beta.15 for linux-arm64 prebuild (#13748) | Not applicable | N15 |
| [92f0af24a359](https://github.com/pingdotgg/t3code/commit/92f0af24a359f64df8f480049913cbdbd9ea6d29) | Show a focus ring on sidebar thread and draft rows (#13344) | Not applicable | N02 |
| [dab9561cae88](https://github.com/pingdotgg/t3code/commit/dab9561cae8817bea24442c1dc7e28996de9c52e) | fix(mobile): keep composer within folded screen after resume (#13310) | Not applicable | N01 |
| [2679d279ce4b](https://github.com/pingdotgg/t3code/commit/2679d279ce4b4879290bc756cf219b10f77e1873) | fix(server): let OpenCode generate session titles (#13368) | Not applicable | N06 |
| [def34c28b884](https://github.com/pingdotgg/t3code/commit/def34c28b8841002bb97a3238b82ac758cbdaa0f) | fix(server): let Antigravity inspect unsupported files by path (#13339) | Not applicable | N13 |
| [fe6388f01d5a](https://github.com/pingdotgg/t3code/commit/fe6388f01d5ae9f9db43079c6f5b2c31869067ca) | fix(mobile): link URLs with ports and single-label hosts (#13795) | Not applicable | N08 |
| [daafcc4a972b](https://github.com/pingdotgg/t3code/commit/daafcc4a972b25eb762962cb72575b1a2e583d61) | feat(web): add keyboard navigation for usage (#10158) | Not applicable | N02 |
| [99efaeab5829](https://github.com/pingdotgg/t3code/commit/99efaeab58297c7e346ee91a4f28044faed03c59) | perf(observability): stop writing empty spans on spawns, projected events, and idle polls (#13756) | Not applicable | N06 |
| [8b873eab0d74](https://github.com/pingdotgg/t3code/commit/8b873eab0d74b731bc0f387be84a8561eace789f) | perf(server): opening Diagnostics no longer loads the whole trace ring into memory (#13763) | Not applicable | N06 |
| [6f97b0f66a51](https://github.com/pingdotgg/t3code/commit/6f97b0f66a51fbafabd54a5616da8827ebdf6020) | perf(clients): sort projects and settled threads without re-parsing dates per comparison (#13759) | Not applicable | N03 |
| [294dd123410c](https://github.com/pingdotgg/t3code/commit/294dd123410c77eead2810a052c31fb0cfd3e0c1) | fix(observability): the renderer trace proxy stops tracing itself (#13761) | Not applicable | N06 |
| [1d6f23b51929](https://github.com/pingdotgg/t3code/commit/1d6f23b51929117d5e69ded8c5c0f8e195d75d4a) | perf(server): background sweeps only read threads that can still settle (#13765) | Not applicable | N06 |
| [9151ea407acc](https://github.com/pingdotgg/t3code/commit/9151ea407accd1b7af575dcdd514e2b2e3fe6cbb) | perf(clients): saving the thread list cache no longer freezes the UI (#13767) | Not applicable | N03 |
| [887266695780](https://github.com/pingdotgg/t3code/commit/8872666957802b37c9fdd212e01f0024bcc102c6) | perf(server): cut idle wakeups from the Connect relay and session reaper (#13774) | Not applicable | N44 |
| [81e0491d2bde](https://github.com/pingdotgg/t3code/commit/81e0491d2bdec153bf1ee12d4e1deab3909b7428) | fix(mobile): keep trailing underscores and tildes in autolinked URLs (#13807) | Not applicable | N08 |
| [295d7cba09bb](https://github.com/pingdotgg/t3code/commit/295d7cba09bb4b4084482a74b5a42a0cac72a0b1) | fix(web): queued messages send while their thread is not open (#13764) | Not applicable | N10 |
| [95030dc67488](https://github.com/pingdotgg/t3code/commit/95030dc674883f0f2a7fd034b32ce742c8cf55d0) | fix(server): background git status fetches no longer fill the disk with failed repacks (#13812) | Not applicable | N06 |
| [75d63d64c554](https://github.com/pingdotgg/t3code/commit/75d63d64c554a425adc485f1205ef03ec50f8564) | fix(mobile): thread list shows the pull request icon instead of # (#13742) | Not applicable | N01 |
| [393d595984f6](https://github.com/pingdotgg/t3code/commit/393d595984f66df08504e1659997e12ad551bce9) | fix(accessibility): correct control announcements and sidebar traversal (#13491) | Not applicable | N30 |
| [10bb59bf0612](https://github.com/pingdotgg/t3code/commit/10bb59bf061252f5edadf96bc8139911b3fed7fd) | fix(usage): tolerate newer provider variants (#10076) | Not applicable | N07 |
| [d6802b4acfbb](https://github.com/pingdotgg/t3code/commit/d6802b4acfbb6bfb669e0fa225539920622f7670) | fix(usage): omit Cursor warning when no login is saved (#13820) | Not applicable | N06 |
| [3dae78f33b1e](https://github.com/pingdotgg/t3code/commit/3dae78f33b1e3c7534feb51ca41a407ab2536b83) | fix(usage): identify client version mismatches (#8208) | Not applicable | N03 |
| [dd582dee3697](https://github.com/pingdotgg/t3code/commit/dd582dee36976ccfd2a896cf50c3ee957ca7b9be) | fix(web): stop mistaking offline servers for updates (#13083) | Not applicable | N02 |
| [d110f9867041](https://github.com/pingdotgg/t3code/commit/d110f9867041f0466bed9b032c15efcfcefcbfc5) | test(usage): assert contract mismatch details (#13861) | Not applicable | N27 |
| [ea7d46ac107e](https://github.com/pingdotgg/t3code/commit/ea7d46ac107e6a3461b81065ebc86dfd5483095f) | fix(build): validate Linux node-pty prebuilds in Windows artifacts (#13867) | Not applicable | N15 |
| [679c34c09619](https://github.com/pingdotgg/t3code/commit/679c34c096193446ceed62e633387d91fe2cfff8) | fix(clients): hide duplicate Cursor Keychain prompts (#13870) | Not applicable | N41 |
| [74ee5153e05e](https://github.com/pingdotgg/t3code/commit/74ee5153e05eb183dabec413ff9025d83d2058e4) | feat(observability): honor OTEL_*_EXPORTER=none per signal (#13736) | Not applicable | N09 |
| [c9a0e8a11927](https://github.com/pingdotgg/t3code/commit/c9a0e8a119271765196dec38ef91395713da4213) | fix(web): align reasoning arrows with tool calls (#13850) | Not applicable | N18 |
| [cb97415948cc](https://github.com/pingdotgg/t3code/commit/cb97415948cc2c0662a5504ccfe7e97981825ed5) | fix(server): let agents use simctl and adb alongside device tools (#13908) | Not applicable | N20 |
| [ab099178a7b7](https://github.com/pingdotgg/t3code/commit/ab099178a7b7f9728843e90fc95ed90bb61d710d) | fix(web): environment status tooltip resizes when the status changes (#13845) | Not applicable | N02 |
| [a727d1d97690](https://github.com/pingdotgg/t3code/commit/a727d1d97690c9bb12cee5760e91cfd1aa7c017d) | test(relay): remove constant-restating database mode test (#13932) | Not applicable | N33 |
| [de251fc2971a](https://github.com/pingdotgg/t3code/commit/de251fc2971a884cb5b1305ba4daf309dc8cccb0) | fix(web): continue onboarding after incomplete history imports (#13935) | Not applicable | N02 |
| [94f92a7a386a](https://github.com/pingdotgg/t3code/commit/94f92a7a386a26c98892b24fabdd0ea9fa804ce3) | fix(server): let Claude abort turns before closing sessions (#13999) | Ported | Claude Stop interrupts through its native control channel. |
| [d15210cd3da7](https://github.com/pingdotgg/t3code/commit/d15210cd3da79f9a1a495a6309d912d76362a046) | fix(web): prevent usage summary shift for unpriced records (#13991) | Not applicable | N02 |
| [ed57bed8e71b](https://github.com/pingdotgg/t3code/commit/ed57bed8e71b2e89a9350c94bda0dacc71cd4817) | fix(web): skip unchanged shortcut modifier updates - 195x typing latency improvement (#13884) | Not applicable | N02 |
| [ba79610d166d](https://github.com/pingdotgg/t3code/commit/ba79610d166dcc7a9b8beb013d18304bc6261307) | fix(web,mobile): say that removing a T3 Connect environment keeps its account registration (#14127) | Not applicable | N03 |
| [72330e22c0d0](https://github.com/pingdotgg/t3code/commit/72330e22c0d08159641ab10579eb75393c4d95b6) | feat(models): add Claude Sonnet 5.5 (#14152) | Ported | Reviewed Claude model-manifest data is retained by the local extraction. |
| [b528a701102f](https://github.com/pingdotgg/t3code/commit/b528a701102f95089544596856a3f208f4404613) | fix(server): restore Windows terminal startup after node-pty upgrade (#13927) | Not applicable | N06 |
| [7733bc839e23](https://github.com/pingdotgg/t3code/commit/7733bc839e23e19467a70b11a0bcedf6ffe8f1a3) | fix(desktop): make Linux URL handlers discoverable with the app icon (#8673) | Not applicable | N09 |
| [b21f3b719133](https://github.com/pingdotgg/t3code/commit/b21f3b71913370a1a650a61fd9082f992d40e657) | fix(server): prevent Windows PTY helper crashes in watch mode (#14179) | Not applicable | N08 |
| [38f3c62304bf](https://github.com/pingdotgg/t3code/commit/38f3c62304bfe79ea6c9f75a17afb732b555f48d) | fix(shared): local-path remotes no longer crash legacy PR link projection (#13463) | Not applicable | N27 |
| [adfc9240eab7](https://github.com/pingdotgg/t3code/commit/adfc9240eab7c5db376a87591b497d8144289257) | fix(server): preserve usage in oversized transcript records (#13650) | Not applicable | N06 |
| [e518866d2860](https://github.com/pingdotgg/t3code/commit/e518866d28603da273a9357516dd1aef919f1353) | fix(server): OpenCode stop no longer hangs when a turn is interrupted before submission (#12003) | Already covered | Stop before prompt dispatch is remembered and prevents dispatch. |
| [5da55957d48d](https://github.com/pingdotgg/t3code/commit/5da55957d48dbd79c1a05b72bdf2479ae0956b1c) | feat: save Bitbucket credentials from Source Control settings (#14103) | Not applicable | N11 |
| [76fa23df2ef0](https://github.com/pingdotgg/t3code/commit/76fa23df2ef09c8d9624cc6ce1273d32452b3d1c) | fix(provider): mark OpenCode  incompatible (#14198) | Not applicable | N04 |
| [d2c9281b8112](https://github.com/pingdotgg/t3code/commit/d2c9281b8112dc3b2991642c4bdb985e4b08b9bb) | fix(web): pull request badge sits on the sidebar row's baseline (#14007) | Not applicable | N02 |
| [27bdf1aa14e6](https://github.com/pingdotgg/t3code/commit/27bdf1aa14e6b946b72fb7013062084ca70796a6) | feat(codex): connect ChatGPT accounts with managed authentication (#14290) | Not applicable | N46 |
| [5e83e99c9541](https://github.com/pingdotgg/t3code/commit/5e83e99c95411c4da9a23600ee34b3621b2cd275) | Update currentModels in model-manifest.json | Ported | Reviewed Claude model-manifest data is retained by the local extraction. |
| [2cbc24fcae2b](https://github.com/pingdotgg/t3code/commit/2cbc24fcae2b5649d7b60b68da72053a37fa82d5) | chore(release): prepare v0.0.43 | Not applicable | N15 |
| [451afcb22d93](https://github.com/pingdotgg/t3code/commit/451afcb22d93f06cb24f9bc16703404564952553) | fix(codex): Pro Max accounts load, so Ultrafast shows up (#14304) | Not applicable | N47 |
| [1a553d0f5dd2](https://github.com/pingdotgg/t3code/commit/1a553d0f5dd2aa39a64895ec770dc51ef1448b1a) | fix(shared): merge OpenCode Go limits by credential (#14209) | Not applicable | N21 |
| [422248515a1e](https://github.com/pingdotgg/t3code/commit/422248515a1eebff607b44d5abef91e13f6228fc) | feat(codex): regenerate protocol bindings for Codex 0.159 (#14311) | Already covered | Used Codex protocol methods are unchanged; the added permissions request was separately ported. |
| [ff1db030b179](https://github.com/pingdotgg/t3code/commit/ff1db030b179ef712cacc0098366d976e2877f45) | chore(release): prepare v0.0.44 | Not applicable | N15 |
| [8792f9576209](https://github.com/pingdotgg/t3code/commit/8792f95762092ea33e74d41a07e8818befd84970) | test(server): stop pinning codex install advisory to a release range (#14323) | Not applicable | N48 |
| [63b61e647c8d](https://github.com/pingdotgg/t3code/commit/63b61e647c8d973e4378a360cbf576d914f5564d) | chore: stop CodeRabbit from editing PR descriptions (#14307) | Not applicable | N15 |
| [60cb7d180d9e](https://github.com/pingdotgg/t3code/commit/60cb7d180d9ebdd1010460ccd94cdfad1ec83042) | fix(web): unresolved pull request links use the compact link tooltip (#14243) | Not applicable | N02 |
| [88fbc2cac77f](https://github.com/pingdotgg/t3code/commit/88fbc2cac77ff2bf0d406c44754ecd8014976f5f) | fix(client-runtime): keep model ids in inline code from becoming file chips (#13909) | Not applicable | N07 |
| [916ec94f93da](https://github.com/pingdotgg/t3code/commit/916ec94f93da686303f878d506a760efd13506c1) | fix(web): show the agent's question on user-input timeline rows (#12900) | Not applicable | N49 |
| [55ec55b1fb2e](https://github.com/pingdotgg/t3code/commit/55ec55b1fb2e65f2736ff8420ae0373210c1911e) | fix(web): open workspace root links in the file explorer (#12449) | Not applicable | N02 |
| [050cfad04f67](https://github.com/pingdotgg/t3code/commit/050cfad04f673fb092630614b8f7e82000b43ea5) | fix(grok): recover from crashed provider sessions (#10607) | Already covered | Lost-client tests cover unknown outcomes and retained queue holds. |
| [2a23c30ea608](https://github.com/pingdotgg/t3code/commit/2a23c30ea60854cb17bec1a9d248032207c5d6b2) | fix(web): let command menu descriptions use the full row width (#8865) | Not applicable | N02 |
| [0fcd5f906114](https://github.com/pingdotgg/t3code/commit/0fcd5f90611451cca842689faea53b5450c022da) | fix(web): multi-PR badges open the linked pull requests panel (#13211) | Not applicable | N02 |
| [c18e5ea6ed74](https://github.com/pingdotgg/t3code/commit/c18e5ea6ed741443a8ec4a5d22d4b6939b0ecd21) | fix(web): stop clipping the bottoms of diff file names (#14375) | Not applicable | N02 |
| [c2fa9fc911da](https://github.com/pingdotgg/t3code/commit/c2fa9fc911daeac97df4760f95fc57dca42b84c8) | fix(web): name the step that registers a mobile client (#10963) | Not applicable | N02 |
| [38969148a23e](https://github.com/pingdotgg/t3code/commit/38969148a23e7a422045be023627c7fbbddf7503) | test(desktop): Keep WSL busy-runtime fixtures visible when sh is bash (#14351) | Not applicable | N09 |
| [35be904f2fc4](https://github.com/pingdotgg/t3code/commit/35be904f2fc40aa6d7a42778b6895e8274f3097f) | chore: bump vite-plus to 1.0 (#14462) | Not applicable | N03 |
| [792c7dd12786](https://github.com/pingdotgg/t3code/commit/792c7dd127865fc542a453ad944b94ae85ead10a) | fix(web): show double bolts for Codex Ultrafast (#14479) | Not applicable | N50 |
| [0d9468fea907](https://github.com/pingdotgg/t3code/commit/0d9468fea907dda747b51204e92ca15b54f81d47) | docs: define contribution triage policy (#14480) | Not applicable | N15 |
| [d5980a0ff151](https://github.com/pingdotgg/t3code/commit/d5980a0ff1511e6ae1f1876406a7c45a7a989cdb) | docs: use explicit contribution triage exemptions (#14485) | Not applicable | N15 |
| [bd89c1302026](https://github.com/pingdotgg/t3code/commit/bd89c1302026255c62cc09278207bfaf2664da4a) | fix(server): Grok CLIs older than 1.0.13 are marked broken (#14486) | Not applicable | N04 |
| [7c67876983ad](https://github.com/pingdotgg/t3code/commit/7c67876983ade7ef886c825eefe569b3a96c81e8) | fix(clients): hide disconnected environments when adding projects (#14490) | Not applicable | N03 |
| [6b286ae8a20e](https://github.com/pingdotgg/t3code/commit/6b286ae8a20ef2e18fa70b09e92f66edd859128b) | feat: start threads without a project (#13612) | Not applicable | N03 |
| [9da066dbe906](https://github.com/pingdotgg/t3code/commit/9da066dbe906310671b86290b09df0b56a88ffa5) | fix(server): Claude /compact no longer ends early and leaves the thread busy (#14497) | Not applicable | N20 |
| [1905846e0322](https://github.com/pingdotgg/t3code/commit/1905846e03226582f64fec16e1500ddffa151fdf) | fix(web): Dark+ and Light+ themes import instead of colliding with built-in ids (#14499) | Not applicable | N02 |
| [0cf482b08bca](https://github.com/pingdotgg/t3code/commit/0cf482b08bca0249ec94fbcbd0eb04fff28d9b3d) | fix(web): make composer suggestions usable with screen readers (#10154) | Not applicable | N22 |
| [67b175a4c916](https://github.com/pingdotgg/t3code/commit/67b175a4c916ac0d5dbac809da22271db3f542d7) | perf(release): build and publish npm platform packages concurrently (#14028) | Not applicable | N06 |
| [6f8e2534f2d9](https://github.com/pingdotgg/t3code/commit/6f8e2534f2d935e6befa7353a003c4b59fe66f6f) | perf(ci): run PR checks side by side and balance server shards (#14025) | Not applicable | N06 |
| [8630e1ac7a94](https://github.com/pingdotgg/t3code/commit/8630e1ac7a94cdae6f3014161db65385931cf3a2) | perf(release): trim Windows packaging setup (#14037) | Not applicable | N15 |
| [c57a04b722f2](https://github.com/pingdotgg/t3code/commit/c57a04b722f2172e3be2c8f73c936d10b4582431) | perf(release): start Windows builds without waiting for the Linux job (#14027) | Not applicable | N15 |
| [783ccf0fdde0](https://github.com/pingdotgg/t3code/commit/783ccf0fdde0d20824ad87af10a870fc28005835) | perf(release): build Vercel deployments early and go live after publish (#14029) | Not applicable | N15 |
| [7ab800a43c77](https://github.com/pingdotgg/t3code/commit/7ab800a43c770b2c48a2d9306c89b6177f61ad4a) | fix(server): Claude subagents with their own model no longer show the parent's model (#14540) | Not applicable | N51 |
| [921cb3c8bc77](https://github.com/pingdotgg/t3code/commit/921cb3c8bc772309ccab4c422b091d0a91497a49) | feat(web): restart the agent session from cmd+k to load new skills and plugins (#14542) | Not applicable | N52 |
| [71a90ae70e3f](https://github.com/pingdotgg/t3code/commit/71a90ae70e3ff880396363809bd9bfcfd520f30c) | fix(web): hotkey settings record plain keys and Tab (#14548) | Not applicable | N02 |
| [41a8239849d7](https://github.com/pingdotgg/t3code/commit/41a8239849d7e6523c08edb363efdeccfb150be5) | fix(release): Windows CLI smoke test no longer fails on temp dir cleanup (#14553) | Not applicable | N15 |
| [148e6deea046](https://github.com/pingdotgg/t3code/commit/148e6deea046658639aae9fef5b349781cec39d1) | feat: start a new project from just a name (#14527) | Not applicable | N03 |
| [5cc99e1c2398](https://github.com/pingdotgg/t3code/commit/5cc99e1c23980d7995a13c47f969b47cb68ed1be) | fix(desktop): agent clicks in the browser no longer pop Save dialogs (#14573) | Not applicable | N09 |
| [8dc07f199c2e](https://github.com/pingdotgg/t3code/commit/8dc07f199c2e3a4d8ec1a3997a2b9c2edf12457e) | chore(mobile): upgrade to Expo SDK 58 and React Native 0.88 RC (#12045) | Not applicable | N01 |
| [e41755cf8a9b](https://github.com/pingdotgg/t3code/commit/e41755cf8a9b07a12fefee5134cadacb514d4ad9) | feat(mobile): render the Android subscription widget with expo-widgets (#12046) | Not applicable | N01 |
| [09388bf2da51](https://github.com/pingdotgg/t3code/commit/09388bf2da51b958fe5c11fb5bade155fe575e58) | feat(mobile): degrade the agent Live Activity once its content goes stale (#12047) | Not applicable | N01 |
| [438af295fefd](https://github.com/pingdotgg/t3code/commit/438af295fefdd04cc38629f902d7faa07cf39712) | feat(notifications): stack agent alerts by thread on both platforms (#12048) | Not applicable | N01 |
| [ce920f2ac893](https://github.com/pingdotgg/t3code/commit/ce920f2ac893051bff2108c03346bd35340fd988) | chore(mobile): drive dev-menu suppression from the dev-client launch URL (#12049) | Not applicable | N15 |
| [9962e6986db6](https://github.com/pingdotgg/t3code/commit/9962e6986db68cd5537d23153e7898d6b3696b85) | fix(mobile): read display scale and width from the view's scene, not UIScreen.main (#12050) | Not applicable | N01 |
| [1e9c36023c89](https://github.com/pingdotgg/t3code/commit/1e9c36023c898bfcab6dce862e105441b85ec22e) | refactor(mobile): adopt Expo Modules 2.0 for function-only native members (#12051) | Not applicable | N01 |
| [bed69f6ea41d](https://github.com/pingdotgg/t3code/commit/bed69f6ea41dab13ec37afbaaa2c911a19865f7a) | feat(mobile): suppress only the on-screen thread's alert on Android (#12052) | Not applicable | N01 |
| [b91e4668fa2b](https://github.com/pingdotgg/t3code/commit/b91e4668fa2b8e4ac0044e65a08dc6b5e51dfbcf) | feat(marketing): replace the screenshot crop with a designed social card (#13487) | Not applicable | N19 |
| [a3fb5392e361](https://github.com/pingdotgg/t3code/commit/a3fb5392e361f6a6160cb4a6996284dbf9c1d1b3) | docs: update user count in AGENTS.md (#11413) | Not applicable | N15 |
| [5a574a77d1be](https://github.com/pingdotgg/t3code/commit/5a574a77d1bef579b01d137b81888f68eabf7d4b) | test(web): remove duplicate sidebar ordering tests (#14558) | Not applicable | N02 |
| [a97a4a9d189f](https://github.com/pingdotgg/t3code/commit/a97a4a9d189f145afe79c3a6f8533bbf9a6b40b1) | Update marketing stats for GitHub stars and users | Not applicable | N19 |
| [6ea01f8d2441](https://github.com/pingdotgg/t3code/commit/6ea01f8d24419019588f2be589681929d8e26b0c) | fix: cloned projects show their favicon instead of a monogram (#14635) | Not applicable | N03 |
| [094fb230e90f](https://github.com/pingdotgg/t3code/commit/094fb230e90ffb0af303a318ca716fecefd66c80) | docs: server features are services, and handlers stay thin (#14613) | Not applicable | N15 |
| [a3abb5266080](https://github.com/pingdotgg/t3code/commit/a3abb5266080c15b2a675d7f92b517b567c427e2) | fix(ci): pin eas-cli so mobile PR previews deploy again (#14659) | Not applicable | N15 |
| [10ac2f2ba495](https://github.com/pingdotgg/t3code/commit/10ac2f2ba495508a2a2fd469e67eda5df73f0df1) | fix(server): background GitHub polling uses ~74% fewer calls with batched GraphQL (#14673) | Not applicable | N11 |
| [67f640093ffe](https://github.com/pingdotgg/t3code/commit/67f640093ffea1adaa585949d87e4bc2010b0810) | fix(web): show the Docker icon on dockerfile code blocks (#14675) | Not applicable | N02 |
| [0a04cc50def4](https://github.com/pingdotgg/t3code/commit/0a04cc50def48121b9ece038c7df8b7fa789074b) | feat(web): update providers on every machine with one click (#14678) | Not applicable | N16 |
| [b33eda13998b](https://github.com/pingdotgg/t3code/commit/b33eda13998b5b0571daaf266c87f6f31af41f7d) | feat(web): beta Working section hides busy threads until they need you (#13926) | Not applicable | N03 |
| [ced865c224eb](https://github.com/pingdotgg/t3code/commit/ced865c224ebb7dd2cbd0f2931d6e33559b9343a) | fix(web): PR checks status collapses to its icon instead of wrapping (#14700) | Not applicable | N02 |
| [fa6262db3107](https://github.com/pingdotgg/t3code/commit/fa6262db3107a4c95b6ff611113e6fa0753e5948) | fix: remove outdated restart setting update advice (#14710) | Not applicable | N02 |
| [20012ebd80b0](https://github.com/pingdotgg/t3code/commit/20012ebd80b06a4c10ab0e78893822e7f873b2a8) | fix(web): keep the terminal toggle clear of the last header action (#13426) | Not applicable | N02 |
| [2731929610ec](https://github.com/pingdotgg/t3code/commit/2731929610ecb64617ff5e066980302da328f658) | fix(web): group composer undo the way the Lexical composer did (#14674) | Not applicable | N02 |
| [3e60fecf39f3](https://github.com/pingdotgg/t3code/commit/3e60fecf39f37faaca9bdf37e64b7a58229ea547) | fix(desktop): markdown pages no longer render invisible in the dark-mode browser (#14601) | Not applicable | N09 |
| [39405f1992b4](https://github.com/pingdotgg/t3code/commit/39405f1992b4d1264730138b0fe31e07d66df4f4) | fix(mobile): one No project row at the top of the project picker (#14532) | Not applicable | N01 |
| [99e08526e5ec](https://github.com/pingdotgg/t3code/commit/99e08526e5ec84f294940cba5929841518c52fec) | Remove unused isCloudDebugEnabled and isTerminalDebugEnabled (#14367) | Not applicable | N01 |
| [54084ae1e6c3](https://github.com/pingdotgg/t3code/commit/54084ae1e6c32809db040e4fa571c80fdf2d8ae4) | fix(mobile): update Expo 58 to restore widgets and Live Activities (#14734) | Not applicable | N08 |
| [ce96e4c3de19](https://github.com/pingdotgg/t3code/commit/ce96e4c3de1973c7e0088ef8806043841f902080) | fix(mobile): project picker cards match the settings card color (#14756) | Not applicable | N01 |
| [4804036e04a0](https://github.com/pingdotgg/t3code/commit/4804036e04a025a35c46c02100d3c1706cd4fa47) | chore(mobile): bump app version to 1.4.0 | Not applicable | N01 |
| [6c8fed35dded](https://github.com/pingdotgg/t3code/commit/6c8fed35dded9ff71c5b46807125457acbb76be6) | fix(markdown): keep Windows paths intact in link and image destinations (#12615) | Not applicable | N23 |
| [42b5885c90bc](https://github.com/pingdotgg/t3code/commit/42b5885c90bcef05ba56acce00cdd2cb9d4998b7) | fix(mobile): opening a thread no longer crashes Android release builds (#14850) | Not applicable | N08 |
| [5b31001eac12](https://github.com/pingdotgg/t3code/commit/5b31001eac125b501bd01decb5e04c4ff85d0c86) | chore(release): prepare v0.0.45 | Not applicable | N15 |
| [e0db2a5e58bc](https://github.com/pingdotgg/t3code/commit/e0db2a5e58bcbe7d738bca7667d2440ddb83e30f) | fix(server): stop a second server from resending Claude turns (#13295) | Not applicable | N06 |
| [024d49520ead](https://github.com/pingdotgg/t3code/commit/024d49520eadb24cc2f2cef0b0a82c33f0290813) | docs(triage): leave protected PRs open (#14852) | Not applicable | N15 |
| [de343914273e](https://github.com/pingdotgg/t3code/commit/de343914273eceb852a1d1d739cd1d38df7796ee) | feat(orchestrator): introduce new orchestrator (#2829) | Ported | Grok auto mode, removal of accept-edits and ACP clientType/ctrl_c metadata; orchestration rewrite excluded. |
| [e9298af6f520](https://github.com/pingdotgg/t3code/commit/e9298af6f5200fc7e63e0ad7213fe44a4151f36c) | chore(mobile): bump app version to 2.0.0 | Not applicable | N01 |
| [1e7c8e0f24c2](https://github.com/pingdotgg/t3code/commit/1e7c8e0f24c217a3229615fa1de1e6ed85d7e835) | fix(server): Claude MCP tools show their names and icons (#14860) | Not applicable | N53 |
| [cc1e634bfa62](https://github.com/pingdotgg/t3code/commit/cc1e634bfa62edd56ff792eea666e436fdef788f) | fix(server): released builds keep the OpenCode policy for their own version (#14863) | Not applicable | N04 |
| [8bc40b4e07bb](https://github.com/pingdotgg/t3code/commit/8bc40b4e07bb7b4b0f71876d59520360c9bf958c) | fix(server): idle shells stop reloading every project once a minute (#14893) | Not applicable | N06 |
| [b4d3d51ac99d](https://github.com/pingdotgg/t3code/commit/b4d3d51ac99d4306d754afb5c78c49845bfac3c1) | fix(server): Claude model changes no longer kill running background agents (#14726) | Not applicable | N54 |
| [bf7121d7a25f](https://github.com/pingdotgg/t3code/commit/bf7121d7a25fe84d76561c3142b9433615a703a1) | fix(server): API-key Codex installs no longer warn on every start (#14903) | Not applicable | N06 |
| [ca7df394ed81](https://github.com/pingdotgg/t3code/commit/ca7df394ed8151fa77f856beefa90bc60a785d60) | fix(server): steers keep the newest next-turn selection (#14725) | Not applicable | N10 |
| [a7b3ce8c0896](https://github.com/pingdotgg/t3code/commit/a7b3ce8c0896d123a7c3f02c586ff8193841cc09) | fix(settings): Pi and ACP Registry no longer show an Early Access badge (#14915) | Not applicable | N55 |
| [a8927712f8b3](https://github.com/pingdotgg/t3code/commit/a8927712f8b3338fa3c47a0c657b6e11d7e9aaec) | fix(server): Codex auth tokens are an omitted key, never undefined (#14908) | Not applicable | N06 |
| [fd7ee2c30a7e](https://github.com/pingdotgg/t3code/commit/fd7ee2c30a7eeef910c99e7bc92e944526555db7) | chore(lint): forbid declaring tests inside for loops (#14921) | Not applicable | N31 |
| [7ff2eabf56d3](https://github.com/pingdotgg/t3code/commit/7ff2eabf56d30af15a8ac86b0eb46013a9e910ee) | fix(desktop): markdown no longer loads invisible in new dark-mode browser tabs (#14727) | Not applicable | N09 |
| [641426885921](https://github.com/pingdotgg/t3code/commit/641426885921b8d7caad1dbe35f804e6cec6a70b) | test(server): the fake Claude CLI exits with its parent (#14926) | Not applicable | N48 |
| [8ed276c246b6](https://github.com/pingdotgg/t3code/commit/8ed276c246b624631e7d39241ebfd22d8314cb68) | fix(mobile): production OTAs no longer reach the previous major's store binaries (#14944) | Not applicable | N01 |
| [bb7997709dd5](https://github.com/pingdotgg/t3code/commit/bb7997709dd5ed91a3aa18c091286f5f5331b0ef) | fix(web): thread panel shows the remote machine a thread runs on (#14972) | Not applicable | N02 |
| [43bd6677392b](https://github.com/pingdotgg/t3code/commit/43bd6677392b80cd12742f865c277c9939239543) | fix(web): Claude skill calls show the skill name (#14980) | Ported | Name the skill in Claude Skill tool titles. |
| [858b98b8fa61](https://github.com/pingdotgg/t3code/commit/858b98b8fa618cf9b9528b5415493fcf0572ac88) | test(server): the fake Claude CLI is a fixture file, not a generated string (#14950) | Not applicable | N48 |
| [4f7760e6a003](https://github.com/pingdotgg/t3code/commit/4f7760e6a0037b06917adaae1b8a220e3ee6e5cc) | fix(server): Claude subagents show the model their agent file picks (#14994) | Not applicable | N03 |
| [151c241cd2cc](https://github.com/pingdotgg/t3code/commit/151c241cd2cc22c3b10ba8735e44f83fb723a511) | fix(client-runtime): format subagent descriptions as display names (#14999) | Not applicable | N07 |
| [c91e93c231e7](https://github.com/pingdotgg/t3code/commit/c91e93c231e7ae6ec993f8496120ef04dd5fe24d) | Update TRIAGE_EXEMPTIONS with new entries | Not applicable | N15 |
| [22e9d3561330](https://github.com/pingdotgg/t3code/commit/22e9d35613305a04a28e96f83abc10c1bcf4aa8a) | fix(web): outdated servers can be updated even when the client can't connect (#15002) | Not applicable | N02 |
| [3a7058da50c9](https://github.com/pingdotgg/t3code/commit/3a7058da50c91906158c89fc3f845e6c04379414) | fix(web): new worktree threads no longer flicker while starting (#15019) | Not applicable | N02 |
| [d668ffcd64d7](https://github.com/pingdotgg/t3code/commit/d668ffcd64d74daba16520cbbfb0fdb431983eac) | fix(server): subagent threads stop publishing tombstones to the relay (#15016) | Not applicable | N06 |
| [9333509c9180](https://github.com/pingdotgg/t3code/commit/9333509c918083cbfb7e66759ddbcbfa71a3a452) | fix(client-runtime): reconnects back off with jitter and keep healthy sockets (#14897) | Not applicable | N06 |
| [736130c2fd39](https://github.com/pingdotgg/t3code/commit/736130c2fd39b85cd515fa2a41cd10c8a8a52630) | fix(server): working timers no longer reset on every background wake (#15029) | Not applicable | N03 |
| [ca4f84e70242](https://github.com/pingdotgg/t3code/commit/ca4f84e702428656665d53635e16251112091657) | fix(server): threads settle as soon as an agent merges their PR (#15024) | Not applicable | N06 |
| [63f74e209dc2](https://github.com/pingdotgg/t3code/commit/63f74e209dc27df1f8733535d6fd4a79d442add8) | fix(opencode): fill blank reasoning labels (#14978) | Not applicable | N05 |
| [798e945aa779](https://github.com/pingdotgg/t3code/commit/798e945aa77939937464ff45deaa8947be8d402f) | fix(preview): agent browser clicks no longer steal your composer focus (#15008) | Not applicable | N02 |
| [c5a929e1ac05](https://github.com/pingdotgg/t3code/commit/c5a929e1ac05dec15945a36905ab8bb2f1982e59) | fix(web): Woke dismiss syncs, preview mute is back, and subscription retries back off (#15034) | Not applicable | N02 |
| [ddcd3102827f](https://github.com/pingdotgg/t3code/commit/ddcd3102827f520e3dff53b902111744c0ba2ced) | chore: docs, dev scripts and CI catch up with orchestration  (#15041) | Not applicable | N03 |
| [8283b481a34f](https://github.com/pingdotgg/t3code/commit/8283b481a34f5cc6f984f290e7cb93cc35b4dc56) | fix(server): Claude  turns start on Windows with the default binary path (#15021) | Not applicable | N56 |
| [d1034d62b28e](https://github.com/pingdotgg/t3code/commit/d1034d62b28e19c19240085566a39a92c5f9b466) | fix(web): diff panel opens on all branch changes, not just uncommitted (#15005) | Not applicable | N11 |
| [5bf19d12b0fd](https://github.com/pingdotgg/t3code/commit/5bf19d12b0fd54fc9999579869bcf264bc555874) | fix(server): threads stay working while Claude starts a wake turn (#15055) | Not applicable | N54 |
| [5c0429b1578c](https://github.com/pingdotgg/t3code/commit/5c0429b1578cff3307c35c51a305e2749a6dcb6e) | feat(web): mod+alt+enter on an existing thread sends and opens a new thread (#15050) | Not applicable | N02 |
| [db514607f402](https://github.com/pingdotgg/t3code/commit/db514607f4026cd2bbb7b96674168bb227ac043a) | fix(chat): sending on an older thread no longer jumps to the top (#15059) | Not applicable | N03 |
| [fed41fa88bb2](https://github.com/pingdotgg/t3code/commit/fed41fa88bb27cb4325cb208d571393850bc63c2) | chore: add bmdavis419 to triage exemptions (#15062) | Not applicable | N15 |
| [6108ef3d3d0e](https://github.com/pingdotgg/t3code/commit/6108ef3d3d0ef2a598088a1bb64980580e935830) | fix(server): runs no longer get stuck (#15048) | Already covered | Lost-client tests cover unknown outcomes and retained queue holds. |
| [56914128c1ff](https://github.com/pingdotgg/t3code/commit/56914128c1ff9dcf6585c380b7b25845d6ca5f98) | fix(usage): Codex Fast and Ultrafast now cost what they bill (#15101) | Not applicable | N06 |
| [a5b34b25378f](https://github.com/pingdotgg/t3code/commit/a5b34b25378fbbf90ce36ee101a11528ccf11c7e) | fix(clients): a dev server left running no longer says the thread is waiting (#15114) | Not applicable | N02 |
| [f68e24fb419c](https://github.com/pingdotgg/t3code/commit/f68e24fb419c985269cd0acb841e203af1419c43) | fix(web): a thread that left a shell running shows its unseen completion (#14910) | Not applicable | N03 |
| [263097a8da03](https://github.com/pingdotgg/t3code/commit/263097a8da038a143f2b7b7817250f0ad390ea32) | fix(web): mod+enter starts a new thread in the background again (#15060) | Not applicable | N03 |
| [e8545b293b79](https://github.com/pingdotgg/t3code/commit/e8545b293b79cdd73d30ef921db149ef1b5abee0) | feat(usage): show cost by token type, speed, and model detail (#15108) | Not applicable | N03 |
| [18b21325c3ab](https://github.com/pingdotgg/t3code/commit/18b21325c3ab1ff4f0ee7b017d9c4eefc2b2d307) | feat(server): agents can watch a PR and get woken when checks, reviews, or conflicts need them (#15057) | Not applicable | N57 |
| [31a9da179ed0](https://github.com/pingdotgg/t3code/commit/31a9da179ed0763335f05681c577474aec5d2309) | fix(server): keep delegated review rounds on the task API (#15115) | Not applicable | N58 |
| [f391794a35c6](https://github.com/pingdotgg/t3code/commit/f391794a35c604d57e166a3ab48d56fc6e4e469a) | fix(shared): classify workspace previews by literal filenames (#10311) | Not applicable | N06 |
| [1945ce82c0d6](https://github.com/pingdotgg/t3code/commit/1945ce82c0d66d2b8ffbe867e688c0ecbf530e69) | fix(mobile): iOS threads no longer jump to the top (#14808) | Not applicable | N08 |
| [44b49650170a](https://github.com/pingdotgg/t3code/commit/44b49650170a76e6171642d6f8de946cf1d1c821) | fix(web): reduce the gap above the draft composer (#15196) | Not applicable | N02 |
| [a0bf1eee48d1](https://github.com/pingdotgg/t3code/commit/a0bf1eee48d1537ec38e0f1fa5243f10ffd4e870) | fix(mobile): a dev server left running no longer shows the waiting bolt (#15194) | Not applicable | N01 |
| [ce90eec1ffc2](https://github.com/pingdotgg/t3code/commit/ce90eec1ffc2087395227a610843091f4b260cf7) | fix(server): a Claude command you stop shows as interrupted (#14896) | Ported | Show stopped tools as interrupted (#14896). |
| [0080e80c00a3](https://github.com/pingdotgg/t3code/commit/0080e80c00a3e333e3dcafd85b2615d56f84e3b8) | fix(server): editors appear once a slow discovery scan finishes (#13917) | Not applicable | N06 |
| [aad732901e4b](https://github.com/pingdotgg/t3code/commit/aad732901e4b7d485574eaef5a9c1fb388c4291a) | fix(server): Claude threads no longer stay stuck in plan mode Claude entered itself (#15224) | Already covered | Each turn starts a native Claude process with the selected permission flag, so SDK permission-mode drift is not retained. |
| [65731f986ba2](https://github.com/pingdotgg/t3code/commit/65731f986ba2175063b0182deb6a143f8f323b74) | fix(mobile): show complete subagent details (#15189) | Not applicable | N03 |
| [77823bd102ae](https://github.com/pingdotgg/t3code/commit/77823bd102ae50430d4acda9a553e5743d2aa5ba) | fix(mobile): an expired Live Activity no longer leaves a second card (#15254) | Not applicable | N08 |
| [cfdff56f7b3c](https://github.com/pingdotgg/t3code/commit/cfdff56f7b3c9c931ae368224041aa14825649fd) | test(web): remove redundant thread sort fallback tests (#15095) | Not applicable | N02 |
| [70e5a40c20dd](https://github.com/pingdotgg/t3code/commit/70e5a40c20ddbaab7831ab36e17e64c9908269a5) | fix(web): thinking row after a failed tool expands the run's tool calls (#15056) | Not applicable | N18 |
| [39efcd855833](https://github.com/pingdotgg/t3code/commit/39efcd855833a1486ff7662e2efcedeeee6be0cc) | perf(web): DOM changes no longer restyle the whole page (#15265) | Not applicable | N28 |
| [71dbaf1f94b5](https://github.com/pingdotgg/t3code/commit/71dbaf1f94b580491d033f2e347a9b1c507fd793) | perf(usage): cut warm usage scans from seconds to milliseconds on large histories (#15149) | Not applicable | N06 |
| [fffe6e6c82c1](https://github.com/pingdotgg/t3code/commit/fffe6e6c82c12769e0cbc4d89a23ec845c5abed1) | perf(web): virtualize command palette results (#15266) | Not applicable | N02 |
| [0cfa113be597](https://github.com/pingdotgg/t3code/commit/0cfa113be597af3a1b2fe905a58a5b583f8089b8) | chore(lint): flag :has() variants that restyle the whole page (#15274) | Not applicable | N12 |
| [429c625a8515](https://github.com/pingdotgg/t3code/commit/429c625a8515f9c2e4af28c488fbf32ce7789645) | fix(web): workspace card docks beside chat when the window is narrow (#14992) | Not applicable | N28 |
| [5e35272fda7c](https://github.com/pingdotgg/t3code/commit/5e35272fda7cc94d9c140377bc8db70580295133) | feat(web): render mermaid code blocks as diagrams (#15067) | Not applicable | N23 |
| [f1dcd93931b3](https://github.com/pingdotgg/t3code/commit/f1dcd93931b322d587a7253856f44625a6afbce0) | feat(web): Nightly tells you to get the beta mobile app (#15070) | Not applicable | N02 |
| [a21702d0e58a](https://github.com/pingdotgg/t3code/commit/a21702d0e58ad20263dc965d84472bb6d7eb38c8) | test(server): ACP adapter tests no longer race the prompt settle (#15330) | Not applicable | N06 |
| [b05adb70b00f](https://github.com/pingdotgg/t3code/commit/b05adb70b00feac0883b2e364256100cfab4b6ca) | feat(usage): fold preview model IDs into the model they belong to (#15333) | Not applicable | N03 |
| [a78760e5e7b1](https://github.com/pingdotgg/t3code/commit/a78760e5e7b11ea51e21a1640fec54bd3f210fd8) | refactor(server): check RPC scopes in group middleware (#15324) | Not applicable | N06 |
| [8d846660cecf](https://github.com/pingdotgg/t3code/commit/8d846660cecfc69ef4e192322fced29fa07f374c) | feat(mobile): beta Working section hides busy threads until they need you (#15346) | Not applicable | N03 |
| [00eb8f6183ba](https://github.com/pingdotgg/t3code/commit/00eb8f6183ba4b36a429db08767031ad8ca9d81d) | fix(settings): symlinked settings files stay linked when saved (#15009) | Not applicable | N03 |
| [d65cd33a0398](https://github.com/pingdotgg/t3code/commit/d65cd33a03983751598f6bdce76b402ac7bf2cca) | fix(server): Stop ends a dev server left running before a provider switch (#15355) | Not applicable | N06 |
| [243d1e7c4499](https://github.com/pingdotgg/t3code/commit/243d1e7c4499d7ba4dc5b22794fcd2c880936c2d) | fix(server): merged threads settle even after the agent wakes on its own (#15388) | Not applicable | N06 |
| [c47f4263f927](https://github.com/pingdotgg/t3code/commit/c47f4263f927a68645270c2892d8fa18ba7dcb3b) | fix(web): no-project drafts can switch machines (#15356) | Not applicable | N02 |
| [f786ff3548ae](https://github.com/pingdotgg/t3code/commit/f786ff3548aef904b7bdd47f263e8704ba20ca4b) | fix(web): highlight tool inputs and remove nested work log indentation (#15384) | Not applicable | N03 |
| [5108c978b1d9](https://github.com/pingdotgg/t3code/commit/5108c978b1d9a8bd029e5ea96781d8cbc90b3877) | fix(server): restarts keep delegated tasks, queued threads, and stops intact (#15323) | Already covered | Queue and hold survive restart without automatic replay. |
| [a976f8c74ca4](https://github.com/pingdotgg/t3code/commit/a976f8c74ca4eab6e16bb47b5fa446154b743371) | fix(web): sending past the resume banner compacts first (#15290) | Not applicable | N02 |
| [44bd4c9cafa6](https://github.com/pingdotgg/t3code/commit/44bd4c9cafa61a415f76fbb261f321a158660577) | fix(codex): resume archived native sessions (#15389) | Ported | Unarchive and resume an archived Codex thread. |
| [f2cc80a7a44e](https://github.com/pingdotgg/t3code/commit/f2cc80a7a44ee05c1f39442779f2b719e31faeb5) | feat(web): morph composer and panel action icons (#14924) | Not applicable | N28 |
| [d3071275d576](https://github.com/pingdotgg/t3code/commit/d3071275d576bb289a6cb177c6cc28fff8e95015) | fix(web): subagents sent a follow-up show as running in Lineage (#15334) | Not applicable | N02 |
| [2d81e4c27559](https://github.com/pingdotgg/t3code/commit/2d81e4c27559bbd78564bc8cc3b38a0011814361) | fix(web): clear stale chat action shortcuts (#15394) | Not applicable | N02 |
| [88744f3ddba9](https://github.com/pingdotgg/t3code/commit/88744f3ddba9d3883ba631f0f4801f1a97fe77ce) | fix(orchestration): restore earlier app agent transcript pages (#14104) | Not applicable | N06 |
| [1b1e5b9674fc](https://github.com/pingdotgg/t3code/commit/1b1e5b9674fcacf7aa177594d44cd61614c2f45f) | fix(web): remove the square thread info panel shadow (#15069) | Not applicable | N16 |
| [8e399944393e](https://github.com/pingdotgg/t3code/commit/8e399944393ee48054c0ca4aece0b6c9befd0998) | fix(mobile): Android usage widget no longer sticks on "Loading widget" in release builds (#15142) | Not applicable | N01 |
| [a6aebbe2e06e](https://github.com/pingdotgg/t3code/commit/a6aebbe2e06e6f2fb247a291272dcfbedb8ed63f) | fix(web): size the model picker to its content (#15152) | Not applicable | N05 |
| [2a455570e6ee](https://github.com/pingdotgg/t3code/commit/2a455570e6ee70a59241c30126b50c7df1696a6a) | test(server): replay checks a Claude subagent's thread takes its reported model (#15022) | Not applicable | N06 |
| [dab26f582cf7](https://github.com/pingdotgg/t3code/commit/dab26f582cf7826fec67fbed64b46f59f3421280) | fix(web): subagent finish notifications look like subagent cards (#15281) | Not applicable | N57 |
| [4543c7368cae](https://github.com/pingdotgg/t3code/commit/4543c7368cae6dfef6b2b206b5d8aeb709eeca34) | fix(web): thread status dot has an accessible name (#14587) | Not applicable | N02 |
| [c5bc98db03ef](https://github.com/pingdotgg/t3code/commit/c5bc98db03ef048e2d88010876b2b3e474ecfd90) | fix(web): legacy sidebar options button has a label (#14602) | Not applicable | N02 |
| [2b8bb0c18815](https://github.com/pingdotgg/t3code/commit/2b8bb0c1881583428adbc07e3b641b21067ab41a) | fix(web): imported themes keep switches and focus rings visible (#14498) | Not applicable | N02 |
| [7bc161f8692d](https://github.com/pingdotgg/t3code/commit/7bc161f8692dfb969489f40bf3eb5bd27bc01f10) | fix(web): links to issues no longer strand the pull request viewer (#14242) | Not applicable | N11 |
| [237d51e6cd2f](https://github.com/pingdotgg/t3code/commit/237d51e6cd2f935d3ddced4efe8fcd4a5d697e43) | fix(web): repo/task breadcrumb no longer bounces when the sidebar collapses (#15046) | Not applicable | N02 |
| [f90b77d80971](https://github.com/pingdotgg/t3code/commit/f90b77d80971ea4855a42abfc89ce85466e1accb) | fix(web): Pull request panel entry works for linked PRs (#15061) | Not applicable | N02 |
| [95edeb753b07](https://github.com/pingdotgg/t3code/commit/95edeb753b070bad191e10afe15918c969135225) | fix(web): add context menu to draft threads in the sidebar (#10637) | Not applicable | N02 |
| [9efb01690065](https://github.com/pingdotgg/t3code/commit/9efb01690065923820dfa889af5745cc34750091) | fix(web): keep sidebar branding and build pills from clipping at varying font sizes and zoom levels (#12141) | Not applicable | N02 |
| [0c8112013760](https://github.com/pingdotgg/t3code/commit/0c8112013760617230d382e55447ddb75627a997) | fix(usage): model shares and order follow the selected metric (#11391) | Not applicable | N03 |
| [1826fb55cc4c](https://github.com/pingdotgg/t3code/commit/1826fb55cc4c9c6246d31175e15a496ba4778e07) | feat(web): sweep sidebar buttons to settle, un-settle, and wake threads (#14768) | Not applicable | N02 |
| [737993303d36](https://github.com/pingdotgg/t3code/commit/737993303d36e10674c54b95e5bd3826682c99c7) | feat: retry a failed workspace preparation (#15326) | Not applicable | N16 |
| [8fb068c4982f](https://github.com/pingdotgg/t3code/commit/8fb068c4982fc50a2a2f57c3b6fd4731735cf0bf) | fix(server): registry test stubs no longer outlive the test run (#15457) | Not applicable | N48 |
| [60a56f11e73d](https://github.com/pingdotgg/t3code/commit/60a56f11e73dfc9c26e74f8f8e93e391191fc3f8) | test(server): the registry's fake Claude CLI is a fixture file, not a generated string (#15463) | Not applicable | N48 |
| [845ddd935453](https://github.com/pingdotgg/t3code/commit/845ddd9354538d42db1cc7dc30143c6bb8f0d40f) | refactor(clients): share opening a machine's No project folder (#14759) | Not applicable | N03 |
| [070127cbb7b4](https://github.com/pingdotgg/t3code/commit/070127cbb7b4a0a44bab760f59b51829ec349528) | test(server): the git-ssh wrapper's fake SSH script is a fixture file, not a generated string (#15480) | Not applicable | N06 |
| [1508552a16ed](https://github.com/pingdotgg/t3code/commit/1508552a16edebe29cab4820c54c16b75c142408) | test(server): the ACP registry's fake npm is a fixture file, not a generated string (#15483) | Not applicable | N48 |
| [f870c419fc21](https://github.com/pingdotgg/t3code/commit/f870c419fc21f7ee23c122e60fb94adb8bf73a40) | test(server): the ACP registry's fake uv is a fixture file, not a generated string (#15484) | Not applicable | N48 |
| [c5a0c78b7c0f](https://github.com/pingdotgg/t3code/commit/c5a0c78b7c0f913610b118234117747065050a5a) | feat(clients): step a new thread to the next machine from the keyboard (#15391) | Not applicable | N03 |
| [d073c261b0e4](https://github.com/pingdotgg/t3code/commit/d073c261b0e4ba8fd77bf92030c9a372b87fff27) | fix(web): promoting a draft thread no longer logs a React key warning (#15458) | Not applicable | N02 |
| [ee7b49d638e4](https://github.com/pingdotgg/t3code/commit/ee7b49d638e412590b09b415cc48399d51644b60) | test(server): the text generation's fake Claude CLI is a fixture file, not a generated string (#15479) | Not applicable | N06 |
| [bf5c146be914](https://github.com/pingdotgg/t3code/commit/bf5c146be914c9b596cdc45d343228271115c099) | fix(mobile): keep dictation running across navigation behind an edge pill (#15502) | Not applicable | N01 |
| [a92c43b80ba3](https://github.com/pingdotgg/t3code/commit/a92c43b80ba307f112e0ae03bf64d841475cf326) | fix(client-runtime): relay disconnects no longer show as thread errors (#15470) | Not applicable | N07 |
| [daa1d0ed947c](https://github.com/pingdotgg/t3code/commit/daa1d0ed947c276cb74b08025b5903e2ffe31278) | fix(web): subagent cards name the provider account (#15493) | Not applicable | N03 |
| [f4f3abf71e98](https://github.com/pingdotgg/t3code/commit/f4f3abf71e98361918e360a89d48780cc65cafd1) | fix(server): read paginated review replies when watching PRs (#15427) | Not applicable | N11 |
| [fe93a5dbdd3c](https://github.com/pingdotgg/t3code/commit/fe93a5dbdd3c4496f402f71d232597c391c4973b) | fix(server): offer one-click provider updates for every install (#15416) | Not applicable | N04 |
| [930aec13da56](https://github.com/pingdotgg/t3code/commit/930aec13da56f83c404bc00d92486b2409a5cac0) | fix(mobile): keep the dictation timer from shifting width (#15504) | Not applicable | N01 |
| [40596072ebab](https://github.com/pingdotgg/t3code/commit/40596072ebabf045414f54ca6ffec3911543d8ff) | fix(relay): T3 Connect links no longer fail on colliding prepared statements (#15411) | Not applicable | N33 |
| [0fe4fa40fe65](https://github.com/pingdotgg/t3code/commit/0fe4fa40fe65ac9545621a65633f08d9fa29fd0d) | fix(server): sqlite transactions wait for the write lock instead of failing (#15488) | Not applicable | N27 |
| [6e0abd5a10a6](https://github.com/pingdotgg/t3code/commit/6e0abd5a10a646337076a9baf7f6502dc4f0e74d) | fix(web): unpin button shows the pin-off icon on hover (#15425) | Not applicable | N02 |
| [ae010fe55b9c](https://github.com/pingdotgg/t3code/commit/ae010fe55b9c4d96f294a412ec40d44d2145724b) | fix(mobile): make queued message removal tappable (#15417) | Not applicable | N01 |
| [e19a48d220a1](https://github.com/pingdotgg/t3code/commit/e19a48d220a18ef198178b48d12aa1a034596489) | fix(web): keep workspace panels below dialogs (#15454) | Not applicable | N16 |
| [1302ccacbd6d](https://github.com/pingdotgg/t3code/commit/1302ccacbd6d3e957f2f95239556adb6eae79a48) | fix(clients): Working section keeps its order while agents finish and wake (#15418) | Not applicable | N03 |
| [d60a71ef6e37](https://github.com/pingdotgg/t3code/commit/d60a71ef6e371d8b345ecb1f7575117d675187c7) | feat(mobile): full-screen simulator viewer with on-demand controls (#15551) | Not applicable | N01 |
| [eac52f0087d9](https://github.com/pingdotgg/t3code/commit/eac52f0087d9ba5dee5542f24788d1482affae43) | fix(client-runtime): closing a busy stream no longer drops the connection (#15563) | Not applicable | N07 |
| [33b7d7b3ae2a](https://github.com/pingdotgg/t3code/commit/33b7d7b3ae2a7b1c3d1bd58c04cb9bf581ec2fe7) | feat(web): add shift-held pull request quick actions (#15549) | Not applicable | N02 |
| [5a96895a85b4](https://github.com/pingdotgg/t3code/commit/5a96895a85b43c838da6c89cf21068b99c003d5a) | fix: expanded tool calls show their output, empty ones don't expand (#15505) | Already covered | Only tool rows with bodies expand. |
| [5318d054a5ea](https://github.com/pingdotgg/t3code/commit/5318d054a5eabf85308a086e2fdbef5b6cdfd0ee) | fix(web): show device diagnostics before hub readiness (#15435) | Not applicable | N02 |
| [c8d7d50a7379](https://github.com/pingdotgg/t3code/commit/c8d7d50a73793b6c1d43ea23211e89bbd8f0c910) | fix(web): open thread picker for unsent drafts (#15436) | Not applicable | N02 |
| [e01e58916fa5](https://github.com/pingdotgg/t3code/commit/e01e58916fa51ec52389cabba1b5fbba6344723d) | fix(desktop): print version before initializing the app (#15440) | Not applicable | N09 |
| [b3b6ae2cebba](https://github.com/pingdotgg/t3code/commit/b3b6ae2cebbae39a706038911dbb8c85c7120be7) | fix(server): recover claude skill scalar frontmatter (#15452) | Not applicable | N20 |
| [253dea360ef0](https://github.com/pingdotgg/t3code/commit/253dea360ef0c743c5de8ba04f8412a7eb4694ac) | fix(server): keep settled threads asleep after restarts (#15604) | Not applicable | N06 |
| [f9db42d14fd3](https://github.com/pingdotgg/t3code/commit/f9db42d14fd36e7d21f5dcbf0027f267d53c2989) | fix(server): avoid inferring forgejo conflicts from mergeability (#15441) | Not applicable | N11 |
| [32b77f4fa370](https://github.com/pingdotgg/t3code/commit/32b77f4fa370a5195ad16ab709405b1038c49fcd) | fix(web): dismiss hovered timeline tooltips on scroll (#15455) | Not applicable | N59 |
| [836098543c6e](https://github.com/pingdotgg/t3code/commit/836098543c6ecdf66734596df0ea3a8f6513f2a6) | fix(server): discover Claude commands in each workspace (#15462) | Not applicable | N16 |
| [2dfef8779e04](https://github.com/pingdotgg/t3code/commit/2dfef8779e046dc9b3fe87668022f4ed0907adbb) | fix(web): restore project action preview opening (#15490) | Not applicable | N02 |
| [71fbe8e013bf](https://github.com/pingdotgg/t3code/commit/71fbe8e013bfddab0935f65a82732e9820bd4b73) | fix(desktop): keep titlebar controls inset when zoomed (#15496) | Not applicable | N28 |
| [1c6269326a50](https://github.com/pingdotgg/t3code/commit/1c6269326a502874677192a844e449577df6279c) | fix(source-control): use the Azure DevOps mark (#15512) | Not applicable | N16 |
| [c346fb679bac](https://github.com/pingdotgg/t3code/commit/c346fb679bac106c4c162ab145c27e3c8b9640bd) | fix(web): open provider update details from both icons (#15501) | Not applicable | N16 |
| [d3bec62ec167](https://github.com/pingdotgg/t3code/commit/d3bec62ec167e783412effbc929da77167f08c4d) | fix(web): reveal sidebar actions for secondary hovering pointers (#15536) | Not applicable | N28 |
| [1d013c7ca1ed](https://github.com/pingdotgg/t3code/commit/1d013c7ca1edc843bd3b5e23fd6573aeeaf9b11d) | fix(markdown): preserve descriptive file-link labels (#15509) | Not applicable | N23 |
| [df4ae529eacf](https://github.com/pingdotgg/t3code/commit/df4ae529eacfff8db38f2f4b9b767d3717446c8a) | feat(clients): tool calls show the call above a muted result, without cards (#15506) | Not applicable | N03 |
| [0ce43a4eea2f](https://github.com/pingdotgg/t3code/commit/0ce43a4eea2f21fe404639c248fb86328c410c5d) | fix(chat): repair unclosed local file links in assistant responses (#15520) | Not applicable | N23 |
| [3b0093d7164a](https://github.com/pingdotgg/t3code/commit/3b0093d7164a59bea721e2582e385e984561cd41) | fix(server): match manual update commands to installed cli (#15539) | Not applicable | N03 |
| [4ee6bfd50ef4](https://github.com/pingdotgg/t3code/commit/4ee6bfd50ef4a089440d5c3662db2298da9cc50e) | fix(server): preserve staging during commit message generation (#15532) | Not applicable | N11 |
| [484459f0a963](https://github.com/pingdotgg/t3code/commit/484459f0a9633e56214a7cd35b790c96f94190b4) | fix(mobile): Keep the last line of iOS markdown replies visible (#15737) | Not applicable | N01 |
| [b4381985db55](https://github.com/pingdotgg/t3code/commit/b4381985db55755ff15340d118160c14006ced13) | feat(release): include nightly changelogs in Discord announcements (#15754) | Not applicable | N15 |
| [2188bdd8b5c0](https://github.com/pingdotgg/t3code/commit/2188bdd8b5c0251f40daebdb66647dd0150cf076) | revert(web): remove automatic compaction before resume (#15771) | Not applicable | N02 |
| [efecd3cf8bce](https://github.com/pingdotgg/t3code/commit/efecd3cf8bcec3d1891b5f5a27dc2f6d797c6448) | fix(server): Claude threads no longer get stuck after background commands (#15770) | Not applicable | N54 |
| [fe417dff4cdc](https://github.com/pingdotgg/t3code/commit/fe417dff4cdc149d6873bb4fbaaa388731628d0b) | fix(cli): reject accidental server launches (#15795) | Not applicable | N06 |
| [979ca66cedf1](https://github.com/pingdotgg/t3code/commit/979ca66cedf10d685c7486da1929297c3693fe43) | feat(clients): reach one environment over several routes (#15467) | Not applicable | N03 |
| [745c225f9240](https://github.com/pingdotgg/t3code/commit/745c225f92401ba0950770fa7c3ec67a7067dbeb) | feat(clients): learn an environment's LAN and tailnet addresses (#15468) | Not applicable | N03 |
| [ec20db4a5a27](https://github.com/pingdotgg/t3code/commit/ec20db4a5a27de45f69f64ca05d17ec1bd96644d) | fix(server): share MCP tool presentation across providers (#15475) | Not applicable | N53 |
| [801ca76ed6a5](https://github.com/pingdotgg/t3code/commit/801ca76ed6a5b9d5d6a40f6c1ba231d9ffb3757e) | revert(chat): remove automatic file-link repair (#15824) | Not applicable | N23 |
| [bc2748241776](https://github.com/pingdotgg/t3code/commit/bc2748241776066c532f3312a26f1756d9740f9c) | perf(web): validate monospace fonts when selected (#15642) | Not applicable | N02 |
| [f4153a75a4b0](https://github.com/pingdotgg/t3code/commit/f4153a75a4b04e14bd45d1b578af4d1d63cc84e5) | fix(server): expand home-relative media paths (#15618) | Not applicable | N06 |
| [f729e0ebfe75](https://github.com/pingdotgg/t3code/commit/f729e0ebfe7578860585a77ddde8f008e30c2351) | fix(server): recover Linux runtime directory for device hub (#12402) | Not applicable | N06 |
| [ad5178a31aac](https://github.com/pingdotgg/t3code/commit/ad5178a31aac09c93159ea152daf00720ca56358) | fix(web): center icons in thread details icon buttons (#15669) | Not applicable | N02 |
| [01f894e23e03](https://github.com/pingdotgg/t3code/commit/01f894e23e031ef32c25a15b518a94f20defea6f) | fix(mobile): back from an agent's thread returns to its parent (#15068) | Not applicable | N01 |
| [f8a308fbc90e](https://github.com/pingdotgg/t3code/commit/f8a308fbc90e89cd5f8d41b7fb169a03343bd9b5) | fix(dev): worktree setup never deletes a real env file (#15845) | Not applicable | N15 |
| [1e2ecbd97588](https://github.com/pingdotgg/t3code/commit/1e2ecbd9758830669684b494d4398f626b0576e0) | fix(server): drop the duplicate Option import that breaks main CI (#15847) | Not applicable | N06 |
| [de09c7566c71](https://github.com/pingdotgg/t3code/commit/de09c7566c71a9c01a554b2f163fa50e3e15bfe5) | fix(dev): write bootstrap warnings directly to stderr (#15865) | Not applicable | N15 |
| [a1d9d72aefc2](https://github.com/pingdotgg/t3code/commit/a1d9d72aefc2a388ea3915956e9d84d279c87da7) | fix(mobile): a message that fails to send now says why in the thread (#15807) | Not applicable | N01 |
| [1beb0355d05b](https://github.com/pingdotgg/t3code/commit/1beb0355d05bbe4a780b238e3138f0a8ed99df5f) | fix(server): queue background notifications during active tools (#15892) | Not applicable | N54 |
| [37de6cbde65c](https://github.com/pingdotgg/t3code/commit/37de6cbde65c7cf9ba90a2557c232e63b7e16988) | refactor(server): share one keyed lock that releases idle keys (#15577) | Not applicable | N60 |
| [06e627448bc1](https://github.com/pingdotgg/t3code/commit/06e627448bc1ee58487c1a2a9d6bf8d5ef479af3) | feat(server): T3 MCP tools take explicit thread and project targets (#15219) | Not applicable | N06 |
| [c40873762863](https://github.com/pingdotgg/t3code/commit/c40873762863abe34e2e784c518d81cbc08f6662) | fix(desktop):  imports stashed prompts and drafts from the V1 profile (#15072) | Not applicable | N09 |
| [2a778f7a6e7b](https://github.com/pingdotgg/t3code/commit/2a778f7a6e7b5f9403c5dbff2c2f4a9ce26c0ade) | feat(web): shell commands in the timeline are syntax highlighted (#15037) | Not applicable | N02 |
| [250e052f44dd](https://github.com/pingdotgg/t3code/commit/250e052f44dd313b658abebc707242a7b25be340) | fix(mobile): upgrade Uniwind and remove local patch (#14597) | Not applicable | N01 |
| [cf3e714b0f58](https://github.com/pingdotgg/t3code/commit/cf3e714b0f58e29c8fa8660db50d2187e3263b65) | fix(server): Stop ends a Codex command after its thread was settled (#15546) | Not applicable | N61 |
| [7812230572f2](https://github.com/pingdotgg/t3code/commit/7812230572f2d7e042d8d14ada2381fc71377560) | fix(server): subagents no longer inherit parent pull-request links (#14918) | Not applicable | N06 |
| [7ffa2184a7cf](https://github.com/pingdotgg/t3code/commit/7ffa2184a7cfc78d870bf6cbfa4064c51868aef2) | fix(prs): queue fast actions and close batches by dragging (#15851) | Not applicable | N11 |
| [1a3f7ad50856](https://github.com/pingdotgg/t3code/commit/1a3f7ad5085606463a602bd756ca9c04d106fffe) | perf(prs): share concurrent github routing metadata probes (#15853) | Not applicable | N07 |
| [1604ccc9d79f](https://github.com/pingdotgg/t3code/commit/1604ccc9d79f5270fb8e14d60664184bf142c4cb) | fix(mobile): back from a finished subagent in the feed returns to its parent (#15844) | Not applicable | N01 |
| [7b797f2b9091](https://github.com/pingdotgg/t3code/commit/7b797f2b90918e7bae0f710dcf24018b8b0843dc) | fix(desktop): bound preview inspector retention and record renderer identity (#16032) | Not applicable | N09 |
| [e22c8804346d](https://github.com/pingdotgg/t3code/commit/e22c8804346d4bd45a733803eda826e04e3bbd74) | fix(web): show fast mode beside reasoning as text (#16069) | Not applicable | N50 |
| [68802a48f9a8](https://github.com/pingdotgg/t3code/commit/68802a48f9a8f7f7db5ca4d605bdc818be62a804) | fix(mobile): make the routes list match the other settings rows (#15958) | Not applicable | N01 |
| [51f7f77f8731](https://github.com/pingdotgg/t3code/commit/51f7f77f87318a5329d3bab15d513fda5654200b) | fix(threads): stop pull request watches when settling (#16095) | Not applicable | N06 |
| [ca3f9738c39c](https://github.com/pingdotgg/t3code/commit/ca3f9738c39c3502dd2258aae872f971080969ef) | feat(contracts): clients tolerate union members they don't know yet (#15951) | Not applicable | N06 |
| [5c03ced3be84](https://github.com/pingdotgg/t3code/commit/5c03ced3be8444567db4b50a474619bc5e63db98) | refactor(contracts): project icons decode forward-compatibly instead of encoding a fallback (#16118) | Not applicable | N06 |
| [31f9d832f4f0](https://github.com/pingdotgg/t3code/commit/31f9d832f4f019ff565beafbd6bb18451fa7039d) | Removed an unused helper from the Android push payload builder (#16116) | Not applicable | N33 |
| [1c33f3b2b297](https://github.com/pingdotgg/t3code/commit/1c33f3b2b2976b7f4e26698ed69c549de687a97e) | perf(mobile): reduce shell cache encoding work (#15096) | Not applicable | N01 |
| [c71ca063dfcd](https://github.com/pingdotgg/t3code/commit/c71ca063dfcd8c2e3dcb2bd92adbb2180cbbece3) | perf(mobile): defer audio recorder creation until dictation (#15248) | Not applicable | N01 |
| [4dbc0129c244](https://github.com/pingdotgg/t3code/commit/4dbc0129c244d5869d7598a22f8d9386cca17884) | feat(server): bump Antigravity ACP agent to 1.3.0 (#15746) | Not applicable | N13 |
| [9d029a1df251](https://github.com/pingdotgg/t3code/commit/9d029a1df25126745dd70ac561ff25d3f8fbac42) | feat(acp): support local provider commands (#16021) | Not applicable | N16 |
| [3e6b45028cee](https://github.com/pingdotgg/t3code/commit/3e6b45028ceec5820dacb37dc3852470ebdc9411) | fix(server): honor submodule settings when creating worktrees (#15594) | Not applicable | N11 |
| [194c73f3f991](https://github.com/pingdotgg/t3code/commit/194c73f3f991558ca540ef6fe0ce75223bbe87a7) | chore(deps): upgrade Effect to stable 4.0.1 (#16138) | Not applicable | N38 |
| [b12b3f706d12](https://github.com/pingdotgg/t3code/commit/b12b3f706d1257b79c69c34699babb6834a72b34) | fix(server): worktree threads survive a local branch named t3code (#16167) | Not applicable | N11 |
| [6f9cea00ae96](https://github.com/pingdotgg/t3code/commit/6f9cea00ae967f38fa3cdc9c07f92031806f4264) | chore(refs): sync Effect and Alchemy references to 4.0.1 and beta.80 (#16170) | Not applicable | N32 |
| [039ddf1143d7](https://github.com/pingdotgg/t3code/commit/039ddf1143d7dba0e25d0b1b7c8ed0cd3e19f588) | fix(web): match subagent timestamp fonts to chat (#16151) | Not applicable | N57 |
| [5b613de175dd](https://github.com/pingdotgg/t3code/commit/5b613de175dd2567ae806907440ef83f6cd7c2a4) | fix(web): wrap full status text in composer hover details (#16158) | Not applicable | N22 |
| [2bffcc52f77b](https://github.com/pingdotgg/t3code/commit/2bffcc52f77bd6ff8be8e91be7a09f455eecbe07) | ci: run the transfer report job on Blacksmith (#16178) | Not applicable | N15 |
| [f32c23cf1aa9](https://github.com/pingdotgg/t3code/commit/f32c23cf1aa9ff177224e93c239165adc67798c0) | fix(server): Stop also stops delegated tasks and pull request watches (#16002) | Not applicable | N06 |
| [a172e1c4e809](https://github.com/pingdotgg/t3code/commit/a172e1c4e8098c7d8dbe538a375ecc69faf9514d) | feat: native /goal for Codex and Claude, with goal status in the UI (#15592) | Not applicable | N20 |
| [73fd3c1ab0d8](https://github.com/pingdotgg/t3code/commit/73fd3c1ab0d8b2f6add7bbf8c498b9a2a87278ff) | fix(server): use current SQL import in thread stop tests | Not applicable | N06 |
| [f973b1d7d5cd](https://github.com/pingdotgg/t3code/commit/f973b1d7d5cd7ffb4dd3c1c61cc0caece09fd1a6) | fix(server): name the cause of a failed git command (#8645) | Not applicable | N11 |
| [21eb9a421aff](https://github.com/pingdotgg/t3code/commit/21eb9a421aff7a8fbe044b1a34bb3e364fb94a68) | fix(server): PR watch wakes the agent when a bot edits its review comment (#15415) | Not applicable | N11 |
| [24238fee321e](https://github.com/pingdotgg/t3code/commit/24238fee321e2c6bc711832b8cf3698b1f6b30ca) | feat(source-control): omit agent credits from PR merge messages (#16192) | Not applicable | N11 |
| [5cebd3b814ae](https://github.com/pingdotgg/t3code/commit/5cebd3b814ae171df83fb81db3036fbbca2a660e) | fix(clients): dropped connections say why in the client trace (#16200) | Not applicable | N07 |
| [404e229f22ef](https://github.com/pingdotgg/t3code/commit/404e229f22ef2dc7e42074d5379d5c554dd2c083) | fix(server): PR watch reports a required check that first appears already passed (#15804) | Not applicable | N06 |
| [2a68a777afda](https://github.com/pingdotgg/t3code/commit/2a68a777afda95aa23538b9954c5a15221ed2857) | fix: a failed DPoP key load and a fresh maintenance read are no longer cached (#15500) | Not applicable | N60 |
| [5eb87730871a](https://github.com/pingdotgg/t3code/commit/5eb87730871ac6125266b48b98d067de00a19ec7) | fix: tool screenshots show as images, not base64 text (#16199) | Already covered | AgentToolImages renders validated tool images. |
| [f5eb25071c04](https://github.com/pingdotgg/t3code/commit/f5eb25071c042c4a729ddae9fd07bd81804907da) | docs(mcp): thread tools reach threads in any project (#15947) | Not applicable | N15 |
| [207f6a10b05c](https://github.com/pingdotgg/t3code/commit/207f6a10b05cbdaee6fc650e915ba09dbe39d428) | fix(threads): threads watching a PR stay in Working instead of bouncing to the inbox (#16204) | Not applicable | N43 |
| [3d2a98f99be1](https://github.com/pingdotgg/t3code/commit/3d2a98f99be10b7621a368f4e05f36a853ac48d1) | chore(deps): bump cursor sdk and astro to clear vulnerable transitives (#16214) | Not applicable | N15 |
| [54b6b667f340](https://github.com/pingdotgg/t3code/commit/54b6b667f340c03132e9fc2cb2c3d657571442a2) | fix(server): PR sync waits out a GitHub rate limit pause instead of failing every PR (#16203) | Not applicable | N06 |
| [7dfb86a32be6](https://github.com/pingdotgg/t3code/commit/7dfb86a32be6ec064dd8988e1a3b44cc046ff958) | feat(server): scheduled tasks can run on a webhook (#15085) | Not applicable | N20 |
| [b070e52d3e6b](https://github.com/pingdotgg/t3code/commit/b070e52d3e6b49c98117249e5b3f0d46edce173a) | feat(relay): forward webhook requests to the environment's tunnel (#15086) | Not applicable | N33 |
| [aee889b0e043](https://github.com/pingdotgg/t3code/commit/aee889b0e043fdc652e48850698d08ebc563d5bc) | feat(mobile): create and copy webhook automations (#15087) | Not applicable | N01 |
| [ea4b44343d3c](https://github.com/pingdotgg/t3code/commit/ea4b44343d3c8f71cf49a72a266d00eac7389435) | feat(web): create webhook automations and inspect their deliveries (#15088) | Not applicable | N02 |
| [f33b060cafc9](https://github.com/pingdotgg/t3code/commit/f33b060cafc96d7c3229744315c230a3b8d1f631) | feat(relay,server,web,mobile): opt-in to hold webhooks while offline (#15487) | Not applicable | N03 |
| [9e5229b47f7a](https://github.com/pingdotgg/t3code/commit/9e5229b47f7af66e2f88447a49f8ea7f00c734c6) | fix(server): PR watches stop burning GitHub's rate limit and giving up (#16208) | Not applicable | N06 |
| [cbb731beb36d](https://github.com/pingdotgg/t3code/commit/cbb731beb36db2330c9a2632b11354315658ea21) | fix(server): delegation sees a fixed provider without the app open (#16219) | Not applicable | N06 |
| [2afe87ea4bb9](https://github.com/pingdotgg/t3code/commit/2afe87ea4bb94f4dbbd1660a556cbffefd7498a2) | feat: new branches use the shorter t3/ prefix (#16220) | Not applicable | N11 |
| [c138163c8cc3](https://github.com/pingdotgg/t3code/commit/c138163c8cc39d4a98b2d3d796bcef183013bac5) | perf: cheaper shell refreshes, one copy of Codex streaming text, no MCP wait polling (#15033) | Already covered | The native Go decoder emits one text delta; shell/live-row/MCP polling layers in this commit are excluded. |
| [677d1527c3d9](https://github.com/pingdotgg/t3code/commit/677d1527c3d96b90b2294d9a0aed7d8c8874b29b) | feat: agents can show HTML pages inline in threads (#15968) | Not applicable | N57 |
| [25d5c7cacb99](https://github.com/pingdotgg/t3code/commit/25d5c7cacb99bc50056edc0ea8d201eac31cfdf4) | chore(relay): match Alchemy to the PS-80 Postgres cluster (#16228) | Not applicable | N33 |
| [9f61ba674115](https://github.com/pingdotgg/t3code/commit/9f61ba674115f3ddb67a8a1fbe0a0ac8839551bf) | feat: agents ask the user for a secret through a private card (#15907) | Already covered | Secret answers use password fields and are never copied. |
| [3da82d5d384a](https://github.com/pingdotgg/t3code/commit/3da82d5d384a5f45e3e4a9c5bdb8336eb54e49ff) | feat(web): see and stop pull request watches in the thread details card (#16235) | Not applicable | N02 |
| [27ab45593fdc](https://github.com/pingdotgg/t3code/commit/27ab45593fdcaffe718c1fa54dcbb2df4bd769a2) | fix(server): ACP mode states with null descriptions are no longer dropped (#16218) | Already covered | ACP modes with null descriptions remain available. |
| [ca952677321a](https://github.com/pingdotgg/t3code/commit/ca952677321abf9f3762ae4bb0129454e5a63955) | fix(server): finished outbox rows and old PR cache files are pruned (#16247) | Not applicable | N11 |
| [cfa4f765ec05](https://github.com/pingdotgg/t3code/commit/cfa4f765ec05950a032b6c1cf9cdfff0c2391545) | fix(relay): releasing a tunnel that still has a connector no longer 500s (#16250) | Not applicable | N33 |
| [7a047d0136de](https://github.com/pingdotgg/t3code/commit/7a047d0136de606accc8270dce64f2260529c120) | refactor(server,relay): webhook capabilities live in services, not handlers (#16232) | Not applicable | N06 |
| [ee6991ad502c](https://github.com/pingdotgg/t3code/commit/ee6991ad502c7c8979ab47b6a8fc645498f5b4ce) | fix(server): a T3 Connect preferences save finishes even if the client disconnects (#16266) | Not applicable | N06 |
| [6302d66f7773](https://github.com/pingdotgg/t3code/commit/6302d66f7773ee43d198155d2cfc22d9d66dbbe0) | refactor(server): import service modules as namespaces, not aliased layers (#16267) | Not applicable | N06 |
| [fc4ecc835e6d](https://github.com/pingdotgg/t3code/commit/fc4ecc835e6d54eb411c333d737f2c4881884338) | feat(server): log how long PR watches stay quiet before they end (#16262) | Not applicable | N06 |
| [102494157506](https://github.com/pingdotgg/t3code/commit/1024941575067f68d909c6b98f56b02edac25b37) | feat(server,web): choose where new worktrees are created (#16231) | Not applicable | N11 |
| [dcbaf0527998](https://github.com/pingdotgg/t3code/commit/dcbaf05279984187b96c99edc2416abf91e77b96) | perf(server): idle status polls and PR sweeps start fewer git processes (#16272) | Not applicable | N11 |
| [758dc290e900](https://github.com/pingdotgg/t3code/commit/758dc290e900e35958d37fec41fed4a357ad60fa) | perf(server): PR watches spend ~90% fewer GitHub points by checking a 1-point fingerprint first (#16270) | Not applicable | N11 |
| [4df84a7d03ff](https://github.com/pingdotgg/t3code/commit/4df84a7d03ff084ff4c6dd7aef54623984690fb7) | fix(pull-requests): PR detail reads no longer drain the GitHub quota (#16280) | Not applicable | N11 |
| [ecfdda5fa804](https://github.com/pingdotgg/t3code/commit/ecfdda5fa804582066c1aa62afea419677c24883) | refactor: layer variables are named layer or layerXyz (#16282) | Not applicable | N62 |
| [9422e2ee12fd](https://github.com/pingdotgg/t3code/commit/9422e2ee12fd3e3bfdf6fe5c377672badf7b20ca) | refactor(server): T3 Connect link capabilities live in a CloudLink service (#16265) | Not applicable | N06 |
| [db4d0a8e6e49](https://github.com/pingdotgg/t3code/commit/db4d0a8e6e49d92a9efcd8d6f82bbed10e2bd748) | fix(web): sidebar drag and drop no longer snaps back (#16291) | Not applicable | N02 |
| [64275ae39653](https://github.com/pingdotgg/t3code/commit/64275ae39653379e63f49f87be021c190fd20e81) | fix(web): inline HTML renders no longer trap the thread's scroll (#16283) | Not applicable | N03 |
| [3d17862e5855](https://github.com/pingdotgg/t3code/commit/3d17862e58554460b0baae7bae0e03d36b8ce5ab) | refactor(server): one module per service instead of Services/ and Layers/ folders (#16295) | Not applicable | N62 |
| [ea54be7f853a](https://github.com/pingdotgg/t3code/commit/ea54be7f853aca37981f60958d66039196f13a0c) | chore(review): configure CodeRabbit in TypeScript (#16281) | Not applicable | N15 |
| [6262e5f50081](https://github.com/pingdotgg/t3code/commit/6262e5f50081845bb247e5af2fc81b068c8ae893) | docs: put the Effect and web UI review rules in the docs (#16286) | Not applicable | N15 |
| [31d0c8a4f47e](https://github.com/pingdotgg/t3code/commit/31d0c8a4f47ead36c9a63ac4d3bf27b55c05095a) | chore(lint): require a reason on every lint and type-checker suppression (#16294) | Not applicable | N03 |
| [8bd7745f0af9](https://github.com/pingdotgg/t3code/commit/8bd7745f0af918582807326d90e58d59324bb2a6) | refactor(relay): import HookInboxObject once, as a namespace (#16307) | Not applicable | N33 |
| [f36df1852423](https://github.com/pingdotgg/t3code/commit/f36df1852423fef7f7855f86946c9f9a95c3429f) | fix(web): a rejected desktop-local credential is not retried every poll (#16273) | Not applicable | N02 |
| [5afe18eaf9d2](https://github.com/pingdotgg/t3code/commit/5afe18eaf9d20c1633b4ec9177c6c9309a8f3d99) | feat(desktop): the renderer's bootstrap token rotates every 12 hours (#16275) | Not applicable | N03 |
| [9c49228f5425](https://github.com/pingdotgg/t3code/commit/9c49228f5425f644cff9b41ae88ca564d40d79fa) | fix(web): recover from a closed IndexedDB connection (#16311) | Not applicable | N02 |
| [18c1210810cb](https://github.com/pingdotgg/t3code/commit/18c1210810cbdb798d28edda1826fa4de91992d1) | fix(relay): stop forcing manual relay deploys by default (#13563) | Not applicable | N15 |
| [7b32b2ca3989](https://github.com/pingdotgg/t3code/commit/7b32b2ca39892bc59e1d8cfff7aea2a8fbefa73a) | feat(relay): measure the managed tunnel backlog (#13564) | Not applicable | N33 |
| [41d2535a90fd](https://github.com/pingdotgg/t3code/commit/41d2535a90fdfb49b2b9a7a7c8fb4bce09c4170f) | feat(relay): clean up tunnels of hosts that never registered recovery (#13565) | Not applicable | N33 |
| [6bfc8baeb8f2](https://github.com/pingdotgg/t3code/commit/6bfc8baeb8f280e57ef6d0d514680b0bce8538a2) | perf(relay): delete expired tunnels four at a time within a time budget (#13566) | Not applicable | N33 |
| [6b456a9618c3](https://github.com/pingdotgg/t3code/commit/6b456a9618c35f2fa525f6dfc9ef9b0ff01471b8) | feat(connect): tell users when an idle tunnel was removed (#13567) | Not applicable | N03 |
| [204a6390967b](https://github.com/pingdotgg/t3code/commit/204a6390967b14311d40ab9205e062302a0c195a) | docs(relay): add the legacy tunnel cleanup rollout runbook (#13568) | Not applicable | N15 |
| [c21f93e43087](https://github.com/pingdotgg/t3code/commit/c21f93e43087c5b7d7f18b2d45d991cbf1f4d28c) | chore(review): point CodeRabbit at the web UI conventions (#16324) | Not applicable | N15 |
| [3a9c1a6df1b7](https://github.com/pingdotgg/t3code/commit/3a9c1a6df1b71d8ba73d287be90443e8e874802f) | chore(review): turn off CodeRabbit's docstring coverage check (#16328) | Not applicable | N15 |
| [ff873b2ec197](https://github.com/pingdotgg/t3code/commit/ff873b2ec1972b701c3f9edd6a0f727c91604dc4) | refactor(server): CloudLink keeps only the link lifecycle; pure checks live beside it (#16340) | Not applicable | N06 |
| [68e50db20b37](https://github.com/pingdotgg/t3code/commit/68e50db20b3784bad69dd754165020a08f2cd42a) | refactor(server): CloudLink fails with its own errors; the connect routes map them to HTTP (#16341) | Not applicable | N06 |
| [99db70cf1a1f](https://github.com/pingdotgg/t3code/commit/99db70cf1a1f24deca80e40a94a6cdbbaf511da6) | refactor(server): replay guards stay in CloudLink (#16349) | Not applicable | N06 |
| [017c9a0eb53b](https://github.com/pingdotgg/t3code/commit/017c9a0eb53b2c100b5be2d366d2374038da9241) | fix(server): forks no longer merge into their upstream repo's project group (#16353) | Not applicable | N03 |
| [2103bfaa77c4](https://github.com/pingdotgg/t3code/commit/2103bfaa77c431e0923ceb53b52418c0eef174da) | fix(server): stop the startup project sync from delaying the app window (#14912) | Not applicable | N06 |
| [ff2fda7d6d59](https://github.com/pingdotgg/t3code/commit/ff2fda7d6d59a05b9077b8fc48a5e0b67054b8d5) | fix(web): avoid blocking image preparation conversions (#13342) | Not applicable | N02 |
| [fec71c5ab4fd](https://github.com/pingdotgg/t3code/commit/fec71c5ab4fd860331fb15ce37c15edf6db1c236) | fix(server): return partial workspace index on timeout (#11500) | Not applicable | N06 |
| [b69f8b5ff079](https://github.com/pingdotgg/t3code/commit/b69f8b5ff079cccc259bf2a001c2da2bc25d6892) | fix(server): probe project favicon candidates concurrently (#12543) | Not applicable | N06 |
| [570b72d83978](https://github.com/pingdotgg/t3code/commit/570b72d83978f718a5e677002bde3abe55a6d797) | fix(observability): a failing trace disk no longer stalls the server (#13758) | Not applicable | N27 |
| [447046b58914](https://github.com/pingdotgg/t3code/commit/447046b589140a50a4a782453133da83ff77e85e) | fix(server): status polling no longer locks the git index (#14718) | Not applicable | N06 |
| [71964e670ff2](https://github.com/pingdotgg/t3code/commit/71964e670ff212006ebb8de26585649c2adced68) | perf(shared): scan PATH once per command before spawning, not on every spawn (#12600) | Not applicable | N27 |
| [61a0ec9eb0ad](https://github.com/pingdotgg/t3code/commit/61a0ec9eb0ad12dd4ef054d2dbab8b5eedbdaf06) | fix(server): main's startup auto-pull test compiles again (#16357) | Not applicable | N06 |
| [24edd55d9459](https://github.com/pingdotgg/t3code/commit/24edd55d9459edfa0a2956094b7868102bba5b6f) | fix(server): project favicons stop being rescanned every minute (#16206) | Not applicable | N06 |
| [255ac27d5200](https://github.com/pingdotgg/t3code/commit/255ac27d5200b01c50acef105ab0ec8bb6ab8345) | fix(server): Claude limits load again for users with large transcript histories (#16358) | Not applicable | N21 |
| [3236d8ce73d0](https://github.com/pingdotgg/t3code/commit/3236d8ce73d06df91e30532d0a5fd03cb537de8e) | Add esthor to the list of GitHub users | Not applicable | N15 |
| [eef36a67c37f](https://github.com/pingdotgg/t3code/commit/eef36a67c37fa74905fc7feaa836e5a81daa8685) | fix(server): caches and ids are written atomically (#16242) | Not applicable | N60 |
| [327a29ef56e2](https://github.com/pingdotgg/t3code/commit/327a29ef56e215c69a5db5d0b6f8df89f04f5d5b) | fix(server): one-shot initializers no longer race (#16260) | Already covered | Each Go session owns an independently initialized Codex process; no app-server initialization is shared across threads. |
| [bfe40e6d2574](https://github.com/pingdotgg/t3code/commit/bfe40e6d2574447b0651bdb41337db743ddbe12c) | fix(server): the PR cache sweep only removes real entry files (#16285) | Not applicable | N11 |
| [4b3568b775d8](https://github.com/pingdotgg/t3code/commit/4b3568b775d86fc699938ec5a9ada773bad8b45f) | chore: keep one copy each of undici 8 and ws 8 (#16211) | Not applicable | N15 |
| [24e4b62c883e](https://github.com/pingdotgg/t3code/commit/24e4b62c883eb67251bdfb07d2bf22cfd3987820) | fix(shared): DrainableWorker keeps running after a failed item (#16223) | Not applicable | N27 |
| [c299308a832d](https://github.com/pingdotgg/t3code/commit/c299308a832db382fce61233bd10a4bd13bbc60c) | fix(server): metrics count interrupted work on the monotonic clock (#16207) | Not applicable | N06 |
| [efeeb7737bed](https://github.com/pingdotgg/t3code/commit/efeeb7737bed89bbfc6dc81662058f8ce7fba2ea) | refactor(web): import connection storage as a namespace in its test (#16315) | Not applicable | N02 |
| [2d162d157dfb](https://github.com/pingdotgg/t3code/commit/2d162d157dfb0eea7e205c508f47d663ab4e8bf9) | fix(contracts): trimmed IDs round-trip (#16300) | Not applicable | N01 |
| [fad673225650](https://github.com/pingdotgg/t3code/commit/fad673225650193fad2a85487ecffc903f8ff30a) | fix(server): main's settings, keybindings and session tests compile again (#16363) | Not applicable | N06 |
| [ff397a29523d](https://github.com/pingdotgg/t3code/commit/ff397a29523d3c63e5844b11608c0189640c7561) | chore(lint): catch known tags with Effect.catchTags (#16361) | Not applicable | N31 |
| [25f0b2a1b732](https://github.com/pingdotgg/t3code/commit/25f0b2a1b7323563aa5d4e7f70e78a3889c5918e) | fix(observability): T3 Connect tracing stops at the relay boundary (#16314) | Not applicable | N11 |
| [0ecb78ed0d35](https://github.com/pingdotgg/t3code/commit/0ecb78ed0d351b65ef68262f632359223f9c389b) | fix(relay): error and deadline responses carry CORS headers (#16253) | Not applicable | N33 |
| [8f75697b9e02](https://github.com/pingdotgg/t3code/commit/8f75697b9e022033f2615cb32e3c6ddc61c52353) | fix(web): bring back the live shimmer on work log rows (#16372) | Not applicable | N18 |
| [83b982a85c87](https://github.com/pingdotgg/t3code/commit/83b982a85c87b4c0eda08beb9028e3a73f7c89b6) | refactor: Effect code gets UUIDs and SHA-256 from Effect's Crypto (#16377) | Not applicable | N31 |
| [442735897f6f](https://github.com/pingdotgg/t3code/commit/442735897f6f92af33798075baa12d4fd4d710dd) | fix(relay): export traces through one tracer, one request span each (#16382) | Not applicable | N33 |
| [4ae976dbae39](https://github.com/pingdotgg/t3code/commit/4ae976dbae39b3e80243b864a8b61da00f642dc4) | fix(server): Pi thread titles use linked PR context (#16210) | Not applicable | N06 |
| [76d3c96fd5b8](https://github.com/pingdotgg/t3code/commit/76d3c96fd5b89cca2c06aeaceaea16100c75084a) | fix(desktop): retry transient bearer bootstrap and degrade on session fetch failure (#12919) | Not applicable | N09 |
| [0290c4a04e52](https://github.com/pingdotgg/t3code/commit/0290c4a04e52ca4e93be7a6fb94c14bcc7b53728) | fix(server): avoid scanning completed history for pending secrets (#16409) | Not applicable | N06 |
| [a7a2230c33fb](https://github.com/pingdotgg/t3code/commit/a7a2230c33fbec9d6616a5de26f09e1fa8856c05) | fix(orchestration): let Stop recover stalled runs (#15442) | Already covered | CloseSession then Reconnect recovers a stalled Stop. |
| [1f2b10c25bc4](https://github.com/pingdotgg/t3code/commit/1f2b10c25bc400461e6c114c2c7236e91996225c) | fix(release): resolve version-qualified catalog overrides (#16411) | Not applicable | N15 |
| [d0dc45885c14](https://github.com/pingdotgg/t3code/commit/d0dc45885c1441da5bd46dcf13124d94ed728b0a) | fix(web): type in front of bold that starts a composer line (#13217) | Not applicable | N02 |
| [411cef0008fa](https://github.com/pingdotgg/t3code/commit/411cef0008fa8e1e5251a8aae2cc6a215f700a9c) | fix(desktop): prevent browser screenshot filename collisions (#14784) | Not applicable | N09 |
| [e65063ccf6ce](https://github.com/pingdotgg/t3code/commit/e65063ccf6ce83aa33d10521aeaf929210f0ff04) | fix(server): end clone options before the repository URL (#14781) | Not applicable | N11 |
| [4980ba282eb3](https://github.com/pingdotgg/t3code/commit/4980ba282eb313b3be87a6b7df28f3b42dcb4ce9) | fix(web): queued messages no longer split the composer notice stack (#16400) | Not applicable | N22 |
| [9ae9f2ba1f30](https://github.com/pingdotgg/t3code/commit/9ae9f2ba1f30e5fc14f69ba6dff6b08c7bfb401c) | fix(server): reject invalid explicit Bitbucket repositories (#15876) | Not applicable | N11 |
| [4fb04fe7fae5](https://github.com/pingdotgg/t3code/commit/4fb04fe7fae566cc91ab810888b3232f274a00cf) | fix(desktop): use the crypto service for screenshot IDs (#16415) | Not applicable | N11 |
| [e6abc01788ee](https://github.com/pingdotgg/t3code/commit/e6abc01788eef557c214a086999b3fd635b69f71) | fix(shared): find versioned JetBrains macOS app bundles (#16246) | Not applicable | N06 |
| [9bd1d8009a6b](https://github.com/pingdotgg/t3code/commit/9bd1d8009a6b7c50f9dd9458e2bf27d481ff3b43) | fix(server): OpenCode 2 threads get T3 Code's MCP tools (#16142) | Not applicable | N06 |
| [ac8e9453ca09](https://github.com/pingdotgg/t3code/commit/ac8e9453ca0948469ccf5651b765086d8d9ecd64) | feat(preview): run the browser on the environment server (#15328) | Not applicable | N16 |
| [17c0878941ab](https://github.com/pingdotgg/t3code/commit/17c0878941ab8ab19108fb639c5138d92a89e919) | fix: restore service references breaking ci (#16495) | Not applicable | N03 |
| [dff412c3411f](https://github.com/pingdotgg/t3code/commit/dff412c3411f34b3150983f00686a98b600526e3) | fix(mcp): mark declared tool failures as errors (#15617) | Not applicable | N06 |
| [fd1c3386c4d6](https://github.com/pingdotgg/t3code/commit/fd1c3386c4d60f3477ab3f13c87537848de099f5) | fix(release): unblock nightly browser tests and cli builds (#16515) | Not applicable | N06 |
| [b38aa18365fe](https://github.com/pingdotgg/t3code/commit/b38aa18365fe314e079d99670932bedfc7407e5a) | fix(mcp): preserve thread command rejection reasons (#15627) | Not applicable | N06 |
| [78552a8c6162](https://github.com/pingdotgg/t3code/commit/78552a8c6162c9b5248231bc70be8bffdb50cbd3) | chore(deps): upgrade @effect/tsgo to 0.46.1 (#16360) | Not applicable | N31 |
| [eea9f7cfebbd](https://github.com/pingdotgg/t3code/commit/eea9f7cfebbdb51d27f07a816119c71a67509997) | fix(orchestration): show reported subagent models (#14108) | Not applicable | N51 |
| [9503155da67a](https://github.com/pingdotgg/t3code/commit/9503155da67add4fbd7fbc51c99ec19a0c8c4a2c) | fix(web): Apple logo no longer dips below the device host label (#14825) | Not applicable | N02 |
| [29366ea29abf](https://github.com/pingdotgg/t3code/commit/29366ea29abf4ceb41ab250d06e190a22258bd18) | fix(web): show subagent effort and speed in hover cards (#13056) | Not applicable | N57 |
| [b83d731cabbf](https://github.com/pingdotgg/t3code/commit/b83d731cabbfd32c7479bb2db7e6afbfb2085ff2) | feat(web): reopen closed tabs across the app (#15207) | Not applicable | N16 |
| [8c61e254fd07](https://github.com/pingdotgg/t3code/commit/8c61e254fd07249b8c420c44d62e8ecd72a78a20) | fix(web): stop wide ordered list markers from clipping (#16523) | Ported | Preserve ordered-list marker gutters, including the missing ol renderer (#16523). |
| [9b2c03ee2532](https://github.com/pingdotgg/t3code/commit/9b2c03ee25321c16928eda74f4036f36f0b1afce) | fix(desktop): build AppImage with the static runtime toolset (fixes libfuse2 launch failure) (#7765) | Not applicable | N15 |
| [7bf6de174171](https://github.com/pingdotgg/t3code/commit/7bf6de174171ffcf0215bf55556f0e3ee7e6d78f) | fix(mobile): keep usage-limit notice opaque (#15602) | Not applicable | N01 |
| [1564aaa5d29d](https://github.com/pingdotgg/t3code/commit/1564aaa5d29deb2520d1211f9f35dd6150b1512e) | feat(server): GitHub API transport that uses gh only for the token (#16319) | Not applicable | N11 |
| [adc3c9327abe](https://github.com/pingdotgg/t3code/commit/adc3c9327abedda6a36694e2344e620fe0d62ae4) | feat(server): pull requests talk to GitHub's API instead of the gh CLI (#16320) | Not applicable | N11 |
| [649418f01fcb](https://github.com/pingdotgg/t3code/commit/649418f01fcb4ae5ebbc07739e0bfee8fae3aaec) | feat(server): source control, media and discovery use GitHub's API instead of gh (#16321) | Not applicable | N11 |
| [9ac8f33f1685](https://github.com/pingdotgg/t3code/commit/9ac8f33f1685d1d6e843b657829d0d7462a13813) | feat: choose the GitHub account per host, save a GitHub token, and fewer reads per PR action (#16322) | Not applicable | N11 |
| [1eae9c2efec4](https://github.com/pingdotgg/t3code/commit/1eae9c2efec43b1d27e60d967001f5721c4a0429) | fix(server): Rebase stack moves each layer onto the rebased layer below it (#16551) | Not applicable | N11 |
| [3dfe373fddb9](https://github.com/pingdotgg/t3code/commit/3dfe373fddb966101371805b0b102ae5af75b15b) | fix(web): diff panel keeps the chosen scope while a turn runs (#16571) | Not applicable | N02 |
| [78c9fcaafcbe](https://github.com/pingdotgg/t3code/commit/78c9fcaafcbee9e2b941d8b0129a9f968373c6b2) | fix(desktop): honor the telemetry opt-out from the shell profile (#16563) | Not applicable | N09 |
| [3b44698d07c7](https://github.com/pingdotgg/t3code/commit/3b44698d07c722471279d6a933499d6108148632) | docs(marketing): disclose product usage data in the privacy policy (#16562) | Not applicable | N19 |
| [115640ae4992](https://github.com/pingdotgg/t3code/commit/115640ae49924c9e4e6215bd8d911445ad932bff) | feat(web): note anonymous usage data in onboarding and link the privacy policy (#16564) | Not applicable | N02 |
| [f21d6da51c9a](https://github.com/pingdotgg/t3code/commit/f21d6da51c9aa78ea12e7f13a378f55f1e63cb38) | perf(web): diff panel no longer re-renders every file header each time a patch arrives (#16033) | Not applicable | N02 |
| [2f85686d9dca](https://github.com/pingdotgg/t3code/commit/2f85686d9dca5ec98c892a2a26c321e09d746d71) | feat(server): every T3 MCP tool declares who may call it (#16335) | Not applicable | N63 |
| [2c8be5893eb3](https://github.com/pingdotgg/t3code/commit/2c8be5893eb3754071162e344a8c8e8a67bef597) | feat(server): outside agents sign in to the T3 MCP server with OAuth (#16336) | Not applicable | N03 |
| [fbe5df2d4b63](https://github.com/pingdotgg/t3code/commit/fbe5df2d4b630d13adc8fe2d38cab354e6d66d67) | feat(web): copy an environment's MCP URL for outside agents (#16337) | Not applicable | N02 |
| [7bee189dfb5c](https://github.com/pingdotgg/t3code/commit/7bee189dfb5cd6807457e98af0c2f95f5ed50edd) | chore(tsconfig): turn off the Schema-over-JSON diagnostic in test files (#16375) | Not applicable | N31 |
| [c9b21b28e2ce](https://github.com/pingdotgg/t3code/commit/c9b21b28e2cebc4c9cbd80726e757df36309dea8) | chore(review): CodeRabbit gates outside contributors' pull requests (#16332) | Not applicable | N15 |
| [f4f148eb6706](https://github.com/pingdotgg/t3code/commit/f4f148eb670622a049ae6561d7795011e383fc43) | fix(desktop): include Linux package license and app metadata (#16597) | Not applicable | N09 |
| [8ddf200e8f63](https://github.com/pingdotgg/t3code/commit/8ddf200e8f637eaf9a4f0d3036ece8f1d3869d10) | fix(server): one failing RPC handler no longer ends the client's other requests (#15515) | Not applicable | N06 |
| [13aabf3d7d0e](https://github.com/pingdotgg/t3code/commit/13aabf3d7d0ec13e902234e86d3df27149c0e4a9) | fix(contracts): a context record that cannot be encoded no longer fails the send (#16398) | Not applicable | N03 |
| [d8d037eae1b7](https://github.com/pingdotgg/t3code/commit/d8d037eae1b77a77f372fd1afd2662a31ebe37c8) | fix(web): open pull request row actions on right-click (#16612) | Not applicable | N02 |
| [876f8656e310](https://github.com/pingdotgg/t3code/commit/876f8656e310f365eab98678e68bcc45615fec7f) | fix(web): show attempted paths in file preview errors (#15628) | Not applicable | N02 |
| [5055de3ae483](https://github.com/pingdotgg/t3code/commit/5055de3ae483d5ece45695ad682b9766ff0cf423) | fix(vcs): passive sidebar rows stop retaining remote pollers (#15666) | Not applicable | N11 |
| [72d5c32ba679](https://github.com/pingdotgg/t3code/commit/72d5c32ba67953805feb6fe9ad3b70b632a64c47) | feat(web): group keybindings settings by area with a page toolbar (#12822) | Not applicable | N02 |
| [c5c6f9cb485b](https://github.com/pingdotgg/t3code/commit/c5c6f9cb485b2db0f205797394bc95f9f575d139) | feat(web): stop T3-owned subagents from Lineage (#15211) | Not applicable | N02 |
| [ab28039ef77c](https://github.com/pingdotgg/t3code/commit/ab28039ef77ca8847c6bfe845c3502ff7e348c30) | feat(web): add fast actions to linked pull requests (#16627) | Not applicable | N02 |
| [7202ba666080](https://github.com/pingdotgg/t3code/commit/7202ba666080cfca1351c080d7aaa2d81bfa9a66) | feat(web): open right panel tab menu with Mod+T (#15686) | Not applicable | N02 |
| [4b5c60485366](https://github.com/pingdotgg/t3code/commit/4b5c60485366421794787aee7513913f92eced29) | fix(server): provider sessions clean up when their start is interrupted (#15571) | Already covered | Shutdown cancels a pending native handshake. |
| [517188b3caa1](https://github.com/pingdotgg/t3code/commit/517188b3caa135bc09ec0a4f93474ef17c8b5c4f) | fix(web): show "No project" near the top of the new thread picker (#16628) | Not applicable | N02 |
| [d021f57bf3a5](https://github.com/pingdotgg/t3code/commit/d021f57bf3a500e419e800c3eb957568bed8f713) | refactor(server): instrument WS RPCs in group middleware (#15548) | Not applicable | N06 |
| [f8ed2a039b7d](https://github.com/pingdotgg/t3code/commit/f8ed2a039b7deaa324beda29b44cc74c846311f7) | chore(deps): upgrade @pierre/diffs to 1.5.2 and @pierre/trees to beta.6 (#16644) | Not applicable | N03 |
| [c4481b9b07d9](https://github.com/pingdotgg/t3code/commit/c4481b9b07d9108d3aa93494a0ec5dd28ed8fd80) | fix(relay): a host restarting onto a deleted tunnel gets a new one (#16649) | Not applicable | N33 |
| [740bda40c92e](https://github.com/pingdotgg/t3code/commit/740bda40c92e1ded65dd2b65ef7c583357b1d258) | fix(server): recover a deleted tunnel when Cloudflare says "Tunnel not found" (#16648) | Not applicable | N06 |
| [2991331515ed](https://github.com/pingdotgg/t3code/commit/2991331515ed5b7e4df0751e5119fa04954e803a) | fix(web): iPhone Duo fold controls follow the phone's orientation (#16630) | Not applicable | N02 |
| [847c53df0126](https://github.com/pingdotgg/t3code/commit/847c53df0126a225fb1a64d4bda9acfffa2b68ed) | fix(web): keep workspace options when expanding lineage (#16635) | Not applicable | N02 |
| [5886bd8f19c8](https://github.com/pingdotgg/t3code/commit/5886bd8f19c8d4b01d7a0b36ad62a9d35f02552d) | fix(web): preserve bare anchor placeholders in markdown (#16637) | Ported | Keep unmatched <A> placeholders as literal text (#16637). |
| [bfec2387b810](https://github.com/pingdotgg/t3code/commit/bfec2387b8102975c84690f99be0f5f834fd0cbe) | fix(pi): preserve provider identity in discovered models (#16661) | Not applicable | N13 |
| [e9c3d950f023](https://github.com/pingdotgg/t3code/commit/e9c3d950f023130791d8ac5fe7aa0992e3430738) | fix(auth): preserve explicitly granted pairing scopes (#9785) | Not applicable | N03 |
| [28ce9d1c1751](https://github.com/pingdotgg/t3code/commit/28ce9d1c1751d70c869e3de3287b96d5e28fc845) | feat(auth): separate environment administration permissions (#9786) | Already covered | Request controls are gated by active/disabled/staleRequestIds. |
| [2eda8ac6a315](https://github.com/pingdotgg/t3code/commit/2eda8ac6a3151b70b8ee49b438bfb6fcd835802e) | feat(auth): separate source control write permissions (#9787) | Not applicable | N03 |
| [321c6f9d26e9](https://github.com/pingdotgg/t3code/commit/321c6f9d26e9240f6ed3d2a3a8f7fc11344c2f17) | feat(auth): separate filesystem read and write permissions (#9788) | Already covered | Request controls are gated by active/disabled/staleRequestIds. |
| [7175edcff4f2](https://github.com/pingdotgg/t3code/commit/7175edcff4f250b583ebed8877e27e36dc48411e) | feat(auth): separate browser preview control permissions (#9789) | Already covered | Request controls are gated by active/disabled/staleRequestIds. |
| [57b2418d4fba](https://github.com/pingdotgg/t3code/commit/57b2418d4fbab56290ff4cb861dcd0a7d0f5ef63) | feat(auth): separate diagnostics and usage permissions (#9790) | Not applicable | N03 |
| [cc41482df023](https://github.com/pingdotgg/t3code/commit/cc41482df0232454af0cc5ac2f1afd984e342134) | feat(auth): allow passive terminal observation (#9791) | Not applicable | N03 |
| [365aa87982a4](https://github.com/pingdotgg/t3code/commit/365aa87982a4d81cc8e0c085e8d1a40ca7daecdc) | fix(auth): keep old clients connected across scope changes (#10298) | Not applicable | N03 |
| [10f39eb9ac80](https://github.com/pingdotgg/t3code/commit/10f39eb9ac80c9a4b7f5097575dd2addc3b6f631) | feat(server): hosted agents like ChatGPT can sign in to the T3 MCP server (#16718) | Not applicable | N03 |
| [0f4f02abb6ab](https://github.com/pingdotgg/t3code/commit/0f4f02abb6ab5428c06a7e4b6c76b3ec84c62498) | docs: connect Claude Code, Codex, ChatGPT and bots over MCP (#16741) | Not applicable | N15 |
| [f60367b8cb5e](https://github.com/pingdotgg/t3code/commit/f60367b8cb5e4a7e5e12dd01dc3889618151b1be) | fix(web): thread details card gives titles room to read (#16746) | Not applicable | N02 |
| [4b063a042ec9](https://github.com/pingdotgg/t3code/commit/4b063a042ec93418a196c020b852459f57e17b57) | fix(mcp): agent HTML pages stop painting slab backgrounds (#16752) | Not applicable | N27 |
| [a1830915ecf7](https://github.com/pingdotgg/t3code/commit/a1830915ecf70e3ad4fbe8efb15ae9f18517eb54) | fix: composer picks up new project skills without a server restart (#16750) | Not applicable | N16 |
| [079e4bccdcff](https://github.com/pingdotgg/t3code/commit/079e4bccdcffc56c54bd69523ed77c006553ea21) | feat(server): run a project action when a worktree thread settles (#16290) | Not applicable | N03 |
| [655dfbe8ba87](https://github.com/pingdotgg/t3code/commit/655dfbe8ba87f0f9ef1d6fad83c20634634653aa) | feat(web): old Claude threads compact on send instead of stacking notices (#16631) | Not applicable | N57 |
| [453e596da7e9](https://github.com/pingdotgg/t3code/commit/453e596da7e98d973e3853a5c2728681dedf363b) | fix(server): settled threads stop polling their pull requests (#16762) | Not applicable | N06 |
| [b77108bc4150](https://github.com/pingdotgg/t3code/commit/b77108bc41506ad7ba28839d9a8e04ba79fe340b) | fix(server): stop storing tool image bytes no client reads (#16652) | Not applicable | N06 |
| [10395bcc5970](https://github.com/pingdotgg/t3code/commit/10395bcc59707302c1476e19186532ff5a9d3557) | fix(server): status refresh no longer pegs CPU in repos with thousands of untracked files (#16771) | Not applicable | N06 |
| [c93080b0a3bf](https://github.com/pingdotgg/t3code/commit/c93080b0a3bf4faffb375169c5ec06adce05eb2f) | perf(server): background branch lookups share one GitHub query per sweep (#16760) | Not applicable | N11 |
| [cd41c4ada0c7](https://github.com/pingdotgg/t3code/commit/cd41c4ada0c70cc2eec95ecd7266f3dab010c58c) | fix(server): threads settle as soon as a client sees their PR merge (#16761) | Not applicable | N11 |
| [611132c171f3](https://github.com/pingdotgg/t3code/commit/611132c171f3a821bd2e32f22261135cef6330ac) | feat(server,web,mobile): agents see snooze state and link to threads (#16782) | Not applicable | N57 |
| [521f5b5a5682](https://github.com/pingdotgg/t3code/commit/521f5b5a5682b18f2d59cf95226fc203cd314240) | fix(web): replace Lineage timers with Stop on hover (#16791) | Not applicable | N02 |
| [6ecb82dc42c4](https://github.com/pingdotgg/t3code/commit/6ecb82dc42c4c721fd2c03c92316ee5361306cdb) | fix(clients): running subagent cards stay visible after their parent turn settles (#16878) | Not applicable | N03 |
| [4d976d1fc233](https://github.com/pingdotgg/t3code/commit/4d976d1fc233d3aa971fa7dcbf2e7316bfa3ac72) | feat: MCP apps render and run inline in threads (#16236) | Not applicable | N57 |
| [a8c480241836](https://github.com/pingdotgg/t3code/commit/a8c480241836a7f6b79862e4504f37801cda6cde) | fix(server): a background Claude subagent's work shows while its parent is idle (#16486) | Not applicable | N54 |
| [8d564a3e5c77](https://github.com/pingdotgg/t3code/commit/8d564a3e5c77d283994119eca82881f8c37a4a4d) | fix(mobile): hide threads from switched-off environments (#16886) | Not applicable | N07 |
| [87c366b1206b](https://github.com/pingdotgg/t3code/commit/87c366b1206b13701fbcc38e459d457f0f9da2df) | fix(server): a refused Claude turn no longer throws away its session (#16287) | Already covered | A refused turn retains its session. |
| [db46c4274487](https://github.com/pingdotgg/t3code/commit/db46c42744874ca123bc170a90c1faa312f20119) | fix(web): onboarding Continue no longer locks on computers that won't connect (#16887) | Not applicable | N02 |
| [d9068d38099f](https://github.com/pingdotgg/t3code/commit/d9068d38099fe5cb4f3fed4cb5455fee878db491) | fix(server): worktree cleanup no longer deletes files hidden by showUntrackedFiles=no (#15834) | Not applicable | N06 |
| [93e864042cf0](https://github.com/pingdotgg/t3code/commit/93e864042cf066f4cdb3efe65dbefaaf0d168894) | fix(server): merged-worktree cleanup removes worktrees after squash merges (#14847) | Not applicable | N11 |
| [e0201d3ab51b](https://github.com/pingdotgg/t3code/commit/e0201d3ab51b4593eff3c41d03ab9c9b2dc07471) | fix(server): free worktrees for terminal thread statuses (#15150) | Not applicable | N06 |
| [4ce6a51a4c4b](https://github.com/pingdotgg/t3code/commit/4ce6a51a4c4bbc9a2fc9e371ecb446851f953482) | fix(server): Windows worktrees with long paths no longer fail or strand (#14917) | Not applicable | N06 |
| [41985b44ff1b](https://github.com/pingdotgg/t3code/commit/41985b44ff1b3a353ab805a31d4e4810842cbf2b) | fix(server): main typechecks again after a test used renamed helpers (#16895) | Not applicable | N06 |
| [f570bd21663f](https://github.com/pingdotgg/t3code/commit/f570bd21663f56ce94c41829d3b7d72886e25a34) | fix(server): Claude prompts no longer hang on a uuid the session already holds (#16897) | Already covered | Native CLI prompt UUIDs are not derived from SDK run-attempt IDs; the Go user frame supplies no uuid. |
| [588d32e22b10](https://github.com/pingdotgg/t3code/commit/588d32e22b10932404856d9bdd9f2f39c7e38070) | feat(models): add Claude Haiku 5.5 and retire Sonnet 5 and Opus 5 to legacy (#16903) | Ported | Reviewed Claude model-manifest data is retained by the local extraction. |
| [ba0ea3d15215](https://github.com/pingdotgg/t3code/commit/ba0ea3d152150f399827d36a5b4b86c5bbda2d7f) | fix(web): iPhone Duo folds animate, center on the hinge, and keep the phone's orientation (#16885) | Not applicable | N02 |
| [9f48092ec95d](https://github.com/pingdotgg/t3code/commit/9f48092ec95dd9eab78931601de95be0107a930c) | fix(server): Claude 5-series models always run 1M context (#16908) | Ported | Reviewed Claude model-manifest data is retained by the local extraction. |
| [73e87f683ae1](https://github.com/pingdotgg/t3code/commit/73e87f683ae1455feb663481f1fa588f01b5784b) | fix(desktop): setup prompts name a t3 that runs on desktop installs (#16676) | Not applicable | N03 |
| [0678e4e23d86](https://github.com/pingdotgg/t3code/commit/0678e4e23d8675ef88f9ac08e1ae90cfe7d6ef2e) | feat(desktop): install the t3 command from Settings (#16683) | Not applicable | N03 |
| [a9fb6a80635c](https://github.com/pingdotgg/t3code/commit/a9fb6a80635c77a1d4fd988e0018059e13628eff) | fix(web): show live names in thread-read activity (#13140) | Not applicable | N57 |
| [468ade30495c](https://github.com/pingdotgg/t3code/commit/468ade30495c450ae4e731483187f77bc3c5e42b) | feat(mobile): adopt v5 navigation and native iPad columns (#16733) | Not applicable | N01 |
| [8a92ebe413dd](https://github.com/pingdotgg/t3code/commit/8a92ebe413dd9e4569724c94e74a1ed54675eff7) | fix(mobile): Android composer picker scrolls past the first four rows (#15856) | Not applicable | N01 |
| [538329603352](https://github.com/pingdotgg/t3code/commit/538329603352173afe3ae9f4586a4f772c0f3a4f) | fix(desktop): sign-in and captchas work again in desktop browser tabs (#16939) | Not applicable | N09 |
| [85489611b547](https://github.com/pingdotgg/t3code/commit/85489611b5477050d7d777a4195c1c50ac0ce6be) | fix(server): missing project folders no longer log favicon warnings (#16757) | Not applicable | N06 |
| [eba052156394](https://github.com/pingdotgg/t3code/commit/eba0521563946a0834af881465d3e1ff7a4ca2f1) | fix(server): unload Codex threads left idle on the shared app-server (#16917) | Not applicable | N06 |
| [26285ab80513](https://github.com/pingdotgg/t3code/commit/26285ab80513260d1ff29d2c3fbe18f21990cb62) | fix(web): fast typing no longer scrambles text when type-to-focus kicks in (#14595) | Not applicable | N02 |
| [7f3628e4153d](https://github.com/pingdotgg/t3code/commit/7f3628e4153d5be4a241c4d54f73a6d7087014ce) | fix(web): simplify workspace card rows (#16823) | Not applicable | N02 |
| [7ac5ec6b1559](https://github.com/pingdotgg/t3code/commit/7ac5ec6b15593f5b682fe73b483bc8e24de76665) | fix(web): Copy MCP URL shows up for environments reached over plain http (#16909) | Not applicable | N02 |
| [abbf5d5bcd04](https://github.com/pingdotgg/t3code/commit/abbf5d5bcd045cdb6baf16128ef54d4df3a7f3fd) | fix(web): C#, Java, PHP and 11 other languages get file icons (#16974) | Not applicable | N02 |
| [95ccbf406626](https://github.com/pingdotgg/t3code/commit/95ccbf406626a8a80fa545777a6388bdf5153086) | feat(clients): live row shows the agent's latest thought (#16284) | Not applicable | N18 |
| [300f7f9d45ff](https://github.com/pingdotgg/t3code/commit/300f7f9d45ff19c01bc987dbf1c3bc3e9a3a59ee) | feat(web): block-level Markdown in the rich text composer (#14677) | Not applicable | N25 |
| [46dd0949053a](https://github.com/pingdotgg/t3code/commit/46dd0949053ae387e785c5e9801725a9ccc28de1) | fix(web): center project monograms in settled rows (#16841) | Not applicable | N02 |
| [3143335fc3a5](https://github.com/pingdotgg/t3code/commit/3143335fc3a568cbbb5174764961889272135cb9) | fix(web): cancelling a new citation no longer leaves a stray space (#16828) | Not applicable | N22 |
| [ff3a04a8ebaa](https://github.com/pingdotgg/t3code/commit/ff3a04a8ebaaf6bb74b8b5b996013b82e044c700) | fix(settings): provider updates show live progress instead of a bare spinner (#16958) | Not applicable | N16 |
| [14fe0158ede3](https://github.com/pingdotgg/t3code/commit/14fe0158ede3d964f183e281f9fec225d30a6b32) | feat(web): find in diffs with Cmd+F (#14623) | Not applicable | N28 |
| [08b1222d0bae](https://github.com/pingdotgg/t3code/commit/08b1222d0bae670433c275ec70165c7b5048726f) | refactor(server): GitHub services are named for the API they call, not gh (#16967) | Not applicable | N11 |
| [a21ac5fb153e](https://github.com/pingdotgg/t3code/commit/a21ac5fb153e3bf418ccd2a4abef2787c241813c) | refactor(server): GitHub GraphQL batches use variables and share one pager (#16960) | Not applicable | N11 |
| [42d92ec59775](https://github.com/pingdotgg/t3code/commit/42d92ec59775c187af69071c81b2b91180810bdd) | refactor(server): GitHub source control reads GitHubApi directly (#16982) | Not applicable | N11 |
| [d720210996a5](https://github.com/pingdotgg/t3code/commit/d720210996a514368ba99f4860063110033d93fa) | refactor(server): GitHub rate limits read the response headers (#16986) | Not applicable | N11 |
| [83a82a46b063](https://github.com/pingdotgg/t3code/commit/83a82a46b0631234ac85748b912566d6f4b1dee3) | chore(review): remove the custom Approvability check (#17018) | Not applicable | N15 |
| [f9e16b3f6647](https://github.com/pingdotgg/t3code/commit/f9e16b3f66476ad1e213152708c9aa8b035085a3) | perf(mobile): show the cached thread list sooner and stop the freeze after it (#16713) | Not applicable | N01 |
| [9fba2093e50f](https://github.com/pingdotgg/t3code/commit/9fba2093e50f208c52d9041c70a793296e311477) | fix(server): replaying a command no longer freezes the server (#17041) | Not applicable | N06 |
| [04ad17425a80](https://github.com/pingdotgg/t3code/commit/04ad17425a80484bf323a7d5076bd477a9499119) | fix(server): opening a long thread no longer blocks the server for a second (#17029) | Not applicable | N06 |
| [0f09627a111f](https://github.com/pingdotgg/t3code/commit/0f09627a111f7e9adc22c256c725565672676f1a) | fix(server,web,mobile): thread links reference the thread id, not a baked title (#17017) | Not applicable | N23 |
| [6f83358ae6d8](https://github.com/pingdotgg/t3code/commit/6f83358ae6d85b5dee7d3280e645e0e9a0aeff62) | fix(web): messages with quotes or links no longer collapse when short (#16624) | Ported | Measure user-message collapse on visible link-label text; attachment portions excluded (#16624). |
| [843c593ab2df](https://github.com/pingdotgg/t3code/commit/843c593ab2dfe432abfba7b928292f3f8e9c7d44) | refactor(web): PR loading skeleton shares the detail panel's layout (#15583) | Not applicable | N02 |
| [2b7fa212a89e](https://github.com/pingdotgg/t3code/commit/2b7fa212a89edda7aad8f73fc3b4afd5b56c442c) | fix(web): PR file headers keep the file name in narrow panels (#15562) | Not applicable | N02 |
| [b24f0fbba09d](https://github.com/pingdotgg/t3code/commit/b24f0fbba09d8623c896081b4ff70aa7190c8fda) | fix(web): PR timeline no longer shifts when its scrollbar appears (#15588) | Not applicable | N02 |
| [0bfd9dd19d0c](https://github.com/pingdotgg/t3code/commit/0bfd9dd19d0c9418a936190525ec8f4c4fdae09b) | fix(server): held queued wakes no longer keep delegated tasks running (#17028) | Not applicable | N06 |
| [c28cd905572e](https://github.com/pingdotgg/t3code/commit/c28cd905572e2a28934a6229d88bcf2975963032) | fix(server): a message sent during a rollback no longer undoes it (#17079) | Not applicable | N06 |
| [6053a7878d24](https://github.com/pingdotgg/t3code/commit/6053a7878d24e5fd596d2b417ed94964a4b9c1c6) | fix(web): PR file stats ignore the hide-whitespace toggle (#16162) | Not applicable | N02 |
| [28f5516a9acf](https://github.com/pingdotgg/t3code/commit/28f5516a9acfcfc1e203487e78869a406fd23b76) | perf(mobile): omit duplicated turn items from bounded thread snapshots (#15385) | Not applicable | N06 |
| [eb17e0db2bd0](https://github.com/pingdotgg/t3code/commit/eb17e0db2bd075e71b4015dc57588e7f838cbcdc) | perf(mobile): pause elapsed-time timers on hidden thread screens (#15397) | Not applicable | N01 |
| [65404eb75de9](https://github.com/pingdotgg/t3code/commit/65404eb75de9233c9d1068ddfd09bacfc3b0465b) | fix(mobile): pause hidden home thread list updates (#15705) | Not applicable | N01 |
| [e803242d936a](https://github.com/pingdotgg/t3code/commit/e803242d936af3e052757d59ab32bc7c4fcc2390) | fix(mobile): restore file viewer insets and glass header (#17073) | Not applicable | N01 |
| [ed27eb724b4d](https://github.com/pingdotgg/t3code/commit/ed27eb724b4d7963b59de8fd2e51d6b877004163) | fix(web): PR code toolbar no longer overlaps in narrow panels (#15561) | Not applicable | N02 |
| [98b5e1bef3c8](https://github.com/pingdotgg/t3code/commit/98b5e1bef3c8d933d7ad826f837086a506a3732f) | fix(web): PR commit menu no longer stretches across the window (#15560) | Not applicable | N02 |
| [647d8affa78c](https://github.com/pingdotgg/t3code/commit/647d8affa78c848c2a55c3e00ed14af9a9eca58e) | fix(web): command palette scrollbar no longer clipped at the top (#17035) | Not applicable | N02 |
| [85a41391e962](https://github.com/pingdotgg/t3code/commit/85a41391e9625d760f5bb2294834c38f42bbf5b8) | fix(web): Usage breadcrumb stays centered on small viewports (#15552) | Not applicable | N02 |
| [70eeb3996d55](https://github.com/pingdotgg/t3code/commit/70eeb3996d557a2e553f817c8e123bf79bbd378c) | fix(mobile): stop refreshing Git status on streamed thread updates (#15893) | Not applicable | N01 |
| [12a4c057f3d6](https://github.com/pingdotgg/t3code/commit/12a4c057f3d6880c611630ccdc0c8c7f2cd23f72) | fix(mobile): skip move indexes for empty and single-thread sections (#16115) | Not applicable | N01 |
| [8a1fc4149ae7](https://github.com/pingdotgg/t3code/commit/8a1fc4149ae7b41e7d3d93f1b51dd9aeb38c4282) | perf(mobile): reuse encoded rows in shell cache saves (#16129) | Not applicable | N01 |
| [6651aaab38c6](https://github.com/pingdotgg/t3code/commit/6651aaab38c6814ed7bba3e805a1ec7a01870b55) | perf(mobile): skip showcase subscriptions in normal builds (#16131) | Not applicable | N01 |
| [6eed953a0602](https://github.com/pingdotgg/t3code/commit/6eed953a060213630af0d7e3cd9a664bd85f7ced) | perf(mobile): remove unused Home project sorting (#16177) | Not applicable | N01 |
| [7873504964c3](https://github.com/pingdotgg/t3code/commit/7873504964c3008ed2a87ada7be62d611e7a281a) | perf(mobile): reduce move-menu index allocations (#16256) | Not applicable | N01 |
| [6c1ce0699697](https://github.com/pingdotgg/t3code/commit/6c1ce06996979fcfe120ec8c8cb4f78a5c8a7365) | perf(mobile): skip impossible thread-key lookups (#16263) | Not applicable | N01 |
| [c001e191a19d](https://github.com/pingdotgg/t3code/commit/c001e191a19db32199ad75ccea219f729f9db7c4) | fix(mobile): collect UI runtime garbage on iOS memory warnings (#16296) | Not applicable | N01 |
| [5c2798d47eb4](https://github.com/pingdotgg/t3code/commit/5c2798d47eb41d5d659b8d2189781fcb06536a5f) | fix(mobile): stop Git sheet refresh loop (#16305) | Not applicable | N01 |
| [4fec120b75bb](https://github.com/pingdotgg/t3code/commit/4fec120b75bb397bda3c09b7317f79c1d26811c0) | fix(mobile): refresh Git status after reconnect (#16329) | Not applicable | N01 |
| [833a03afe32b](https://github.com/pingdotgg/t3code/commit/833a03afe32b2f332ab19ce72c0e0dc3cf72c341) | fix(mobile): update the iOS Git header menu when status changes (#16330) | Not applicable | N01 |
| [b402f5fc65f6](https://github.com/pingdotgg/t3code/commit/b402f5fc65f6f4554266dea385a7da7ff353f028) | perf(mobile): reuse the settled sort when settled rows are unchanged (#16369) | Not applicable | N01 |
| [33afd5a5aa36](https://github.com/pingdotgg/t3code/commit/33afd5a5aa361890c9ce22479e8f52f6b18248aa) | perf(mobile): highlight source files in small batches that keep grammar state (#16729) | Not applicable | N01 |
| [12865b4a73c2](https://github.com/pingdotgg/t3code/commit/12865b4a73c23bb402717d5f592dcd954ed196e5) | feat(web): quote chips show what you said about the quote (#15703) | Not applicable | N02 |
| [ff9eb8bd0c8e](https://github.com/pingdotgg/t3code/commit/ff9eb8bd0c8e71f98969a58d5789889e07b57c8b) | feat(web): projectless threads show their machine in the sidebar (#17022) | Not applicable | N02 |
| [57f96e69d396](https://github.com/pingdotgg/t3code/commit/57f96e69d39669f47e7310062222cd67adbb4912) | fix(lineage): keep agent effort and speed after completion (#16925) | Not applicable | N51 |
| [97a65a3121c1](https://github.com/pingdotgg/t3code/commit/97a65a3121c11801a877897928efc51ef40fa49f) | fix(mobile): align diff scrolling with glass headers (#17085) | Not applicable | N01 |
| [73e097b8c860](https://github.com/pingdotgg/t3code/commit/73e097b8c86000ddc4b38926ba86e1595109d78e) | fix(web): workspace page headers can no longer grow past the top-bar height (#17086) | Not applicable | N02 |
| [30cc78897550](https://github.com/pingdotgg/t3code/commit/30cc788975500a8c00d32a50f348174d1ce578d1) | feat(mobile): redesign the Add environment sheet (#17092) | Not applicable | N01 |
| [d6f47ff9725f](https://github.com/pingdotgg/t3code/commit/d6f47ff9725f0902b526f854519cb7e92dcee52f) | feat(providers): run Muse Code as a native provider (#17082) | Not applicable | N39 |
| [0647c481017f](https://github.com/pingdotgg/t3code/commit/0647c481017f25ddff2ec1dff68ef3944489b6db) | feat(web): compact-before-send is a chip that shows the token count (#17127) | Not applicable | N57 |
| [2ea7684d563f](https://github.com/pingdotgg/t3code/commit/2ea7684d563fe0ebbe9144dbfd6b0eade7eaf49a) | fix(clients): copy button is back in its old spot, before fork (#17137) | Not applicable | N18 |
| [4d1f24ba4123](https://github.com/pingdotgg/t3code/commit/4d1f24ba4123279708e169fcb196f78da8f11351) | fix(usage): repeat usage scans no longer decode unchanged Antigravity databases (#17139) | Not applicable | N06 |
| [1f31906c8e30](https://github.com/pingdotgg/t3code/commit/1f31906c8e3003b2a9d7118c4da163544fa86586) | fix(server): shell snapshots no longer block other database reads while decoding (#17141) | Not applicable | N06 |
| [10e29a26c289](https://github.com/pingdotgg/t3code/commit/10e29a26c289140b8f2312045c1ce0c278c7fe1a) | perf(server): ACP tool updates no longer persist a snapshot per streamed chunk (#16682) | Ported | ACP tool output is journaled linearly. |
| [13c5328dd02a](https://github.com/pingdotgg/t3code/commit/13c5328dd02ab0f5863f7d46455efc8dc243d4ce) | feat(web): filter the Usage page to just the providers you want (#16970) | Not applicable | N02 |
| [9381533771ef](https://github.com/pingdotgg/t3code/commit/9381533771ef50d58304f79fac2b74ebd8932be7) | fix(usage): Cursor account history loads about 4x faster (#17140) | Not applicable | N06 |
| [0ce6b9deb0c3](https://github.com/pingdotgg/t3code/commit/0ce6b9deb0c3f6b4dd9df7ed809f7acb9d237ae0) | fix(server): threads settle as soon as branch status sees their PR merge (#17148) | Not applicable | N11 |
| [859755855791](https://github.com/pingdotgg/t3code/commit/85975585579188be20f46d96b4972ce46c33f4bc) | perf(usage): the Usage page shows numbers in under a second and dims only what is still loading (#17147) | Not applicable | N03 |
| [37eaf5d293ef](https://github.com/pingdotgg/t3code/commit/37eaf5d293ef83ce86d2ce9a1583792696c82bbc) | fix(server): an agent can settle its own thread when its turn ends (#17145) | Not applicable | N06 |
| [5e2225671f70](https://github.com/pingdotgg/t3code/commit/5e2225671f705fcd33f1ea5591b79ba612fb6974) | feat(web): Add provider button sits with the provider list (#17152) | Not applicable | N16 |
| [a4c9494b0e36](https://github.com/pingdotgg/t3code/commit/a4c9494b0e3606775cc5fc929fc138399288bd43) | fix(web): link pull requests to threads in folders that aren't Git repos (#15946) | Not applicable | N02 |
| [12069eefd707](https://github.com/pingdotgg/t3code/commit/12069eefd707f78eafc27812027c994eea0613cf) | feat(web): find messages and plans in the current thread (#10439) | Not applicable | N16 |
| [9a3070bcf023](https://github.com/pingdotgg/t3code/commit/9a3070bcf023c3c025633659addeef4cf628133a) | feat(web): improve terminal scrollback navigation and snapshots (#17091) | Not applicable | N02 |
| [a65b43892f14](https://github.com/pingdotgg/t3code/commit/a65b43892f14bd6c1280296a37a11bd5234be00d) | docs(internals): add a checklist for adding a provider (#17229) | Not applicable | N15 |
| [a6ec88f7a716](https://github.com/pingdotgg/t3code/commit/a6ec88f7a716fc421bd22c2484881c44110f9375) | fix(mobile): keep native screens ordered during stack pops (#17231) | Not applicable | N01 |
| [4daec109cdf8](https://github.com/pingdotgg/t3code/commit/4daec109cdf8e3e92b392017d7564e735e54f6fd) | fix(server): pairing tokens work on Node versions that cannot bind booleans (#16730) | Not applicable | N06 |
| [61b9790816a8](https://github.com/pingdotgg/t3code/commit/61b9790816a8af359ab7e8ba0b52a3481353b18d) | fix(mobile): HTML pages in a thread no longer trap scrolling on Android (#17211) | Not applicable | N01 |
| [07a8b9ece40b](https://github.com/pingdotgg/t3code/commit/07a8b9ece40bc20a5f84ce78d12f621518065863) | fix(web): centered scrollers no longer shift when the scrollbar appears (#17077) | Not applicable | N16 |
| [62a5d3720f82](https://github.com/pingdotgg/t3code/commit/62a5d3720f82f2dc3ace48fcb26c90c6e3976b50) | fix(web): distinguish thread search matches from code tints (#17263) | Not applicable | N28 |
| [f999eb2a8e7d](https://github.com/pingdotgg/t3code/commit/f999eb2a8e7d8db74d4f4ab7dd66f86163e1a226) | fix(server): Pi extension wakes get an owned continuation turn (#17214) | Not applicable | N06 |
| [de72286ee577](https://github.com/pingdotgg/t3code/commit/de72286ee5777319fde054e7a82023916ce6f481) | fix(server): Pi discovers optional T3 tools on demand (#17220) | Not applicable | N06 |
| [eb1cf7bd0efb](https://github.com/pingdotgg/t3code/commit/eb1cf7bd0efba7364d53907036f5b7eefb379c39) | fix(web): stack merge dialog closes as soon as you confirm (#17116) | Not applicable | N02 |
| [e8f5850b8245](https://github.com/pingdotgg/t3code/commit/e8f5850b824572bab76bf810696f875eb648eff9) | fix(server): Pi editor dialogs prefill the answer composer (#17206) | Not applicable | N39 |
| [a0066d5cb7d1](https://github.com/pingdotgg/t3code/commit/a0066d5cb7d18abd786c1c23bf835f898490fd37) | fix(desktop): generate valid User-Agent that follows RFC 9110 guidelines (#17264) | Not applicable | N09 |
| [0b6ec43c5416](https://github.com/pingdotgg/t3code/commit/0b6ec43c54165a87941b7a070849be8979646040) | fix(server): Pi discovers workspace skills and commands (#17190) | Not applicable | N13 |
| [805967a878e6](https://github.com/pingdotgg/t3code/commit/805967a878e6804d58c151f29d2a3d0a06828fd1) | fix(mobile): preserve navigation after native swipe back (#17268) | Not applicable | N01 |
| [fd25c42ae6e5](https://github.com/pingdotgg/t3code/commit/fd25c42ae6e56a82be7219950a746572b1c4d0e9) | fix(server): keep newly discovered models out of legacy groups (#14314) | Not applicable | N64 |
| [df616cc5491d](https://github.com/pingdotgg/t3code/commit/df616cc5491dcc7dd80d3ade8decd88f7a652220) | feat(editors): open remote projects in JetBrains IDEs over SSH (#17271) | Not applicable | N03 |
| [740f591221bd](https://github.com/pingdotgg/t3code/commit/740f591221bd9651f172237d68b7daba9d050dbd) | test(desktop): expect JetBrains IDEs among remote editors (#17291) | Not applicable | N09 |
| [cdd331b6c3f7](https://github.com/pingdotgg/t3code/commit/cdd331b6c3f78bf24b039b09937b837d2d71a4e7) | fix(server): recognize authenticated GitHub Enterprise hosts (#11059) | Not applicable | N11 |
| [2a93885bac57](https://github.com/pingdotgg/t3code/commit/2a93885bac5798a79d55069a0b5dc3e53c6176bc) | fix(connect): relay client updates itself and skips incompatible cloudflared (#17275) | Not applicable | N06 |
| [bd565bd1794f](https://github.com/pingdotgg/t3code/commit/bd565bd1794f49f176ad137555faa46224716138) | fix(shared): relay client install waits out a brief Windows file lock (#16998) | Not applicable | N27 |
| [c377d1ca17db](https://github.com/pingdotgg/t3code/commit/c377d1ca17dbce42008fbb05fccd95468b13d329) | fix(shared): release relay install locks on cancellation (#10585) | Not applicable | N27 |
| [af3d842a001d](https://github.com/pingdotgg/t3code/commit/af3d842a001d030ed528396bf59cf02bbf373cd0) | chore(shared): bump managed cloudflared to 2026.10.0 (#11184) | Not applicable | N02 |
| [580948708b66](https://github.com/pingdotgg/t3code/commit/580948708b66f2a971199670f76cb6b43119e9a4) | fix(shared): bound cloudflared download with 10-minute timeout (#14139) | Not applicable | N27 |
| [9eea28bacd27](https://github.com/pingdotgg/t3code/commit/9eea28bacd27b2ee6d5183e195c397bba8b515a0) | refactor(provider-core): add provider-core and provider-testing packages (#17299) | Not applicable | N38 |
| [2c1160783ea4](https://github.com/pingdotgg/t3code/commit/2c1160783ea4b4dcc1c1f1ab04639ffc0aaa289d) | refactor(settings): drop the legacy per-driver providers map (#17300) | Not applicable | N38 |
| [9b0df1358f9f](https://github.com/pingdotgg/t3code/commit/9b0df1358f9f9c3277bcb33777c2435b88ee75c4) | refactor(provider-pi): move Pi into its own provider package (#17302) | Not applicable | N38 |
| [7b57b723d3b7](https://github.com/pingdotgg/t3code/commit/7b57b723d3b7490b50fffbd99fe89c6aa98fae1f) | feat(models): tell users when a CLI update unlocks a new model (#17307) | Not applicable | N05 |
| [48b71f0eb323](https://github.com/pingdotgg/t3code/commit/48b71f0eb323d08c8fcadfe54adf34becb2984da) | fix(web): collapsed composer reserves room for wide send actions (#17016) | Not applicable | N22 |
| [d81afa0a6f40](https://github.com/pingdotgg/t3code/commit/d81afa0a6f403bd6e5562ffcee2484089a1fdc3f) | fix(muse): workflow subagents no longer stall on hidden approvals (#17329) | Not applicable | N13 |
| [e38b6a188faf](https://github.com/pingdotgg/t3code/commit/e38b6a188fafd499df347331f8c8279ff320c207) | refactor(provider-core): share attachment prompts, notifications, and event loggers (#17330) | Not applicable | N62 |
| [741377ce5fb1](https://github.com/pingdotgg/t3code/commit/741377ce5fb1578916659cbb93fd403b26dbeb62) | fix(web): file previews handle downloads, in-page links, and repo paths, and favicons stop leaking internal hosts (#16950) | Not applicable | N23 |
| [5754121c23f7](https://github.com/pingdotgg/t3code/commit/5754121c23f73201d47a96feb97eca7b96334a2a) | fix(server): environment-hosted browser tabs behave like a normal browser (#16963) | Not applicable | N03 |
| [d9f7c0caed5c](https://github.com/pingdotgg/t3code/commit/d9f7c0caed5cc0fe84cec97f2b6f002084542a4e) | fix(desktop): browser tab fixes for fullscreen, shortcuts, links, reload and hidden tabs (#16961) | Not applicable | N03 |
| [66298aa926e6](https://github.com/pingdotgg/t3code/commit/66298aa926e6a4c95ebecbc576a68634315899c0) | fix(web): desktop opens remote environments' browser tabs locally (#17316) | Not applicable | N02 |
| [3bfbc37dbf37](https://github.com/pingdotgg/t3code/commit/3bfbc37dbf37095816eda500f3f1e05cb98bb95b) | fix(desktop): the t3 command warns instead of installing behind another t3 (#17351) | Not applicable | N03 |
| [e0c80c198cfb](https://github.com/pingdotgg/t3code/commit/e0c80c198cfbb2afe04528cd6a7ec1d2f21606dd) | fix(web): images, video, HTML and PDF preview in a thread before its first message (#17352) | Not applicable | N02 |
| [5b78b34bba86](https://github.com/pingdotgg/t3code/commit/5b78b34bba8645a268900f946858ea1470621c41) | refactor(provider-muse): move Muse Code into its own provider package (#17331) | Not applicable | N38 |
| [6308db4ff500](https://github.com/pingdotgg/t3code/commit/6308db4ff500b89821e22605f964b1701cead8ee) | fix(web): semantic branch naming hint lines up with its setting (#16972) | Not applicable | N02 |
| [393ff4c96807](https://github.com/pingdotgg/t3code/commit/393ff4c968073d8419e5bade06000998602af6ad) | fix(mobile): restore chat image previews in the v5 stack (#17361) | Not applicable | N01 |
| [c724c7cb02aa](https://github.com/pingdotgg/t3code/commit/c724c7cb02aaeb0a23ff6e5e224d17c0fdb9a36b) | feat(mobile): fade working threads and match web's status labels (#17368) | Not applicable | N01 |
| [dbd83435347d](https://github.com/pingdotgg/t3code/commit/dbd83435347d13d5f456271f4f1f5efe294a111d) | fix(server): agent browser tools stop bloating history, fall back sensibly, and respect ownership (#16956) | Not applicable | N20 |
| [29980a314092](https://github.com/pingdotgg/t3code/commit/29980a31409234b676f97bd477c4e46fdb61a929) | fix(web): add room for thread timeline markers (#17372) | Not applicable | N28 |
| [b0ee7dd48f86](https://github.com/pingdotgg/t3code/commit/b0ee7dd48f868c5c02884345f723bebaf616b13d) | fix(web): drop sidebar context before cancelling pointer drag (#17373) | Not applicable | N02 |
| [2c7a446e657e](https://github.com/pingdotgg/t3code/commit/2c7a446e657e705c257db28540819f4c0115b2fc) | refactor(providers): namespace-import service modules in core, Muse, Pi, and testing (#17375) | Not applicable | N65 |
| [456930a09db2](https://github.com/pingdotgg/t3code/commit/456930a09db29efdd3d4a397fe214e9ede2d6975) | fix(auth): show connection permissions and enforce session lifetime (#17370) | Already covered | Request controls are gated by active/disabled/staleRequestIds. |
| [e62161bc5b53](https://github.com/pingdotgg/t3code/commit/e62161bc5b53389a462d066836d47d25df0f4134) | refactor(provider-opencode): move OpenCode into its own provider package (#17345) | Not applicable | N38 |
| [b5e4f1e0b0c2](https://github.com/pingdotgg/t3code/commit/b5e4f1e0b0c2c06f183ab6a89072bec21c8334c4) | refactor(provider-cursor): move Cursor into its own provider package (#17349) | Not applicable | N62 |
| [664bd5e524bb](https://github.com/pingdotgg/t3code/commit/664bd5e524bb58b23ecab0f2e413fc2ecb4caf55) | refactor(provider-acp): move the shared ACP adapter into its own package (#17354) | Not applicable | N62 |
| [b707eeb05278](https://github.com/pingdotgg/t3code/commit/b707eeb052782cfd8b1ff0445f84b2aaf01da38b) | refactor(provider-grok): move Grok into its own provider package (#17357) | Ported | Grok auto mode, removal of accept-edits and ACP clientType/ctrl_c metadata; orchestration rewrite excluded. |
| [42c6623d4928](https://github.com/pingdotgg/t3code/commit/42c6623d492850c747b7ba06c1d0f1e1559d0ad5) | fix(server): speed up long thread message sync (#17387) | Not applicable | N06 |
| [c9fa1367caff](https://github.com/pingdotgg/t3code/commit/c9fa1367caff4cd6595386814e1f8bb950d3d3ef) | fix(desktop): cancel backend pipe reads to avoid slow shutdown (#17386) | Not applicable | N06 |
| [b000eac17131](https://github.com/pingdotgg/t3code/commit/b000eac171318bf0130865387ae33523eecd3019) | refactor(providers): adapter factories yield their services (#17381) | Not applicable | N62 |
| [5785df19ea74](https://github.com/pingdotgg/t3code/commit/5785df19ea748be0755244582cb70d785f95f830) | fix(web): show a row spinner instead of a banner when expanding a folder (#17378) | Not applicable | N02 |
| [8f760371978f](https://github.com/pingdotgg/t3code/commit/8f760371978f05b49e6f6b6aa4b2c523ab14b4fe) | fix(server): a timed-out browser drag no longer exits the server (#17360) | Not applicable | N06 |
| [563645cf6e10](https://github.com/pingdotgg/t3code/commit/563645cf6e104ad930f4c9b209678a77eef07e65) | fix(server): a logged-out Claude CLI no longer reports as authenticated (#15459) | Already covered | Claude login is inspected with claude auth status --json. |
| [3b6af0bd1600](https://github.com/pingdotgg/t3code/commit/3b6af0bd1600f034b0466e1b8ff017fbabeba910) | fix(server): Pi loads every selected skill without losing prompt text (#17194) | Not applicable | N06 |
| [c908cea52510](https://github.com/pingdotgg/t3code/commit/c908cea52510ed29199c3cc477204da425601301) | fix(server): keep the Claude MCP token out of process arguments (#17408) | Already covered | The existing file/environment MCP registration design covered this earlier implementation; the later control-channel port supersedes it. |
| [7a2b1c72beed](https://github.com/pingdotgg/t3code/commit/7a2b1c72beed4f560aebb41aa7b26e0792eeb938) | fix(server): reconcile Pi native session rewinds (#13839) | Not applicable | N13 |
| [cb46c62504ac](https://github.com/pingdotgg/t3code/commit/cb46c62504aca99295a4a8879644625a20685b7f) | test(provider-pi): cover continuation offers through the driver (#17407) | Not applicable | N65 |
| [43f8a8de17a7](https://github.com/pingdotgg/t3code/commit/43f8a8de17a7ac1baa7a3cf36d681856de2d8add) | refactor(provider-acp-registry): move the ACP Registry into its own package (#17405) | Not applicable | N62 |
| [101f8b2f55df](https://github.com/pingdotgg/t3code/commit/101f8b2f55df50d4f8d70ed8e6b8e0edf71a2f31) | fix(server): relay client updates no longer drop the host off T3 Connect (#17366) | Not applicable | N06 |
| [ec80933ac8cd](https://github.com/pingdotgg/t3code/commit/ec80933ac8cd02fec5c97b342462ccc9567cdb1e) | fix(connect): Cloudflare and other VPN addresses no longer show as Tailscale (#17158) | Not applicable | N03 |
| [2cf0ff0643e2](https://github.com/pingdotgg/t3code/commit/2cf0ff0643e23d13658114fbf3a7e76753559755) | fix(web): Local environment switch stays reachable after turning it off (#17359) | Not applicable | N02 |
| [c5d4d2d04cce](https://github.com/pingdotgg/t3code/commit/c5d4d2d04cce7c7fa60ac1c51974c4328cbaa8e5) | fix(web): keep chat banners inside the lane beside the docked details card (#17094) | Not applicable | N28 |
| [a947f7c099dd](https://github.com/pingdotgg/t3code/commit/a947f7c099dd279108a4bdfc5e0cced6c5c30b71) | fix(web): settled and snoozed lines line up with the messages above them (#17191) | Not applicable | N02 |
| [22ccf8a3dfc9](https://github.com/pingdotgg/t3code/commit/22ccf8a3dfc9094b4ccdb689e128c67bacbbcdf5) | fix(web): distinguish project filter from new project (#12113) | Not applicable | N02 |
| [b1508672a63c](https://github.com/pingdotgg/t3code/commit/b1508672a63c4799fed6d9091001de00175d9316) | feat(web): assign a thread details panel shortcut (#16694) | Not applicable | N03 |
| [a6698271a41a](https://github.com/pingdotgg/t3code/commit/a6698271a41a4c1f2c42f25ca5d69cd7d1fa9c30) | fix(web): chat content keeps pace with sidebar resizing (#17383) | Not applicable | N02 |
| [2c1915b3193f](https://github.com/pingdotgg/t3code/commit/2c1915b3193f16e2493709b480e6622645d0ca4e) | refactor(provider-core): expose model metadata through a ModelCatalog port (#17417) | Not applicable | N62 |
| [b66dd402e868](https://github.com/pingdotgg/t3code/commit/b66dd402e86899c3fb4341d46c32d0d69df1eefc) | refactor(provider-core): follow the Effect service conventions throughout (#17427) | Not applicable | N62 |
| [68ab16f30842](https://github.com/pingdotgg/t3code/commit/68ab16f3084236871685bed5032cb05c7c436dbd) | refactor(provider-core): latest-version lookups go through a ProviderLatestVersions service (#17434) | Not applicable | N62 |
| [784b5625f9e2](https://github.com/pingdotgg/t3code/commit/784b5625f9e228cca8dd67e6e24e402325672544) | refactor(provider-core): MCP provider sessions live in a McpProviderSessions service (#17446) | Not applicable | N62 |
| [31b04e2ee929](https://github.com/pingdotgg/t3code/commit/31b04e2ee929660d80d5931ed53fd17bf1d33ce6) | refactor(provider): bring opencode, muse, pi, core and testing in line with Effect conventions (#17542) | Not applicable | N62 |
| [f853d514cab4](https://github.com/pingdotgg/t3code/commit/f853d514cab475e73468f7cb7bab1aae48dbe549) | refactor(provider-acp): ACP, ACP Registry and Grok follow the Effect service conventions (#17544) | Not applicable | N62 |
| [33806e735551](https://github.com/pingdotgg/t3code/commit/33806e73555107b8ac8fb18fd15b9dfb87ec6557) | refactor(provider-cursor): follow the Effect service conventions (#17545) | Not applicable | N62 |
| [64972461c060](https://github.com/pingdotgg/t3code/commit/64972461c0606a4eafdd3cd08056155b6a2fea95) | fix(marketing): use app wordmark in header (#13240) | Not applicable | N19 |
| [aa8c6662e212](https://github.com/pingdotgg/t3code/commit/aa8c6662e212a4307bd4e585661da99a358e46d4) | fix(web): pr merge actions stay visible while the stack refreshes (#17559) | Not applicable | N02 |
| [91a6646efd98](https://github.com/pingdotgg/t3code/commit/91a6646efd98622dc9cef7061387e20bc389463f) | chore(deps): upgrade Effect to 4.0.2 (#17571) | Not applicable | N06 |
| [ecfb7342edde](https://github.com/pingdotgg/t3code/commit/ecfb7342edde9160b7df4231d4f145c7e4a30530) | fix(devices): recover stalled video without losing simulator input (#17566) | Not applicable | N03 |
| [7856908b8492](https://github.com/pingdotgg/t3code/commit/7856908b8492b7a5c381aa2d5a5c4d4292eb976b) | fix(web): keep checkout stable while pr actions load (#16625) | Not applicable | N02 |
| [1fd558d9e254](https://github.com/pingdotgg/t3code/commit/1fd558d9e25493e2a49da625aa35dc80a52b0257) | fix(mobile): show waiting thread status (#16693) | Not applicable | N01 |
| [5722496b7907](https://github.com/pingdotgg/t3code/commit/5722496b79079ac317a984f3baca8c88f41af070) | feat(web): add parent thread breadcrumb navigation (#16666) | Not applicable | N02 |
| [0a4d78912810](https://github.com/pingdotgg/t3code/commit/0a4d789128100ed7ccd6d42f454946aea8edfca8) | fix(server): restart inactivity after snoozed threads wake (#16674) | Not applicable | N06 |
| [454b94a13aea](https://github.com/pingdotgg/t3code/commit/454b94a13aea918f27bb060d8d054d4cfb791bc2) | feat(desktop): passkeys in the in-app browser on macOS (#16952) | Not applicable | N09 |
| [a1db449fe4fe](https://github.com/pingdotgg/t3code/commit/a1db449fe4fecafcb473340771163e791b7fb3d6) | fix(client): load earlier turns works for MCP threads over T3 Connect (#17599) | Not applicable | N07 |
| [0e7abea7c980](https://github.com/pingdotgg/t3code/commit/0e7abea7c98060db537fe81f31ce5ba7c6d376d0) | refactor(client): sign relay request URLs built from the HttpApi contract (#17602) | Not applicable | N07 |
| [28f11ed7a5a2](https://github.com/pingdotgg/t3code/commit/28f11ed7a5a2251ebe4689643b64425760ee5fe4) | refactor(source-control): add @t3tools/source-control-core (#17573) | Not applicable | N11 |
| [f6e45028efb6](https://github.com/pingdotgg/t3code/commit/f6e45028efb69d208a45aee8c2fa514adde44977) | refactor(source-control): Forgejo lives in @t3tools/source-control-forgejo (#17581) | Not applicable | N11 |
| [d01febb509a3](https://github.com/pingdotgg/t3code/commit/d01febb509a376d93af2d298a7787449792b5fcb) | refactor(source-control): Azure DevOps lives in @t3tools/source-control-azure-devops (#17592) | Not applicable | N11 |
| [eb459b5afaf4](https://github.com/pingdotgg/t3code/commit/eb459b5afaf42621a6cb2e698715fa017e26e5d7) | refactor(source-control): GitLab lives in @t3tools/source-control-gitlab (#17594) | Not applicable | N11 |
| [8d1858d9d608](https://github.com/pingdotgg/t3code/commit/8d1858d9d608d78b67268cdc52bceb4302ae8870) | refactor(source-control): Bitbucket lives in @t3tools/source-control-bitbucket (#17597) | Not applicable | N11 |
| [507361bc7301](https://github.com/pingdotgg/t3code/commit/507361bc7301982279b9f115bbae8f22268ce9d6) | refactor(source-control): GitHub lives in @t3tools/source-control-github (#17607) | Not applicable | N11 |
| [e5dca2332fed](https://github.com/pingdotgg/t3code/commit/e5dca2332fed450c6f1b3c7472666fbe80531c5a) | refactor(usage): transcript readers come from their drivers (#17576) | Not applicable | N66 |
| [23957c704b43](https://github.com/pingdotgg/t3code/commit/23957c704b43c25a253baf2644829e88cf45917f) | refactor(usage): OpenCode usage comes from provider-opencode (#17577) | Not applicable | N66 |
| [5a9c664b3ec8](https://github.com/pingdotgg/t3code/commit/5a9c664b3ec8214cc5847d561efa130ba213eb38) | refactor(usage): Cursor account usage comes from provider-cursor (#17578) | Not applicable | N66 |
| [9a0766b0238f](https://github.com/pingdotgg/t3code/commit/9a0766b0238f2c044a965dcba32c6cb7ea0b63cd) | refactor(usage): Antigravity usage is a reader on its driver (#17579) | Not applicable | N66 |
| [b1ec4b36876e](https://github.com/pingdotgg/t3code/commit/b1ec4b36876e7f3ad03a1e9e921286ec7d58fe95) | refactor(usage): usage readers use Effect FileSystem and SqlClient (#17615) | Not applicable | N66 |
| [6586a01f41bd](https://github.com/pingdotgg/t3code/commit/6586a01f41bd692f02d565d38f97b52a8067317e) | fix(web): composer context strip pads both edges evenly (#17562) | Not applicable | N22 |
| [a0a0e94822a0](https://github.com/pingdotgg/t3code/commit/a0a0e94822a0c675fbe0792ef94ff218101aeb66) | test(usage): v4 cache upgrade test waits for the migrated cache write (#17553) | Not applicable | N06 |
| [c85dc642681a](https://github.com/pingdotgg/t3code/commit/c85dc642681a944e644cbbbea7257d274822c21a) | feat(mobile): support Duo in the shared iOS app (#12648) | Not applicable | N01 |
| [49f849ecce36](https://github.com/pingdotgg/t3code/commit/49f849ecce3667abab7ce9e6aaa42dfd2e3c45d7) | refactor(source-control): GitManager reads provider resolvers, not host kinds (#17617) | Not applicable | N11 |
| [2ce3e4ae74de](https://github.com/pingdotgg/t3code/commit/2ce3e4ae74deb63bda7aaf059949a12aaaefed56) | refactor(source-control): PullRequestService reads GitHub resolvers, not its kind (#17619) | Not applicable | N11 |
| [51347aca3aa7](https://github.com/pingdotgg/t3code/commit/51347aca3aa73112bc1e288ae45507ce515f42a2) | refactor(source-control): Forgejo identity and Azure DevOps addressing move into their packages (#17624) | Not applicable | N11 |
| [58916b6554f4](https://github.com/pingdotgg/t3code/commit/58916b6554f45ac518bd8150dd10f1f536e40aac) | refactor: home directory comes from a HostProcessHomeDirectory reference (#17628) | Not applicable | N62 |
| [e30852213ed3](https://github.com/pingdotgg/t3code/commit/e30852213ed3954037091582057f71918918e19f) | refactor(shared): host process references live in a HostProcess module (#17641) | Not applicable | N62 |
| [6cd02e98f7ed](https://github.com/pingdotgg/t3code/commit/6cd02e98f7edd78b7bde9ceff26c829476dbeb77) | feat(web): filter PR comments by bots and resolved threads (#17645) | Not applicable | N02 |
| [0c4012055dea](https://github.com/pingdotgg/t3code/commit/0c4012055deacd43876b80902caad6dd7d6a2eef) | fix(clients): remove redundant prefix from PR watch status (#17635) | Not applicable | N07 |
| [99d2651dd5f8](https://github.com/pingdotgg/t3code/commit/99d2651dd5f82eeed800672e371427d7266dfe72) | fix(web): pending requests wait until you stop typing (#17637) | Not applicable | N22 |
| [08e18b7833c9](https://github.com/pingdotgg/t3code/commit/08e18b7833c9b5d33bfcd1ce8976dcb67c11dde5) | fix(models): remove new badges from Claude Opus and Sonnet 5.5 (#17646) | Ported | Reviewed Claude model-manifest data is retained by the local extraction. |
| [5dcb59ed6c19](https://github.com/pingdotgg/t3code/commit/5dcb59ed6c191589e730fb964aabad005466eb6d) | fix(ui): keep focus and selection borders visible across the app (#16675) | Not applicable | N28 |
| [5fe9d024d9b1](https://github.com/pingdotgg/t3code/commit/5fe9d024d9b1c28ec93d495cf1441c57e47ae7ce) | fix(mobile): prevent row presses during native back swipes (#17648) | Not applicable | N01 |
| [140ee145ac33](https://github.com/pingdotgg/t3code/commit/140ee145ac33653ac2c36f7940e4f54f6e92497f) | fix(server): Codex shadow homes replace stray sqlite maintenance locks (#17663) | Not applicable | N67 |
| [38571328a4bc](https://github.com/pingdotgg/t3code/commit/38571328a4bc3c30c56fc0cae299437fe9b53898) | feat(desktop): T3 Code can be your default web browser on macOS (#17587) | Not applicable | N03 |
| [202b95403a7d](https://github.com/pingdotgg/t3code/commit/202b95403a7d6b7b01390b6a90457fa76e1be439) | test(server): the ACP process-tree test no longer collides with the runner's own pid (#17647) | Not applicable | N48 |
| [cc62349c37fa](https://github.com/pingdotgg/t3code/commit/cc62349c37fa486000b89c7ecc975c1bf66ec8a4) | fix(web): keep branch restore action inline in narrow composers (#14811) | Not applicable | N02 |
| [3e94afa7c267](https://github.com/pingdotgg/t3code/commit/3e94afa7c26734f525a7d8a9cdb47896e8618cf8) | fix(web): composer banner actions stay inline whenever they fit (#17640) | Not applicable | N68 |
| [b744bded016b](https://github.com/pingdotgg/t3code/commit/b744bded016b15b9b5ade552a3e28270ac20176f) | feat(web): draft screen project picker is searchable (#17664) | Not applicable | N02 |
| [b87b13d41426](https://github.com/pingdotgg/t3code/commit/b87b13d41426418c4f42c6477ce62cfe2a43fbae) | fix(server): report incomplete transcript usage scans (#15661) | Not applicable | N06 |
| [d97060709ca7](https://github.com/pingdotgg/t3code/commit/d97060709ca7741c0c45d2ffad4e241b992259f2) | fix(web): every resize-driven layout commits in the same frame (#17656) | Not applicable | N69 |
| [ca40de081c3a](https://github.com/pingdotgg/t3code/commit/ca40de081c3ab087cd6458ce9bef12e43c5a7233) | fix(web): right panel and terminal drawer follow the pointer while dragging (#17657) | Not applicable | N02 |
| [42e1a0fba293](https://github.com/pingdotgg/t3code/commit/42e1a0fba2938bd56761db63f466962342d7b779) | fix(web): terminal drawer keeps its height after the window shrinks (#17658) | Not applicable | N02 |
| [033866c7aa55](https://github.com/pingdotgg/t3code/commit/033866c7aa55c53593ca31538f7d8ac99fdfe0fd) | perf(web): sidebar drags restyle only the sidebar (#17659) | Not applicable | N02 |
| [eddc8fb817db](https://github.com/pingdotgg/t3code/commit/eddc8fb817dbdb6c9ce34a30a2781c02ae8be9e0) | fix(web): server browser page resizes while the panel is dragged (#17660) | Not applicable | N02 |
| [0caa95d6eb47](https://github.com/pingdotgg/t3code/commit/0caa95d6eb471ab033ea670914d61070c65be806) | fix(storage): make worktree cleanup work and show why it skipped (#17563) | Not applicable | N03 |
| [e858ddb65668](https://github.com/pingdotgg/t3code/commit/e858ddb65668d54be5de3996f30c0bd5c3149657) | test(usage): usage service tests keep their state directory until cache writes land (#17636) | Not applicable | N06 |
| [f3a69c70c6a5](https://github.com/pingdotgg/t3code/commit/f3a69c70c6a5bc624e1b5eb80901d8b7c901ad6f) | fix(checkpoint): pulls and rebases no longer flood a turn's changed files (#17161) | Not applicable | N06 |
| [07b808001249](https://github.com/pingdotgg/t3code/commit/07b80800124982bce94879a97767a5365987e8f1) | fix(web): place notification icons after titles (#12209) | Not applicable | N02 |
| [0b9623fda108](https://github.com/pingdotgg/t3code/commit/0b9623fda1085416ef22e3a70cc5dcf1158f25bc) | fix(web): attachments on an open question are visible again (#15537) | Not applicable | N22 |
| [f34adad34f21](https://github.com/pingdotgg/t3code/commit/f34adad34f21510e867b47fe018c7975779a0ffc) | fix(web): scale Files tree with interface font size (#8011) | Not applicable | N02 |
| [318201caa520](https://github.com/pingdotgg/t3code/commit/318201caa520525517ef2bee8f4bbeedb2fec1d7) | fix(server): probe only owned preview listeners (#16687) | Not applicable | N06 |
| [ed4ea1083d47](https://github.com/pingdotgg/t3code/commit/ed4ea1083d477b67304f50d5691ba5fc2b9496cd) | fix(chat): surface pending subagent questions on parents (#16634) | Not applicable | N03 |
| [c1f21e53b3b1](https://github.com/pingdotgg/t3code/commit/c1f21e53b3b138cc591f9df90bd5c8aef6a00ce4) | fix(web): section header chevrons point up when collapsed (#14273) | Not applicable | N02 |
| [e08c31e1a1fd](https://github.com/pingdotgg/t3code/commit/e08c31e1a1fd5e89e2d4130b9ad18b44d15f92cd) | fix(web): timeline divider pill shows a pointer, visible hover and focus ring (#15188) | Not applicable | N02 |
| [6e4ca6ce9d04](https://github.com/pingdotgg/t3code/commit/6e4ca6ce9d04009af290412b77a0acb9be9f8a18) | fix(git): allow creating prs from dirty worktrees (#15625) | Not applicable | N11 |
| [b822420e5074](https://github.com/pingdotgg/t3code/commit/b822420e5074d750e7258ea48568666720cf0415) | docs(install): polish binary install destination phrasing (#15732) | Not applicable | N15 |
| [0981e6b9d421](https://github.com/pingdotgg/t3code/commit/0981e6b9d421ec46010d9b9edebd0beb241ef472) | perf(server): keep passive terminal output flowing (#17178) | Not applicable | N06 |
| [b6aa268b12e8](https://github.com/pingdotgg/t3code/commit/b6aa268b12e8eb6d5aab52e5e09c12e255fca905) | fix(web): stop button icon no longer shifts on hover (#16012) | Ported | Keep the Stop button stable on hover (#16012). |
| [9bbbbbab2a53](https://github.com/pingdotgg/t3code/commit/9bbbbbab2a53ccc05a6b3c1ad8cc64ff411d185f) | fix(web): add bottom padding to expanded tool panels (#16525) | Not applicable | N02 |
| [2def3de4b931](https://github.com/pingdotgg/t3code/commit/2def3de4b931206d3387d222454af0daf4a274e2) | fix(web): keep incremental highlighter return type portable (#17259) | Not applicable | N02 |
| [391f813357c7](https://github.com/pingdotgg/t3code/commit/391f813357c72d1623aba9fc5b199f237d6710c6) | fix(web): sidebar "Code" label no longer clips its letter tops (#16134) | Not applicable | N02 |
| [699da956899f](https://github.com/pingdotgg/t3code/commit/699da956899fb4db219b827855610095568e5a7c) | fix(tests): use POSIX paths for the simulated macOS device host (#17241) | Not applicable | N06 |
| [88edfacf5d5f](https://github.com/pingdotgg/t3code/commit/88edfacf5d5f2ad6deec7ee5ec1c19ee68f23a13) | fix(mobile): Android composer keeps the caret in view on AOSP-based keyboards (#17492) | Not applicable | N01 |
| [467fea963f36](https://github.com/pingdotgg/t3code/commit/467fea963f36086731b47cb43b6a7d8e5645bf4c) | test(web): allow cold timeline imports on CI (#16608) | Not applicable | N02 |
| [aa165e9b9ce0](https://github.com/pingdotgg/t3code/commit/aa165e9b9ce05514df708502481cc57fdfa00ff8) | fix(mobile): pinch zooms chat images on Android (#15047) | Not applicable | N01 |
| [d1e56e79ff6a](https://github.com/pingdotgg/t3code/commit/d1e56e79ff6a4886366292f1424a5e4560b5fab7) | fix(web): selected provider ring no longer clipped during panel resize (#17534) | Not applicable | N02 |
| [d50a2a6fc5ad](https://github.com/pingdotgg/t3code/commit/d50a2a6fc5ad387892bd20b0323bf07c405d9eea) | docs(usage): OpenCode Go limits need a Go API key (#15664) | Not applicable | N15 |
| [c116dacb8315](https://github.com/pingdotgg/t3code/commit/c116dacb8315e4a731b25c8b93c55ef9d01e9ed3) | fix(web): improve usage scanning indicator alignment (#15498) | Not applicable | N02 |
| [97009b049a13](https://github.com/pingdotgg/t3code/commit/97009b049a13fac5be5bee13a34b04cd6b87a7e7) | fix(server): print pairing credential expiry as ISO timestamp (#14128) | Not applicable | N06 |
| [2cc55f2f2226](https://github.com/pingdotgg/t3code/commit/2cc55f2f2226db7d349723293ca772eb83e7c516) | chore(ci): use GPT 6.1 Sol Max for check agents (#14312) | Not applicable | N15 |
| [8b81a8880f56](https://github.com/pingdotgg/t3code/commit/8b81a8880f56daf475136e9d89794bb2a33526d0) | fix(mobile): honor requested terminal native architectures (#10709) | Not applicable | N01 |
| [5c715c694118](https://github.com/pingdotgg/t3code/commit/5c715c6941184da09bf11f6651d303d284bd54da) | fix(server): keep preview browser connected after operation timeouts (#17693) | Not applicable | N06 |
| [aacefcd7e92a](https://github.com/pingdotgg/t3code/commit/aacefcd7e92a784c179b3ca2473a7608f783784e) | fix(web): timeline divider focus ring stays inside the pill (#17702) | Not applicable | N02 |
| [03bd839554c9](https://github.com/pingdotgg/t3code/commit/03bd839554c9da27aae37b027bcdb2fa9896af09) | perf(desktop): reuse the prepared shell environment in the local backend (#17384) | Not applicable | N03 |
| [f02002ab95c9](https://github.com/pingdotgg/t3code/commit/f02002ab95c94510ab23a592f19f669fc50da0e9) | fix(web): align settings page widths (#12158) | Not applicable | N16 |
| [a3d150514152](https://github.com/pingdotgg/t3code/commit/a3d150514152dc99333f48a687d4e89ab31a43e2) | test(server): resolve the temp dir before matching the symlinked entrypoint (#9400) | Not applicable | N06 |
| [6e83160cdf98](https://github.com/pingdotgg/t3code/commit/6e83160cdf98b120dd9ffddfbbcb3212fceab444) | fix(web): keep inline code pills intact when they wrap (#12038) | Not applicable | N28 |
| [2aa033b11dbb](https://github.com/pingdotgg/t3code/commit/2aa033b11dbb54c32e2dfc8c3e24f4aede45ded2) | Revert "chore(ci): use GPT 6.1 Sol Max for check agents" (#17698) | Not applicable | N15 |
| [f55c66b323d2](https://github.com/pingdotgg/t3code/commit/f55c66b323d23ee876bf7b5754a814dd83467789) | fix(web): show the correct new thread shortcut in command palette (#8513) | Not applicable | N02 |
| [c1ade7db946b](https://github.com/pingdotgg/t3code/commit/c1ade7db946b58fe5f1f8069c7b8f2b9c3df550e) | fix(desktop): declare macOS local network usage (#11922) | Not applicable | N15 |
| [968ac2e82b8f](https://github.com/pingdotgg/t3code/commit/968ac2e82b8fbc3ce6d87fbb4159a20a4971d0d3) | docs: add Scoop as Windows installation method (#10509) | Not applicable | N15 |
| [a0a601c262a8](https://github.com/pingdotgg/t3code/commit/a0a601c262a8a7b260d18fd806302183ea529036) | fix(server): agents run in their own systemd scopes so an OOM kill spares the server (#17662) | Not applicable | N70 |
| [4ea735614bb8](https://github.com/pingdotgg/t3code/commit/4ea735614bb8e80ef0f41bf01dff362ecc705d92) | fix(web): welcome wizard says where imported projects come from (#14584) | Not applicable | N02 |
| [d02e292610ea](https://github.com/pingdotgg/t3code/commit/d02e292610ea5eff171d388211ba3ede2f0467a0) | fix(web): cite works on responses that end before a tool call (#17713) | Not applicable | N02 |
| [fa9f936ef5cb](https://github.com/pingdotgg/t3code/commit/fa9f936ef5cbbe4bd4f9890b638766fc92f45075) | fix(desktop): sign Windows native addons (#8206) | Not applicable | N15 |
| [c572cb71d271](https://github.com/pingdotgg/t3code/commit/c572cb71d2711d8cdb4ff7e0ba88af5e9a23aa81) | fix(server): track resumed subagent follow-ups as separate tasks (#17696) | Not applicable | N03 |
| [c2a301b12af6](https://github.com/pingdotgg/t3code/commit/c2a301b12af6e18148eb1d46123c4187f4bdb78e) | fix(web): nested corners follow their container's radius (#17695) | Not applicable | N28 |
| [1f093d119b91](https://github.com/pingdotgg/t3code/commit/1f093d119b91263deb83f0d18e1504b791fdbbd4) | feat(web): pr panel actions confirm in place (#17710) | Not applicable | N02 |
| [4107930420cf](https://github.com/pingdotgg/t3code/commit/4107930420cffe14776ad80783f7060236b5868b) | feat(web): reorder right panel tabs by dragging (#17730) | Not applicable | N02 |
| [c899a0c06ab2](https://github.com/pingdotgg/t3code/commit/c899a0c06ab2b0ec8fe8d6a497bb345ac72a5e5e) | fix(azure-devops): list pull requests with token sign-in and check out into worktrees (#17725) | Not applicable | N11 |
| [0be20c86c296](https://github.com/pingdotgg/t3code/commit/0be20c86c2962b7b581dfc79df9ab34b4fbdab14) | fix(usage): keep one email in two workspaces as two accounts (#17711) | Not applicable | N21 |
| [c3243f05088c](https://github.com/pingdotgg/t3code/commit/c3243f05088c3c20e9439f285fc6d7b0bfe1747d) | feat(pull-requests): hosts can report edit and resolve permissions per item (#17667) | Not applicable | N11 |
| [9cc57680cb5f](https://github.com/pingdotgg/t3code/commit/9cc57680cb5f762f553acd1fa5f3efcf17d8fc67) | perf(server): run Git for Windows' real git.exe, not its launcher (#17707) | Not applicable | N06 |
| [a7223f396942](https://github.com/pingdotgg/t3code/commit/a7223f3969428170cc13bd7859aa59098b2f94b1) | fix(clients): restart continuations show as a T3 Code notice, not another agent's message (#17723) | Not applicable | N57 |
| [57b3780770a8](https://github.com/pingdotgg/t3code/commit/57b3780770a829ff81e62d8dc658683bf51d5155) | fix(mobile): browser picture in picture opens from the header button (#17731) | Not applicable | N01 |
| [65d33177b777](https://github.com/pingdotgg/t3code/commit/65d33177b777e014c56f01851bef446785b3aa28) | feat(source-control): GitCafe lives in @t3tools/source-control-gitcafe (#17681) | Not applicable | N16 |
| [500cb4266f2b](https://github.com/pingdotgg/t3code/commit/500cb4266f2bb290aba2c2038295500ff7b25188) | fix(web): add provider wizard no longer shifts sideways while it grows (#17292) | Not applicable | N02 |
| [42f71dfe0768](https://github.com/pingdotgg/t3code/commit/42f71dfe0768ee768283d50140e7d360a635ee28) | fix(web): PR search keeps the caret where you type (#17675) | Not applicable | N02 |
| [6d5ea190a4a5](https://github.com/pingdotgg/t3code/commit/6d5ea190a4a54b01dfe6500e032526839e3258d1) | fix(server): thread PR badges catch up when another environment reads the PR (#17729) | Not applicable | N11 |
| [bd2346eda2e2](https://github.com/pingdotgg/t3code/commit/bd2346eda2e2c380d1844869c7fd16c279d2190f) | fix(server): refuse editor paths with line breaks or quotes when the editor is a Windows command shim (#17749) | Not applicable | N06 |
| [c77a7b7eebd6](https://github.com/pingdotgg/t3code/commit/c77a7b7eebd6ea8a3a63bbcaaa7c1707fe65c86e) | fix(web): align compact button touch targets (#17748) | Not applicable | N28 |
| [6266af3f1f68](https://github.com/pingdotgg/t3code/commit/6266af3f1f688497ef9717b87b82eafc5240418c) | fix(server): t3_thread_launch refuses a new worktree whose base ref has no commit (#17791) | Not applicable | N11 |
| [d6f3af77192a](https://github.com/pingdotgg/t3code/commit/d6f3af77192adfb2304769096eeec59d3464aa21) | fix(web): omit underlines on markdown image links (#17728) | Ported | Exclude image links from text underlines (#17728). |
| [4606f9d6f213](https://github.com/pingdotgg/t3code/commit/4606f9d6f213952124d93792854b0c4e48658e60) | fix(server): restore auto resume for wrapped Claude gateway rate limits (#17778) | Not applicable | N71 |
| [e1ad595e0f10](https://github.com/pingdotgg/t3code/commit/e1ad595e0f10cc16d66fe54b4ee195435dd3cc0d) | fix(opencode): name OpenCode 2 sessions after their thread (#17414) | Not applicable | N72 |
| [472ea7ec5f67](https://github.com/pingdotgg/t3code/commit/472ea7ec5f67fae660dc606b82698d15a2bc850f) | fix(web): find update settings from the command palette (#17396) | Not applicable | N02 |
| [7b09ac7e0022](https://github.com/pingdotgg/t3code/commit/7b09ac7e0022769425471fad1c0d152c03d5954f) | fix(provider-opencode): tell OpenCode Zen and Go models apart (#17424) | Not applicable | N05 |
| [adbecdac7b6f](https://github.com/pingdotgg/t3code/commit/adbecdac7b6f1a167cc389b3d5fe5be8ec70afdf) | fix(server): a pull that fast-forwards no longer fails on large Git output (#17376) | Not applicable | N06 |
| [91474e56f668](https://github.com/pingdotgg/t3code/commit/91474e56f66888c5ab3856a782a72613e758c401) | fix(web): cancel question auto-advance after navigation (#17364) | Ported | Question-scoped auto-advance is cancelled on navigation or parking (#17364). |
| [33c1e36fd097](https://github.com/pingdotgg/t3code/commit/33c1e36fd097fe4f8e4498db653ee97b17184791) | fix(server): a bare repository name resolves to the signed-in account again (#17379) | Not applicable | N11 |
| [71f6f1db89a5](https://github.com/pingdotgg/t3code/commit/71f6f1db89a50f0108663ebf1724cf3a1a4134ed) | fix(web): a maximized right panel stays maximized when you return to its thread (#17327) | Not applicable | N02 |
| [02612c285640](https://github.com/pingdotgg/t3code/commit/02612c28564056ddb7ae6b06cedd31dafc235ffd) | fix(mobile): allow starting a task with only an image (#17409) | Not applicable | N01 |
| [a418075154d1](https://github.com/pingdotgg/t3code/commit/a418075154d1d4f4263d5fa55804b9288f61d3dd) | fix(server): PR watch no longer reports passed while a second run of a check is still going (#17344) | Not applicable | N11 |
| [a5b482452dc8](https://github.com/pingdotgg/t3code/commit/a5b482452dc8ca87625d5a80f83be0a9b56674de) | fix(server): say why a thread can't be settled (#17258) | Not applicable | N06 |
| [03b210528160](https://github.com/pingdotgg/t3code/commit/03b210528160e68c8dbfc27f11f693085f92ab09) | fix(server): prevent busy terminals from starving history persistence (#17181) | Not applicable | N27 |
| [de49d637d4b7](https://github.com/pingdotgg/t3code/commit/de49d637d4b72b3e3306533e3777cb37e6a3f738) | feat(server): use macOS .icns app icons as project icons (#17149) | Not applicable | N03 |
| [a1dee8b2b34a](https://github.com/pingdotgg/t3code/commit/a1dee8b2b34ad949935b703b85efb16b6fd1b987) | fix(mobile): usage reset icon lines up with its row (#17175) | Not applicable | N01 |
| [ee7924ff7fca](https://github.com/pingdotgg/t3code/commit/ee7924ff7fcabe24d5e8d940ac5e878d36477a0d) | fix(web): paths pasted after @ keep their underscores (#16619) | Not applicable | N22 |
| [dd4549eede92](https://github.com/pingdotgg/t3code/commit/dd4549eede928f9dc514778aeae15d74656a87f8) | fix(server): settle every OpenCode subagent call one report answers (#17134) | Not applicable | N72 |
| [8bf0a38873ae](https://github.com/pingdotgg/t3code/commit/8bf0a38873aed486a585b2cebd8dd577ae5656b3) | perf(web): switching project keeps Diagnostics and Providers mounted (#17122) | Not applicable | N02 |
| [f7664fdf4255](https://github.com/pingdotgg/t3code/commit/f7664fdf425555e0f85b74d904cfa4eaf57cf7af) | perf(web): Open Source Licenses downloads its manifest once per session (#17119) | Not applicable | N02 |
| [94d1a8e93f04](https://github.com/pingdotgg/t3code/commit/94d1a8e93f0434d34fa894523a04fe6a6f6172f6) | fix(server): restore OpenCode adapter test typecheck (#17810) | Not applicable | N06 |
| [98beed1a226c](https://github.com/pingdotgg/t3code/commit/98beed1a226c42b85c52ff5ee9d5dbb67d776a77) | fix(server): Claude subagents show the reasoning effort they run at (#17496) | Not applicable | N51 |
| [8c777fbfade3](https://github.com/pingdotgg/t3code/commit/8c777fbfade340f2092c011483c7fb0cde44c675) | feat(server): stop Claude subagents without stopping their owner (#17826) | Not applicable | N73 |
| [afd98d7d1840](https://github.com/pingdotgg/t3code/commit/afd98d7d1840ed76f57d3ece556257aebc00f1bc) | fix(server): stopped native subagents no longer read as Running (#17223) | Not applicable | N06 |
| [a11f46413329](https://github.com/pingdotgg/t3code/commit/a11f464133291122e0f6381b35e1b245d0aa9c73) | perf(server): t3_thread_list reads only the listed project's threads (#17843) | Not applicable | N06 |
| [cfbd1457ad09](https://github.com/pingdotgg/t3code/commit/cfbd1457ad093b1cd7f6c53851009a62b676b203) | fix(server): show diffs for projects outside the server cwd (#17724) | Not applicable | N11 |
| [50647de0143e](https://github.com/pingdotgg/t3code/commit/50647de0143e68fde051eb1a0ccf4fe63f9947b2) | fix(server): check the specific scope for scripts, preview input, and full-access MCP grants (#17772) | Not applicable | N06 |
| [dacd2cb649ee](https://github.com/pingdotgg/t3code/commit/dacd2cb649eea06a00244bd181eb400735e923d5) | fix(source-control): stop Forgejo status refresh from scanning every pull request (#12223) | Not applicable | N11 |
| [fd3549c8ffa5](https://github.com/pingdotgg/t3code/commit/fd3549c8ffa5eac8230a8b5e1a87e59d28ba488e) | refactor(contracts): source control provider kind is an open branded slug (#17739) | Not applicable | N11 |
| [7bc732804222](https://github.com/pingdotgg/t3code/commit/7bc732804222946c0c9b23af057484e4412698fc) | feat(source-control): each host package ships a client definition (#17746) | Not applicable | N11 |
| [c8a603307c2d](https://github.com/pingdotgg/t3code/commit/c8a603307c2d50f8fb9d302f4b51f20fd9f91b33) | refactor(client-runtime): add project clone sources come from host definitions (#17756) | Not applicable | N01 |
| [579159d6c6e9](https://github.com/pingdotgg/t3code/commit/579159d6c6e96f9c18358e492fbf0b8af740e80c) | refactor(web): host presentation and behavior come from client definitions (#17757) | Not applicable | N11 |
| [b0249ac87117](https://github.com/pingdotgg/t3code/commit/b0249ac8711703f1a1b8cd0f3acd11155f5bf172) | refactor(source-control): reference parsing and project matching are host resolvers (#17770) | Not applicable | N11 |
| [9b40d3746388](https://github.com/pingdotgg/t3code/commit/9b40d3746388e573370488f73b50e1e869b79a45) | feat(pull-requests): quick actions follow each host's capabilities, not GitHub (#17774) | Not applicable | N11 |
| [a4c8745abee3](https://github.com/pingdotgg/t3code/commit/a4c8745abee3f73f960c914800974dee530dbc3b) | feat(projects): new projects can be published to any ready host (#17860) | Not applicable | N11 |
| [f82796a16abb](https://github.com/pingdotgg/t3code/commit/f82796a16abb707c1e2c41301ad8d6e3153668c5) | fix(server): send Claude MCP servers over the control channel (#17898) | Ported | Register the host MCP server through the Claude control channel. |
| [ec4378614046](https://github.com/pingdotgg/t3code/commit/ec4378614046ab58864ff5e640de2b46f6704505) | fix(web): a finished reply replaced by a steer is no longer labeled partial (#17761) | Not applicable | N10 |
| [61eb4af69c38](https://github.com/pingdotgg/t3code/commit/61eb4af69c38c1f39e58ed76aa695b751f9f9baf) | fix(client-runtime): queued runs that start after a steer show up in the thread (#17764) | Not applicable | N10 |
| [2a4ee339827e](https://github.com/pingdotgg/t3code/commit/2a4ee339827e480da6ad0139585cf634a46ff73e) | feat(mobile): choose the microphone order for voice input (#17896) | Not applicable | N01 |
| [4f63275ac788](https://github.com/pingdotgg/t3code/commit/4f63275ac7885281d245022e65f1e9e8816b1382) | feat(source-control): host settings live on each host's definition, with a GitCafe token (#17901) | Not applicable | N16 |
| [38c187bfb7a0](https://github.com/pingdotgg/t3code/commit/38c187bfb7a0020db36ccc1479e7c3501b205fb8) | fix(pi): preserve tool images and structured results (#17851) | Not applicable | N65 |
| [a8bdfdb54181](https://github.com/pingdotgg/t3code/commit/a8bdfdb54181fb5c9d43f2747112c137a804d016) | fix(server): Claude 5 task lists reach the tasks drawer (#14964) | Not applicable | N74 |
| [f92afe79c921](https://github.com/pingdotgg/t3code/commit/f92afe79c921dd7de20204e83d62fed0491b4e54) | fix(web): find bar and thread details panel stop covering each other (#17858) | Not applicable | N16 |
| [0442f5bfcca6](https://github.com/pingdotgg/t3code/commit/0442f5bfcca6eca556ee79d41f4addb7a2022188) | fix(web): use server metadata for file chip icons (#17923) | Not applicable | N23 |
| [c8b89a01d70f](https://github.com/pingdotgg/t3code/commit/c8b89a01d70fd54cbc6b6a70ba0fdff702b821aa) | fix(desktop): copy images from HTML previews (#17555) | Not applicable | N09 |
| [e408cac0d7d9](https://github.com/pingdotgg/t3code/commit/e408cac0d7d903ccfcecfb906d4e48292294ab55) | docs(pi): update installation and remote login guidance (#17836) | Not applicable | N65 |
| [7e8071f1f156](https://github.com/pingdotgg/t3code/commit/7e8071f1f1566220fb52cbfaaf97a2bc4e93a193) | fix(pi): preserve native abort outcomes (#17853) | Not applicable | N65 |
| [160b18c8667f](https://github.com/pingdotgg/t3code/commit/160b18c8667f8370f46708d273d368648316b200) | fix(pi): keep thinking defaults specific to each model (#17835) | Not applicable | N65 |
| [62da1cd48bb4](https://github.com/pingdotgg/t3code/commit/62da1cd48bb43def11a79cd6aa033dbc478e4dfd) | fix(pi): preserve shell command exit codes (#17834) | Not applicable | N65 |
| [6e9c5a851c92](https://github.com/pingdotgg/t3code/commit/6e9c5a851c92c52e2f4a8a80c655e60db133f9ac) | fix(pi): expire and cancel extension approvals (#17840) | Not applicable | N65 |
| [9afde1c6dbe0](https://github.com/pingdotgg/t3code/commit/9afde1c6dbe00d2b87829732c88789c10f3429c0) | feat(pi): include native sessions in usage reports (#17848) | Not applicable | N66 |
| [818e2be75e1a](https://github.com/pingdotgg/t3code/commit/818e2be75e1a68bbcac122eac1b6fb815ccf4b73) | fix(server): route Copilot ACP subagent output into subagent threads (#17714) | Not applicable | N06 |
| [74dfbb697235](https://github.com/pingdotgg/t3code/commit/74dfbb697235b0db87355d3f8575dc2ba45dddf7) | fix(web): composer banner titles truncate beside their icon instead of wrapping (#17699) | Not applicable | N68 |
| [953edeec9dcd](https://github.com/pingdotgg/t3code/commit/953edeec9dcddd7afc582926eb1b65cfd82dc784) | fix(server): Muse turns no longer fail on Windows (#17163) | Not applicable | N06 |
| [5f7294d452a5](https://github.com/pingdotgg/t3code/commit/5f7294d452a5c6d6e0b7bd2c992478f830fbe212) | fix(pi): allow known read-only T3 tools without approval (#17852) | Not applicable | N65 |
| [477282263a91](https://github.com/pingdotgg/t3code/commit/477282263a9115f911380b5aec7d4973267e4edc) | fix: worktree threads keep their worktree when the agent starts, and messages sent during setup queue (#17654) | Not applicable | N03 |
