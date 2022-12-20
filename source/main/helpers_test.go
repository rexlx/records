package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_readJSON(t *testing.T) {
	var decodedJson struct {
		Foo string `json:"foo"`
	}
	// create sample json
	jason := map[string]interface{}{
		"foo": "bar",
	}
	body, _ := json.Marshal(jason)
	// create req
	req, err := http.NewRequest("POST", "/", bytes.NewReader(body))
	if err != nil {
		t.Log(err)
	}

	// need  a test recorder
	rr := httptest.NewRecorder()
	defer req.Body.Close()

	err = testApp.readJSON(rr, req, &decodedJson)
	if err != nil {
		t.Error("coudlnt decode json", err)
	}

}

func Test_writeJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	data := jsonResponse{
		Error:   false,
		Message: "yeyeyeyeye",
	}
	headers := make(http.Header)
	headers.Add("YO", "MAN")
	err := testApp.writeJSON(rr, http.StatusOK, data, headers)
	if err != nil {
		t.Errorf("test failed %v", err)
	}
	err = testApp.writeJSON(rr, http.StatusOK, data, headers)
	if err != nil {
		t.Errorf("test failed (env specific) %v", err)
	}
}

func Test_errorJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	err := testApp.errorJSON(rr, errors.New("your fink isn't in the bluq"))
	if err != nil {
		t.Error(err)
	}
}

func TestSanitizeServiceName(t *testing.T) {
	type test struct {
		input    string
		expected string
	}
	tests := []test{
		{input: "test service", expected: "test_service"},
		{input: "Best Service", expected: "Best_Service"},
		{input: "another_service", expected: "another_service"},
		{input: "broke\tservice", expected: "broke	service"},
	}
	for _, tc := range tests {
		output := SanitizeServiceName(tc.input)
		if output != tc.expected {
			t.Errorf("expected %v, got %v", tc.expected, output)
		}
	}
}

func Test_createApiKey(t *testing.T) {
	if testApp.ApiKey != "" {
		t.Errorf("api key should be empty")
	}
	testApp.createApiKey()
	if testApp.ApiKey == "" {
		t.Errorf("empty api key!")
	}
}

func Test_genRandomString(t *testing.T) {
	x, _ := genRandomString()
	if len(x) != 40 {
		t.Errorf("genRandomString made a naughty string, length is %v of 40", len(x))
	}
}

func Test_serviceValidator(t *testing.T) {
	tests := []serviceDetails{
		{Runtime: 0, Refresh: 2, Name: "one"},
		{Runtime: 0, Refresh: 0, Name: "two"},
		{Runtime: 2, Refresh: 2, Name: "three"},
		{Runtime: 2, Refresh: -2, Name: "four"},
	}
	for _, tc := range tests {
		if err := serviceValidator(&tc); err != nil {
			if tc.Runtime > 0 && tc.Refresh > 0 {
				t.Errorf("serviceValidator returned unexpected result")
			}
		} else {
			if tc.Runtime < 1 && tc.Refresh < 1 {
				t.Errorf("serviceValidator returned unexpected result")
			}
		}
	}

}
