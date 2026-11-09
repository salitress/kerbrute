package util

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Result is one finding, written as a single line of newline-delimited JSON
// to stdout when --json is set, so output can be piped straight into jq or
// another tool instead of scraped from the colorized log lines.
type Result struct {
	Type      string `json:"type"` // "login", "username" or "asrep_hash"
	Username  string `json:"username"`
	Domain    string `json:"domain"`
	Detail    string `json:"detail,omitempty"`
	Hash      string `json:"hash,omitempty"`
	Timestamp string `json:"timestamp"`
}

func EmitJSON(resultType, username, domain, detail, hash string) {
	r := Result{
		Type:      resultType,
		Username:  username,
		Domain:    domain,
		Detail:    detail,
		Hash:      hash,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	b, err := json.Marshal(r)
	if err != nil {
		return
	}
	fmt.Fprintln(os.Stdout, string(b))
}
