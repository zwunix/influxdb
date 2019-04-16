package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/influxdata/flux"
	"github.com/influxdata/flux/ast"
	"github.com/influxdata/flux/complete"
	"github.com/influxdata/flux/csv"
	"github.com/influxdata/flux/iocounter"
	"github.com/influxdata/flux/parser"
	platform "github.com/influxdata/influxdb"
	pcontext "github.com/influxdata/influxdb/context"
	"github.com/influxdata/influxdb/http/metric"
	"github.com/influxdata/influxdb/kit/check"
	"github.com/influxdata/influxdb/kit/tracing"
	"github.com/influxdata/influxdb/query"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	fluxPath = "/api/v2/query"
)

// FluxBackend is all services and associated parameters required to construct
// the FluxHandler.
type FluxBackend struct {
	Logger              *zap.Logger
	QueryMetricRecorder metric.Recorder

	OrganizationService platform.OrganizationService
	ProxyQueryService   query.ProxyQueryService
}

// NewFluxBackend returns a new instance of FluxBackend.
func NewFluxBackend(b *APIBackend) *FluxBackend {
	return &FluxBackend{
		Logger:              b.Logger.With(zap.String("handler", "query")),
		QueryMetricRecorder: b.QueryMetricRecorder,

		ProxyQueryService:   b.FluxService,
		OrganizationService: b.OrganizationService,
	}
}

// FluxHandler implements handling flux queries.
type FluxHandler struct {
	*httprouter.Router

	Logger *zap.Logger

	Now                 func() time.Time
	OrganizationService platform.OrganizationService
	ProxyQueryService   query.ProxyQueryService

	MetricRecorder metric.Recorder
}

// NewFluxHandler returns a new handler at /api/v2/query for flux queries.
func NewFluxHandler(b *FluxBackend) *FluxHandler {
	h := &FluxHandler{
		Router: NewRouter(),
		Now:    time.Now,
		Logger: b.Logger,

		ProxyQueryService:   b.ProxyQueryService,
		OrganizationService: b.OrganizationService,
		MetricRecorder:      b.QueryMetricRecorder,
	}

	h.HandlerFunc("POST", fluxPath, h.handleQuery)
	h.HandlerFunc("POST", "/api/v2/query/ast", h.postFluxAST)
	h.HandlerFunc("POST", "/api/v2/query/analyze", h.postQueryAnalyze)
	h.HandlerFunc("POST", "/api/v2/query/spec", h.postFluxSpec)
	h.HandlerFunc("GET", "/api/v2/query/suggestions", h.getFluxSuggestions)
	h.HandlerFunc("GET", "/api/v2/query/suggestions/:name", h.getFluxSuggestion)
	return h
}

func (h *FluxHandler) handleQuery(w http.ResponseWriter, r *http.Request) {
	span, r := tracing.ExtractFromHTTPRequest(r, "FluxHandler")
	defer span.Finish()

	ctx := r.Context()

	// TODO(desa): I really don't like how we're recording the usage metrics here
	// Ideally this will be moved when we solve https://github.com/influxdata/influxdb/issues/13403
	var orgID platform.ID
	var requestBytes int
	sw := newStatusResponseWriter(w)
	w = sw
	defer func() {
		h.MetricRecorder.Record(ctx, metric.Metric{
			OrgID:         orgID,
			Endpoint:      r.URL.Path, // This should be sufficient for the time being as it should only be single endpoint.
			RequestBytes:  requestBytes,
			ResponseBytes: sw.responseBytes,
			Status:        sw.code(),
		})
	}()

	a, err := pcontext.GetAuthorizer(ctx)
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}

	req, n, err := decodeProxyQueryRequest(ctx, r, a, h.OrganizationService)
	if err != nil && err != platform.ErrAuthorizerNotSupported {
		EncodeError(ctx, err, w)
		return
	}
	requestBytes = n

	// Transform the context into one with the request's authorization.
	ctx = pcontext.SetAuthorizer(ctx, req.Request.Authorization)

	hd, ok := req.Dialect.(HTTPDialect)
	if !ok {
		EncodeError(ctx, fmt.Errorf("unsupported dialect over HTTP %T", req.Dialect), w)
		return
	}
	hd.SetHeaders(w)

	cw := iocounter.Writer{Writer: w}
	if _, err := h.ProxyQueryService.Query(ctx, &cw, req); err != nil {
		if cw.Count() == 0 {
			// Only record the error headers IFF nothing has been written to w.
			EncodeError(ctx, err, w)
			return
		}
		h.Logger.Info("Error writing response to client",
			zap.String("handler", "flux"),
			zap.Error(err),
		)
	}
}

type langRequest struct {
	Query string `json:"query"`
}

type postFluxASTResponse struct {
	AST *ast.Package `json:"ast"`
}

// postFluxAST returns a flux AST for provided flux string
func (h *FluxHandler) postFluxAST(w http.ResponseWriter, r *http.Request) {
	span, r := tracing.ExtractFromHTTPRequest(r, "FluxHandler")
	defer span.Finish()

	var request langRequest
	ctx := r.Context()

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		EncodeError(ctx, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "invalid json",
			Err:  err,
		}, w)
		return
	}

	pkg := parser.ParseSource(request.Query)
	if ast.Check(pkg) > 0 {
		err := ast.GetError(pkg)
		EncodeError(ctx, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "invalid AST",
			Err:  err,
		}, w)
		return
	}

	res := postFluxASTResponse{
		AST: pkg,
	}

	if err := encodeResponse(ctx, w, http.StatusOK, res); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

// postQueryAnalyze parses a query and returns any query errors.
func (h *FluxHandler) postQueryAnalyze(w http.ResponseWriter, r *http.Request) {
	span, r := tracing.ExtractFromHTTPRequest(r, "FluxHandler")
	defer span.Finish()

	ctx := r.Context()

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		EncodeError(ctx, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "invalid json",
			Err:  err,
		}, w)
		return
	}

	a, err := req.Analyze()
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}
	if err := encodeResponse(ctx, w, http.StatusOK, a); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type postFluxSpecResponse struct {
	Spec *flux.Spec `json:"spec"`
}

// postFluxSpec returns a flux Spec for provided flux string
func (h *FluxHandler) postFluxSpec(w http.ResponseWriter, r *http.Request) {
	span, r := tracing.ExtractFromHTTPRequest(r, "FluxHandler")
	defer span.Finish()

	var req langRequest
	ctx := r.Context()

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		EncodeError(ctx, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "invalid json",
			Err:  err,
		}, w)
		return
	}

	spec, err := flux.Compile(ctx, req.Query, h.Now())
	if err != nil {
		EncodeError(ctx, &platform.Error{
			Code: platform.EUnprocessableEntity,
			Msg:  "invalid spec",
			Err:  err,
		}, w)
		return
	}

	res := postFluxSpecResponse{
		Spec: spec,
	}

	if err := encodeResponse(ctx, w, http.StatusOK, res); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

// fluxParams contain flux funciton parameters as defined by the semantic graph
type fluxParams map[string]string

// suggestionResponse provides the parameters available for a given Flux function
type suggestionResponse struct {
	Name   string     `json:"name"`
	Params fluxParams `json:"params"`
}

// suggestionsResponse provides a list of available Flux functions
type suggestionsResponse struct {
	Functions []suggestionResponse `json:"funcs"`
}

// getFluxSuggestions returns a list of available Flux functions for the Flux Builder
func (h *FluxHandler) getFluxSuggestions(w http.ResponseWriter, r *http.Request) {
	span, r := tracing.ExtractFromHTTPRequest(r, "FluxHandler")
	defer span.Finish()

	ctx := r.Context()
	completer := complete.DefaultCompleter()
	names := completer.FunctionNames()
	var functions []suggestionResponse
	for _, name := range names {
		suggestion, err := completer.FunctionSuggestion(name)
		if err != nil {
			EncodeError(ctx, err, w)
			return
		}

		filteredParams := make(fluxParams)
		for key, value := range suggestion.Params {
			if key == "table" {
				continue
			}

			filteredParams[key] = value
		}

		functions = append(functions, suggestionResponse{
			Name:   name,
			Params: filteredParams,
		})
	}
	res := suggestionsResponse{Functions: functions}

	if err := encodeResponse(ctx, w, http.StatusOK, res); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

// getFluxSuggestion returns the function parameters for the requested function
func (h *FluxHandler) getFluxSuggestion(w http.ResponseWriter, r *http.Request) {
	span, r := tracing.ExtractFromHTTPRequest(r, "FluxHandler")
	defer span.Finish()

	ctx := r.Context()
	name := httprouter.ParamsFromContext(ctx).ByName("name")
	completer := complete.DefaultCompleter()

	suggestion, err := completer.FunctionSuggestion(name)
	if err != nil {
		EncodeError(ctx, err, w)
		return
	}

	res := suggestionResponse{Name: name, Params: suggestion.Params}
	if err := encodeResponse(ctx, w, http.StatusOK, res); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

// PrometheusCollectors satisifies the prom.PrometheusCollector interface.
func (h *FluxHandler) PrometheusCollectors() []prom.Collector {
	// TODO: gather and return relevant metrics.
	return nil
}

var _ query.ProxyQueryService = (*FluxService)(nil)

// FluxService connects to Influx via HTTP using tokens to run queries.
type FluxService struct {
	Addr               string
	Token              string
	InsecureSkipVerify bool
}

// Query runs a flux query against a influx server and sends the results to the io.Writer.
// Will use the token from the context over the token within the service struct.
func (s *FluxService) Query(ctx context.Context, w io.Writer, r *query.ProxyRequest) (flux.Statistics, error) {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()
	u, err := newURL(s.Addr, fluxPath)
	if err != nil {
		return flux.Statistics{}, tracing.LogError(span, err)
	}

	qreq, err := QueryRequestFromProxyRequest(r)
	if err != nil {
		return flux.Statistics{}, tracing.LogError(span, err)
	}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(qreq); err != nil {
		return flux.Statistics{}, tracing.LogError(span, err)
	}

	hreq, err := http.NewRequest("POST", u.String(), &body)
	if err != nil {
		return flux.Statistics{}, tracing.LogError(span, err)
	}

	SetToken(s.Token, hreq)

	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Accept", "text/csv")
	hreq = hreq.WithContext(ctx)
	tracing.InjectToHTTPRequest(span, hreq)

	hc := newClient(u.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(hreq)
	if err != nil {
		return flux.Statistics{}, tracing.LogError(span, err)
	}
	defer resp.Body.Close()

	if err := CheckError(resp); err != nil {
		return flux.Statistics{}, tracing.LogError(span, err)
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		return flux.Statistics{}, tracing.LogError(span, err)
	}
	return flux.Statistics{}, nil
}

func (s FluxService) Check(ctx context.Context) check.Response {
	return QueryHealthCheck(s.Addr, s.InsecureSkipVerify)
}

var _ query.QueryService = (*FluxQueryService)(nil)

// FluxQueryService implements query.QueryService by making HTTP requests to the /api/v2/query API endpoint.
type FluxQueryService struct {
	Addr               string
	Token              string
	InsecureSkipVerify bool
}

// Query runs a flux query against a influx server and decodes the result
func (s *FluxQueryService) Query(ctx context.Context, r *query.Request) (flux.ResultIterator, error) {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	u, err := newURL(s.Addr, fluxPath)
	if err != nil {
		return nil, tracing.LogError(span, err)
	}
	params := url.Values{}
	params.Set(OrgID, r.OrganizationID.String())
	u.RawQuery = params.Encode()

	preq := &query.ProxyRequest{
		Request: *r,
		Dialect: csv.DefaultDialect(),
	}
	qreq, err := QueryRequestFromProxyRequest(preq)
	if err != nil {
		return nil, tracing.LogError(span, err)
	}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(qreq); err != nil {
		return nil, tracing.LogError(span, err)
	}

	hreq, err := http.NewRequest("POST", u.String(), &body)
	if err != nil {
		return nil, tracing.LogError(span, err)
	}

	SetToken(s.Token, hreq)

	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Accept", "text/csv")
	hreq = hreq.WithContext(ctx)
	tracing.InjectToHTTPRequest(span, hreq)

	hc := newClient(u.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(hreq)
	if err != nil {
		return nil, tracing.LogError(span, err)
	}
	// Can't defer resp.Body.Close here because the CSV decoder depends on reading from resp.Body after this function returns.

	if err := CheckError(resp); err != nil {
		return nil, tracing.LogError(span, err)
	}

	decoder := csv.NewMultiResultDecoder(csv.ResultDecoderConfig{})
	itr, err := decoder.Decode(resp.Body)
	if err != nil {
		return nil, tracing.LogError(span, err)
	}

	return itr, nil
}

func (s FluxQueryService) Check(ctx context.Context) check.Response {
	return QueryHealthCheck(s.Addr, s.InsecureSkipVerify)
}

// SimpleQuery runs a flux query with common parameters and returns CSV results.
func SimpleQuery(addr, flux, org, token string) ([]byte, error) {
	u, err := newURL(addr, fluxPath)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set(OrgName, org)
	u.RawQuery = params.Encode()

	header := true
	qr := &QueryRequest{
		Type:  "flux",
		Query: flux,
		Dialect: QueryDialect{
			Header:         &header,
			Delimiter:      ",",
			CommentPrefix:  "#",
			DateTimeFormat: "RFC3339",
		},
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(qr); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", u.String(), &body)
	if err != nil {
		return nil, err
	}

	SetToken(token, req)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/csv")

	insecureSkipVerify := false
	hc := newClient(u.Scheme, insecureSkipVerify)
	res, err := hc.Do(req)
	if err != nil {
		return nil, err
	}

	if err := CheckError(res); err != nil {
		return nil, err
	}

	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

func QueryHealthCheck(url string, insecureSkipVerify bool) check.Response {
	u, err := newURL(url, "/health")
	if err != nil {
		return check.Response{
			Name:    "query health",
			Status:  check.StatusFail,
			Message: errors.Wrap(err, "could not form URL").Error(),
		}
	}

	hc := newClient(u.Scheme, insecureSkipVerify)
	resp, err := hc.Get(u.String())
	if err != nil {
		return check.Response{
			Name:    "query health",
			Status:  check.StatusFail,
			Message: errors.Wrap(err, "error getting response").Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return check.Response{
			Name:    "query health",
			Status:  check.StatusFail,
			Message: fmt.Sprintf("http error %v", resp.StatusCode),
		}
	}

	var healthResponse check.Response
	if err = json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
		return check.Response{
			Name:    "query health",
			Status:  check.StatusFail,
			Message: errors.Wrap(err, "error decoding JSON response").Error(),
		}
	}

	return healthResponse
}
