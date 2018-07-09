package server

import (
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"

	errors "gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-mysql-server.v0"
	"gopkg.in/src-d/go-mysql-server.v0/sql"

	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-vitess.v0/mysql"
	"gopkg.in/src-d/go-vitess.v0/sqltypes"
	"gopkg.in/src-d/go-vitess.v0/vt/proto/query"
)

var regKillCmd = regexp.MustCompile(`^kill (?:(query|connection) )?(\d+)$`)

var errConnectionNotFound = errors.NewKind("Connection not found: %c")

// TODO parametrize
const rowsBatch = 100

// Handler is a connection handler for a SQLe engine.
type Handler struct {
	mu sync.Mutex
	e  *sqle.Engine
	sm *SessionManager
	c  map[uint32]*mysql.Conn
}

// NewHandler creates a new Handler given a SQLe engine.
func NewHandler(e *sqle.Engine, sm *SessionManager) *Handler {
	return &Handler{
		e:  e,
		sm: sm,
		c:  make(map[uint32]*mysql.Conn),
	}
}

// NewConnection reports that a new connection has been established.
func (h *Handler) NewConnection(c *mysql.Conn) {
	h.mu.Lock()
	if _, ok := h.c[c.ConnectionID]; !ok {
		h.c[c.ConnectionID] = c
	}
	h.mu.Unlock()

	h.sm.NewSession(c)
	logrus.Infof("NewConnection: client %v", c.ConnectionID)
}

// ConnectionClosed reports that a connection has been closed.
func (h *Handler) ConnectionClosed(c *mysql.Conn) {
	h.sm.CloseConn(c)

	h.mu.Lock()
	delete(h.c, c.ConnectionID)
	h.mu.Unlock()

	logrus.Infof("ConnectionClosed: client %v", c.ConnectionID)
}

// ComQuery executes a SQL query on the SQLe engine.
func (h *Handler) ComQuery(
	c *mysql.Conn,
	query string,
	callback func(*sqltypes.Result) error,
) error {
	println(query)
	if query == "/* mysql-connector-java-8.0.11 (Revision: 6d4eaa273bc181b4cf1c8ad0821a2227f116fedf) */SELECT  @@session.auto_increment_increment AS auto_increment_increment, @@character_set_client AS character_set_client, @@character_set_connection AS character_set_connection, @@character_set_results AS character_set_results, @@character_set_server AS character_set_server, @@collation_server AS collation_server, @@init_connect AS init_connect, @@interactive_timeout AS interactive_timeout, @@license AS license, @@lower_case_table_names AS lower_case_table_names, @@max_allowed_packet AS max_allowed_packet, @@net_write_timeout AS net_write_timeout, @@query_cache_size AS query_cache_size, @@query_cache_type AS query_cache_type, @@sql_mode AS sql_mode, @@system_time_zone AS system_time_zone, @@time_zone AS time_zone, @@tx_isolation AS transaction_isolation, @@wait_timeout AS wait_timeout" {
		schema := sql.Schema{
			&sql.Column{
				Name: "auto_increment_increment",
				Type: sql.Int32,
			},
			&sql.Column{
				Name: "character_set_client",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "character_set_connection",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "character_set_results",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "character_set_server",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "collation_server",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "init_connect",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "interactive_timeout",
				Type: sql.Int32,
			},
			&sql.Column{
				Name: "license",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "lower_case_table_names",
				Type: sql.Int32,
			},
			&sql.Column{
				Name: "max_allowed_packet",
				Type: sql.Int32,
			},
			&sql.Column{
				Name: "net_write_timeout",
				Type: sql.Int32,
			},
			&sql.Column{
				Name: "query_cache_size",
				Type: sql.Int32,
			},
			&sql.Column{
				Name: "query_cache_type",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "sql_mode",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "system_time_zone",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "time_zone",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "transaction_isolation",
				Type: sql.Text,
			},
			&sql.Column{
				Name: "wait_timeout",
				Type: sql.Int32,
			},
		}
		r := &sqltypes.Result{Fields: schemaToFields(schema)}
		r.Rows = append(r.Rows, rowToSQL(schema, sql.NewRow(
			1,
			"utf8",
			"utf8",
			"utf8",
			"latin1",
			"latin1_swedish_ci",
			"",
			28800,
			"GPL",
			0,
			419430400,
			60,
			1048576,
			"OFF",
			"ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION",
			"UTC",
			"SYSTEM",
			"REPEATABLE-READ",
			28800)))
		r.RowsAffected++
		callback(r)

		return nil

	}
	if query == "SET character_set_results = NULL" ||
		query == "SET autocommit=1" {
		println("ENTERING", query)
		r := &sqltypes.Result{}
		callback(r)
		return nil
	}

	ctx, done, err := h.sm.NewContext(c)
	if err != nil {
		return err
	}

	defer done()

	handled, err := h.handleKill(query)
	if err != nil {
		return err
	}

	if handled {
		return nil
	}

	schema, rows, err := h.e.Query(ctx, query)
	if err != nil {
		return err
	}

	var r *sqltypes.Result
	var proccesedAtLeastOneBatch bool
	for {
		if r == nil {
			r = &sqltypes.Result{Fields: schemaToFields(schema)}
		}

		if r.RowsAffected == rowsBatch {
			if err := callback(r); err != nil {
				return err
			}

			r = nil
			proccesedAtLeastOneBatch = true

			continue
		}

		row, err := rows.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		r.Rows = append(r.Rows, rowToSQL(schema, row))
		r.RowsAffected++
	}

	// Even if r.RowsAffected = 0, the callback must be
	// called to update the state in the go-vitess' listener
	// and avoid returning errors when the query doesn't
	// produce any results.
	if r != nil && (r.RowsAffected == 0 && proccesedAtLeastOneBatch) {
		return nil
	}

	return callback(r)
}

func (h *Handler) handleKill(query string) (bool, error) {
	q := strings.ToLower(query)
	s := regKillCmd.FindStringSubmatch(q)
	if s == nil {
		return false, nil
	}

	id, err := strconv.Atoi(s[2])
	if err != nil {
		return false, err
	}

	logrus.Infof("handleKill: id %v", id)

	h.mu.Lock()
	c, ok := h.c[uint32(id)]
	h.mu.Unlock()
	if !ok {
		return false, errConnectionNotFound.New(id)
	}

	h.sm.CloseConn(c)

	// KILL CONNECTION and KILL should close the connection. KILL QUERY only
	// cancels the query.
	//
	// https://dev.mysql.com/doc/refman/5.7/en/kill.html

	if s[1] != "query" {
		c.Close()

		h.mu.Lock()
		delete(h.c, uint32(id))
		h.mu.Unlock()
	}

	return true, nil
}

func rowToSQL(s sql.Schema, row sql.Row) []sqltypes.Value {
	o := make([]sqltypes.Value, len(row))
	for i, v := range row {
		o[i] = s[i].Type.SQL(v)
	}

	return o
}

func schemaToFields(s sql.Schema) []*query.Field {
	fields := make([]*query.Field, len(s))
	for i, c := range s {
		fields[i] = &query.Field{
			Name: c.Name,
			Type: c.Type.Type(),
		}
	}

	return fields
}
