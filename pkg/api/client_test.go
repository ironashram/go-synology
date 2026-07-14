package api

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/synology-community/go-synology/pkg/util"
)

func TestMarshalURL(t *testing.T) {
	type embeddedStruct struct {
		EmbeddedString string `form:"embedded_string" url:"embedded_string"`
		EmbeddedInt    int    `form:"embedded_int"    url:"embedded_int"`
	}

	testCases := []struct {
		name     string
		in       any
		expected url.Values
	}{
		{
			name: "scalar types",
			in: struct {
				Name    string `form:"name" url:"name"`
				ID      int    `form:"id" url:"id"`
				Enabled bool   `form:"enabled" url:"enabled"`
			}{
				Name:    "name value",
				ID:      2,
				Enabled: true,
			},
			expected: url.Values{
				"name":    []string{"name value"},
				"id":      []string{"2"},
				"enabled": []string{"true"},
			},
		},
		{
			name: "slice types",
			in: struct {
				Names []string `form:"names" url:"names"`
				IDs   []int    `form:"ids" url:"ids"`
			}{
				Names: []string{"value 1", "value 2"},
				IDs:   []int{1, 2, 3},
			},
			expected: url.Values{
				"names": []string{"[\"value 1\",\"value 2\"]"},
				"ids":   []string{"[1,2,3]"},
			},
		},
		{
			name: "embedded struct",
			in: struct {
				embeddedStruct
				Name string `form:"name" url:"name"`
			}{
				embeddedStruct: embeddedStruct{
					EmbeddedString: "my string",
					EmbeddedInt:    5,
				},
				Name: "field name",
			},
			expected: url.Values{
				"name":            []string{"field name"},
				"embedded_string": []string{"my string"},
				"embedded_int":    []string{"5"},
			},
		},
		{
			name: "unexported field without tag",
			in: struct {
				Name       string `form:"name" url:"name"`
				ID         int    `form:"id" url:"id"`
				unexported string
			}{
				Name:       "name value",
				ID:         2,
				unexported: "must be skipped",
			},
			expected: url.Values{
				"name": []string{"name value"},
				"id":   []string{"2"},
			},
		},
		{
			name: "unexported field with tag",
			in: struct {
				Name       string `form:"name" url:"name"`
				ID         int    `form:"id" url:"id"`
				unexported string `form:"unexported" url:"unexported"`
			}{
				Name:       "name value",
				ID:         2,
				unexported: "with explicit tag",
			},
			expected: url.Values{
				"name":       []string{"name value"},
				"id":         []string{"2"},
				"unexported": []string{"with explicit tag"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := util.MarshalURL(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHandleErrors(t *testing.T) {
	globalErrors := func() ErrorSummary {
		return ErrorSummary{
			100: "global error 100",
			101: "global error 101",
			102: "global error 102",
		}
	}

	testCases := []struct {
		name                string
		response            ApiResponse[Response]
		responseKnownErrors []ErrorSummary
		expected            ApiError
	}{
		{
			name: "global errors only",
			response: ApiResponse[Response]{
				Success: false,
				Data:    struct{}{},
				Error: ApiError{
					Code: 100,
					Errors: []ErrorFields{
						{
							Code:   102,
							Fields: map[string]any{"path": "/some/path", "reason": "a reason"},
						},
					},
				},
			},
			expected: ApiError{
				Code:    100,
				Summary: "global error 100",
				Errors: []ErrorFields{
					{Code: 102, Fields: map[string]any{"path": "/some/path", "reason": "a reason"}},
				},
			},
		},
		{
			name: "response-specific error",
			response: ApiResponse[Response]{
				Success: false,
				Data:    struct{}{},
				Error: ApiError{
					Code: 100,
					Errors: []ErrorFields{
						{Code: 202, Fields: map[string]any{}},
					},
				},
			},
			responseKnownErrors: []ErrorSummary{
				{
					202: "response error 202",
				},
			},
			expected: ApiError{
				Code:    100,
				Summary: "global error 100",
				Errors: []ErrorFields{
					{Code: 202, Fields: map[string]any{}},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := handleErrors(tc.response,
				globalErrors,
			)
			assert.Error(t, actual)
		})
	}
}
