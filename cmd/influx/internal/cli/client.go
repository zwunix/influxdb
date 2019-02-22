package cli

import (
	"context"
	"fmt"
	"io"
	nethttp "net/http"
	"path"
	"strings"

	"github.com/influxdata/flux"
	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/bolt"
	"github.com/influxdata/influxdb/http"
	"github.com/influxdata/influxdb/internal/fs"
	"github.com/influxdata/influxdb/kv"
	"github.com/influxdata/influxdb/query"
)

type client struct {
	config Config

	closers []io.Closer

	authzService      influxdb.AuthorizationService
	dashboardService  influxdb.DashboardService
	bucketService     influxdb.BucketService
	labelService      influxdb.LabelService
	orgService        influxdb.OrganizationService
	onboardingService influxdb.OnboardingService
	passwordsService  influxdb.PasswordsService
	queryService      query.QueryService
	secretService     influxdb.SecretService
	scrapeService     influxdb.ScraperTargetStoreService
	taskService       influxdb.TaskService
	telegrafService   influxdb.TelegrafConfigStore
	userService       influxdb.UserService
	variableService   influxdb.VariableService
	writeService      influxdb.WriteService
}

type Config struct {
	Local bool
	Host  string
	Token string
}

func NewClient(c Config) *client {
	return &client{
		config: c,
	}
}

func (c *client) Open(ctx context.Context) error {
	if c.config.Local {
		file, err := fs.BoltFile()
		if err != nil {
			return err
		}

		store := bolt.NewKVStore(file)
		if err := store.Open(ctx); err != nil {
			return err
		}

		c.closers = append(c.closers, store)

		svc := kv.NewService(store)
		if err := svc.Initialize(ctx); err != nil {
			return err
		}

		c.authzService = svc
		c.dashboardService = svc
		c.bucketService = svc
		c.labelService = svc
		c.orgService = svc
		c.passwordsService = svc
		c.queryService = nil // TODO(desa): dunno what to make this
		c.secretService = svc
		c.scrapeService = svc
		c.taskService = nil // TODO(desa): ???
		c.telegrafService = svc
		c.userService = svc
		c.variableService = svc
		c.writeService = nil // TODO(desa): ???

		return nil
	}

	if err := ping(c.config); err != nil {
		return fmt.Errorf("could not reach host: %v", err)
	}

	// TODO(desa): construct http services.

	return nil
}

type errors []error

func (errs errors) Error() string {
	var strs []string
	for _, err := range errs {
		strs = append(strs, err.Error())
	}
	return strings.Join(strs, ";")
}

func (c *client) Close(ctx context.Context) error {
	var errs errors
	for _, closer := range c.closers {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) >= 1 {
		return errs
	}

	return nil
}

func ping(c Config) error {
	req, err := nethttp.NewRequest("GET", path.Join(c.Host, "/api/v2"), nil)
	if err != nil {
		// TODO(desa): add better user facing error.
		return err
	}
	http.SetToken(c.Token, req)

	resp, err := nethttp.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != nethttp.StatusOK {
		return fmt.Errorf("ping recieve status code %v, expected 200", resp.StatusCode)
	}

	return nil
}

func (c *client) FindAuthorizationByID(ctx context.Context, id influxdb.ID) (*influxdb.Authorization, error) {
	return c.authzService.FindAuthorizationByID(ctx, id)
}

func (c *client) FindAuthorizationByToken(ctx context.Context, t string) (*influxdb.Authorization, error) {
	return c.authzService.FindAuthorizationByToken(ctx, t)
}

func (c *client) FindAuthorizations(ctx context.Context, filter influxdb.AuthorizationFilter, opt ...influxdb.FindOptions) ([]*influxdb.Authorization, int, error) {
	return c.authzService.FindAuthorizations(ctx, filter, opt...)
}

func (c *client) CreateAuthorization(ctx context.Context, a *influxdb.Authorization) error {
	return c.authzService.CreateAuthorization(ctx, a)
}

func (c *client) SetAuthorizationStatus(ctx context.Context, id influxdb.ID, status influxdb.Status) error {
	return c.authzService.SetAuthorizationStatus(ctx, id, status)
}

func (c *client) DeleteAuthorization(ctx context.Context, id influxdb.ID) error {
	return c.authzService.DeleteAuthorization(ctx, id)
}

func (c *client) FindDashboardByID(ctx context.Context, id influxdb.ID) (*influxdb.Dashboard, error) {
	return c.dashboardService.FindDashboardByID(ctx, id)
}

func (c *client) FindDashboards(ctx context.Context, filter influxdb.DashboardFilter, opts influxdb.FindOptions) ([]*influxdb.Dashboard, int, error) {
	return c.dashboardService.FindDashboards(ctx, filter, opts)
}

func (c *client) CreateDashboard(ctx context.Context, d *influxdb.Dashboard) error {
	return c.dashboardService.CreateDashboard(ctx, d)
}

func (c *client) UpdateDashboard(ctx context.Context, id influxdb.ID, upd influxdb.DashboardUpdate) (*influxdb.Dashboard, error) {
	return c.dashboardService.UpdateDashboard(ctx, id, upd)
}

func (c *client) AddDashboardCell(ctx context.Context, id influxdb.ID, cell *influxdb.Cell, opts influxdb.AddDashboardCellOptions) error {
	return c.dashboardService.AddDashboardCell(ctx, id, cell, opts)
}

func (c *client) RemoveDashboardCell(ctx context.Context, dashboardID influxdb.ID, cellID influxdb.ID) error {
	return c.dashboardService.RemoveDashboardCell(ctx, dashboardID, cellID)
}

func (c *client) UpdateDashboardCell(ctx context.Context, dashboardID influxdb.ID, cellID influxdb.ID, upd influxdb.CellUpdate) (*influxdb.Cell, error) {
	return c.dashboardService.UpdateDashboardCell(ctx, dashboardID, cellID, upd)
}

func (c *client) GetDashboardCellView(ctx context.Context, dashboardID influxdb.ID, cellID influxdb.ID) (*influxdb.View, error) {
	return c.dashboardService.GetDashboardCellView(ctx, dashboardID, cellID)
}

func (c *client) UpdateDashboardCellView(ctx context.Context, dashboardID influxdb.ID, cellID influxdb.ID, upd influxdb.ViewUpdate) (*influxdb.View, error) {
	return c.dashboardService.UpdateDashboardCellView(ctx, dashboardID, cellID, upd)
}

func (c *client) DeleteDashboard(ctx context.Context, id influxdb.ID) error {
	return c.dashboardService.DeleteDashboard(ctx, id)
}

func (c *client) ReplaceDashboardCells(ctx context.Context, id influxdb.ID, cs []*influxdb.Cell) error {
	return c.dashboardService.ReplaceDashboardCells(ctx, id, cs)
}

func (c *client) FindBucketByID(ctx context.Context, id influxdb.ID) (*influxdb.Bucket, error) {
	return c.bucketService.FindBucketByID(ctx, id)
}

func (c *client) FindBucket(ctx context.Context, filter influxdb.BucketFilter) (*influxdb.Bucket, error) {
	return c.bucketService.FindBucket(ctx, filter)
}

func (c *client) FindBuckets(ctx context.Context, filter influxdb.BucketFilter, opt ...influxdb.FindOptions) ([]*influxdb.Bucket, int, error) {
	return c.bucketService.FindBuckets(ctx, filter, opt...)
}

func (c *client) CreateBucket(ctx context.Context, b *influxdb.Bucket) error {
	return c.bucketService.CreateBucket(ctx, b)
}

func (c *client) UpdateBucket(ctx context.Context, id influxdb.ID, upd influxdb.BucketUpdate) (*influxdb.Bucket, error) {
	return c.bucketService.UpdateBucket(ctx, id, upd)
}

func (c *client) DeleteBucket(ctx context.Context, id influxdb.ID) error {
	return c.bucketService.DeleteBucket(ctx, id)
}

func (c *client) FindLabelByID(ctx context.Context, id influxdb.ID) (*influxdb.Label, error) {
	return c.labelService.FindLabelByID(ctx, id)
}

func (c *client) FindLabels(ctx context.Context, filter influxdb.LabelFilter, opt ...influxdb.FindOptions) ([]*influxdb.Label, error) {
	return c.labelService.FindLabels(ctx, filter, opt...)
}

func (c *client) FindResourceLabels(ctx context.Context, filter influxdb.LabelMappingFilter) ([]*influxdb.Label, error) {
	return c.labelService.FindResourceLabels(ctx, filter)
}

func (c *client) CreateLabel(ctx context.Context, l *influxdb.Label) error {
	return c.labelService.CreateLabel(ctx, l)
}

func (c *client) CreateLabelMapping(ctx context.Context, m *influxdb.LabelMapping) error {
	return c.labelService.CreateLabelMapping(ctx, m)
}

func (c *client) UpdateLabel(ctx context.Context, id influxdb.ID, upd influxdb.LabelUpdate) (*influxdb.Label, error) {
	return c.labelService.UpdateLabel(ctx, id, upd)
}

func (c *client) DeleteLabel(ctx context.Context, id influxdb.ID) error {
	return c.labelService.DeleteLabel(ctx, id)
}

func (c *client) DeleteLabelMapping(ctx context.Context, m *influxdb.LabelMapping) error {
	return c.labelService.DeleteLabelMapping(ctx, m)
}

func (c *client) FindOrganizationByID(ctx context.Context, id influxdb.ID) (*influxdb.Organization, error) {
	return c.orgService.FindOrganizationByID(ctx, id)
}

func (c *client) FindOrganization(ctx context.Context, filter influxdb.OrganizationFilter) (*influxdb.Organization, error) {
	return c.orgService.FindOrganization(ctx, filter)
}

func (c *client) FindOrganizations(ctx context.Context, filter influxdb.OrganizationFilter, opt ...influxdb.FindOptions) ([]*influxdb.Organization, int, error) {
	return c.orgService.FindOrganizations(ctx, filter, opt...)
}

func (c *client) CreateOrganization(ctx context.Context, o *influxdb.Organization) error {
	return c.orgService.CreateOrganization(ctx, o)
}

func (c *client) UpdateOrganization(ctx context.Context, id influxdb.ID, upd influxdb.OrganizationUpdate) (*influxdb.Organization, error) {
	return c.orgService.UpdateOrganization(ctx, id, upd)
}

func (c *client) DeleteOrganization(ctx context.Context, id influxdb.ID) error {
	return c.orgService.DeleteOrganization(ctx, id)
}

func (c *client) SetPassword(ctx context.Context, name string, password string) error {
	if !c.config.Local {
		return fmt.Errorf("cannot set password remotely try with --local flag")
	}

	return c.passwordsService.SetPassword(ctx, name, password)
}

func (c *client) ComparePassword(ctx context.Context, name string, password string) error {
	if !c.config.Local {
		return fmt.Errorf("cannot compare password remotely try with --local flag")
	}

	return c.passwordsService.ComparePassword(ctx, name, password)
}

func (c *client) CompareAndSetPassword(ctx context.Context, name string, old string, new string) error {
	return c.passwordsService.CompareAndSetPassword(ctx, name, old, new)
}

func (c *client) FindUserByID(ctx context.Context, id influxdb.ID) (*influxdb.User, error) {
	return c.userService.FindUserByID(ctx, id)
}

func (c *client) FindUser(ctx context.Context, filter influxdb.UserFilter) (*influxdb.User, error) {
	return c.userService.FindUser(ctx, filter)
}

func (c *client) FindUsers(ctx context.Context, filter influxdb.UserFilter, opt ...influxdb.FindOptions) ([]*influxdb.User, int, error) {
	return c.userService.FindUsers(ctx, filter, opt...)
}

func (c *client) CreateUser(ctx context.Context, u *influxdb.User) error {
	return c.userService.CreateUser(ctx, u)
}

func (c *client) UpdateUser(ctx context.Context, id influxdb.ID, upd influxdb.UserUpdate) (*influxdb.User, error) {
	return c.userService.UpdateUser(ctx, id, upd)
}

func (c *client) DeleteUser(ctx context.Context, id influxdb.ID) error {
	return c.userService.DeleteUser(ctx, id)
}

func (c *client) IsOnboarding(ctx context.Context) (bool, error) {
	return c.onboardingService.IsOnboarding(ctx)
}

func (c *client) Generate(ctx context.Context, req *influxdb.OnboardingRequest) (*influxdb.OnboardingResults, error) {
	return c.onboardingService.Generate(ctx, req)
}

func (c *client) Query(ctx context.Context, req *query.Request) (flux.ResultIterator, error) {
	return c.queryService.Query(ctx, req)
}

func (c *client) GetSecretKeys(ctx context.Context, orgID influxdb.ID) ([]string, error) {
	return c.secretService.GetSecretKeys(ctx, orgID)
}

func (c *client) PutSecret(ctx context.Context, orgID influxdb.ID, k string, v string) error {
	return c.secretService.PutSecret(ctx, orgID, k, v)
}

func (c *client) PutSecrets(ctx context.Context, orgID influxdb.ID, m map[string]string) error {
	return c.secretService.PutSecrets(ctx, orgID, m)
}

func (c *client) PatchSecrets(ctx context.Context, orgID influxdb.ID, m map[string]string) error {
	return c.secretService.PatchSecrets(ctx, orgID, m)
}

func (c *client) DeleteSecret(ctx context.Context, orgID influxdb.ID, ks ...string) error {
	return c.secretService.DeleteSecret(ctx, orgID, ks...)
}

func (c *client) ListTargets(ctx context.Context) ([]influxdb.ScraperTarget, error) {
	return c.scrapeService.ListTargets(ctx)
}

func (c *client) AddTarget(ctx context.Context, t *influxdb.ScraperTarget, userID influxdb.ID) error {
	return c.scrapeService.AddTarget(ctx, t, userID)
}

func (c *client) GetTargetByID(ctx context.Context, id influxdb.ID) (*influxdb.ScraperTarget, error) {
	return c.scrapeService.GetTargetByID(ctx, id)
}

func (c *client) RemoveTarget(ctx context.Context, id influxdb.ID) error {
	return c.scrapeService.RemoveTarget(ctx, id)
}

func (c *client) UpdateTarget(ctx context.Context, t *influxdb.ScraperTarget, userID influxdb.ID) (*influxdb.ScraperTarget, error) {
	return c.scrapeService.UpdateTarget(ctx, t, userID)
}

func (c *client) FindTaskByID(ctx context.Context, id influxdb.ID) (*influxdb.Task, error) {
	return c.taskService.FindTaskByID(ctx, id)
}

func (c *client) FindTasks(ctx context.Context, filter influxdb.TaskFilter) ([]*influxdb.Task, int, error) {
	return c.taskService.FindTasks(ctx, filter)
}

func (c *client) CreateTask(ctx context.Context, t influxdb.TaskCreate) (*influxdb.Task, error) {
	return c.taskService.CreateTask(ctx, t)
}

func (c *client) UpdateTask(ctx context.Context, id influxdb.ID, upd influxdb.TaskUpdate) (*influxdb.Task, error) {
	return c.taskService.UpdateTask(ctx, id, upd)
}

func (c *client) DeleteTask(ctx context.Context, id influxdb.ID) error {
	return c.taskService.DeleteTask(ctx, id)
}

func (c *client) FindTaskLogs(ctx context.Context, filter influxdb.LogFilter) ([]*influxdb.Log, int, error) {
	return c.taskService.FindLogs(ctx, filter)
}

func (c *client) FindTaskRuns(ctx context.Context, filter influxdb.RunFilter) ([]*influxdb.Run, int, error) {
	return c.taskService.FindRuns(ctx, filter)
}

func (c *client) FindTaskRunByID(ctx context.Context, taskID influxdb.ID, runID influxdb.ID) (*influxdb.Run, error) {
	return c.taskService.FindRunByID(ctx, taskID, runID)
}

func (c *client) CancelTaskRun(ctx context.Context, taskID influxdb.ID, runID influxdb.ID) error {
	return c.taskService.CancelRun(ctx, taskID, runID)
}

func (c *client) RetryTaskRun(ctx context.Context, taskID influxdb.ID, runID influxdb.ID) (*influxdb.Run, error) {
	return c.taskService.RetryRun(ctx, taskID, runID)
}

func (c *client) ForceTaskRun(ctx context.Context, taskID influxdb.ID, scheduledFor int64) (*influxdb.Run, error) {
	return c.taskService.ForceRun(ctx, taskID, scheduledFor)
}

func (c *client) FindTelegrafConfigByID(ctx context.Context, id influxdb.ID) (*influxdb.TelegrafConfig, error) {
	return c.telegrafService.FindTelegrafConfigByID(ctx, id)
}

func (c *client) FindTelegrafConfigs(ctx context.Context, filter influxdb.TelegrafConfigFilter, opt ...influxdb.FindOptions) ([]*influxdb.TelegrafConfig, int, error) {
	return c.telegrafService.FindTelegrafConfigs(ctx, filter, opt...)
}

func (c *client) CreateTelegrafConfig(ctx context.Context, tc *influxdb.TelegrafConfig, userID influxdb.ID) error {
	return c.telegrafService.CreateTelegrafConfig(ctx, tc, userID)
}

func (c *client) UpdateTelegrafConfig(ctx context.Context, id influxdb.ID, tc *influxdb.TelegrafConfig, userID influxdb.ID) (*influxdb.TelegrafConfig, error) {
	return c.telegrafService.UpdateTelegrafConfig(ctx, id, tc, userID)
}

func (c *client) DeleteTelegrafConfig(ctx context.Context, id influxdb.ID) error {
	return c.telegrafService.DeleteTelegrafConfig(ctx, id)
}

func (c *client) FindVariableByID(ctx context.Context, id influxdb.ID) (*influxdb.Variable, error) {
	return c.variableService.FindVariableByID(ctx, id)
}

func (c *client) FindVariables(ctx context.Context, filter influxdb.VariableFilter, opt ...influxdb.FindOptions) ([]*influxdb.Variable, error) {
	return c.variableService.FindVariables(ctx, filter, opt...)
}

func (c *client) CreateVariable(ctx context.Context, m *influxdb.Variable) error {
	return c.variableService.CreateVariable(ctx, m)
}

func (c *client) UpdateVariable(ctx context.Context, id influxdb.ID, update *influxdb.VariableUpdate) (*influxdb.Variable, error) {
	return c.variableService.UpdateVariable(ctx, id, update)
}

func (c *client) ReplaceVariable(ctx context.Context, variable *influxdb.Variable) error {
	return c.variableService.ReplaceVariable(ctx, variable)
}

func (c *client) DeleteVariable(ctx context.Context, id influxdb.ID) error {
	return c.variableService.DeleteVariable(ctx, id)
}

func (c *client) Write(ctx context.Context, org influxdb.ID, bucket influxdb.ID, r io.Reader) error {
	return c.writeService.Write(ctx, org, bucket, r)
}
