package function

import (
	"fmt"
	"strings"

	"gopkg.in/src-d/enry.v1"

	"github.com/alecthomas/chroma"

	"github.com/alecthomas/chroma/lexers"
	"gopkg.in/src-d/go-mysql-server.v0/sql"
)

type GetTokens struct {
	TokenName   sql.Expression
	FilePath    sql.Expression
	FileContent sql.Expression
}

func NewGetTokens(token, filePath, content sql.Expression) sql.Expression {
	return &GetTokens{
		TokenName:   token,
		FilePath:    filePath,
		FileContent: content,
	}
}

// Children implements the Expression interface.
func (f *GetTokens) Children() []sql.Expression {
	return []sql.Expression{f.TokenName, f.FilePath, f.FileContent}
}

// IsNullable implements the Expression interface.
func (f *GetTokens) IsNullable() bool {
	return true
}

// Resolved implements the Expression interface.
func (f *GetTokens) Resolved() bool {
	return f.TokenName.Resolved() && f.FilePath.Resolved() && f.FileContent.Resolved()
}

// TransformUp implements the Expression interface.
func (f *GetTokens) TransformUp(fn sql.TransformExprFunc) (sql.Expression, error) {
	tokenName, err := f.TokenName.TransformUp(fn)
	if err != nil {
		return nil, err
	}

	filePath, err := f.FilePath.TransformUp(fn)
	if err != nil {
		return nil, err
	}

	fileContent, err := f.FileContent.TransformUp(fn)
	if err != nil {
		return nil, err
	}

	return fn(&GetTokens{
		TokenName:   tokenName,
		FilePath:    filePath,
		FileContent: fileContent,
	})
}

func (f *GetTokens) String() string {
	return fmt.Sprintf("get_tokens(%s, %s, %s)", f.TokenName, f.FilePath, f.FileContent)
}

// Type implements the Expression interface.
func (*GetTokens) Type() sql.Type {
	return sql.Array(sql.Text)
}

// Eval implements the Expression interface.
func (f *GetTokens) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	filePathExpr, err := f.FilePath.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	if filePathExpr == nil {
		return nil, nil
	}

	filePath, err := sql.Text.Convert(filePathExpr)
	if err != nil {
		return nil, err
	}

	contentExpr, err := f.FileContent.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	if contentExpr == nil {
		return nil, nil
	}

	content, err := sql.Blob.Convert(contentExpr)
	if err != nil {
		return nil, err
	}

	tokenExpr, err := f.TokenName.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	if tokenExpr == nil {
		return nil, nil
	}

	token, err := sql.Text.Convert(tokenExpr)
	if err != nil {
		return nil, err
	}

	path := filePath.(string)
	blob := content.([]byte)
	tokenString := token.(string)
	if enry.IsBinary(blob) {
		return nil, nil
	}

	if len(blob) == 0 {
		return nil, nil
	}

	var lexer chroma.Lexer

	lexer = lexers.Match(path)

	if lexer == nil {
		l := 1024
		if len(blob) < l {
			l = len(blob) - 1
		}

		lexer = lexers.Analyse(string(blob[0:l]))
	}

	if lexer == nil {
		return nil, nil
	}

	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, string(blob))
	if err != nil {
		return nil, err
	}

	var out []interface{}
	for t := iterator(); t != chroma.EOF; t = iterator() {
		if strings.ToLower(t.Type.String()) == strings.ToLower(tokenString) {
			out = append(out, t.String())
		}
	}

	if len(out) == 0 {
		return nil, nil
	}

	return out, nil
}
