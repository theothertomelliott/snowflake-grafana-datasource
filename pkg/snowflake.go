package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// newDatasource returns datasource.ServeOpts.
func newDatasource() datasource.ServeOpts {
	// creates a instance manager for your plugin. The function passed
	// into `NewInstanceManger` is called when the instance is created
	// for the first time or when a datasource configuration changed.
	im := datasource.NewInstanceManager(newDataSourceInstance)
	ds := &SnowflakeDatasource{
		im: im,
	}

	return datasource.ServeOpts{
		QueryDataHandler:   ds,
		CheckHealthHandler: ds,
	}
}

type SnowflakeDatasource struct {
	// The instance manager can help with lifecycle management
	// of datasource instances in plugins. It's not a requirements
	// but a best practice that we recommend that you follow.
	im instancemgmt.InstanceManager
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifer).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (td *SnowflakeDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {

	// create response struct
	response := backend.NewQueryDataResponse()

	password := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["password"]
	privateKey := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["privateKey"]

	config, err := getConfig(req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		log.DefaultLogger.Error("Could not get config for plugin", "err", err)
		return response, err
	}

	queryTag, err := queryTagFromContext(req.PluginContext)
	if err != nil {
		return response, err
	}

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = td.query(q, config, password, privateKey, queryTag)
	}

	return response, nil
}

// queryTagFromContext builds JSON-formatted string to be applied to the QUERY_TAG parameter
// for the session (https://docs.snowflake.com/en/sql-reference/parameters.html#query-tag).
//
// This tag will include information about the context in which the query was executed
// such as the user who requested it. This can be used for auditing.
func queryTagFromContext(pc backend.PluginContext) (string, error) {
	qtd := queryTagData{
		Job:   "Grafana",
		OrgId: pc.OrgID,
	}

	// Add user information as appropriate.
	// If the User is nil, this is a backend request (such as for alert evaluation).
	// If the User is not nil, but only has a role, this is an anonymous request.
	if pc.User != nil {
		if u := *pc.User; (backend.User{Role: u.Role}) == u {
			qtd.IsAnonymous = true
		}
		qtd.UserName = pc.User.Name
		qtd.UserLogin = pc.User.Login
		qtd.UserEmail = pc.User.Email
		qtd.UserRole = pc.User.Role
	} else {
		qtd.IsBackend = true
	}

	queryTagBytes, err := json.Marshal(qtd)
	if err != nil {
		return "", err
	}
	return string(queryTagBytes), nil
}

// queryTagData is the data type representing the QUERY_TAG JSON
// format.
type queryTagData struct {
	Job         string `json:"job"`
	OrgId       int64  `json:"org_id"`
	UserLogin   string `json:"user_login,omitempty"`
	UserName    string `json:"user_name,omitempty"`
	UserEmail   string `json:"user_email,omitempty"`
	UserRole    string `json:"user_role,omitempty"`
	IsBackend   bool   `json:"is_backend,omitempty"`
	IsAnonymous bool   `json:"is_anonymous,omitempty"`
}

type pluginConfig struct {
	Account     string `json:"account"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	Warehouse   string `json:"warehouse"`
	Database    string `json:"database"`
	Schema      string `json:"schema"`
	ExtraConfig string `json:"extraConfig"`
}

func getConfig(settings *backend.DataSourceInstanceSettings) (pluginConfig, error) {
	var config pluginConfig
	err := json.Unmarshal(settings.JSONData, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func getConnectionString(config *pluginConfig, password string, privateKey string, queryTag string) string {
	params := url.Values{}
	params.Add("role", config.Role)
	params.Add("warehouse", config.Warehouse)
	params.Add("database", config.Database)
	params.Add("schema", config.Schema)
	if queryTag != "" {
		params.Add("QUERY_TAG", queryTag)
	}

	var userPass = ""
	if len(privateKey) != 0 {
		params.Add("authenticator", "SNOWFLAKE_JWT")
		params.Add("privateKey", privateKey)
		userPass = url.User(config.Username).String()
	} else {
		userPass = url.UserPassword(config.Username, password).String()
	}

	return fmt.Sprintf("%s@%s?%s&%s", userPass, config.Account, params.Encode(), config.ExtraConfig)
}

type instanceSettings struct {
	httpClient *http.Client
}

func newDataSourceInstance(setting backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	log.DefaultLogger.Info("Creating instance")
	return &instanceSettings{
		httpClient: &http.Client{},
	}, nil
}

func (s *instanceSettings) Dispose() {
	log.DefaultLogger.Info("Disposing of instance")
}
