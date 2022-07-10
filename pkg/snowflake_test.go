package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestGetConfig(t *testing.T) {

	tcs := []struct {
		json     string
		config   pluginConfig
		response string
		err      string
	}{
		{json: "{}", config: pluginConfig{}},
		{json: "{\"account\":\"test\"}", config: pluginConfig{Account: "test"}},
		{json: "{", err: "unexpected end of JSON input"},
	}
	for i, tc := range tcs {
		t.Run(fmt.Sprintf("testcase %d", i), func(t *testing.T) {
			configStruct := backend.DataSourceInstanceSettings{
				JSONData: []byte(tc.json),
			}
			conf, err := getConfig(&configStruct)
			if tc.err == "" {
				require.NoError(t, err)
				require.Equal(t, tc.config, conf)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.err, err.Error())
			}
		})
	}
}

func TestGetConnectionString(t *testing.T) {

	config := pluginConfig{
		Account:     "account",
		Database:    "database",
		Role:        "role",
		Schema:      "schema",
		Username:    "username",
		Warehouse:   "warehouse",
		ExtraConfig: "conf=xxx",
	}

	t.Run("with User/pass", func(t *testing.T) {
		connectionString := getConnectionString(&config, "password", "", "")
		require.Equal(t, "username:password@account?database=database&role=role&schema=schema&warehouse=warehouse&conf=xxx", connectionString)
	})

	t.Run("with private key", func(t *testing.T) {
		connectionString := getConnectionString(&config, "", "privateKey", "")
		require.Equal(t, "username@account?authenticator=SNOWFLAKE_JWT&database=database&privateKey=privateKey&role=role&schema=schema&warehouse=warehouse&conf=xxx", connectionString)
	})

	t.Run("with User/pass special char", func(t *testing.T) {
		connectionString := getConnectionString(&config, "p@sswor/d", "", "")
		require.Equal(t, "username:p%40sswor%2Fd@account?database=database&role=role&schema=schema&warehouse=warehouse&conf=xxx", connectionString)
	})

	t.Run("with query tag", func(t *testing.T) {
		connectionString := getConnectionString(&config, "p@sswor/d", "", "mytag")
		require.Equal(t, "username:p%40sswor%2Fd@account?QUERY_TAG=mytag&database=database&role=role&schema=schema&warehouse=warehouse&conf=xxx", connectionString)
	})

	config = pluginConfig{
		Account:     "acc@ount",
		Database:    "dat@base",
		Role:        "ro@le",
		Schema:      "sch@ema",
		Username:    "user@name",
		Warehouse:   "ware@house",
		ExtraConfig: "conf=xxx",
	}

	t.Run("with string to escape", func(t *testing.T) {
		connectionString := getConnectionString(&config, "pa$$s&", "", "")
		require.Equal(t, "user%40name:pa$$s&@acc@ount?database=dat%40base&role=ro%40le&schema=sch%40ema&warehouse=ware%40house&conf=xxx", connectionString)
	})
}

func TestBuildQueryTag(t *testing.T) {
	var tests = []struct {
		name     string
		pc       backend.PluginContext
		expected queryTagData
	}{
		{
			name: "backend",
			pc: backend.PluginContext{
				OrgID: 123,
			},
			expected: queryTagData{
				Job:       "Grafana",
				OrgId:     123,
				IsBackend: true,
			},
		},
		{
			name: "anonymous",
			pc: backend.PluginContext{
				OrgID: 123,
				User:  &backend.User{},
			},
			expected: queryTagData{
				Job:         "Grafana",
				OrgId:       123,
				IsAnonymous: true,
			},
		},
		{
			name: "authenticated user",
			pc: backend.PluginContext{
				OrgID: 123,
				User: &backend.User{
					Name:  "Firstname Lastname",
					Login: "auserlogin",
					Email: "someone@example.com",
				},
			},
			expected: queryTagData{
				Job:       "Grafana",
				OrgId:     123,
				UserName:  "Firstname Lastname",
				UserLogin: "auserlogin",
				UserEmail: "someone@example.com",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			queryTag, err := queryTagFromContext(test.pc)
			// An error indicates something went wrong marshaling the JSON,
			// which always means some kind of bug.
			if err != nil {
				t.Error(err)
				return
			}

			var got queryTagData
			err = json.Unmarshal([]byte(queryTag), &got)
			if err != nil {
				t.Error(err)
				return
			}

			if got != test.expected {
				t.Errorf("expected %+v, got %+v", test.expected, got)
			}
		})
	}

}
