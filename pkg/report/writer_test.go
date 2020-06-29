package report_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	dbTypes "github.com/aquasecurity/trivy-db/pkg/types"
	"github.com/aquasecurity/trivy/pkg/report"
	"github.com/aquasecurity/trivy/pkg/types"
)

func TestReportWriter_Table(t *testing.T) {
	testCases := []struct {
		name           string
		detectedVulns  []types.DetectedVulnerability
		expectedOutput string
		light          bool
	}{
		{
			name: "happy path full",
			detectedVulns: []types.DetectedVulnerability{
				{
					VulnerabilityID:  "123",
					PkgName:          "foo",
					InstalledVersion: "1.2.3",
					FixedVersion:     "3.4.5",
					Vulnerability: dbTypes.Vulnerability{
						Title:       "foobar",
						Description: "baz",
						Severity:    "HIGH",
					},
				},
			},
			expectedOutput: `+---------+------------------+----------+-------------------+---------------+--------+
| LIBRARY | VULNERABILITY ID | SEVERITY | INSTALLED VERSION | FIXED VERSION | TITLE  |
+---------+------------------+----------+-------------------+---------------+--------+
| foo     |              123 | HIGH     | 1.2.3             | 3.4.5         | foobar |
+---------+------------------+----------+-------------------+---------------+--------+
`,
		},
		{
			name:  "happy path light",
			light: true,
			detectedVulns: []types.DetectedVulnerability{
				{
					VulnerabilityID:  "123",
					PkgName:          "foo",
					InstalledVersion: "1.2.3",
					FixedVersion:     "3.4.5",
					Vulnerability: dbTypes.Vulnerability{
						Title:       "foobar",
						Description: "baz",
						Severity:    "HIGH",
					},
				},
			},
			expectedOutput: `+---------+------------------+----------+-------------------+---------------+
| LIBRARY | VULNERABILITY ID | SEVERITY | INSTALLED VERSION | FIXED VERSION |
+---------+------------------+----------+-------------------+---------------+
| foo     |              123 | HIGH     | 1.2.3             | 3.4.5         |
+---------+------------------+----------+-------------------+---------------+
`,
		},
		{
			name: "no title for vuln",
			detectedVulns: []types.DetectedVulnerability{
				{
					VulnerabilityID:  "123",
					PkgName:          "foo",
					InstalledVersion: "1.2.3",
					FixedVersion:     "3.4.5",
					Vulnerability: dbTypes.Vulnerability{
						Description: "foobar",
						Severity:    "HIGH",
					},
				},
			},
			expectedOutput: `+---------+------------------+----------+-------------------+---------------+--------+
| LIBRARY | VULNERABILITY ID | SEVERITY | INSTALLED VERSION | FIXED VERSION | TITLE  |
+---------+------------------+----------+-------------------+---------------+--------+
| foo     |              123 | HIGH     | 1.2.3             | 3.4.5         | foobar |
+---------+------------------+----------+-------------------+---------------+--------+
`,
		},
		{
			name: "long title for vuln",
			detectedVulns: []types.DetectedVulnerability{
				{
					VulnerabilityID:  "123",
					PkgName:          "foo",
					InstalledVersion: "1.2.3",
					FixedVersion:     "3.4.5",
					Vulnerability: dbTypes.Vulnerability{
						Title:    "a b c d e f g h i j k l m n o p q r s t u v",
						Severity: "HIGH",
					},
				},
			},
			expectedOutput: `+---------+------------------+----------+-------------------+---------------+----------------------------+
| LIBRARY | VULNERABILITY ID | SEVERITY | INSTALLED VERSION | FIXED VERSION |           TITLE            |
+---------+------------------+----------+-------------------+---------------+----------------------------+
| foo     |              123 | HIGH     | 1.2.3             | 3.4.5         | a b c d e f g h i j k l... |
+---------+------------------+----------+-------------------+---------------+----------------------------+
`,
		},
		{
			name:           "no vulns",
			detectedVulns:  []types.DetectedVulnerability{},
			expectedOutput: ``,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputResults := report.Results{
				{
					Target:          "foo",
					Vulnerabilities: tc.detectedVulns,
				},
			}
			tableWritten := bytes.Buffer{}
			assert.NoError(t, report.WriteResults("table", &tableWritten, inputResults, "", tc.light), tc.name)
			assert.Equal(t, tc.expectedOutput, tableWritten.String(), tc.name)
		})
	}
}

func TestReportWriter_JSON(t *testing.T) {
	testCases := []struct {
		name          string
		detectedVulns []types.DetectedVulnerability
		expectedJSON  report.Results
	}{
		{
			name: "happy path",
			detectedVulns: []types.DetectedVulnerability{
				{
					VulnerabilityID:  "123",
					PkgName:          "foo",
					InstalledVersion: "1.2.3",
					FixedVersion:     "3.4.5",
					Vulnerability: dbTypes.Vulnerability{
						Title:       "foobar",
						Description: "baz",
						Severity:    "HIGH",
					},
				},
			},
			expectedJSON: report.Results{
				report.Result{
					Target: "foojson",
					Vulnerabilities: []types.DetectedVulnerability{
						{
							VulnerabilityID:  "123",
							PkgName:          "foo",
							InstalledVersion: "1.2.3",
							FixedVersion:     "3.4.5",
							Vulnerability: dbTypes.Vulnerability{
								Title:       "foobar",
								Description: "baz",
								Severity:    "HIGH",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			jw := report.JsonWriter{}
			jsonWritten := bytes.Buffer{}
			jw.Output = &jsonWritten

			inputResults := report.Results{
				{
					Target:          "foojson",
					Vulnerabilities: tc.detectedVulns,
				},
			}

			assert.NoError(t, report.WriteResults("json", &jsonWritten, inputResults, "", false), tc.name)

			writtenResults := report.Results{}
			errJson := json.Unmarshal([]byte(jsonWritten.String()), &writtenResults)
			assert.NoError(t, errJson, "invalid json written", tc.name)

			assert.Equal(t, tc.expectedJSON, writtenResults, tc.name)
		})
	}

}

func TestReportWriter_Template(t *testing.T) {
	testCases := []struct {
		name          string
		detectedVulns []types.DetectedVulnerability
		template      string
		expected      string
	}{
		{
			name: "happy path",
			detectedVulns: []types.DetectedVulnerability{
				{
					VulnerabilityID: "CVE-2019-0000",
					PkgName:         "foo",
					Vulnerability: dbTypes.Vulnerability{
						Severity: dbTypes.SeverityHigh.String(),
					},
				},
				{
					VulnerabilityID: "CVE-2019-0000",
					PkgName:         "bar",
					Vulnerability: dbTypes.Vulnerability{
						Severity: dbTypes.SeverityHigh.String()},
				},
				{
					VulnerabilityID: "CVE-2019-0001",
					PkgName:         "baz",
					Vulnerability: dbTypes.Vulnerability{
						Severity: dbTypes.SeverityCritical.String(),
					},
				},
			},
			template: "{{ range . }}{{ range .Vulnerabilities}}{{ println .VulnerabilityID .Severity }}{{ end }}{{ end }}",
			expected: "CVE-2019-0000 HIGH\nCVE-2019-0000 HIGH\nCVE-2019-0001 CRITICAL\n",
		},
		{
			name: "happy path",
			detectedVulns: []types.DetectedVulnerability{
				{
					VulnerabilityID:  "123",
					PkgName:          "foo",
					InstalledVersion: "1.2.3",
					FixedVersion:     "3.4.5",
					Vulnerability: dbTypes.Vulnerability{
						Title:       "foobar",
						Description: "baz",
						Severity:    "HIGH",
					},
				},
			},

			template: `<testsuites>
{{- range . -}}
{{- $failures := len .Vulnerabilities }}
    <testsuite tests="1" failures="{{ $failures }}" time="" name="{{  .Target }}">
	{{- if not (eq .Type "") }}
        <properties>
            <property name="type" value="{{ .Type }}"></property>
        </properties>
        {{- end -}}
        {{ range .Vulnerabilities }}
        <testcase classname="{{ .PkgName }}-{{ .InstalledVersion }}" name="[{{ .Vulnerability.Severity }}] {{ .VulnerabilityID }}" time="">
            <failure message={{ .Title | printf "%q" }} type="description">{{ .Description | printf "%q" }}</failure>
        </testcase>
    {{- end }}
	</testsuite>
{{- end }}
</testsuites>`,

			expected: `<testsuites>
    <testsuite tests="1" failures="1" time="" name="foojunit">
        <properties>
            <property name="type" value="test"></property>
        </properties>
        <testcase classname="foo-1.2.3" name="[HIGH] 123" time="">
            <failure message="foobar" type="description">"baz"</failure>
        </testcase>
	</testsuite>
</testsuites>`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmplWritten := bytes.Buffer{}
			inputResults := report.Results{
				{
					Target:          "foojunit",
					Type:            "test",
					Vulnerabilities: tc.detectedVulns,
				},
			}

			assert.NoError(t, report.WriteResults("template", &tmplWritten, inputResults, tc.template, false))
			assert.Equal(t, tc.expected, tmplWritten.String())
		})
	}
}
