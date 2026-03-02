package scaffold

// GenerateGitignore returns standard ignore rules for Tuist/iOS projects.
func GenerateGitignore() string {
	return `# macOS
.DS_Store

# Xcode
*.xcodeproj
*.xcworkspace
xcuserdata/
*.xcuserstate
DerivedData/
build/

# Tuist
Derived/

# Swift Package Manager
.build/
.swiftpm/

# CocoaPods (if used)
Pods/

# IDE
.idea/
*.swp

# Project tooling
.task-board/
task-board.config.json
`
}
