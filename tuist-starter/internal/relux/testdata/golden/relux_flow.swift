import Notes
import Relux

extension Notes.Business {
    public actor Flow: Relux.Flow {
        public let dispatcher: Relux.Dispatcher
        private let state: Notes.Business.State

        public init(
            dispatcher: Relux.Dispatcher? = .none,
            state: Notes.Business.State
        ) async {
            let defaultDispatcher = await Self.defaultDispatcher
            self.dispatcher = dispatcher ?? defaultDispatcher
            self.state = state
        }

        public func apply(_ effect: any Relux.Effect) async -> Relux.ActionResult {
            guard let effect = effect as? Notes.Business.Effect else { return .success }
            return await internalApply(effect)
        }

        private func internalApply(_ effect: Notes.Business.Effect) async -> Relux.ActionResult {
            switch effect {}
        }
    }
}
