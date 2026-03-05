import Observation
import SwiftUI

public struct NotesView: View {
    @Bindable private var store: NotesStore

    public init(store: NotesStore) {
        self.store = store
    }

    public var body: some View {
        content
            .navigationTitle("Notes")
            .task {
                store.dispatch(.appeared)
            }
    }

    @ViewBuilder
    private var content: some View {
        if store.state.isLoading {
            ProgressView("Loading Notes...")
        } else if let errorMessage = store.state.errorMessage {
            VStack(spacing: 8) {
                Text("Failed to load")
                    .font(.headline)
                Text(errorMessage)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
        } else if store.state.items.isEmpty {
            Text("No notes items")
                .foregroundStyle(.secondary)
        } else {
            List(store.state.items) { item in
                Text(item.title)
            }
        }
    }
}
