package backend

type Backend interface {
	GetValues(map[string]string) (map[string]string, error)
}
