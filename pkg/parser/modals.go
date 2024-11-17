package parser

type Method struct {
	Name         string
	HTTPMethod   string
	HandlerName  string
	URLPath      string
	Params       []Param
	Returns      []Return
	QueryParams  []string
	ResponseType string
	RequestType  string
	HasContext   bool
}

type Param struct {
	Name      string
	Type      string
	Package   string
	IsPointer bool
	IsElipse  bool
}

type Return struct {
	Type      string
	Package   string
	IsPointer bool
	IsElipse  bool
}
