package function

import (
	lru "github.com/hashicorp/golang-lru"
	"gopkg.in/src-d/go-mysql-server.v0/sql"
)

// Functions for gitbase queries.
var Functions sql.Functions

func init() {
	langCache, err := lru.New(10000)
	if err != nil {
		panic(err)
	}

	Functions = sql.Functions{
		"is_tag":     sql.Function1(NewIsTag),
		"is_remote":  sql.Function1(NewIsRemote),
		"language":   sql.FunctionN(NewLanguage(langCache)),
		"uast":       sql.FunctionN(NewUAST),
		"uast_xpath": sql.Function2(NewUASTXPath),
	}
}
