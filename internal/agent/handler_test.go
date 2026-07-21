package agent

import "testing"

func TestCleanJSON(t *testing.T) {
	cases := map[string]string{
		`{"tool":"x","args":{}}`:                                       `{"tool":"x","args":{}}`,
		"```json\n{\"tool\":\"x\"}\n```":                              `{"tool":"x"}`,
		"<think>reasoning here</think>\n{\"tool\":\"list\"}":           `{"tool":"list"}`,
		"<think>a</think> ```json\n{\"tool\":\"q\",\"args\":{}}\n```": `{"tool":"q","args":{}}`,
		"blah {\"tool\":\"x\"} trailing":                              `{"tool":"x"}`,
	}
	for in, want := range cases {
		if got := cleanJSON(in); got != want {
			t.Errorf("cleanJSON(%q) = %q, want %q", in, got, want)
		}
	}
}
