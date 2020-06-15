package cgi

import (
	"crypto/subtle"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandler(t *testing.T) {
	type test struct {
		Name           string
		Script         string
		ExpectedStatus int
		ExpectedHeader http.Header
		ExpectedBody   string
		OutputHandler  OutputHandler
	}

	tt := []test{
		test{
			Name:           "Default headers default handler",
			Script:         "./noheaders.sh",
			ExpectedStatus: http.StatusInternalServerError,
			OutputHandler:  DefaultOutputHandler,
		},
		test{
			Name:           "Default headers",
			Script:         "./noheaders.sh",
			ExpectedStatus: http.StatusOK,
			ExpectedHeader: http.Header{
				"Content-Type": []string{"text/plain"},
			},
			ExpectedBody:  "./expected_body",
			OutputHandler: EZOutputHandlerReplacer,
		},
		test{
			Name:           "Replace Headers",
			Script:         "./headers.sh",
			ExpectedStatus: http.StatusOK,
			ExpectedHeader: http.Header{
				"Content-Type": []string{"text/html"},
				"Test-Header":  []string{"PASS"},
			},
			ExpectedBody:  "./expected_body",
			OutputHandler: EZOutputHandlerReplacer,
		},
		test{
			Name:           "Headers with no blank line",
			Script:         "./headers_noblank.sh",
			ExpectedStatus: http.StatusOK,
			ExpectedHeader: http.Header{
				"Content-Type": []string{"text/html"},
				"Test-Header":  []string{"PASS"},
			},
			ExpectedBody:  "./expected_body",
			OutputHandler: EZOutputHandlerReplacer,
		},
		test{
			Name:           "Non-empty request body",
			Script:         "./requestbody.sh",
			ExpectedStatus: http.StatusOK,
			ExpectedBody:   "./expected_body",
			OutputHandler:  EZOutputHandlerReplacer,
		},
	}

	err := os.Chdir("./test-assets")
	if err != nil {
		t.Fatalf("error while moving into test-assets directory: %s", err)
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			h := &Handler{
				Path:          tc.Script,
				Dir:           ".",
				OutputHandler: tc.OutputHandler,
			}

			r := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			result := w.Result()

			if result.StatusCode != tc.ExpectedStatus {
				t.Fatalf("wrong status - expected: %d\treceived: %d", tc.ExpectedStatus, result.StatusCode)
			}

			if tc.ExpectedStatus != http.StatusOK {
				return
			}

			for k, _ := range tc.ExpectedHeader {
				received := result.Header.Get(k)
				expected := tc.ExpectedHeader.Get(k)
				if received != expected {
					t.Fatalf("wrong header: %s - expected: %s\treceived: %s",
						k, expected, received)
				}
			}

			expectedBytes, err := ioutil.ReadFile(tc.ExpectedBody)
			if err != nil {
				t.Fatalf("error while opening expected body file: %s", err)
			}
			receivedBytes, err := ioutil.ReadAll(result.Body)
			if err != nil {
				t.Fatalf("error while opening expected body file: %s", err)
			}

			bodyDiffs := subtle.ConstantTimeCompare(expectedBytes, receivedBytes)
			if bodyDiffs != 0 {
				t.Fatalf("wrong body - expected: %s\treceived: %s", string(expectedBytes), string(receivedBytes))
			}
		})
	}
}
