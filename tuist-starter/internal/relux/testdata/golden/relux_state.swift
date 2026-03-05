import Notes
import Relux

extension Notes.Business {
    @Observable
    public final class State: Relux.HybridState {
        public init() {}

        public func reduce(with action: any Relux.Action) async {
            guard let action = action as? Notes.Business.Action else { return }
            internalReduce(with: action)
        }

        public func cleanup() async {}

        private func internalReduce(with action: Notes.Business.Action) {
            switch action {}
        }
    }
}
