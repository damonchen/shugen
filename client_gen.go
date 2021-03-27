package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/damonchen/x/camelcase"
	"github.com/dave/dst"
)

// Client client
type Client struct {
	Name  string
	funcs []Func
}

// Func func
type Func struct {
	Name   string
	params []Param
	// TODO: 此处固化了一些
	Resp string
}

// Param param
type Param struct {
	Name string
}

// Return return
type Return struct {
	rets []Param
}

type Package struct {
	name    string
	clients []Client
}

func extractClients(f *dst.File) (*Package, error) {
	packageName := f.Name.Name

	pkg := Package{
		name: packageName,
	}
	for _, decl := range f.Decls {
		switch decl.(type) {
		case *dst.GenDecl:
		case *dst.FuncDecl:
			funcDecl := decl.(*dst.FuncDecl)
			if funcDecl.Name.Name == "init" {
				if funcDecl.Body == nil {
					return nil, errors.New("not found init function")
				}

				clients, err := extractInitFunc(f, funcDecl.Body.List)
				if err != nil {
					return nil, err
				}
				pkg.clients = clients
				return &pkg, nil
			}
		}
	}
	return nil, errors.New("not support file content")
}

func extractInitFunc(f *dst.File, stmts []dst.Stmt) ([]Client, error) {
	for _, stmt := range stmts {
		exprStmt, ok := stmt.(*dst.ExprStmt)
		if !ok {
			return nil, errors.New("init function has no stmt")
		}

		if exprStmt.X == nil {
			continue
		}

		callExpr, ok := exprStmt.X.(*dst.CallExpr)
		if !ok {
			continue
		}

		selectorExpr := callExpr.Fun.(*dst.SelectorExpr)
		funcIdent := selectorExpr.X.(*dst.Ident)

		name := fmt.Sprintf("%s.%s", funcIdent.Name, selectorExpr.Sel.Name)
		if name == "bundle.Client" {
			return extractBundleClientParams(f, callExpr)
		}

	}
	return nil, errors.New("not found client definition")
}

func stripStringQuote(s string) string {
	return strings.TrimSuffix(strings.TrimPrefix(s, "\""), "\"")
}

func extractBundleClientParams(f *dst.File, callExpr *dst.CallExpr) ([]Client, error) {
	var clients []Client
	for _, arg := range callExpr.Args {
		compositeLit := arg.(*dst.CompositeLit)

		client := Client{}

		for _, ele := range compositeLit.Elts {
			kv := ele.(*dst.KeyValueExpr)
			ident := kv.Key.(*dst.Ident)

			switch ident.Name {
			case "Name":
				bl := kv.Value.(*dst.BasicLit)
				client.Name = stripStringQuote(bl.Value)
			case "APIs":
				cl := kv.Value.(*dst.CompositeLit)
				client.funcs = extractAPIs(cl)
			default:
				return nil, errors.New(fmt.Sprintf("unknown ident name %s", ident.Name))
			}
		}

		clients = append(clients, client)
	}
	return clients, nil
}

func extractAPIs(cl *dst.CompositeLit) []Func {
	fns := make([]Func, len(cl.Elts))
	j := 0
	for _, cele := range cl.Elts {
		tt := cele.(*dst.CompositeLit)
		fn := Func{}
		foundFn := false
		for _, ele := range tt.Elts {
			kv := ele.(*dst.KeyValueExpr)
			if kv == nil {
				continue
			}

			ident := kv.Key.(*dst.Ident)
			switch ident.Name {
			case "Name":
				name := kv.Value.(*dst.BasicLit).Value
				name = stripStringQuote(name)
				fn.Name = name
				foundFn = true
			case "Path":
			case "Method":
			case "Params":
				params := kv.Value.(*dst.CompositeLit)
				paramType := params.Type.(*dst.Ident)
				fn.params = append(fn.params, Param{Name: paramType.Name})
				foundFn = true
			case "Response":
				params := kv.Value.(*dst.CompositeLit)
				fn.Resp = params.Type.(*dst.Ident).Name
				foundFn = true
			}
		}
		if foundFn {
			fns[j] = fn
			j++
		}
	}
	return fns
}

func generateFuncParams(params []Param) (string, string) {
	ps := make([]string, len(params))
	pv := make([]string, len(params))
	for i, param := range params {
		paramValue := camelcase.CamelCase(param.Name)
		ps[i] = paramValue + " " + param.Name
		pv[i] = paramValue
	}

	return strings.Join(ps, ", "), strings.Join(pv, ",")
}

func generateFunc(name string, fn Func) string {
	clientName := camelcase.PascalCase(name) + "Client"
	funcName := camelcase.PascalCase(fn.Name)
	params, paramValues := generateFuncParams(fn.params)
	resp := fn.Resp

	return fmt.Sprintf(`func (c %s) %s(%s) (*%s, error) {
	resp, err := bundle.Get("%s").Call("%s", %s)
	if err != nil {
		return nil, err
	}
	return resp.(*%s), nil
}
`, clientName, funcName, params, resp, name, fn.Name, paramValues, resp)
}

func generateFuncs(client Client, funcs []Func) string {
	r := make([]string, len(funcs))
	for i, fn := range funcs {
		fns := generateFunc(client.Name, fn)

		r[i] = fns
	}
	return strings.Join(r, "\n")
}

func generateClient(client Client) string {
	clientName := camelcase.PascalCase(client.Name) + "Client"
	clientStruct := fmt.Sprintf(`type %s struct {}

`, clientName)

	newClient := fmt.Sprintf(`func Get%s() %s {
	return %s{}
}
`, clientName, clientName, clientName)

	funcs := generateFuncs(client, client.funcs)

	return clientStruct + funcs + "\n" + newClient
}

func generatePkg(pkg *Package) string {
	pkgName := fmt.Sprintf("package %s\n\n", pkg.name)

	cs := make([]string, len(pkg.clients))
	for i, client := range pkg.clients {
		clientStr := generateClient(client)
		cs[i] = clientStr
	}
	return pkgName + strings.Join(cs, "\n")
}
