import Notes
import Relux

extension Notes.Business {
    public actor Flow: Relux.Flow {
        private let state: Notes.Business.State

        public init(state: Notes.Business.State) {
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
