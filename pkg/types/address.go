package types

type Address string

func (a Address) String() string {
	return string(a)
}

func (a Address) IsZero() bool {
	return a == ""
}
