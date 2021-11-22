package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func withAWSCLI(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_WITH_AWSCLI") != "true" {
		t.SkipNow()
	}
}

func TestHandleWithAWSCLI(t *testing.T) {
	withAWSCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := handler(ctx, &RedshiftUDFInput{
		NumRecords: 6,
		Arguments: [][]interface{}{
			{nil},
			{"aws comprehend detect-dominant-language --region ap-northeast-1 --text 'this is a pen.'"},
			{"aws comprehend detect-dominant-language --region ap-northeast-1 --text 'これはペンです。';"},
			{"aws comprehend detect-dominant-language --region ap-northeast-1 --text 'hoc est stylo'|jq -c '.hoge'"},
			{"aws comprehend detect-dominant-language --region ap-northeast-1 --text 'dies ist ein Stift'& echo piyo"},
			{"aws comprehend detect-dominant-language --region ap-northeast-1 --text 'これはペンです。'&&echo hoge"},
		},
	})
	require.NoError(t, err, "handle expect no error")
	t.Log("output.Success", output.Success)
	t.Log("output.ErrorMsg", output.ErrorMsg)
	t.Log("output.Resutls", toJSON(t, output.Results))
	require.True(t, output.Success)
	require.Equal(t, 6, output.NumRecords, "num")
	require.Nil(t, output.Results[0], "nil text return nil")
	excepted := []interface{}{
		map[string]interface{}{
			"Languages": []interface{}{
				map[string]interface{}{
					"LanguageCode": "en",
					"Score":        0.9984308481216431,
				},
			},
		},
		map[string]interface{}{
			"Languages": []interface{}{
				map[string]interface{}{
					"LanguageCode": "ja",
					"Score":        1.00,
				},
			},
		},
		map[string]interface{}{
			"Languages": []interface{}{
				map[string]interface{}{
					"LanguageCode": "la",
					"Score":        0.9894738793373108,
				},
			},
		},
		map[string]interface{}{
			"Languages": []interface{}{
				map[string]interface{}{
					"LanguageCode": "de",
					"Score":        0.9999744296073914,
				},
			},
		},
		map[string]interface{}{
			"Languages": []interface{}{
				map[string]interface{}{
					"LanguageCode": "ja",
					"Score":        1.00,
				},
			},
		},
	}
	for i := 0; i < 3; i++ {
		require.EqualValues(t, excepted[i], output.Results[i+1])
	}
}

func toJSON(t *testing.T, v interface{}) string {
	t.Helper()
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func fromJSON(t *testing.T, str string) interface{} {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(str))
	var v interface{}
	if err := decoder.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}
