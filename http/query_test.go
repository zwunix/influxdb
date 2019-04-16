package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/influxdata/flux"
	"github.com/influxdata/flux/ast"
	"github.com/influxdata/flux/csv"
	"github.com/influxdata/flux/lang"
	platform "github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/mock"
	"github.com/influxdata/influxdb/query"
	_ "github.com/influxdata/influxdb/query/builtin"
)

var cmpOptions = cmp.Options{
	cmpopts.IgnoreTypes(ast.BaseNode{}),
	cmpopts.IgnoreUnexported(query.ProxyRequest{}),
	cmpopts.IgnoreUnexported(query.Request{}),
	cmpopts.IgnoreUnexported(flux.Spec{}),
	cmpopts.EquateEmpty(),
}

func TestQueryRequest_WithDefaults(t *testing.T) {
	type fields struct {
		Spec    *flux.Spec
		AST     *ast.Package
		Query   string
		Type    string
		Dialect QueryDialect
		org     *platform.Organization
	}
	tests := []struct {
		name   string
		fields fields
		want   QueryRequest
	}{
		{
			name: "empty query has defaults set",
			want: QueryRequest{
				Type: "flux",
				Dialect: QueryDialect{
					Delimiter:      ",",
					DateTimeFormat: "RFC3339",
					Header:         func(x bool) *bool { return &x }(true),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := QueryRequest{
				Spec:    tt.fields.Spec,
				AST:     tt.fields.AST,
				Query:   tt.fields.Query,
				Type:    tt.fields.Type,
				Dialect: tt.fields.Dialect,
				Org:     tt.fields.org,
			}
			if got := r.WithDefaults(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("QueryRequest.WithDefaults() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryRequest_Validate(t *testing.T) {
	type fields struct {
		Extern  *ast.File
		Spec    *flux.Spec
		AST     *ast.Package
		Query   string
		Type    string
		Dialect QueryDialect
		org     *platform.Organization
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "requires query, spec, or ast",
			fields: fields{
				Type: "flux",
			},
			wantErr: true,
		},
		{
			name: "query cannot have both extern and spec",
			fields: fields{
				Extern: &ast.File{},
				Spec:   &flux.Spec{},
				Type:   "flux",
				Dialect: QueryDialect{
					Delimiter:      ",",
					DateTimeFormat: "RFC3339",
				},
			},
			wantErr: true,
		},
		{
			name: "requires flux type",
			fields: fields{
				Query: "howdy",
				Type:  "doody",
			},
			wantErr: true,
		},
		{
			name: "comment must be a single character",
			fields: fields{
				Query: "from()",
				Type:  "flux",
				Dialect: QueryDialect{
					CommentPrefix: "error!",
				},
			},
			wantErr: true,
		},
		{
			name: "delimiter must be a single character",
			fields: fields{
				Query: "from()",
				Type:  "flux",
				Dialect: QueryDialect{
					Delimiter: "",
				},
			},
			wantErr: true,
		},
		{
			name: "characters must be unicode runes",
			fields: fields{
				Query: "from()",
				Type:  "flux",
				Dialect: QueryDialect{
					Delimiter: string([]byte{0x80}),
				},
			},
			wantErr: true,
		},
		{
			name: "unknown annotations",
			fields: fields{
				Query: "from()",
				Type:  "flux",
				Dialect: QueryDialect{
					Delimiter:   ",",
					Annotations: []string{"error"},
				},
			},
			wantErr: true,
		},
		{
			name: "unknown date time format",
			fields: fields{
				Query: "from()",
				Type:  "flux",
				Dialect: QueryDialect{
					Delimiter:      ",",
					DateTimeFormat: "error",
				},
			},
			wantErr: true,
		},
		{
			name: "valid query",
			fields: fields{
				Query: "from()",
				Type:  "flux",
				Dialect: QueryDialect{
					Delimiter:      ",",
					DateTimeFormat: "RFC3339",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := QueryRequest{
				Extern:  tt.fields.Extern,
				Spec:    tt.fields.Spec,
				AST:     tt.fields.AST,
				Query:   tt.fields.Query,
				Type:    tt.fields.Type,
				Dialect: tt.fields.Dialect,
				Org:     tt.fields.org,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("QueryRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQueryRequest_proxyRequest(t *testing.T) {
	type fields struct {
		Extern  *ast.File
		Spec    *flux.Spec
		AST     *ast.Package
		Query   string
		Type    string
		Dialect QueryDialect
		org     *platform.Organization
	}
	tests := []struct {
		name    string
		fields  fields
		now     func() time.Time
		want    *query.ProxyRequest
		wantErr bool
	}{
		{
			name: "requires query, spec, or ast",
			fields: fields{
				Type: "flux",
			},
			wantErr: true,
		},
		{
			name: "valid query",
			fields: fields{
				Query: "howdy",
				Type:  "flux",
				Dialect: QueryDialect{
					Delimiter:      ",",
					DateTimeFormat: "RFC3339",
				},
				org: &platform.Organization{},
			},
			now: func() time.Time { return time.Unix(1, 1) },
			want: &query.ProxyRequest{
				Request: query.Request{
					Compiler: lang.ASTCompiler{
						AST: &ast.Package{
							Package: "main",
							Files: []*ast.File{
								{
									Body: []ast.Statement{
										&ast.ExpressionStatement{
											Expression: &ast.Identifier{Name: "howdy"},
										},
									},
								},
							},
						},
						Now: time.Unix(1, 1),
					},
				},
				Dialect: &csv.Dialect{
					ResultEncoderConfig: csv.ResultEncoderConfig{
						NoHeader:  false,
						Delimiter: ',',
					},
				},
			},
		},
		{
			name: "valid AST",
			fields: fields{
				AST:  &ast.Package{},
				Type: "flux",
				Dialect: QueryDialect{
					Delimiter:      ",",
					DateTimeFormat: "RFC3339",
				},
				org: &platform.Organization{},
			},
			now: func() time.Time { return time.Unix(1, 1) },
			want: &query.ProxyRequest{
				Request: query.Request{
					Compiler: lang.ASTCompiler{
						AST: &ast.Package{},
						Now: time.Unix(1, 1),
					},
				},
				Dialect: &csv.Dialect{
					ResultEncoderConfig: csv.ResultEncoderConfig{
						NoHeader:  false,
						Delimiter: ',',
					},
				},
			},
		},
		{
			name: "valid AST with extern",
			fields: fields{
				Extern: &ast.File{
					Body: []ast.Statement{
						&ast.OptionStatement{
							Assignment: &ast.VariableAssignment{
								ID:   &ast.Identifier{Name: "x"},
								Init: &ast.IntegerLiteral{Value: 0},
							},
						},
					},
				},
				AST:  &ast.Package{},
				Type: "flux",
				Dialect: QueryDialect{
					Delimiter:      ",",
					DateTimeFormat: "RFC3339",
				},
				org: &platform.Organization{},
			},
			now: func() time.Time { return time.Unix(1, 1) },
			want: &query.ProxyRequest{
				Request: query.Request{
					Compiler: lang.ASTCompiler{
						AST: &ast.Package{
							Files: []*ast.File{
								{
									Body: []ast.Statement{
										&ast.OptionStatement{
											Assignment: &ast.VariableAssignment{
												ID:   &ast.Identifier{Name: "x"},
												Init: &ast.IntegerLiteral{Value: 0},
											},
										},
									},
								},
							},
						},
						Now: time.Unix(1, 1),
					},
				},
				Dialect: &csv.Dialect{
					ResultEncoderConfig: csv.ResultEncoderConfig{
						NoHeader:  false,
						Delimiter: ',',
					},
				},
			},
		},
		{
			name: "valid spec",
			fields: fields{
				Type: "flux",
				Spec: &flux.Spec{
					Now: time.Unix(0, 0).UTC(),
				},
				Dialect: QueryDialect{
					Delimiter:      ",",
					DateTimeFormat: "RFC3339",
				},
				org: &platform.Organization{},
			},
			want: &query.ProxyRequest{
				Request: query.Request{
					Compiler: lang.SpecCompiler{
						Spec: &flux.Spec{
							Now: time.Unix(0, 0).UTC(),
						},
					},
				},
				Dialect: &csv.Dialect{
					ResultEncoderConfig: csv.ResultEncoderConfig{
						NoHeader:  false,
						Delimiter: ',',
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := QueryRequest{
				Extern:  tt.fields.Extern,
				Spec:    tt.fields.Spec,
				AST:     tt.fields.AST,
				Query:   tt.fields.Query,
				Type:    tt.fields.Type,
				Dialect: tt.fields.Dialect,
				Org:     tt.fields.org,
			}
			got, err := r.proxyRequest(tt.now)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryRequest.ProxyRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want, cmpOptions...) {
				t.Errorf("QueryRequest.ProxyRequest() -want/+got\n%s", cmp.Diff(tt.want, got, cmpOptions...))
			}
		})
	}
}

func Test_decodeQueryRequest(t *testing.T) {
	type args struct {
		ctx context.Context
		r   *http.Request
		svc platform.OrganizationService
	}
	tests := []struct {
		name    string
		args    args
		want    *QueryRequest
		wantErr bool
	}{
		{
			name: "valid query request",
			args: args{
				r: httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"query": "from()"}`)),
				svc: &mock.OrganizationService{
					FindOrganizationF: func(ctx context.Context, filter platform.OrganizationFilter) (*platform.Organization, error) {
						return &platform.Organization{
							ID: func() platform.ID { s, _ := platform.IDFromString("deadbeefdeadbeef"); return *s }(),
						}, nil
					},
				},
			},
			want: &QueryRequest{
				Query: "from()",
				Type:  "flux",
				Dialect: QueryDialect{
					Delimiter:      ",",
					DateTimeFormat: "RFC3339",
					Header:         func(x bool) *bool { return &x }(true),
				},
				Org: &platform.Organization{
					ID: func() platform.ID { s, _ := platform.IDFromString("deadbeefdeadbeef"); return *s }(),
				},
			},
		},
		{
			name: "valid query request with explict content-type",
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"query": "from()"}`))
					r.Header.Set("Content-Type", "application/json")
					return r
				}(),
				svc: &mock.OrganizationService{
					FindOrganizationF: func(ctx context.Context, filter platform.OrganizationFilter) (*platform.Organization, error) {
						return &platform.Organization{
							ID: func() platform.ID { s, _ := platform.IDFromString("deadbeefdeadbeef"); return *s }(),
						}, nil
					},
				},
			},
			want: &QueryRequest{
				Query: "from()",
				Type:  "flux",
				Dialect: QueryDialect{
					Delimiter:      ",",
					DateTimeFormat: "RFC3339",
					Header:         func(x bool) *bool { return &x }(true),
				},
				Org: &platform.Organization{
					ID: func() platform.ID { s, _ := platform.IDFromString("deadbeefdeadbeef"); return *s }(),
				},
			},
		},
		{
			name: "error decoding json",
			args: args{
				r: httptest.NewRequest("POST", "/", bytes.NewBufferString(`error`)),
			},
			wantErr: true,
		},
		{
			name: "error validating query",
			args: args{
				r: httptest.NewRequest("POST", "/", bytes.NewBufferString(`{}`)),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := decodeQueryRequest(tt.args.ctx, tt.args.r, tt.args.svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeQueryRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeQueryRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeProxyQueryRequest(t *testing.T) {
	type args struct {
		ctx  context.Context
		r    *http.Request
		auth *platform.Authorization
		svc  platform.OrganizationService
	}
	tests := []struct {
		name    string
		args    args
		want    *query.ProxyRequest
		wantErr bool
	}{
		{
			name: "valid post query request",
			args: args{
				r: httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"query": "from()"}`)),
				svc: &mock.OrganizationService{
					FindOrganizationF: func(ctx context.Context, filter platform.OrganizationFilter) (*platform.Organization, error) {
						return &platform.Organization{
							ID: func() platform.ID { s, _ := platform.IDFromString("deadbeefdeadbeef"); return *s }(),
						}, nil
					},
				},
			},
			want: &query.ProxyRequest{
				Request: query.Request{
					OrganizationID: func() platform.ID { s, _ := platform.IDFromString("deadbeefdeadbeef"); return *s }(),
					Compiler: lang.ASTCompiler{
						AST: &ast.Package{
							Package: "main",
							Files: []*ast.File{
								{
									Body: []ast.Statement{
										&ast.ExpressionStatement{
											Expression: &ast.CallExpression{
												Callee: &ast.Identifier{Name: "from"},
											},
										},
									},
								},
							},
						},
					},
				},
				Dialect: &csv.Dialect{
					ResultEncoderConfig: csv.ResultEncoderConfig{
						NoHeader:  false,
						Delimiter: ',',
					},
				},
			},
		},
		{
			name: "valid query including extern definition",
			args: args{
				r: httptest.NewRequest("POST", "/", bytes.NewBufferString(`
{
	"extern": {
		"type": "File",
		"body": [
			{
				"type": "OptionStatement",
				"assignment": {
					"type": "VariableAssignment",
					"id": {
						"type": "Identifier",
						"name": "x"
					},
					"init": {
						"type": "IntegerLiteral",
						"value": "0"
					}
				}
			}
		]
	},
	"query": "from(bucket: \"mybucket\")"
}
`)),
				svc: &mock.OrganizationService{
					FindOrganizationF: func(ctx context.Context, filter platform.OrganizationFilter) (*platform.Organization, error) {
						return &platform.Organization{
							ID: func() platform.ID { s, _ := platform.IDFromString("deadbeefdeadbeef"); return *s }(),
						}, nil
					},
				},
			},
			want: &query.ProxyRequest{
				Request: query.Request{
					OrganizationID: func() platform.ID { s, _ := platform.IDFromString("deadbeefdeadbeef"); return *s }(),
					Compiler: lang.ASTCompiler{
						AST: &ast.Package{
							Package: "main",
							Files: []*ast.File{
								{
									Body: []ast.Statement{
										&ast.OptionStatement{
											Assignment: &ast.VariableAssignment{
												ID:   &ast.Identifier{Name: "x"},
												Init: &ast.IntegerLiteral{Value: 0},
											},
										},
									},
								},
								{
									Body: []ast.Statement{
										&ast.ExpressionStatement{
											Expression: &ast.CallExpression{
												Callee: &ast.Identifier{Name: "from"},
												Arguments: []ast.Expression{
													&ast.ObjectExpression{
														Properties: []*ast.Property{
															{
																Key:   &ast.Identifier{Name: "bucket"},
																Value: &ast.StringLiteral{Value: "mybucket"},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				Dialect: &csv.Dialect{
					ResultEncoderConfig: csv.ResultEncoderConfig{
						NoHeader:  false,
						Delimiter: ',',
					},
				},
			},
		},
		{
			name: "valid post vnd.flux query request",
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest("POST", "/api/v2/query?org=myorg", strings.NewReader(`from(bucket: "mybucket")`))
					r.Header.Set("Content-Type", "application/vnd.flux")
					return r
				}(),
				svc: &mock.OrganizationService{
					FindOrganizationF: func(ctx context.Context, filter platform.OrganizationFilter) (*platform.Organization, error) {
						return &platform.Organization{
							ID: func() platform.ID { s, _ := platform.IDFromString("deadbeefdeadbeef"); return *s }(),
						}, nil
					},
				},
			},
			want: &query.ProxyRequest{
				Request: query.Request{
					OrganizationID: func() platform.ID { s, _ := platform.IDFromString("deadbeefdeadbeef"); return *s }(),
					Compiler: lang.ASTCompiler{
						AST: &ast.Package{
							Package: "main",
							Files: []*ast.File{
								{
									Body: []ast.Statement{
										&ast.ExpressionStatement{
											Expression: &ast.CallExpression{
												Callee: &ast.Identifier{Name: "from"},
												Arguments: []ast.Expression{
													&ast.ObjectExpression{
														Properties: []*ast.Property{
															{
																Key:   &ast.Identifier{Name: "bucket"},
																Value: &ast.StringLiteral{Value: "mybucket"},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				Dialect: &csv.Dialect{
					ResultEncoderConfig: csv.ResultEncoderConfig{
						NoHeader:  false,
						Delimiter: ',',
					},
				},
			},
		},
	}
	cmpOptions := append(cmpOptions, cmpopts.IgnoreFields(lang.ASTCompiler{}, "Now"))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := decodeProxyQueryRequest(tt.args.ctx, tt.args.r, tt.args.auth, tt.args.svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeProxyQueryRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(tt.want, got, cmpOptions...) {
				t.Errorf("decodeProxyQueryRequest() -want/+got\n%s", cmp.Diff(tt.want, got, cmpOptions...))
			}
		})
	}
}
