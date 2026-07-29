package native

type Proto struct {
	Name     string
	Aliases  []Alias
	Enums    []Enum
	Messages []Message
}

type Alias struct {
	Name string
	Type TypeExpr
}

type Enum struct {
	Name   string
	Base   string
	Fields []EnumField
}

type EnumField struct {
	Name  string
	Value int
}

type Message struct {
	Name   string
	Fields []MessageField
}

type MessageField struct {
	Type   TypeExpr
	Name   string
	Number int
}

type TypeExpr struct {
	Name      string
	ArraySize int
}

func (t TypeExpr) IsArray() bool {
	return t.ArraySize > 0
}
