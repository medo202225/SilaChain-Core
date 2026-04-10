package logging

type Field map[string]any

func MergeFields(base Field, extra Field) Field {
	out := Field{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
