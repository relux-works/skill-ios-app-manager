import Notes
import Relux

extension Notes.Module {
    public struct Impl: Notes.Module.Interface {
        public let states: [any Relux.AnyState]
        public let sagas: [any Relux.Saga]

        @MainActor
        public init() async {
            let state = Notes.Business.State()
            let flow = await Notes.Business.Flow(state: state)
            self.states = [state]
            self.sagas = [flow]
        }
    }
}
