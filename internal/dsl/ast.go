package dsl

type Pipeline struct {
	Commands []Command
	Redirect *Redirect
}

type Command struct {
	Name string
	Args []string
}

type Redirect struct {
	Path string
}
