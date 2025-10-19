## Developer README — go-git

This document is a compact, developer-focused guide to the `go-git` codebase. It summarizes the project layout, core concepts and algorithms, main public APIs, common usage flows, and suggested follow-ups for deeper exploration.

Note: this is a living overview intended to help contributors quickly find the parts of the codebase they need. It is not exhaustive; follow-ups at the end point to deeper reads.

## High-level purpose

`go-git` is a pure-Go implementation of Git. It exposes both low-level "plumbing" APIs (objects, packfile parsing/encoding, refs, rev-list operations) and higher-level "porcelain" helpers (clone, init, worktree operations). The implementation is intentionally modular: storage is pluggable (in-memory, filesystem, custom storers), transports are registerable (http, ssh, custom), and object formats (SHA1 vs SHA256) are supported via configuration.

## Top-level layout (important files/folders)

- `repository.go` — high-level `Repository` type and porcelain entry points (Init, Open, Clone, PlainInit, PlainOpen, etc.). Many client-facing flows begin here.
- `worktree*.go`, `worktree_*.go` — worktree operations (checkout, commit, status).
- `plumbing/` — core plumbing primitives:
  - `plumbing/object` — typed objects (Commit, Tree, Blob, Tag) and helpers to decode objects.
  - `plumbing/storer` — interfaces for storers (EncodedObjectStorer, ReferenceStorer, Transactioner, etc.).
  - `plumbing/format/packfile` — packfile parser & encoder, delta handling, thin packs, cache.
  - `plumbing/revlist` — algorithms to walk commit ancestry and collect reachable objects.
  - `plumbing/transport` — interfaces and implementations for fetch/push (http, ssh, local transports).
- `storage/` — implementations of storage abstractions (filesystem, memory, transactional storers).
- `_examples/` — many small examples showcasing usage (clone, custom transport, auth variants, log, etc.). Good starting points for API usage patterns.
- `plumbing/*` tests and package-level tests scattered across the repo are good runnable examples of behavior and edge cases.

## Key abstractions and contracts

- Repository: porcelain entry point. Holds a `storage.Storer` and optionally a `billy.Filesystem` (worktree). Main public methods: `Init`, `Open`, `Clone`, `PlainInit`, `PlainOpen`, `Head()`, `Reference()`, `References()`, `Worktree()`.
- Storer / storage.Storer: central storage interface (composed of encoded-object and reference storers) used everywhere. Allows different backends (filesystem, memory). Key methods you will commonly see:
  - `NewEncodedObject()`, `SetEncodedObject()`, `EncodedObject(type, hash)`, `HasEncodedObject(hash)`, `RawObjectWriter(type, size)`
  - `Reference` operations: `SetReference`, `Reference`, `References()` — see `plumbing/storer/reference.go`.
  - Optional interfaces: `PackfileWriter`, `PackedObjectStorer`, `LooseObjectStorer`, `Transactioner` — code branches often check these with type assertions.
- plumbing.EncodedObject / plumbing.ObjectType: representation of object headers and raw contents. `plumbing` package defines `ObjectType` constants (Commit/Tree/Blob/Tag/OFSDelta/REFDelta)`.

## Important algorithms & implementation details

- Packfile parsing and encoding (`plumbing/format/packfile`):
  - Parser does a streaming scan of a `.pack` file using a `Scanner`, caches object headers, handles OFS and REF deltas, and can store objects directly into a storage while parsing. It supports "low memory" mode for large packfiles and thin packs (external refs).
  - Encoder selects objects to pack via a `deltaSelector`, supports OFS and REF delta encodings, sliding window delta selection (`packWindow`) and writes compressed object data with zlib streams and an overall SHA1 checksum.

- Rev-list (`plumbing/revlist`):
  - Implements the core reachable-object collection logic used by fetch/push and pack generation. Given a set of commit hashes and optional ignores, it walks commit ancestry and trees to produce the list of object hashes to include.
  - Uses commit preorder iterators and tree walkers from `plumbing/object` to avoid revisiting seen objects and to skip submodules.

- Object storage patterns:
  - The code frequently performs type assertions on storers to detect optional capabilities (e.g., `PackfileWriter`) and fallbacks when capabilities are missing.
  - Encoded object iteration uses lazy iterators (`EncodedObjectLookupIter`) to fetch objects on demand.

## Common usage flows (where to look)

- Clone (porcelain): `repository.Clone` -> `repository.clone` — creates a repo, sets up remotes and transport, fetches objects using `transport.Fetch` and processes received packfiles using the packfile parser, then sets refs and optionally checks out a worktree.
- Pack generation (porcelain pack creation): `packfile.NewEncoder` is used from `repository` to create packfiles for upload/push; it queries storage to select objects to pack and writes the pack stream.
- Fetch/Push: implemented via `plumbing/transport` implementations. Transport code interacts with `revlist` to compute wants/haves, uses packfile parsing on fetch, and pack encoding on push.
- Commit history walking: `Repository.Log` -> uses `object.CommitPreorderIter` and revlist utilities.

## Examples and entry points

- `_examples/clone` shows typical Clone use (memory or filesystem storers).
- `_examples/log` demonstrates walking commits and printing history.
- `_examples/custom_http` shows how to register a custom transport and use a custom HTTP client.

## How to explore the code when making changes

1. Start from the public API you want to change (e.g., `Repository.Clone` or `storage.Storer`).
2. Search for usages with `grep` (e.g., `grep -R "PackfileWriter" -n`).
3. Run unit tests for the package (e.g., `go test ./plumbing/format/packfile -run TestName`) to get immediate feedback.
4. When changing storage behavior, implement new storage by satisfying `storage.Storer` and its composed interfaces.

## Tests and examples to read first

- `plumbing/format/packfile` tests (parser_test, encoder_test) — packfile edge cases and thin pack handling.
- `plumbing/revlist/revlist_test.go` — revlist expected behaviour and submodule handling.
- `_examples` folder — small runnable programs that exercise public APIs.

## Common pitfalls / gotchas

- Many storers implement optional interfaces; code often does type assertions and has fallback code paths. When adding a new storer, ensure it supports the optional interfaces expected by the caller or the code will fall back to slower/more general paths.
- Packfile parsing supports thin packs and external references; when adding features to pack handling, be careful about `REFDelta` external references (`externalRef`) and how they resolve from storage.
- Object hashing format (SHA1 vs SHA256): object format is configurable; some components create hashers assuming SHA1 — check `plumbing/format/config` and packfile encoder usage.

## Developer TODOs / follow-ups

- Expand this READMEDEV with per-package diagrams (object flow during Clone/Fetch/Push).
- Add a small "how packfile parsing works" diagram that maps `Scanner -> Parser -> Storage` paths.
- Catalogue optional storer interfaces and where they are used (type assertions) into a single table.
- Add end-to-end integration smoke tests for different storage combinations (memory vs filesystem) and low-memory pack parsing.

## Quick pointers (files to open next)

- `repository.go` — main porcelain flows.
- `plumbing/format/packfile/parser.go` and `encoder.go` — packfile algorithms.
- `plumbing/revlist/revlist.go` — reachable-object logic.
- `plumbing/storer/*` — interfaces and iterator helpers.
- `_examples/*` — runnable usage samples.

---

If you'd like, I can now:
- Expand any section above into a deeper per-package developer guide (one package at a time).
- Generate diagrams (plantuml or mermaid) for Clone/Fetch object flow.
- Add a `CONTRIBUTING_DEV.md` with a checklist for adding new storers or transports.

Tell me which of these follow-ups you'd like next and I will continue.

## Developer quick-start (build / test / run examples)

These commands assume you have Go installed (see `go.mod` for the minimal supported version).

Run all unit tests in the repository (may be slow):

```bash
go test ./... -v
```

Run tests for a package only (faster):

```bash
go test ./plumbing/format/packfile -run TestName -v
```

Run an example (for instance the log example):

```bash
go run ./_examples/log
```

Build the module:

```bash
go build ./...
```

## Module / Go version / docs

- Module path: `github.com/go-git/go-git/v6` (see `go.mod`).
- Minimum Go version (declared in `go.mod`): `go 1.24.0`.
- API docs: https://pkg.go.dev/github.com/go-git/go-git/v6

## Public `Repository` API (quick reference)

Common porcelain entry points and methods on `Repository` (this list is a quick index; see `repository.go` for full signatures and options):

- Init, Open — create or open a repository backed by a `storage.Storer` and optional worktree FS.
- Clone, CloneContext, PlainInit, PlainOpen — common clone/init/open helpers.
- Config, SetConfig, ConfigScoped — access repository config.
- Remote, Remotes, CreateRemote, CreateRemoteAnonymous, DeleteRemote — remote management.
- Branch, CreateBranch, DeleteBranch — branch config helpers.
- CreateTag, Tag, DeleteTag — tag creation and lookup (annotated tags supported).
- Object, Objects — access raw objects and iterators.
- Head, Reference, References — reference lookups.
- Worktree — obtain a `Worktree` object for checkout/commit operations.

If you plan to alter `Repository` behavior, add or update unit tests in `repository_*.go` and the `_examples` to demonstrate end-to-end behavior.

## Optional/auxiliary storer interfaces (where callers use type assertions)

The codebase depends on several optional interfaces that a `storer` or `storage.Storer` might implement. Callers often detect these with type assertions and use optimized code paths when available. Important ones include:

- `PackfileWriter` — allows directly writing a packfile to storage (`PackfileWriter() (io.WriteCloser, error)`). Used when creating a pack to push.
- `PackedObjectStorer` — packfile related helpers (e.g., `ObjectPacks()`).
- `LooseObjectStorer` — iteration over loose objects and per-object mtime (`ForEachObjectHash`, `LooseObjectTime`).
- `DeltaObjectStorer` — returns delta objects without resolving deltas.
- `Transactioner` — begin/commit/rollback transactional operations.
- `FilesystemStorer` / `storage.FilesystemStorer` — exposes underlying filesystem (used to create `.git` file when necessary).
- `LowMemoryCapable` (parser helper) — storers can signal low-memory support for pack parsing.
- storage-level interfaces from `storage/storer.go` (e.g., `IndexStorer`, `ShallowStorer`, `config.ConfigStorer`, `ModuleStorer`) — used by higher-level flows and commands.

When adding a new storage backend, survey the repository for these assertions (search e.g. `PackfileWriter`) and implement the interfaces required by the operations you want to optimize.

## What's missing from READMEDEV (quick audit)

I reviewed the code and `READMEDEV.md`. The READMEDEV now contains a compact overview but some additional useful items were missing and are now added or noted below:

- Quick-start build/test/run commands — added in this update.
- Explicit list of public `Repository` methods — added as a quick reference.
- More complete list of optional storer interfaces and storage-level interfaces — added.

Remaining suggestions (not added yet):

- An architectural diagram showing object flow for Clone/Fetch/Push (I can generate Mermaid/PlantUML).
- A table mapping optional storer interfaces to the files that assert them (useful to implement a new `Storer`). I can generate this by scanning the codebase.
- A small HOWTO for contributing new storage backends or transports with a checklist and minimal example.

If you'd like, I can implement any of the remaining suggestions now (diagram, interface-to-file table, or CONTRIBUTING_DEV checklist). Which one should I do next?
