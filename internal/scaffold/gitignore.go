package scaffold

// GenerateGitignore returns standard ignore rules for Tuist/iOS projects.
func GenerateGitignore() string {
	return `# macOS
.DS_Store

# Xcode
*.xcodeproj
*.xcworkspace
xcuserdata/
DerivedData/
build/

# Tuist
Derived/
`
}
