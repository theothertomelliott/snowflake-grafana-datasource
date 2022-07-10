package main

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestCreateAndValidationConnectionString(t *testing.T) {

	tcs := []struct {
		request          *backend.CheckHealthRequest
		result           *backend.CheckHealthResult
		connectionString string
	}{
		{
			request: &backend.CheckHealthRequest{
				PluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						DecryptedSecureJSONData: map[string]string{"password": ""},
					},
				},
			},
			result: &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: "Password or private key are required."},
		},
		{
			request: &backend.CheckHealthRequest{
				PluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						JSONData:                []byte("{"),
						DecryptedSecureJSONData: map[string]string{"password": "pass"},
					},
				},
			},
			result: &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: "Error getting config: unexpected end of JSON input"},
		},
		{
			request: &backend.CheckHealthRequest{
				PluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						JSONData:                []byte("{}"),
						DecryptedSecureJSONData: map[string]string{"password": "pass"},
					},
				},
			},
			result: &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: "Account not provided"},
		},
		{
			request: &backend.CheckHealthRequest{
				PluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						JSONData:                []byte("{\"account\":\"test\"}"),
						DecryptedSecureJSONData: map[string]string{"password": "pass"},
					},
				},
			},
			result: &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: "Username not provided"},
		},
		{
			request: &backend.CheckHealthRequest{
				PluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						JSONData:                []byte("{\"account\":\"test\",\"username\":\"user\"}"),
						DecryptedSecureJSONData: map[string]string{"password": "pass"},
					},
				},
			},
			connectionString: "user:pass@test?QUERY_TAG=%7B%22job%22%3A%22Grafana%22%2C%22org_id%22%3A0%2C%22is_backend%22%3Atrue%7D&database=&role=&schema=&warehouse=&validateDefaultParameters=true",
		},
		{
			request: &backend.CheckHealthRequest{
				PluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						JSONData:                []byte("{\"account\":\"test\",\"username\":\"user\",\"extraConfig\":\"config=conf\"}"),
						DecryptedSecureJSONData: map[string]string{"password": "pass"},
					},
				},
			},
			connectionString: "user:pass@test?QUERY_TAG=%7B%22job%22%3A%22Grafana%22%2C%22org_id%22%3A0%2C%22is_backend%22%3Atrue%7D&database=&role=&schema=&warehouse=&config=conf&validateDefaultParameters=true",
		},
	}
	for i, tc := range tcs {
		t.Run(fmt.Sprintf("testcase %d", i), func(t *testing.T) {
			con, result := createAndValidationConnectionString(tc.request)
			if result == nil {
				require.Equal(t, tc.connectionString, con)
			} else {
				require.Equal(t, tc.result, result)
			}
		})
	}
}
