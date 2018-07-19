package route

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/evergreen-ci/evergreen"
	"github.com/evergreen-ci/evergreen/db"
	"github.com/evergreen-ci/evergreen/model"
	"github.com/evergreen-ci/evergreen/model/distro"
	"github.com/evergreen-ci/evergreen/model/event"
	"github.com/evergreen-ci/evergreen/model/user"
	"github.com/evergreen-ci/evergreen/rest/data"
	restModel "github.com/evergreen-ci/evergreen/rest/model"
	"github.com/evergreen-ci/evergreen/testutil"
	"github.com/evergreen-ci/gimlet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AdminRouteSuite struct {
	sc          data.Connector
	getHandler  gimlet.RouteHandler
	postHandler gimlet.RouteHandler

	suite.Suite
}

func TestAdminRouteSuiteWithDB(t *testing.T) {
	s := new(AdminRouteSuite)
	db.SetGlobalSessionProvider(testutil.TestConfig().SessionFactory())
	s.sc = &data.DBConnector{}
	testutil.HandleTestingErr(db.ClearCollections(evergreen.ConfigCollection), t,
		"Error clearing collections")

	// run the rest of the tests
	suite.Run(t, s)
}

func TestAdminRouteSuiteWithMock(t *testing.T) {
	s := new(AdminRouteSuite)
	s.sc = &data.MockConnector{}

	// run the rest of the tests
	suite.Run(t, s)
}

func (s *AdminRouteSuite) SetupSuite() {
	// test getting the route handler
	s.getHandler = makeFetchAdminSettings(s.sc)
	s.postHandler = makeSetAdminSettings(s.sc)
	s.IsType(&adminGetHandler{}, s.getHandler)
	s.IsType(&adminPostHandler{}, s.postHandler)
}

func (s *AdminRouteSuite) TestAdminRoute() {
	ctx := context.Background()
	ctx = gimlet.AttachUser(ctx, &user.DBUser{Id: "user"})

	db.SetGlobalSessionProvider(testutil.TestConfig().SessionFactory())
	s.NoError(db.Clear(distro.Collection))
	d1 := &distro.Distro{
		Id: "valid-distro",
	}
	d2 := &distro.Distro{
		Id:            "invalid-distro",
		ContainerPool: "test-pool-1",
	}
	s.NoError(d1.Insert())
	s.NoError(d2.Insert())

	testSettings := testutil.MockConfig()
	jsonBody, err := json.Marshal(testSettings)
	s.NoError(err)
	buffer := bytes.NewBuffer(jsonBody)
	request, err := http.NewRequest("POST", "/admin/settings", buffer)
	s.NoError(err)
	s.NoError(s.postHandler.Parse(ctx, request))

	// test executing the POST request
	resp := s.postHandler.Run(ctx)
	s.NotNil(resp)
	s.Equal(http.StatusOK, resp.Status())

	// test getting the settings
	s.NoError(s.getHandler.Parse(ctx, nil))
	resp = s.getHandler.Run(ctx)
	s.NotNil(resp)
	settingsResp, err := resp.Data().(restModel.Model).ToService()
	s.NoError(err)
	settings, ok := settingsResp.(evergreen.Settings)
	s.True(ok)

	s.EqualValues(testSettings.Alerts.SMTP.From, settings.Alerts.SMTP.From)
	s.EqualValues(testSettings.Alerts.SMTP.Port, settings.Alerts.SMTP.Port)
	s.Equal(len(testSettings.Alerts.SMTP.AdminEmail), len(settings.Alerts.SMTP.AdminEmail))
	s.EqualValues(testSettings.Amboy.Name, settings.Amboy.Name)
	s.EqualValues(testSettings.Amboy.LocalStorage, settings.Amboy.LocalStorage)
	s.EqualValues(testSettings.Api.HttpListenAddr, settings.Api.HttpListenAddr)
	s.EqualValues(testSettings.AuthConfig.Crowd.Username, settings.AuthConfig.Crowd.Username)
	s.EqualValues(testSettings.AuthConfig.Naive.Users[0].Username, settings.AuthConfig.Naive.Users[0].Username)
	s.EqualValues(testSettings.AuthConfig.Github.ClientId, settings.AuthConfig.Github.ClientId)
	s.Equal(len(testSettings.AuthConfig.Github.Users), len(settings.AuthConfig.Github.Users))
	s.EqualValues(testSettings.ContainerPools.Pools[0].Distro, settings.ContainerPools.Pools[0].Distro)
	s.EqualValues(testSettings.ContainerPools.Pools[0].Id, settings.ContainerPools.Pools[0].Id)
	s.EqualValues(testSettings.ContainerPools.Pools[0].MaxContainers, settings.ContainerPools.Pools[0].MaxContainers)
	s.EqualValues(testSettings.HostInit.SSHTimeoutSeconds, settings.HostInit.SSHTimeoutSeconds)
	s.EqualValues(testSettings.Jira.Username, settings.Jira.Username)
	s.EqualValues(testSettings.LoggerConfig.DefaultLevel, settings.LoggerConfig.DefaultLevel)
	s.EqualValues(testSettings.LoggerConfig.Buffer.Count, settings.LoggerConfig.Buffer.Count)
	s.EqualValues(testSettings.Notify.SMTP.From, settings.Notify.SMTP.From)
	s.EqualValues(testSettings.Notify.SMTP.Port, settings.Notify.SMTP.Port)
	s.Equal(len(testSettings.Notify.SMTP.AdminEmail), len(settings.Notify.SMTP.AdminEmail))
	s.EqualValues(testSettings.Providers.AWS.Id, settings.Providers.AWS.Id)
	s.EqualValues(testSettings.Providers.Docker.APIVersion, settings.Providers.Docker.APIVersion)
	s.EqualValues(testSettings.Providers.GCE.ClientEmail, settings.Providers.GCE.ClientEmail)
	s.EqualValues(testSettings.Providers.OpenStack.IdentityEndpoint, settings.Providers.OpenStack.IdentityEndpoint)
	s.EqualValues(testSettings.Providers.VSphere.Host, settings.Providers.VSphere.Host)
	s.EqualValues(testSettings.RepoTracker.MaxConcurrentRequests, settings.RepoTracker.MaxConcurrentRequests)
	s.EqualValues(testSettings.Scheduler.TaskFinder, settings.Scheduler.TaskFinder)
	s.EqualValues(testSettings.ServiceFlags.HostinitDisabled, settings.ServiceFlags.HostinitDisabled)
	s.EqualValues(testSettings.Slack.Level, settings.Slack.Level)
	s.EqualValues(testSettings.Slack.Options.Channel, settings.Slack.Options.Channel)
	s.EqualValues(testSettings.Splunk.Channel, settings.Splunk.Channel)
	s.EqualValues(testSettings.Ui.HttpListenAddr, settings.Ui.HttpListenAddr)

	// test that invalid input errors
	badSettingsOne := testutil.MockConfig()
	badSettingsOne.ApiUrl = ""
	badSettingsOne.Ui.CsrfKey = "12345"
	jsonBody, err = json.Marshal(badSettingsOne)
	s.NoError(err)
	buffer = bytes.NewBuffer(jsonBody)
	request, err = http.NewRequest("POST", "/admin", buffer)
	s.NoError(err)
	s.NoError(s.postHandler.Parse(ctx, request))
	resp = s.postHandler.Run(ctx)
	s.Contains(resp.Data().(gimlet.ErrorResponse).Message, "API hostname must not be empty")
	s.Contains(resp.Data().(gimlet.ErrorResponse).Message, "CSRF key must be 32 characters long")
	s.NotNil(resp)

	// test that invalid container pools errors
	badSettingsTwo := testutil.MockConfig()
	badSettingsTwo.ContainerPools.Pools = []evergreen.ContainerPool{
		evergreen.ContainerPool{
			Distro:        "valid-distro",
			Id:            "test-pool-1",
			MaxContainers: 100,
		},
		evergreen.ContainerPool{
			Distro:        "invalid-distro",
			Id:            "test-pool-2",
			MaxContainers: 100,
		},
		evergreen.ContainerPool{
			Distro:        "missing-distro",
			Id:            "test-pool-3",
			MaxContainers: 100,
		},
	}
	jsonBody, err = json.Marshal(badSettingsTwo)
	s.NoError(err)
	buffer = bytes.NewBuffer(jsonBody)
	request, err = http.NewRequest("POST", "/admin", buffer)
	s.NoError(err)
	s.NoError(s.postHandler.Parse(ctx, request))
	resp = s.postHandler.Run(ctx)
	s.Contains(resp.Data().(gimlet.ErrorResponse).Message, "container pool test-pool-2 has invalid distro")
	s.Contains(resp.Data().(gimlet.ErrorResponse).Message, "error finding distro for container pool test-pool-3")
	s.NotNil(resp)
}

func (s *AdminRouteSuite) TestRevertRoute() {
	const route = "/admin/revert"
	const version = 2

	routeManager := makeRevertRouteManager(s.sc)
	user := &user.DBUser{Id: "userName"}
	ctx := gimlet.AttachUser(context.Background(), user)
	s.NotNil(routeManager)
	changes := restModel.APIAdminSettings{
		SuperUsers: []string{"me"},
	}
	before := evergreen.Settings{}
	_, err := s.sc.SetEvergreenSettings(&changes, &before, user, true)
	s.NoError(err)
	dbEvents, err := event.FindAdmin(event.RecentAdminEvents(1))
	s.NoError(err)
	eventData := dbEvents[0].Data.(*event.AdminEventData)
	guid := eventData.GUID
	s.NotEmpty(guid)

	body := struct {
		GUID string `json:"guid"`
	}{guid}
	jsonBody, err := json.Marshal(&body)
	s.NoError(err)
	buffer := bytes.NewBuffer(jsonBody)
	request, err := http.NewRequest("POST", "/admin/revert", buffer)
	s.NoError(err)
	err = routeManager.Parse(ctx, request)
	s.NoError(err)
	resp := routeManager.Run(ctx)
	s.NotNil(resp)
	s.Equal(http.StatusOK, resp.Status())
	body = struct {
		GUID string `json:"guid"`
	}{""}
	jsonBody, err = json.Marshal(&body)
	s.NoError(err)
	buffer = bytes.NewBuffer(jsonBody)
	request, err = http.NewRequest("POST", "/admin/revert", buffer)
	s.NoError(err)
	err = routeManager.Parse(ctx, request)
	s.Error(err)
	s.NotNil(ctx)
}

func TestRestartRoute(t *testing.T) {
	assert := assert.New(t)

	ctx := gimlet.AttachUser(context.Background(), &user.DBUser{Id: "userName"})

	queue := evergreen.GetEnvironment().LocalQueue()
	sc := &data.MockConnector{}
	handler := makeRestartRoute(sc, queue)

	assert.NotNil(handler)

	startTime := time.Date(2017, time.June, 12, 11, 0, 0, 0, time.Local)
	endTime := time.Date(2017, time.June, 12, 13, 0, 0, 0, time.Local)

	// test that invalid time range errors
	body := struct {
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		DryRun    bool      `json:"dry_run"`
	}{endTime, startTime, false}
	jsonBody, err := json.Marshal(&body)
	assert.NoError(err)
	buffer := bytes.NewBuffer(jsonBody)
	request, err := http.NewRequest("POST", "/admin/restart", buffer)
	assert.NoError(err)
	assert.Error(handler.Parse(ctx, request))

	// test a valid request
	body = struct {
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		DryRun    bool      `json:"dry_run"`
	}{startTime, endTime, false}
	jsonBody, err = json.Marshal(&body)
	assert.NoError(err)
	buffer = bytes.NewBuffer(jsonBody)
	request, err = http.NewRequest("POST", "/admin/restart", buffer)
	assert.NoError(err)
	assert.NoError(handler.Parse(ctx, request))
	resp := handler.Run(ctx)
	assert.NotNil(resp)
	model, ok := resp.Data().(*restModel.RestartTasksResponse)
	assert.True(ok)
	assert.True(len(model.TasksRestarted) > 0)
	assert.Nil(model.TasksErrored)
}

func TestAdminEventRoute(t *testing.T) {
	assert := assert.New(t)
	db.SetGlobalSessionProvider(testutil.TestConfig().SessionFactory())
	testutil.HandleTestingErr(db.ClearCollections(evergreen.ConfigCollection, event.AllLogCollection), t,
		"Error clearing collections")

	// log some changes in the event log with the /admin/settings route
	ctx := context.Background()
	ctx = gimlet.AttachUser(ctx, &user.DBUser{Id: "user"})
	routeManager := makeSetAdminSettings(&data.DBConnector{})

	testSettings := testutil.MockConfig()
	jsonBody, err := json.Marshal(testSettings)
	assert.NoError(err)
	buffer := bytes.NewBuffer(jsonBody)
	request, err := http.NewRequest("POST", "/admin/settings", buffer)
	assert.NoError(err)
	assert.NoError(routeManager.Parse(ctx, request))
	now := time.Now()
	resp := routeManager.Run(ctx)
	assert.NotNil(resp)
	assert.Equal(http.StatusOK, resp.Status())

	// get the changes with the /admin/events route
	ctx = context.Background()
	route := makeFetchAdminEvents(&data.DBConnector{
		URL: "http://evergreen.example.net",
	})
	request, err = http.NewRequest("GET", "/admin/events?limit=10&ts=2026-01-02T15%3A04%3A05Z", nil)
	assert.NoError(err)
	assert.NoError(route.Parse(ctx, request))
	response := route.Run(ctx)
	assert.NotNil(resp)
	count := 0
	fmt.Printf("%+v\n", response)
	data := response.Data().([]interface{})
	for _, model := range data {
		evt, ok := model.(*restModel.APIAdminEvent)
		assert.True(ok)
		count++
		assert.NotEmpty(evt.Guid)
		assert.NotNil(evt.Before)
		assert.NotNil(evt.After)
		assert.Equal("user", evt.User)
	}
	assert.Equal(10, count)
	pagination := response.Pages().Next
	assert.Equal("ts", pagination.KeyQueryParam)
	assert.Equal("limit", pagination.LimitQueryParam)
	assert.Equal("next", pagination.Relation)
	assert.Equal(10, pagination.Limit)
	ts, err := time.Parse(time.RFC3339, pagination.Key)
	assert.NoError(err)
	assert.InDelta(now.Unix(), ts.Unix(), float64(time.Millisecond.Nanoseconds()))
}

func TestClearTaskQueueRoute(t *testing.T) {
	assert := assert.New(t)
	route := &clearTaskQueueHandler{
		sc: &data.DBConnector{},
	}
	distro := "d1"
	tasks := []model.TaskQueueItem{
		{
			Id: "task1",
		},
		{
			Id: "task2",
		},
		{
			Id: "task3",
		},
	}
	queue := model.NewTaskQueue(distro, tasks)
	assert.Len(queue.Queue, 3)
	assert.NoError(queue.Save())

	route.distro = distro
	resp := route.Run(context.Background())
	assert.Equal(http.StatusOK, resp.Status())

	queueFromDb, err := model.LoadTaskQueue(distro)
	assert.NoError(err)
	assert.Len(queueFromDb.Queue, 0)
}