package runner

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

// PrintText outputs results in human-readable format.
func PrintText(w io.Writer, results []Result, verbose bool) {
	totalPassed := 0
	totalFailed := 0
	totalErrors := 0

	for _, r := range results {
		icon := "\u2713" // checkmark
		if r.Error != nil {
			icon = "\u2717" // x mark
			totalErrors++
		} else if !r.TestsPassed {
			icon = "\u2717"
		}

		sizeStr := formatSize(r.Size)
		durationStr := formatDuration(r.Duration)

		if r.Error != nil {
			fmt.Fprintf(w, "%s %-20s %-6s %-40s  %-10s %s\n",
				icon, truncate(r.Name, 20), r.Method, truncate(r.URL, 40),
				durationStr, sizeStr)
			fmt.Fprintf(w, "  \u2514 Error: %s\n", r.Error)
		} else {
			statusStr := fmt.Sprintf("%d %s", r.StatusCode, statusText(r.StatusCode))
			fmt.Fprintf(w, "%s %-20s %-6s %-40s  %s  %s  %s\n",
				icon, truncate(r.Name, 20), r.Method, truncate(r.URL, 40),
				statusStr, durationStr, sizeStr)
		}

		// Print test results
		for _, tr := range r.TestResults {
			if tr.Passed {
				totalPassed++
				fmt.Fprintf(w, "  \u2713 %s\n", tr.Name)
			} else {
				totalFailed++
				fmt.Fprintf(w, "  \u2717 %s: %s\n", tr.Name, tr.Error)
			}
		}

		// Print script logs in verbose mode
		if verbose && len(r.ScriptLogs) > 0 {
			for _, log := range r.ScriptLogs {
				fmt.Fprintf(w, "  [log] %s\n", log)
			}
		}

		// Print response body in verbose mode
		if verbose && len(r.Body) > 0 {
			fmt.Fprintf(w, "  --- Response Body ---\n")
			body := string(r.Body)
			for _, line := range strings.Split(body, "\n") {
				fmt.Fprintf(w, "  %s\n", line)
			}
			fmt.Fprintf(w, "  ---------------------\n")
		}
	}

	// Summary
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Requests: %d total, %d errors\n", len(results), totalErrors)
	if totalPassed+totalFailed > 0 {
		fmt.Fprintf(w, "Tests: %d passed, %d failed\n", totalPassed, totalFailed)
	}
}

// PrintJSON outputs results as JSON.
func PrintJSON(w io.Writer, results []Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

// junitTestSuites is the root JUnit XML element.
type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Errors   int             `xml:"errors,attr"`
	Time     float64         `xml:"time,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Error     *junitError   `xml:"error,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

type junitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// PrintJUnit outputs results as JUnit XML for CI.
func PrintJUnit(w io.Writer, results []Result) error {
	suites := junitTestSuites{}

	for _, r := range results {
		suite := junitTestSuite{
			Name: r.Name,
			Time: r.Duration.Seconds(),
		}

		// If request had an error, add it as an error test case
		if r.Error != nil {
			suite.Errors = 1
			suite.Tests = 1
			suite.Cases = append(suite.Cases, junitTestCase{
				Name:      r.Name,
				ClassName: r.Method + " " + r.URL,
				Time:      r.Duration.Seconds(),
				Error: &junitError{
					Message: r.Error.Error(),
					Type:    "RequestError",
					Content: r.Error.Error(),
				},
			})
		} else if len(r.TestResults) > 0 {
			// Add script test cases
			suite.Tests = len(r.TestResults)
			for _, tr := range r.TestResults {
				tc := junitTestCase{
					Name:      tr.Name,
					ClassName: r.Name,
					Time:      r.Duration.Seconds(),
				}
				if !tr.Passed {
					suite.Failures++
					tc.Failure = &junitFailure{
						Message: tr.Error,
						Type:    "AssertionFailure",
						Content: tr.Error,
					}
				}
				suite.Cases = append(suite.Cases, tc)
			}
		} else {
			// No script tests - add a single test case for the request itself
			suite.Tests = 1
			tc := junitTestCase{
				Name:      r.Name,
				ClassName: r.Method + " " + r.URL,
				Time:      r.Duration.Seconds(),
			}
			if r.StatusCode >= 400 {
				suite.Failures++
				tc.Failure = &junitFailure{
					Message: fmt.Sprintf("HTTP %d", r.StatusCode),
					Type:    "HTTPError",
					Content: r.Status,
				}
			}
			suite.Cases = append(suite.Cases, tc)
		}

		suites.Suites = append(suites.Suites, suite)
	}

	fmt.Fprint(w, xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(suites); err != nil {
		return err
	}
	fmt.Fprintln(w)
	return nil
}

func formatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const kb = 1024
	const mb = kb * 1024
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%d\u00b5s", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func statusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 301:
		return "Moved"
	case 302:
		return "Found"
	case 304:
		return "Not Modified"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 409:
		return "Conflict"
	case 422:
		return "Unprocessable"
	case 429:
		return "Too Many Requests"
	case 500:
		return "Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Unavailable"
	case 504:
		return "Gateway Timeout"
	default:
		return ""
	}
}
