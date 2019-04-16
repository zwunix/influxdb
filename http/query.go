package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/influxdata/flux"
	"github.com/influxdata/flux/ast"
	"github.com/influxdata/flux/csv"
	"github.com/influxdata/flux/lang"
	"github.com/influxdata/flux/parser"
	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/query"
	"github.com/influxdata/influxql"
)

// QueryRequest is a flux query request.
type QueryRequest struct {
	Extern  *ast.File    `json:"extern,omitempty"`
	Spec    *flux.Spec   `json:"spec,omitempty"`
	AST     *ast.Package `json:"ast,omitempty"`
	Query   string       `json:"query"`
	Type    string       `json:"type"`
	Dialect QueryDialect `json:"dialect"`

	Org *influxdb.Organization `json:"-"`
}

// QueryDialect is the formatting options for the query response.
type QueryDialect struct {
	Header         *bool    `json:"header"`
	Delimiter      string   `json:"delimiter"`
	CommentPrefix  string   `json:"commentPrefix"`
	DateTimeFormat string   `json:"dateTimeFormat"`
	Annotations    []string `json:"annotations"`
}

// WithDefaults adds default values to the request.
func (r QueryRequest) WithDefaults() QueryRequest {
	if r.Type == "" {
		r.Type = "flux"
	}
	if r.Dialect.Delimiter == "" {
		r.Dialect.Delimiter = ","
	}
	if r.Dialect.DateTimeFormat == "" {
		r.Dialect.DateTimeFormat = "RFC3339"
	}
	if r.Dialect.Header == nil {
		header := true
		r.Dialect.Header = &header
	}
	return r
}

// Validate checks the query request and returns an error if the request is invalid.
func (r QueryRequest) Validate() error {
	if r.Query == "" && r.Spec == nil && r.AST == nil {
		return errors.New(`request body requires either query, spec, or AST`)
	}

	if r.Spec != nil && r.Extern != nil {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "request body cannot specify both a spec and external declarations",
		}
	}

	if r.Type != "flux" {
		return fmt.Errorf(`unknown query type: %s`, r.Type)
	}

	if len(r.Dialect.CommentPrefix) > 1 {
		return fmt.Errorf("invalid dialect comment prefix: must be length 0 or 1")
	}

	if len(r.Dialect.Delimiter) != 1 {
		return fmt.Errorf("invalid dialect delimeter: must be length 1")
	}

	rune, size := utf8.DecodeRuneInString(r.Dialect.Delimiter)
	if rune == utf8.RuneError && size == 1 {
		return fmt.Errorf("invalid dialect delimeter character")
	}

	for _, a := range r.Dialect.Annotations {
		switch a {
		case "group", "datatype", "default":
		default:
			return fmt.Errorf(`unknown dialect annotation type: %s`, a)
		}
	}

	switch r.Dialect.DateTimeFormat {
	case "RFC3339", "RFC3339Nano":
	default:
		return fmt.Errorf(`unknown dialect date time format: %s`, r.Dialect.DateTimeFormat)
	}

	return nil
}

// QueryAnalysis is a structured response of errors.
type QueryAnalysis struct {
	Errors []queryParseError `json:"errors"`
}

type queryParseError struct {
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Character int    `json:"character"`
	Message   string `json:"message"`
}

// Analyze attempts to parse the query request and returns any errors
// encountered in a structured way.
func (r QueryRequest) Analyze() (*QueryAnalysis, error) {
	switch r.Type {
	case "flux":
		return r.analyzeFluxQuery()
	case "influxql":
		return r.analyzeInfluxQLQuery()
	}

	return nil, fmt.Errorf("unknown query request type %s", r.Type)
}

func (r QueryRequest) analyzeFluxQuery() (*QueryAnalysis, error) {
	a := &QueryAnalysis{}
	pkg := parser.ParseSource(r.Query)
	errCount := ast.Check(pkg)
	if errCount == 0 {
		a.Errors = []queryParseError{}
		return a, nil
	}
	a.Errors = make([]queryParseError, 0, errCount)
	ast.Walk(ast.CreateVisitor(func(node ast.Node) {
		loc := node.Location()
		for _, err := range node.Errs() {
			a.Errors = append(a.Errors, queryParseError{
				Line:    loc.Start.Line,
				Column:  loc.Start.Column,
				Message: err.Msg,
			})
		}
	}), pkg)
	return a, nil
}

func (r QueryRequest) analyzeInfluxQLQuery() (*QueryAnalysis, error) {
	a := &QueryAnalysis{}
	_, err := influxql.ParseQuery(r.Query)
	if err == nil {
		a.Errors = []queryParseError{}
		return a, nil
	}

	ms := influxqlParseErrorRE.FindAllStringSubmatch(err.Error(), -1)
	a.Errors = make([]queryParseError, 0, len(ms))
	for _, m := range ms {
		if len(m) != 4 {
			return nil, fmt.Errorf("influxql query error is not formatted as expected: got %d matches expected 4", len(m))
		}
		msg := m[1]
		lineStr := m[2]
		line, err := strconv.Atoi(lineStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse line number from error mesage: %s -> %v", lineStr, err)
		}
		charStr := m[3]
		char, err := strconv.Atoi(charStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse character number from error mesage: %s -> %v", charStr, err)
		}

		a.Errors = append(a.Errors, queryParseError{
			Line:      line,
			Column:    columnFromCharacter(r.Query, char),
			Character: char,
			Message:   msg,
		})
	}

	return a, nil
}

func columnFromCharacter(q string, char int) int {
	col := 0
	for i, c := range q {
		if c == '\n' {
			col = 0
		}

		if i == char {
			break
		}
		col++
	}

	return col
}

var influxqlParseErrorRE = regexp.MustCompile(`^(.+) at line (\d+), char (\d+)$`)

// ProxyRequest returns a request to proxy from the flux.
func (r QueryRequest) ProxyRequest() (*query.ProxyRequest, error) {
	return r.proxyRequest(time.Now)
}

func (r QueryRequest) proxyRequest(now func() time.Time) (*query.ProxyRequest, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	// Query is preferred over spec
	var compiler flux.Compiler
	if r.Query != "" {
		pkg, err := flux.Parse(r.Query)
		if err != nil {
			return nil, err
		}
		c := lang.ASTCompiler{
			AST: pkg,
			Now: now(),
		}
		if r.Extern != nil {
			c.PrependFile(r.Extern)
		}
		compiler = c
	} else if r.AST != nil {
		c := lang.ASTCompiler{
			AST: r.AST,
			Now: now(),
		}
		if r.Extern != nil {
			c.PrependFile(r.Extern)
		}
		compiler = c
	} else if r.Spec != nil {
		compiler = lang.SpecCompiler{
			Spec: r.Spec,
		}
	}

	delimiter, _ := utf8.DecodeRuneInString(r.Dialect.Delimiter)

	noHeader := false
	if r.Dialect.Header != nil {
		noHeader = !*r.Dialect.Header
	}

	// TODO(nathanielc): Use commentPrefix and dateTimeFormat
	// once they are supported.
	return &query.ProxyRequest{
		Request: query.Request{
			OrganizationID: r.Org.ID,
			Compiler:       compiler,
		},
		Dialect: &csv.Dialect{
			ResultEncoderConfig: csv.ResultEncoderConfig{
				NoHeader:    noHeader,
				Delimiter:   delimiter,
				Annotations: r.Dialect.Annotations,
			},
		},
	}, nil
}

// QueryRequestFromProxyRequest converts a query.ProxyRequest into a QueryRequest.
// The ProxyRequest must contain supported compilers and dialects otherwise an error occurs.
func QueryRequestFromProxyRequest(req *query.ProxyRequest) (*QueryRequest, error) {
	qr := new(QueryRequest)
	switch c := req.Request.Compiler.(type) {
	case lang.FluxCompiler:
		qr.Type = "flux"
		qr.Query = c.Query
	case lang.SpecCompiler:
		qr.Type = "flux"
		qr.Spec = c.Spec
	case lang.ASTCompiler:
		qr.Type = "flux"
		qr.AST = c.AST
	default:
		return nil, fmt.Errorf("unsupported compiler %T", c)
	}
	switch d := req.Dialect.(type) {
	case *csv.Dialect:
		var header = !d.ResultEncoderConfig.NoHeader
		qr.Dialect.Header = &header
		qr.Dialect.Delimiter = string(d.ResultEncoderConfig.Delimiter)
		qr.Dialect.CommentPrefix = "#"
		qr.Dialect.DateTimeFormat = "RFC3339"
		qr.Dialect.Annotations = d.ResultEncoderConfig.Annotations
	default:
		return nil, fmt.Errorf("unsupported dialect %T", d)
	}

	return qr, nil
}

func decodeQueryRequest(ctx context.Context, r *http.Request, svc influxdb.OrganizationService) (*QueryRequest, int, error) {
	var req QueryRequest
	body := &countReader{Reader: r.Body}

	var contentType = "application/json"
	if ct := r.Header.Get("Content-Type"); ct != "" {
		contentType = ct
	}
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, body.bytesRead, err
	}
	switch mt {
	case "application/vnd.flux":
		octets, err := ioutil.ReadAll(body)
		if err != nil {
			return nil, body.bytesRead, err
		}
		req.Query = string(octets)
	case "application/json":
		fallthrough
	default:
		if err := json.NewDecoder(body).Decode(&req); err != nil {
			return nil, body.bytesRead, err
		}
	}

	req = req.WithDefaults()
	if err := req.Validate(); err != nil {
		return nil, body.bytesRead, err
	}

	req.Org, err = queryOrganization(ctx, r, svc)
	return &req, body.bytesRead, err
}

type countReader struct {
	bytesRead int
	io.Reader
}

func (r *countReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.bytesRead += n
	return n, err
}

func decodeProxyQueryRequest(ctx context.Context, r *http.Request, auth influxdb.Authorizer, svc influxdb.OrganizationService) (*query.ProxyRequest, int, error) {
	req, n, err := decodeQueryRequest(ctx, r, svc)
	if err != nil {
		return nil, n, err
	}

	pr, err := req.ProxyRequest()
	if err != nil {
		return nil, n, err
	}

	var token *influxdb.Authorization
	switch a := auth.(type) {
	case *influxdb.Authorization:
		token = a
	case *influxdb.Session:
		token = a.EphemeralAuth(req.Org.ID)
	default:
		return pr, n, influxdb.ErrAuthorizerNotSupported
	}

	pr.Request.Authorization = token
	return pr, n, nil
}
