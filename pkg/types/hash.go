package types

type Hash string

func (h Hash) String() string {
	return string(h)
}

func (h Hash) IsZero() bool {
	return h == ""
}
