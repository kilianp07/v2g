package formatter

type Formatter interface {
	Format(entry map[string]any) ([]byte, error)
}
