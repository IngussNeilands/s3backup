package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const loginJSON = `{
  "auth": {
    "renewable": true,
    "lease_duration": 1200,
    "metadata": null,
    "policies": [
      "default"
    ],
    "accessor": "fd6c9a00-d2dc-3b11-0be5-af7ae0e1d374",
    "client_token": "5b1a0318-679c-9c45-e5c6-d1b9a9035d49"
  },
  "warnings": null,
  "wrap_info": null,
  "data": null,
  "lease_duration": 0,
  "renewable": false,
  "lease_id": ""
}`

const secretJSON = `{
  "auth": null,
  "data": {
    "cipher_key": "use me to encrypt",
    "s3_access_key": "aws access",
    "s3_secret_key": "aws secret",
    "s3_token": "aws token",
    "s3_region": "us-east-1",
    "s3_endpoint": "https://spaces.test"
  },
  "lease_duration": 2764800,
  "lease_id": "",
  "renewable": false
}`

func checkLogin(w http.ResponseWriter, r *http.Request) {
	m := make(map[string]interface{})
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		return
	}
	role := m["role_id"].(string)
	secret := m["secret_id"].(string)
	if role != "test-role" || secret != "test-secret" {
		http.Error(w, "Unknown role/secret IDs", http.StatusForbidden)
		return
	}
	fmt.Fprintln(w, loginJSON)
}

func respondWith(body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, body)
	}
}

func testHandler() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/v1/auth/approle/login", checkLogin).
		Methods("PUT")

	r.HandleFunc("/v1/secret/myteam/backup", respondWith(secretJSON)).
		Headers("X-Vault-Token", "5b1a0318-679c-9c45-e5c6-d1b9a9035d49").
		Methods("GET")

	return r
}

func TestVaultLookup(t *testing.T) {
	ts := httptest.NewServer(testHandler())
	defer ts.Close()

	v, err := NewVault(ts.URL, "")
	require.NoError(t, err)

	cfg, err := v.LookupWithAppRole("test-role", "test-secret", "secret/myteam/backup")
	require.NoError(t, err)

	assert.Equal(t, "use me to encrypt", cfg.CipherKey)
	assert.Equal(t, "aws access", cfg.S3AccessKey)
	assert.Equal(t, "aws secret", cfg.S3SecretKey)
	assert.Equal(t, "aws token", cfg.S3Token)
	assert.Equal(t, "us-east-1", cfg.S3Region)
	assert.Equal(t, "https://spaces.test", cfg.S3Endpoint)
}
