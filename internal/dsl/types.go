package dsl

type Operation string

const (
	OpDSLCompile Operation = "dsl.compile"
)

type Request struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId,omitempty"`
	Op        Operation `json:"op"`
	DSL       string    `json:"dsl,omitempty"`
}

type Response struct {
	ID     string         `json:"id"`
	OK     bool           `json:"ok"`
	Result *CompileResult `json:"result,omitempty"`
	Error  *Error         `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CompileResult struct {
	Plan Plan `json:"plan"`
}

type Plan struct {
	Steps []Step `json:"steps"`
}

type Step struct {
	Tool       string            `json:"tool"`
	InputPath  string            `json:"inputPath,omitempty"`
	OutputPath string            `json:"outputPath"`
	Params     map[string]string `json:"params,omitempty"`
}
