# Native file capabilities

The host injects `OpenOptions.NativeFiles`, the optional final argument of
`NewBroker`, or `BrokerOptions.NativeFiles` during initial composition. Direct
Driver consumers use `WithNativeFiles`. `NewNativeFiles` borrows one scratch
root and a distinct persistent root for each admitted provider. All roots must
have the same owner and user/workspace scope. The host closes them after the
Broker and native children stop. A custom remote `Factory` can operate without
local native capabilities; local probes and launches fail without them.

Each `managedfiles.Grant` declares owner, category, schema, scope, canonical
absolute existing path, byte/file/entry/lease limits, trigger, cleanup and
recovery. Opening a grant checks existing contents and follows no symlink in
the root path. It creates no root, resolves no HOME/cwd and chooses no alternate
path on failure. Interrupted files remain visible and count toward admission.
Scratch allocations reserve their entire finite budget before creating work.
The bounded `File` rejects excessive writes before writing any excess bytes.
Live writers and subprocesses retain their reservation through actual closure.

Native version/status/model inspection uses a granted scratch cwd. MCP launch
files contain only credential references, have a 4 KiB allowance and are
removed after the process exits. Native temporary files have a 32 MiB / 256-entry
allowance per launch. HOME, provider state, XDG paths and temporary paths are
set explicitly; ambient HOME/TMPDIR and application data paths are excluded.
Native login, native resume data and native diagnostics belong to the declared
persistent provider root. The library neither copies old provider databases
nor changes the vendor's native file formats. Broker events and provider
settings have only the shared storage authority.

Grant admission and post-exit inventory checks do not sandbox a subprocess.
The host must constrain writable mounts/permissions and apply physical disk
limits to persistent native state and scratch. Bubblewrap, container mounts and
native filesystem quotas remain deployment responsibilities. There is no
automatic deployment activation or claim of live mount-isolation acceptance.
