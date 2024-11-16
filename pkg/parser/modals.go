package parser

type Param struct {
	Name string
	Type string
}

type Return struct {
	Type string
}

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
}
