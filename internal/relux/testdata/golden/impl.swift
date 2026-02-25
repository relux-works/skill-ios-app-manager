import Notes

extension Notes.Module {
    public struct Impl: Notes.Module.Interface {
        public init() {}

        public func register() {
            print("Notes module registered")
        }
    }
}
