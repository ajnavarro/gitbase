package plan

import (
	"io"
	"testing"

	"github.com/mvader/gitql/mem"
	"github.com/mvader/gitql/sql"
	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	assert := assert.New(t)
	childSchema := sql.Schema{
		sql.Field{"col1", sql.String},
		sql.Field{"col2", sql.Integer},
	}
	child := mem.NewTable("test", childSchema)
	child.Insert("a", int32(3))
	child.Insert("b", int32(3))
	child.Insert("c", int32(1))
	sf := []SortField{
		{Column: "col2", Order: Ascending},
		{Column: "col1", Order: Descending},
	}
	s := NewSort(sf, child)
	assert.Equal(childSchema, s.Schema())
	iter, err := s.RowIter()
	assert.Nil(err)
	assert.NotNil(iter)
	row, err := iter.Next()
	assert.Nil(err)
	assert.NotNil(row)
	assert.Equal("c", row.Fields()[0])
	row, err = iter.Next()
	assert.Nil(err)
	assert.NotNil(row)
	assert.Equal("a", row.Fields()[0])
	row, err = iter.Next()
	assert.Nil(err)
	assert.NotNil(row)
	assert.Equal("b", row.Fields()[0])
	row, err = iter.Next()
	assert.Equal(io.EOF, err)
	assert.Nil(row)
}
