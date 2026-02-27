Perfect! Now I have all the information I need. Let me compile the comprehensive research report:

---

# Relux Module Internal Structure Research

## Overview

This document provides a comprehensive analysis of Relux module layouts in the relux-sample-app. Based on thorough examination of 4+ modules (Notes, App, ErrorHandling, Navigation, Account, Logger), the patterns are consistent and highly structured.

## Key Architectural Principles

1. **Namespace-first structure**: Every module starts with an enum namespace that defines the hierarchical organization
2. **Domain separation**: Clear separation between Data/API, Business logic, and UI layers
3. **Protocol-driven architecture**: Heavy use of interfaces (protocols) to enable dependency injection via SwiftIoC
4. **Actor-based concurrency**: Business logic uses Swift actors for thread safety and concurrent execution
5. **Relux framework integration**: State machines, effects, and sagas for predictable state management
6. **File naming convention**: Consistent dot-notation naming (e.g., `Notes+Business+State.swift`)

---

## Directory Structure Pattern

### Top-Level Module Organization

```
Modules/
├── <ModuleName>/
│   ├── <ModuleName>+Namespace.swift           [REQUIRED] Module structure definition
│   ├── <ModuleName>+Module.swift              [REQUIRED] Relux.Module implementation + IoC setup
│   ├── Business/
│   │   ├── <ModuleName>+Business+Action.swift [IF STATEFUL] Actions (Relux.Action enum)
│   │   ├── <ModuleName>+Business+State.swift  [IF STATEFUL] State container (actor/class)
│   │   ├── <ModuleName>+Business+State+Reducer.swift [IF STATEFUL] Action reducer logic
│   │   ├── <ModuleName>+Business+Err.swift    [IF NEEDED] Error types
│   │   ├── Model/
│   │   │   ├── <ModuleName>+Business+Model+<Entity>.swift
│   │   │   └── ...
│   │   └── Middleware/
│   │       ├── <ModuleName>+Business+Effect.swift   [IF HAS EFFECTS] Relux.Effect enum
│   │       ├── <ModuleName>+Business+Flow.swift     [FOR FULL FEATURES] Flow (reducer) - Relux.Flow
│   │       ├── <ModuleName>+Business+Saga.swift     [IF APP-LEVEL] Saga (side effects)
│   │       ├── <ModuleName>+Business+Service.swift  [IF DATA ACCESS] Service interface + impl
│   │       └── ...
│   ├── Data/
│   │   └── Api/
│   │       ├── <ModuleName>+Data+Api+Fetcher.swift  [IF API CALLS] API client interface + impl
│   │       └── DTO/
│   │           └── <ModuleName>+Data+Api+DTO+<Entity>.swift
│   ├── Err/
│   │   └── <ModuleName>+Business+Err.swift    [ALTERNATIVE: Domain-specific errors]
│   ├── UI/
│   │   ├── <ModuleName>+UI+State.swift        [IF HAS UI STATE] Presentation state (ObservableObject)
│   │   ├── <ModuleName>+UI+Router.swift       [IF ROUTING] Navigation router
│   │   ├── Model/
│   │   │   ├── <ModuleName>+UI+Model+<Entity>.swift
│   │   │   └── <ModuleName>+UI+Page.swift     [IF ROUTING] Page enum for routing
│   │   ├── Components/
│   │   │   └── <ComponentName>/
│   │   │       ├── <ModuleName>+UI+Components+<Name>.swift
│   │   │       └── <ModuleName>+UI+Components+<Name>+Props.swift
│   │   ├── <Feature>/  (List, Details, Create, Edit, Widget, etc.)
│   │   │   ├── <ModuleName>+UI+<Feature>+Container.swift      [VIEW CONTAINER - Relux.UI.Container]
│   │   │   └── Page/
│   │   │       ├── <ModuleName>+UI+<Feature>+Container+Page.swift       [DUMB VIEW - Relux.UI.View]
│   │   │       └── <ModuleName>+UI+<Feature>+Container+Page+Props.swift [PROPS DTO]
│   │   └── ...
│   └── Utils/
│       └── <ModuleName>+Utils+<Helper>.swift  [OPTIONAL: Local helpers]
```

---

## File-by-File Content Patterns

### 1. Namespace Definition File
**Filename**: `<ModuleName>+Namespace.swift`

```swift
// Simple namespace hierarchy
enum Notes {
    enum Data {
        enum Api {
            enum DTO {}
        }
    }
    enum Business {
        enum Model {}
    }
    enum UI {
        enum Model {}
        enum Component {}
        enum Widget {}
        enum List {}
        enum Details {}
        enum Create {}
        enum Edit {}
    }
}
```

**Purpose**: Defines module's type hierarchy. Used as a namespace for all types within the module. This is the "structure definition" file—no implementations.

**Key Characteristics**:
- Pure enum structure (no implementations, no logic)
- Hierarchical organization mirrors file system
- Acts as the "namespace" that all other files extend

---

### 2. Module Definition File
**Filename**: `<ModuleName>+Module.swift`

**Pattern A: Full Relux Module (with states and sagas)**
```swift
import SwiftIoC

extension Notes {
    struct Module: Relux.Module {
        private let ioc: IoC

        var states: [any Relux.AnyState]
        var sagas: [any Relux.Saga]

        init() async {
            self.ioc = Self.buildIoC()

            self.states = [
                self.ioc.get(by: Notes.Business.State.self)!,
                await self.ioc.getAsync(by: Notes.UI.State.self)!
            ]
            self.sagas = [
                await self.ioc.getAsync(by: Notes.Business.IFlow.self)!
            ]
        }
    }
}

extension Notes.Module {
    static func buildIoC() -> IoC {
        let ioc: IoC = .init(logger: IoC.Logger(enabled: false))

        ioc.register(Notes.Business.State.self, lifecycle: .container, resolver: buildBusinessState)
        ioc.register(Notes.UI.State.self, lifecycle: .container, resolver: { await buildUIState(ioc: ioc) })
        ioc.register(Notes.Business.IService.self, lifecycle: .container, resolver: { buildSvc(ioc: ioc) })
        ioc.register(Notes.Data.Api.IFetcher.self, lifecycle: .container, resolver: { buildFetcher(ioc: ioc) })
        ioc.register(Notes.Business.IFlow.self, lifecycle: .container, resolver: { await buildFlow(ioc: ioc) })

        return ioc
    }

    private static func buildBusinessState() -> Notes.Business.State {
        Notes.Business.State()
    }

    private static func buildUIState(ioc: IoC) async -> Notes.UI.State {
        await Notes.UI.State(state: ioc.get(by: Notes.Business.State.self)!)
    }

    // ... other builders ...
}
```

**Pattern B: Simple Module (no states, only sagas)**
```swift
extension ErrorHandling {
    struct Module: IModule {
        private let ioc: IoC = Self.buildIoC()

        let states: [any Relux.AnyState] = []
        var sagas: [any Relux.Saga]
        
        init() {
            self.sagas = [
                self.ioc.get(by: ErrorHandling.Business.ISaga.self)!
            ]
        }
    }
}
```

**Pattern C: App-level Module (with store injection)**
```swift
@MainActor
struct Module: IModule {
    private let ioc: IoC

    let states: [any Relux.AnyState] = []
    let sagas: [any Relux.Saga]

    init(store: Relux.Store) {
        self.ioc = Self.buildIoC(store: store)
        self.sagas = [
            self.ioc.get(by: SampleApp.Business.ISaga.self)!
        ]
    }
}
```

**Key Characteristics**:
- Implements `Relux.Module` protocol
- Contains internal IoC container (SwiftIoC library)
- `init()` or `init(store: Relux.Store)` depending on module type
- `states` array: list of all Relux.AnyState instances
- `sagas` array: list of all Relux.Saga instances (can be Flow or Saga)
- `buildIoC()` static method registers all dependencies
- Builder methods (`buildXxx()`) instantiate and wire up dependencies
- `async` initialization for modules with async state setup

---

### 3. Business Action File
**Filename**: `<ModuleName>+Business+Action.swift`

```swift
extension Notes.Business {
    enum Action: Relux.Action {
        case obtainNotesSuccess(notes: [Model.Note])
        case obtainNotesFail(err: Err)

        case upsertNoteSuccess(note: Model.Note)
        case upsertNoteFail(err: Err)

        case deleteNoteSuccess(note: Model.Note)
        case deleteNoteFail(err: Err)
    }
}
```

**Key Characteristics**:
- Enum conforming to `Relux.Action`
- Associated values for payload data
- Naming: `<VerbSuccess>` and `<VerbFail>` pairs (or just single case for side-effect actions)
- No logic—pure data structure
- All cases at same level (flat enum)

---

### 4. Business State File
**Filename**: `<ModuleName>+Business+State.swift`

```swift
import Combine

extension Notes.Business {
    actor State {
        @Published var notes: MaybeData<[Model.Note], Err> = .initial()
    }
}

extension Notes.Business.State: Relux.BusinessState {
    func reduce(with action: any Relux.Action) async {
        switch action as? Notes.Business.Action {
            case .none: break
            case let .some(action): await internalReduce(with: action)
        }
    }
    
    func cleanup() async {
        self.notes = .initial()
    }
}
```

**Key Characteristics**:
- `actor` (not class or struct) for thread-safe state
- `@Published` properties for Combine publishers
- `MaybeData<T, E>` wrapper for loading/success/failure states
- Conforms to `Relux.BusinessState`
- `reduce(with:)` method switches on action type and delegates to `internalReduce`
- `cleanup()` resets state to initial
- No direct mutation outside of `reduce()`

---

### 5. Reducer (Action Handler) File
**Filename**: `<ModuleName>+Business+State+Reducer.swift`

```swift
extension Notes.Business.State {
    func internalReduce(with action: Notes.Business.Action) async {
        switch action {
            case let .obtainNotesSuccess(notes):
                self.notes = .success(notes)
            case let .obtainNotesFail(err):
                self.notes = .failure(err)

            case let .upsertNoteSuccess(note):
                var notes = (self.notes.value ?? [])
                notes.upsertByIdentity(note)
                self.notes = .success(notes)
            case .upsertNoteFail:
                break

            case let .deleteNoteSuccess(note):
                var notes = (self.notes.value ?? [])
                notes.removeById(note)
                self.notes = .success(notes)

            case .deleteNoteFail:
                break
        }
    }
}
```

**Key Characteristics**:
- Extension on State class
- `internalReduce(with:)` method (matches type of action enum)
- Exhaustive switch on action cases
- Direct mutation of `@Published` properties
- Payload extraction and state transformation
- May ignore failure cases with `break`
- Async function (though doesn't always await)

---

### 6. Effect File
**Filename**: `<ModuleName>+Business+Effect.swift`

```swift
extension Notes.Business {
    enum Effect: Relux.Effect {
        case obtainNotes
        case upsert(note: Model.Note)
        case delete(note: Model.Note)
    }
}

// Alternative: App-level effects
extension SampleApp.Business {
    enum Effect: Relux.Effect {
        case setAppContext
    }
}
```

**Key Characteristics**:
- Enum conforming to `Relux.Effect`
- Cases represent side effects to trigger
- Can have associated values (data needed to perform the effect)
- No logic—pure enumeration
- Used by Flow/Saga to drive async operations

---

### 7. Flow File (Reducer/Saga for full features)
**Filename**: `<ModuleName>+Business+Flow.swift`

```swift
extension Notes.Business {
    protocol IFlow: Relux.Flow {}
}

extension Notes.Business {
    actor Flow {
        private typealias Model = Notes.Business.Model
        let dispatcher: Relux.Dispatcher
        private let svc: Notes.Business.IService

        init(
            dispatcher: Relux.Dispatcher? = .none,
            svc: Notes.Business.IService
        ) async {
            let defaultDispatcher = await Self.defaultDispatcher
            self.dispatcher = dispatcher ?? defaultDispatcher
            self.svc = svc
        }
    }
}

extension Notes.Business.Flow: Notes.Business.IFlow {
    func apply(_ effect: any Relux.Effect) async -> Relux.Flow.Result {
        switch effect as? Notes.Business.Effect {
            case .none: .success
            case .obtainNotes: await obtainNotes()
            case let .upsert(note): await upsert(note)
            case let .delete(note): await delete(note)
        }
    }
}

extension Notes.Business.Flow {
    private func obtainNotes() async -> Relux.Flow.Result {
        switch await svc.getNotes() {
            case let .success(notes):
                await actions {
                    Notes.Business.Action.obtainNotesSuccess(notes: notes)
                }
                return .success
            case let .failure(err):
                await actions(.concurrently) {
                    Notes.Business.Action.obtainNotesFail(err: err)
                    ErrorHandling.Business.Effect.track(error: err)
                }
                return .success
        }
    }

    private func upsert(_ note: Model.Note) async -> Relux.Flow.Result {
        switch await svc.upsert(note: note) {
            case .success:
                await actions {
                    Notes.Business.Action.upsertNoteSuccess(note: note)
                }
                return .success
            case let .failure(err):
                await actions(.concurrently) {
                    Notes.Business.Action.upsertNoteFail(err: err)
                    ErrorHandling.Business.Effect.track(error: err)
                }
                return .failure(err)
        }
    }
    // ...
}
```

**Key Characteristics**:
- `actor Flow` for concurrent effect handling
- Conforms to `Relux.Flow` protocol (via `IFlow` interface)
- Has `dispatcher: Relux.Dispatcher` to trigger actions
- `apply(_ effect:) -> Relux.Flow.Result` method
- Returns `.success` or `.failure(Error)` to indicate flow completion
- Each effect has a corresponding `private` method
- Methods use `await actions { ... }` to dispatch actions back to state
- Can dispatch multiple actions with `.concurrently` modifier
- Can reference other module effects (e.g., `ErrorHandling.Business.Effect.track`)
- **Note**: Relux.Flow is sometimes called a "Saga" in comments, but technically it's a Flow that returns a Result

---

### 8. Saga File (App-level or simple side effects)
**Filename**: `<ModuleName>+Business+Saga.swift`

```swift
extension SampleApp.Business {
    protocol ISaga: Relux.Saga {}
}

extension SampleApp.Business {
    actor Saga {
        private let store: Relux.Store

        init(store: Relux.Store) {
            self.store = store
        }
    }
}

extension SampleApp.Business.Saga: SampleApp.Business.ISaga {
    func apply(_ effect: any Relux.Effect) async {
        switch effect as? SampleApp.Business.Effect {
            case .none: break
            case .setAppContext: await setAppContext()
        }

        switch effect as? Auth.Business.Effect {
            case .runLogoutFlow: await cleanupAppCache()
            default: break
        }
    }
}

extension SampleApp.Business.Saga {
    private func setAppContext() async {
        await actions {
            AppRouter.Action.set([.auth(page: .localAuth)])
        }
    }

    private func cleanupAppCache() async {
        await self.store.cleanup(exclusions: [AppRouter.self])
    }
}
```

**Key Characteristics**:
- `actor Saga` for handling side effects
- Conforms to `Relux.Saga` protocol (via `ISaga` interface)
- `apply(_ effect:) async` method (returns nothing, just async operations)
- Can handle effects from multiple modules
- Access to full `Relux.Store` for cross-module coordination
- Uses `await actions { ... }` to dispatch
- Typically used for app-level orchestration, cleanup, cross-module flows

---

### 9. Service Interface & Implementation
**Filename**: `<ModuleName>+Business+Service.swift` (both interface and impl in one file)

```swift
extension Notes.Business {
    protocol IService: Sendable {
        typealias Note = Notes.Business.Model.Note
        typealias Err = Notes.Business.Err

        func getNotes() async -> Result<[Note], Err>
        func upsert(note: Note) async -> Result<Void, Err>
        func delete(noteId: Note.Id) async -> Result<Void, Err>
    }
}

extension Notes.Business {
    actor Service {
        private let fetcher: Notes.Data.Api.IFetcher

        init(fetcher: Notes.Data.Api.IFetcher) {
            self.fetcher = fetcher
        }
    }
}

extension Notes.Business.Service: Notes.Business.IService {
    func getNotes() async -> Result<[Note], Err> {
        switch await self.fetcher.getNotes() {
            case let .success(notes): 
                .success(notes.map { Note(from: $0) }.sorted())
            case let .failure(err): 
                .failure(err)
        }
    }
    
    func upsert(note: Notes.Business.Model.Note) async -> Result<Void, Notes.Business.Err> {
        await self.fetcher.upsert(note: note.asDto)
    }
    
    func delete(noteId: Notes.Business.Model.Note.Id) async -> Result<Void, Notes.Business.Err> {
        await self.fetcher.deletetNote(by: noteId)
    }
}
```

**Key Characteristics**:
- Protocol interface (`IService`) conforming to `Sendable`
- Async methods returning `Result<T, Error>`
- Actor implementation for thread safety
- Depends on fetcher/API layer
- Handles DTO↔️ Model conversion
- May add business logic (sorting, filtering, etc.)

---

### 10. Data/API Fetcher File
**Filename**: `<ModuleName>+Data+Api+Fetcher.swift`

```swift
extension Notes.Data.Api {
    protocol IFetcher: Sendable {
        typealias Err = Notes.Business.Err
        typealias DTO = Notes.Data.Api.DTO

        func getNotes() async -> Result<[DTO.Note], Err>
        func upsert(note: DTO.Note) async -> Result<Void, Err>
        func deletetNote(by id: DTO.Note.Id) async -> Result<Void, Err>
    }
}

extension Notes.Data.Api {
    actor Fetcher {
        private var notes: Dictionary<DTO.Note.Id, DTO.Note> = [
            .init(id: .init(), date: .now, title: "title 1", content: "content 1"),
            // ...
        ].keyed(by: \.id)
    }
}

extension Notes.Data.Api.Fetcher: Notes.Data.Api.IFetcher {
    func getNotes() async -> Result<[DTO.Note], Err> {
        .success(Array(notes.values))
    }

    func upsert(note: DTO.Note) async -> Result<Void, Err> {
        self.notes[note.id] = note
        return .success(())
    }
    
    func deletetNote(by id: DTO.Note.Id) async -> Result<Void, Err> {
        self.notes.removeValue(forKey: id)
        return .success(())
    }
}

// Mock/Test variant
extension Notes.Data.Api {
    actor TestFetcher {
        private var notes: Dictionary<DTO.Note.Id, DTO.Note> = [
            .init(id: .init(), date: .now, title: "title 1", content: "content 1"),
        ].keyed(by: \.id)
    }
}

extension Notes.Data.Api.TestFetcher: Notes.Data.Api.IFetcher {
    // ... same implementation ...
}
```

**Key Characteristics**:
- Protocol interface (`IFetcher`) conforming to `Sendable`
- Actor implementation for thread-safe data access
- In this sample: in-memory mock data (not real HTTP calls)
- `Result<T, Error>` return type
- May have parallel `TestFetcher` implementation for testing
- DTO objects (not domain models)

---

### 11. DTO (Data Transfer Object) File
**Filename**: `<ModuleName>+Data+Api+DTO+<Entity>.swift`

```swift
extension Notes.Data.Api.DTO {
    struct Note: Codable {
        typealias Id = UUID
        
        let id: Id
        let date: Date
        let title: String
        let content: String
    }
}

extension Notes.Data.Api.DTO.Note {
    enum Err: Error {
        case failedToDecode(_ msg: String)
        case failedToEncode(_ msg: String)
    }
}

extension Notes.Data.Api.DTO.Note {
    enum CodingKeys: String, CodingKey {
        case id
        case date
        case title
        case content
    }
}

extension Notes.Data.Api.DTO.Note {
    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try container.decode(UUID.self, forKey: .id)
        self.title = try container.decode(String.self, forKey: .title)
        self.content = try container.decode(String.self, forKey: .content)
        let dateStr = try container.decode(String.self, forKey: .date)
        self.date = switch dateStr.utcStringAsLocalDate {
            case let .some(date): date
            case .none: throw Err.failedToDecode("date from \(dateStr)")
        }
    }
}

extension Notes.Data.Api.DTO.Note {
    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(title, forKey: .title)
        try container.encode(content, forKey: .content)
        try container.encode(date.localDateAsUtcString, forKey: .date)
    }
}
```

**Key Characteristics**:
- Struct (not class) conforming to `Codable`
- Mirrors API response structure (not domain model)
- Custom `CodingKeys` for API field name mapping
- Custom `init(from:)` and `encode(to:)` for complex field handling
- May have associated error enum
- Separate from domain model (Business.Model.Note)

---

### 12. Domain Model File
**Filename**: `<ModuleName>+Business+Model+<Entity>.swift`

```swift
extension Notes.Business.Model {
    struct Note {
        typealias Id = UUID

        let id: Id
        let createdAt: Date
        let title: String
        let content: String
    }
}

extension Notes.Business.Model.Note {
    init(from apiModel: Notes.Data.Api.DTO.Note) {
        self.id = apiModel.id
        self.createdAt = apiModel.date
        self.title = apiModel.title
        self.content = apiModel.content
    }
}

extension Notes.Business.Model.Note {
    var asDto: Notes.Data.Api.DTO.Note {
        .init(
            id: self.id,
            date: self.createdAt,
            title: self.title,
            content: self.content
        )
    }
}

extension Notes.Business.Model.Note: Identifiable {}
extension Notes.Business.Model.Note: Sendable {}
extension Notes.Business.Model.Note: Equatable {}
extension Notes.Business.Model.Note: Hashable {}
extension Notes.Business.Model.Note: Comparable {
    static func < (lhs: Self, rhs: Self) -> Bool {
        lhs.createdAt < rhs.createdAt
    }
}
```

**Key Characteristics**:
- Struct with typealias for ID type
- Conversion method `init(from:)` from DTO
- Conversion property `asDto` to DTO
- Conforms to multiple protocols via extensions: `Identifiable`, `Sendable`, `Equatable`, `Hashable`, `Comparable`
- No `Codable` (DTOs handle serialization)
- Clean separation from data layer

---

### 13. Error File
**Filename**: `<ModuleName>+Business+Err.swift`

```swift
extension Notes.Business {
    enum Err: Error, Sendable {
        case notImplemented
        case obtainFailed(cause: Error)
        case upsertFailed(note: Model.Note, cause: Error)
        case deleteFailed(noteId: Model.Note.Id, cause: Error)
    }
}

extension Notes.Business.Err: Hashable {
    func hash(into hasher: inout Hasher) {
        hasher.combine(localizedDescription)
    }
}

extension Notes.Business.Err: Equatable {
    static func == (lhs: Self, rhs: Self) -> Bool {
        lhs.localizedDescription == rhs.localizedDescription
    }
}
```

**Key Characteristics**:
- Enum conforming to `Error` and `Sendable`
- Cases represent specific error conditions
- Can have associated values (context data)
- Conformance to `Hashable` and `Equatable` for state comparison
- Hash/equality based on `localizedDescription` (not object identity)

---

### 14. UI State File
**Filename**: `<ModuleName>+UI+State.swift`

```swift
extension Notes.UI {
    final class State: ObservableObject, Relux.UIState {
        typealias Note = Notes.Business.Model.Note
        typealias Err = Notes.Business.Err

        @Published var notesGroupedByDay: MaybeData<[[Note]], Err> = .initial()
        @Published var notes: MaybeData<[Note.Id: Note], Err> = .initial()

        init(
            state: Notes.Business.State
        ) async {
            await initPipelines(state: state)
        }
    }
}

extension Notes.UI.State {
    func note(by id: Note.Id) -> MaybeData<Note?, Err> {
        switch notes {
            case .initial: .initial()
            case let .failure(err): .failure(err)
            case let .success(notes): .success(notes[id])
        }
    }
}

extension Notes.UI.State {
    private func initPipelines(state: Notes.Business.State) async {
        await state.$notes
            .map(Self.mapNotesToGroups)
            .receive(on: mainQueue)
            .assign(to: &$notesGroupedByDay)

        await state.$notes
            .map(Self.mapNotesToDict)
            .receive(on: mainQueue)
            .assign(to: &$notes)
    }

    nonisolated
    private static func mapNotesToDict(_ notes: MaybeData<[Note], Err>) -> MaybeData<[Note.Id: Note], Err> {
        switch notes {
            case .initial: .initial()
            case let .failure(err): .failure(err)
            case let .success(notes): .success(notes.keyed(by: \.id))
        }
    }

    nonisolated
    private static func mapNotesToGroups(_ notes: MaybeData<[Note], Err>) -> MaybeData<[[Note]], Err> {
        switch notes {
            case .initial: return .initial()
            case let .failure(err): return .failure(err)
            case let .success(notes): return .success(
                notes
                    .sorted { $0.createdAt > $1.createdAt }
                    .chunked { prev, next in
                        prev.createdAt.startOfDay == next.createdAt.startOfDay
                    }
                    .map { Array($0) }
            )
        }
    }
}
```

**Key Characteristics**:
- `final class` (not actor) conforming to `ObservableObject` and `Relux.UIState`
- `@Published` properties for SwiftUI reactivity
- Async `init()` to set up Combine pipelines
- Transforms Business state into UI-friendly structures
- Helper methods (getters) for specific queries
- Combines/groups data for presentation
- Maps Business errors to UI errors
- `nonisolated` static helper methods for transformations

---

### 15. UI Page Enum (Routing)
**Filename**: `<ModuleName>+UI+Page.swift`

```swift
extension Notes.UI.Model {
    enum Page: NavPathComponent {
        case list
        case details(id: Notes.Business.Model.Note.Id)
        case create
        case edit(note: Notes.Business.Model.Note)
    }
}
```

**Key Characteristics**:
- Enum conforming to `NavPathComponent`
- Cases represent distinct screens/pages
- Associated values for data passed between screens
- Used by Relux.Navigation router for path tracking

---

### 16. UI Container File (Connected Component)
**Filename**: `<ModuleName>+UI+<Feature>+Container.swift`

```swift
extension Notes.UI.List {
    struct Container: Relux.UI.Container {
        typealias Note = Notes.Business.Model.Note

        @EnvironmentObject private var notesState: Notes.UI.State

        var body: some View {
            content
        }

        private var content: some View {
            Page(
                props: .init(
                    notes: notesState.notesGroupedByDay
                ),
                actions: .init(
                    onReload: ViewCallback(reloadNotes),
                    onCreate: ViewCallback(openCreateNote),
                    onRemove: ViewInputCallback(remove)
                )
            )
        }
    }
}

extension Notes.UI.List.Container {
    private func reloadNotes() async {
        performAsync {
            Notes.Business.Effect.obtainNotes
        }
    }

    private func openCreateNote() async {
        await actions {
            AppRouter.Action.push(.app(page: .notes(.create)))
        }
    }

    private func remove(_ note: Note) async {
        await actions {
            Notes.Business.Effect.delete(note: note)
        }
    }
}
```

**Key Characteristics**:
- Struct conforming to `Relux.UI.Container`
- Injects UI state via `@EnvironmentObject` (for `ObservableObject`)
- Wraps "dumb" Page view with props and actions
- Handles UI event callbacks (button taps, etc.)
- Methods use `await actions { ... }` or `performAsync { ... }` to dispatch
- Can dispatch both Actions and Effects
- First point of contact with Relux store
- Bridges SwiftUI event system with Relux dispatcher

---

### 17. UI Page File (Dumb View)
**Filename**: `<ModuleName>+UI+<Feature>+Container+Page.swift`

```swift
extension Notes.UI.List.Container {
    struct Page: Relux.UI.View {
        typealias Note = Notes.Business.Model.Note
        typealias Err = Notes.Business.Err

        let props: Props
        let actions: Actions

        var body: some View {
            content
                .navigationTitle("Notes")
                .navigationBarTitleDisplayMode(.large)
                .navigationBarItems(trailing: createBtn)
                .animation(.easeInOut, value: props.notes.asAnimatableValue)
                .refreshable(action: actions.onReload.callAsFunction)
        }
    }
}

// Sections organized with extension marks
extension Notes.UI.List.Container.Page {
    private var createBtn: some View {
        NavBarBtn.iconBtn(
            systemName: "plus",
            action: actions.onCreate
        )
    }
}

extension Notes.UI.List.Container.Page {
    private var content: some View {
        List {
            listView(for: props.notes.value)
        }.overlay(content: loadingState)
    }

    @ViewBuilder
    private func loadingState() -> some View {
        switch props.notes {
            case .initial: initialView
            case .failure: failureView
            case let .success(notes): switch notes.isEmpty {
                case true: emptyListView
                case false: EmptyView()
            }
        }
    }
}

extension Notes.UI.List.Container.Page {
    private var initialView: some View {
        ProgressView("Loading...")
            .extendingContent()
    }

    private var failureView: some View {
        Text("Failed to load")
            .extendingContent()
    }

    private var emptyListView: some View {
        Text("No notes yet...")
            .extendingContent()
    }
}

extension Notes.UI.List.Container.Page {
    @ViewBuilder
    private func listView(for noteGroups: [[Note]]?) -> some View {
        switch noteGroups {
            case .none: EmptyView()
            case let .some(groups): noteGroupsView(for: groups)
        }
    }

    @ViewBuilder
    private func noteGroupsView(for noteGroups: [[Note]]) -> some View {
        ForEach(noteGroups, id: \.id) { group in
            notesGroupSection(for: group)
        }
    }

    private func notesGroupSection(for group: [Note]) -> some View {
        Section(header: notesGroupSectionHeader(for: group.first?.createdAt ?? .now)) {
            ForEach(group) { note in
                noteRow(for: note)
                    .swipeActions(edge: .trailing, allowsFullSwipe: false) { removeSwipeAction(for: note) }
            }
        }
    }

    private func removeSwipeAction(for note: Note) -> some View {
        SwipeButton(
            props: .init(icon: Image(systemName: "trash"), tint: .red),
            actions: .init(action: { await onRemove(note) })
        )
    }
}

extension Notes.UI.List.Container.Page {
    private func onRemove(_ note: Note) async {
        await actions.onRemove(note)
    }
}
```

**Key Characteristics**:
- Struct conforming to `Relux.UI.View`
- Receives `props` (data) and `actions` (callbacks)
- **Pure presentation** — no state access, no direct dispatch
- All methods are `private` (no external API)
- Organized into logical sections via extensions with MARK comments
- Uses `@ViewBuilder` for conditional SwiftUI views
- Calls `actions` callbacks for user interactions
- No direct dependency on Relux or Business layer

---

### 18. UI Page Props DTO
**Filename**: `<ModuleName>+UI+<Feature>+Container+Page+Props.swift`

```swift
// Example (implied structure, may be inline in Page file):

extension Notes.UI.List.Container.Page {
    struct Props {
        let notes: MaybeData<[[Note]], Err>
    }

    struct Actions {
        let onReload: ViewCallback<Void>
        let onCreate: ViewCallback<Void>
        let onRemove: ViewInputCallback<Note>
    }
}
```

**Key Characteristics**:
- Nested `Props` and `Actions` structs in Page
- `Props` contains all data needed for rendering
- `Actions` contains all callbacks
- Used to clearly separate concerns in Page init
- May be split into separate file if complex

---

### 19. Simple UI Router File (Navigation state)
**Filename**: `<ModuleName>+UI+Router.swift` or defined inline in Module

```swift
// May be wrapped in Router type or defined inline
typealias ModalRouter = Navigation.Business.ModalRouter
typealias AppRouter = Relux.Navigation.ProjectingRouter<AppPage>

extension Navigation.Business.ModalRouter {
    enum Action: Relux.Action {
        case present(page: Model.ModalPage)
        case dismiss
    }
}

extension Navigation.Business.ModalRouter {
    func internalReduce(with action: Action) async {
        switch action {
            case let .present(page):
                guard self.modalSheet != page else { return }
                self.modalSheet = page
            case .dismiss:
                self.modalSheet = .none
        }
    }
}
```

**Key Characteristics**:
- Navigation state as a Relux state (not just SwiftUI NavigationStack)
- Can be modal router or path-based router
- Actions for push/pop/set/present/dismiss
- Reducer updates navigation state
- Integrated with Relux store for time-travel debugging

---

## Common Patterns & Conventions

### Naming Conventions

| Pattern | Example | Meaning |
|---------|---------|---------|
| `<Module>+<Layer>+<Entity>.swift` | `Notes+Business+State.swift` | File naming convention |
| `enum <Module> { ... }` | `enum Notes { ... }` | Namespace definition |
| `extension <Module>.<Layer>.<Type>` | `extension Notes.Business.State` | Type extension location |
| `<VerbSuccess>` / `<VerbFail>` | `obtainNotesSuccess`, `obtainNotesFail` | Action case naming |
| `I<Type>` | `IService`, `IFlow`, `IFetcher` | Protocol interface naming |
| `<Type>` | `Service`, `Flow`, `Fetcher` | Concrete implementation naming |
| `Page` | `Notes+UI+<Feature>+Container+Page.swift` | Dumb view component |
| `Container` | `Notes+UI+<Feature>+Container.swift` | Connected component |
| `Props` / `Actions` | `Props`, `Actions` | DTO structs in Page |

### Protocol Conformances

**Frequently Used Protocols**:

| Protocol | Usage |
|----------|-------|
| `Relux.Module` | Module root, provides states and sagas |
| `Relux.BusinessState` | Business logic state, handles reduce() |
| `Relux.UIState` | Presentation state, transforms business state |
| `Relux.Flow` | Effect processor returning Result |
| `Relux.Saga` | Side-effect handler (no return) |
| `Relux.Action` | Enum of state mutations |
| `Relux.Effect` | Enum of side effects |
| `Relux.UI.Container` | Connected SwiftUI component |
| `Relux.UI.View` | Dumb SwiftUI view |
| `ObservableObject` | For SwiftUI @EnvironmentObject injection |
| `Sendable` | For actor isolation |
| `Codable` | For DTOs only |

### Sendable Annotations

- **Actors** (State, Service, Fetcher, Flow, Saga): Thread-safe by default
- **Enums** (Action, Effect, Error): Must conform to `Sendable`
- **Models**: Must conform to `Sendable` for use in actors
- **DTOs**: Should conform to `Sendable`

### Import Patterns

```swift
import SwiftUI              // UI Container files
import Combine             // State files with @Published
import Relux              // All business logic files
import SwiftIoC           // Module.swift for IoC setup
import AuthReluxInt       // When importing other modules
```

### Async/Await Patterns

- `init()` can be `async` for modules with async state setup
- `reduce(with:) async` for state mutations
- `apply(_:) async` for Flow/Saga
- Methods return `async -> Result<T, Error>`
- `await actions { ... }` to dispatch to Relux
- `await actions(.concurrently) { ... }` for parallel actions
- `performAsync { ... }` wrapper in UI Container (convenience)

### State Mutations

- Direct mutation only in `reduce()` / `internalReduce()` methods
- Use `@Published` properties for Combine integration
- Wrap values in `MaybeData<T, E>` for loading states
- Never mutate state outside reducer
- `cleanup()` resets state when module unloads

---

## Module-Specific Variations

### Pattern A: Full Feature Module (Notes)
- Has: Business state, effects, flow, UI state, data layer
- Structure: Namespace → Module → Business (actions/state/reducer/effects/flow/service) → Data (fetcher/DTO) → UI (container/pages)
- Used for: Complex feature with data persistence and UI

### Pattern B: Simple Saga Module (ErrorHandling, SampleApp)
- Has: No state, only saga/effect handler
- Structure: Namespace → Module → Business (effect/saga/service)
- Used for: Cross-cutting concerns, app-level orchestration, error tracking

### Pattern C: State-Only Module (Navigation)
- Has: State (ModalRouter), actions, reducer, no effects
- Structure: Namespace → Module → Business (state/actions/reducer)
- Used for: Pure state machines (routing, modal management)

### Pattern D: Utility Module (Logger)
- Has: Pure functions, no Relux integration (optional)
- Structure: Single file with static methods
- Used for: Logging, debugging, helpers

### Pattern E: Simple UI Module (Account)
- Has: Just UI Container, no business logic
- Structure: Namespace → UI (container/page)
- Used for: Leaf UI that reads from parent module state

---

## Key Architectural Insights

### Dependency Injection (SwiftIoC)

Every module builds its own IoC container:
```swift
static func buildIoC() -> IoC {
    let ioc = IoC(logger: IoC.Logger(enabled: false))
    
    // Register all types
    ioc.register(Notes.Business.State.self, lifecycle: .container, resolver: buildBusinessState)
    ioc.register(Notes.Business.IService.self, resolver: { buildSvc(ioc: ioc) })
    
    return ioc
}
```

- `lifecycle: .container` = singleton per container
- Default (no lifecycle) = new instance per get
- Resolvers can be sync or async
- Dependencies passed via closure parameter

### State Layering

**Business State** (`actor`):
- Immutable data model
- Driven by actions
- @Published for reactivity
- Uses `MaybeData<T, E>` wrapper

**UI State** (`final class ObservableObject`):
- Transforms business state for presentation
- Combines multiple data sources
- Groups/sorts/filters for UI
- Computed properties and helper methods

### Effect → Action → State Flow

```
Effect triggered
    ↓
Flow.apply(effect) processes effect
    ↓
Flow dispatches action(s) via dispatcher
    ↓
State.reduce(action) handles action
    ↓
@Published properties update
    ↓
UI re-renders
```

### Module Integration

Modules can reference other modules:
- Effects from one module can trigger effects from another
- Services can depend on other modules' services
- UI containers can read state from multiple modules

Example: Notes module can dispatch `ErrorHandling.Business.Effect.track(error:)`

---

## Summary Table: File Types & Purposes

| File Type | Responsibility | Count per Module |
|-----------|---------------|----|
| `+Namespace.swift` | Module structure (enum hierarchy) | 1 |
| `+Module.swift` | IoC setup, state/saga registration | 1 |
| `+Business+Action.swift` | Action enumeration | 0-1 |
| `+Business+State.swift` | Business state actor | 0-1 |
| `+Business+State+Reducer.swift` | Action handler | 0-1 |
| `+Business+Effect.swift` | Effect enumeration | 0-1 |
| `+Business+Flow.swift` | Effect processor (returns Result) | 0-1 |
| `+Business+Saga.swift` | Side-effect handler (no Result) | 0-1 |
| `+Business+Service.swift` | Business logic service | 0-N |
| `+Data+Api+Fetcher.swift` | API client | 0-1 |
| `+Data+Api+DTO+*.swift` | Data transfer objects | 0-N |
| `+Business+Model+*.swift` | Domain models | 0-N |
| `+Business+Err.swift` | Error types | 0-1 |
| `+UI+State.swift` | Presentation state | 0-1 |
| `+UI+Page.swift` | Routing page enum | 0-1 |
| `+UI+<Feature>+Container.swift` | Connected component | 0-N |
| `+UI+<Feature>+Container+Page.swift` | Dumb view | 0-N |
| `+UI+Router.swift` | Navigation router | 0-1 |
| `+UI+Components+*.swift` | Reusable UI components | 0-N |

---

## Concrete Examples: File Structure Trees

### Notes Module (Full Feature)
```
Notes/
├── Notes+Namespace.swift                          [Enum hierarchy]
├── Notes+Module.swift                             [IoC + states/sagas]
├── Business/
│   ├── Notes+Business+Action.swift               [Action enum]
│   ├── Notes+Business+State.swift                [State actor + Relux.BusinessState]
│   ├── Notes+Business+State+Reducer.swift        [internalReduce method]
│   ├── Notes+Business+Err.swift                  [Error enum]
│   ├── Middleware/
│   │   ├── Notes+Business+Effect.swift           [Effect enum]
│   │   ├── Notes+Business+Flow.swift             [Flow actor + Relux.Flow]
│   │   ├── Notes+Business+Service.swift          [IService interface + Service impl]
│   │   └── (not present: no Saga, only Flow)
│   └── Model/
│       └── Notes+Business+Model+Note.swift       [Domain model]
├── Data/
│   └── Api/
│       ├── Notes+Data+Api+Fetcher.swift          [IFetcher interface + Fetcher impl]
│       └── DTO/
│           └── Notes+Data+Api+DTO+Note.swift     [DTO struct]
├── Err/
│   └── Notes+Business+Err.swift                  [Domain errors]
├── UI/
│   ├── Notes+UI+State.swift                      [UIState class]
│   ├── Notes+UI+Router.swift                     (not present, no internal routing)
│   ├── Model/
│   │   └── Notes+UI+Page.swift                   [Page enum for routing]
│   ├── List/
│   │   ├── Notes+UI+List+Container.swift         [Container view]
│   │   └── Page/
│   │       └── Notes+UI+ List+Container+Page.swift [Dumb view]
│   ├── Details/
│   │   ├── Notes+UI+Details+Container.swift
│   │   └── Page/
│   │       └── Notes+UI+ Details+Container+Page.swift
│   ├── Create/
│   │   ├── Notes+UI+Create+Container.swift
│   │   └── Page/
│   │       ├── Notes+UI+Create+Container+Page.swift
│   │       └── Notes+UI+Create+Container+Page+LS.swift (Local State?)
│   ├── Edit/
│   │   ├── Notes+UI+Edit+Container.swift
│   │   └── Page/
│   │       ├── Notes+UI+Edit+Container+Page.swift
│   │       └── Notes+UI+Edit+Container+Page+LS.swift
│   ├── Widget/
│   │   ├── Notes+UI+Widget+Container.swift
│   │   └── Page/
│   │       └── Notes+UI+Widget+Container+Page.swift
│   ├── Components/
│   │   └── EditForm/
│   │       ├── Notes+UI+Components+EditForm.swift
│   │       ├── Notes+UI+Components+EditForm+Props.swift
│   │       └── Notes+UI+Components+EditForm+Note.swift
│   └── Utils/
│       └── Collection+Helpers.swift
└── Utils/
    └── Notes+Utils+<helper>.swift
```

### ErrorHandling Module (Simple Saga)
```
ErrorHandling/
├── ErrorHandling+Namespace.swift                 [Enum: enum ErrorHandling { enum Business {} }]
├── ErrorHandling+Module.swift                    [IoC + saga registration]
└── Business/
    └── Middleware/
        ├── ErrorHandling+Business+Effect.swift   [Effect enum: case track(error:)]
        ├── ErrorHandling+Business+Saga.swift     [Saga actor + Relux.Saga]
        └── ErrorHandling+Business+Service.swift  [IService + Service]
```

### Navigation Module (State-Only Router)
```
Navigation/
├── Navigation+Namespace.swift                    [Enum hierarchy]
├── Navigation+Module.swift                       [IoC + router/modalRouter registration]
├── Business/
│   ├── Navigation+Business+ModalRouter.swift     [ModalRouter state + reduce]
│   ├── Navigation+Business+ModalRouter+Action.swift [Action enum]
│   ├── Navigation+Business+ModalRouter+Reducer.swift [internalReduce]
│   └── Models/
│       └── Navigation+Business+Model+ModalPage.swift [Page enum]
└── UI/
    └── Models/
        └── Navigation+UI+Model+AppPage.swift     [App-level routing page enum]
```

---

## Generation Templates

When generating a new Relux module, use these templates:

### 1. Namespace
```swift
enum <ModuleName> {
    enum Data {
        enum Api {
            enum DTO {}
        }
    }
    enum Business {
        enum Model {}
    }
    enum UI {
        enum Model {}
        enum Component {}
    }
}
```

### 2. Module (Full Feature with State)
```swift
import SwiftIoC

extension <ModuleName> {
    struct Module: Relux.Module {
        private let ioc: IoC
        var states: [any Relux.AnyState]
        var sagas: [any Relux.Saga]

        init() async {
            self.ioc = Self.buildIoC()
            self.states = [
                self.ioc.get(by: <ModuleName>.Business.State.self)!,
                await self.ioc.getAsync(by: <ModuleName>.UI.State.self)!
            ]
            self.sagas = [
                await self.ioc.getAsync(by: <ModuleName>.Business.IFlow.self)!
            ]
        }
    }
}

extension <ModuleName>.Module {
    static func buildIoC() -> IoC {
        let ioc: IoC = .init(logger: IoC.Logger(enabled: false))
        ioc.register(<ModuleName>.Business.State.self, lifecycle: .container, resolver: buildBusinessState)
        ioc.register(<ModuleName>.UI.State.self, lifecycle: .container, resolver: { await buildUIState(ioc: ioc) })
        ioc.register(<ModuleName>.Business.IService.self, lifecycle: .container, resolver: { buildSvc(ioc: ioc) })
        ioc.register(<ModuleName>.Data.Api.IFetcher.self, lifecycle: .container, resolver: { buildFetcher(ioc: ioc) })
        ioc.register(<ModuleName>.Business.IFlow.self, lifecycle: .container, resolver: { await buildFlow(ioc: ioc) })
        return ioc
    }

    private static func buildBusinessState() -> <ModuleName>.Business.State {
        <ModuleName>.Business.State()
    }

    private static func buildUIState(ioc: IoC) async -> <ModuleName>.UI.State {
        await <ModuleName>.UI.State(state: ioc.get(by: <ModuleName>.Business.State.self)!)
    }

    private static func buildSvc(ioc: IoC) -> <ModuleName>.Business.IService {
        <ModuleName>.Business.Service(fetcher: ioc.get(by: <ModuleName>.Data.Api.IFetcher.self)!)
    }

    private static func buildFetcher(ioc: IoC) -> <ModuleName>.Data.Api.IFetcher {
        <ModuleName>.Data.Api.Fetcher()
    }

    private static func buildFlow(ioc: IoC) async -> <ModuleName>.Business.IFlow {
        await <ModuleName>.Business.Flow(svc: ioc.get(by: <ModuleName>.Business.IService.self)!)
    }
}
```

### 3. Action Enum
```swift
extension <ModuleName>.Business {
    enum Action: Relux.Action {
        case <verb>Success(<data>)
        case <verb>Fail(err: Err)
    }
}
```

### 4. State Actor
```swift
import Combine

extension <ModuleName>.Business {
    actor State {
        @Published var <property>: MaybeData<<DataType>, Err> = .initial()
    }
}

extension <ModuleName>.Business.State: Relux.BusinessState {
    func reduce(with action: any Relux.Action) async {
        switch action as? <ModuleName>.Business.Action {
            case .none: break
            case let .some(action): await internalReduce(with: action)
        }
    }
    
    func cleanup() async {
        self.<property> = .initial()
    }
}
```

### 5. Reducer
```swift
extension <ModuleName>.Business.State {
    func internalReduce(with action: <ModuleName>.Business.Action) async {
        switch action {
            case let .<verb>Success(<data>):
                self.<property> = .success(<data>)
            case let .<verb>Fail(err):
                self.<property> = .failure(err)
        }
    }
}
```

### 6. Effect Enum
```swift
extension <ModuleName>.Business {
    enum Effect: Relux.Effect {
        case <action>
        case <actionWithData>(data: <Type>)
    }
}
```

### 7. Flow Actor
```swift
extension <ModuleName>.Business {
    protocol IFlow: Relux.Flow {}
}

extension <ModuleName>.Business {
    actor Flow {
        private typealias Model = <ModuleName>.Business.Model
        let dispatcher: Relux.Dispatcher
        private let svc: <ModuleName>.Business.IService

        init(dispatcher: Relux.Dispatcher? = .none, svc: <ModuleName>.Business.IService) async {
            let defaultDispatcher = await Self.defaultDispatcher
            self.dispatcher = dispatcher ?? defaultDispatcher
            self.svc = svc
        }
    }
}

extension <ModuleName>.Business.Flow: <ModuleName>.Business.IFlow {
    func apply(_ effect: any Relux.Effect) async -> Relux.Flow.Result {
        switch effect as? <ModuleName>.Business.Effect {
            case .none: .success
            case .<action>: await <action>()
        }
    }
}

extension <ModuleName>.Business.Flow {
    private func <action>() async -> Relux.Flow.Result {
        switch await svc.<method>() {
            case let .success(<data>):
                await actions {
                    <ModuleName>.Business.Action.<verb>Success(<data>)
                }
                return .success
            case let .failure(err):
                await actions(.concurrently) {
                    <ModuleName>.Business.Action.<verb>Fail(err: err)
                    ErrorHandling.Business.Effect.track(error: err)
                }
                return .success
        }
    }
}
```

---

## Authoritative Architecture (from "UDF with Relux" presentation)

Source: AppsConf X 2025, Alexey Grigorev (MTS Web Services). This section captures the canonical definitions and design rationale from the framework author.

### Relux vs Redux: Key Philosophical Differences

| Concept | Redux | Relux |
|---------|-------|-------|
| **Store** | Centralized state storage + reducing | ONLY connecting and storing states (no reducing) |
| **State** | Immutable | **Mutable** (actor-isolated) |
| **Reducer** | Pure function (external to state) | **Part of State**, mutates its own state |
| **Dispatcher** | Part of Store | **Separate entity** — event bus |
| **Middleware** | Thunk, Saga, Logger... | Effect, Saga/Flow, Logger |

### Canonical UDF Data Flow

```
User → View ← props ← Container ← data ← State ← newState ← Reducer ← action ← Store ← action ← Dispatcher
                         │                                                                            ↑
                         └──── dispatches action/effect ────────────────────────────────────────────────┘
                                                                                                       ↑
Middleware layer: Saga/Flow → calls SOA → gets data → dispatches actions/effects ──────────────────────┘
                                                                                                       ↑
Logger ← receives all messages from ────────────────────────────────────────────────────────────────────┘
```

### State Types Hierarchy

```
Store → AnyState → UIState        (ObservableObject, final class, @MainActor)
                 → BusinessState  (actor, @Published properties)
                 → HybridState    (for cases where both UI + business in one)
```

- **BusinessState**: actor, thread-safe, owns data, reduces actions
- **UIState**: ObservableObject, transforms business state via Combine pipelines, receives on main queue
- **HybridState**: combines both — can be connected to Store with View lifecycle binding

### Flow vs Saga Distinction

- **Saga** (`Relux.Saga`): `apply(_ effect:) async` — no return value, fire-and-forget side effects. Used for app-level orchestration (e.g., cleanup on logout, cross-module flows)
- **Flow** (`Relux.Flow`): `apply(_ effect:) async -> Relux.Flow.Result` — returns `.success` or `.failure(Error)`. Successor of Saga, preferred for feature modules

### Three Modularity Ideas (Complementary)

1. **Namespaces** (enum-based) — readability, type hierarchy
2. **Logical modules** (`Relux.Module`) — runtime boundaries, state/saga grouping
3. **Physical modules** (SPM packages) — compilation boundaries, access control

### Relux's Role in the Project

- Stores **entire app state** centrally
- Helps establish **module boundaries**
- Serves **business logic** (through Flows/Sagas + SOA)
- **Reflects all events** flowing through dispatcher (logging)
- Through sagas interacts with **SOA layer** (service oriented architecture)
- Through containers interacts with **UI layer**

### Testing Pattern (from presentation)

Flow accepts optional dispatcher for testability:

```swift
actor Flow {
    let dispatcher: Relux.Dispatcher
    private let svc: Notes.Business.IService

    init(dispatcher: Relux.Dispatcher? = .none, svc: Notes.Business.IService) async {
        let defaultDispatcher = await Self.defaultDispatcher
        self.dispatcher = dispatcher ?? defaultDispatcher
        self.svc = svc
    }
}
```

Test structure (Swift Testing framework):

```swift
@Suite("Notes") struct NotesTests {
    @Suite("Business") struct Business {
        @Suite("Flow") struct FlowTests {
            @Test func obtainNotes_Failure() async throws {
                // Arrange
                let logger = Relux.Testing.Logger()
                let dispatcher = Relux.Dispatcher(logger: logger)
                let service = NotesTests.Business.ServiceMock()
                let flow = await Notes.Business.Flow(dispatcher: dispatcher, svc: service)

                let err: Err = .obtainFailed(cause: StubErr())
                service.obtainNotesHandler = { .failure(err) }

                // Act
                _ = await flow.apply(Effect.obtainNotes)

                // Assert
                let failureAction = logger.getAction(Action.obtainNotesFail(err: err))
                #expect(failureAction.isNotNil)
                let trackEffect = logger.getEffect(ErrorHandling.Business.Effect.track(error: err))
                #expect(trackEffect.isNotNil)
            }
        }
    }
}
```

Key testing insights:
- `@Suite` hierarchy mirrors namespace hierarchy
- `Relux.Testing.Logger` captures dispatched actions/effects
- `logger.getAction()` / `logger.getEffect()` for assertions
- Mock services with closure-based handlers
- No need to mock entire Relux store — just dispatcher + service

### Known Advantages (author's assessment)

1. Built-in logging (all events through dispatcher)
2. Centralized state storage and management
3. Clear boundaries: business ↔ service logic
4. Clear boundaries: business ↔ view logic
5. Modularity (three complementary ideas)
6. Swift 6 compatibility (actors + Sendable)
7. Testability (DI, mock dispatcher, mock services)

### Known Disadvantages & Mitigations

| Disadvantage | Mitigation |
|-------------|------------|
| Needs async IoC | Already written, Swift 6 compatible |
| Connecting Business and UI States is manual (Combine pipelines) | HybridState for most cases |
| Can't guarantee UIState updated after Action completes (structured concurrency) | Relux.Flow returns Result |
| Centralized state where private state would be better | HybridState + View lifecycle binding |

### Compatibility Notes (from presentation)

- Compatible with other architectures (can coexist with MV*, VIPER, etc.)
- Easy to add to existing project incrementally
- Used in: fully Relux projects, MV→Relux migrations, KMP core + platform features on Relux
- Juniors handle feature implementation without issues (structured patterns)
- GPTs/LLMs understand the structured code well and make fewer mistakes

---

## Final Notes

This Relux architecture is:
- **Type-safe**: Heavy use of protocols and enums
- **Testable**: Clear separation of concerns, dependency injection
- **Reusable**: Modules can reference and integrate with each other
- **Debuggable**: Action/effect dispatch is logged, state is immutable from outside
- **Scalable**: Pattern works from simple utilities to complex features
- **Concurrent**: Actors ensure thread safety, async/await for natural async code

The key insight: **Namespace-based file organization + Protocol-driven architecture + Relux state machine = Highly maintainable, predictable UI applications.**
