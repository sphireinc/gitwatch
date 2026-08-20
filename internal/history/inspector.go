package history

type Inspector struct {
	Commit  Commit
	Files   []string
	Diff    string
	Loading bool
	Error   error
}

func (i Inspector) Summary() string {
	return i.Commit.Short + " · " + i.Commit.Author + " · " + i.Commit.Subject
}
