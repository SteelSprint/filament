package driftpin

type Scanner interface {
	Scan() (Scan, error)
}

type FileScanner struct {
	dir string
}

func NewFileScanner(dir string) *FileScanner {
	return &FileScanner{dir: dir}
}

func (s *FileScanner) Scan() (Scan, error) {
	return Scan{
		SpecHashes:   map[string]string{},
		MarkerHashes: map[string]string{},
	}, nil
}
