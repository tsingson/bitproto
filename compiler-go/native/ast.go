package native

type Proto struct {
	Name     string
	Aliases  []Alias
	Enums    []Enum
	Consts   []Const
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

type Const struct {
	Name  string
	Value string
	Type  string
}

type Message struct {
	Name       string
	Extensible bool
	Fields     []MessageField
}

type MessageField struct {
	Type   TypeExpr
	Name   string
	Number int
}

type TypeExpr struct {
	Name       string
	ArraySize  int
	Extensible bool // true when array type is marked extensible with '
}

func (t TypeExpr) IsArray() bool {
	return t.ArraySize > 0
}
