package bot

type Bot interface {
}

type bot struct {
}

func New() Bot {
	

	return &bot{}
}
