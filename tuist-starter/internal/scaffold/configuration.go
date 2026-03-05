package scaffold

// GenerateConfiguration returns the root Configuration namespace file.
func GenerateConfiguration() string {
	return `/// Shared configuration namespace.
/// Setup commands add extensions here (e.g. Configuration+HttpClient).
enum Configuration {}
`
}
