package message

type jiraMessage struct {
	issue JiraIssue
	Base
}

// JiraIssue requires project and summary to create a real jira issue.
// Other fields depend on permissions given to the specific project, and
// all fields must be legitimate custom fields defined for the project.
// To see whether you have the right permissions to create an issue with certain
// fields, check your JIRA interface on the web.
type JiraIssue struct {
	Project     string
	Summary     string
	Description string
	Reporter    string
	Assignee    string
	Type        string
	Components  []string
	Labels      []string
	// ... other fields
	Fields map[string]string
}

// JiraField is a struct composed of a key-value pair.
type JiraField struct {
	Key   string
	Value interface{}
}

// MakeJiraMessage creates a jiraMessage instance with the given JiraIssue.
func MakeJiraMessage(issue JiraIssue) Composer {
	return &jiraMessage{
		issue: issue,
	}
}

// NewJiraMessage creates and returns a fully formed jiraMessage, which implements
// message.Composer. project string and summary string are required, and any
// number of additional fields may be included. Fields with keys Reporter, Assignee,
// Type, and Labels will be specifically assigned to respective fields in the new
// jiraIssue included in the jiraMessage, (e.g. JiraIssue.Reporter, etc), and
// all other fields will be included in jiraIssue.Fields.
func NewJiraMessage(project, summary string, fields ...JiraField) Composer {
	issue := JiraIssue{
		Project: project,
		Summary: summary,
		Fields:  map[string]string{},
	}

	// Assign given fields to jira issue fields
	for _, f := range fields {
		switch f.Key {
		case "reporter", "Reporter":
			issue.Reporter = f.Value.(string)
		case "assignee", "Assignee":
			issue.Assignee = f.Value.(string)
		case "type", "Type":
			issue.Type = f.Value.(string)
		case "labels", "Labels":
			issue.Labels = f.Value.([]string)
		case "component", "Component":
			issue.Components = f.Value.([]string)
		default:
			issue.Fields[f.Key] = f.Value.(string)
		}
	}

	// Setting "Task" as the default value for IssueType
	if issue.Type == "" {
		issue.Type = "Task"
	}

	return MakeJiraMessage(issue)
}

func (m *jiraMessage) String() string   { return m.issue.Summary }
func (m *jiraMessage) Raw() interface{} { return m.issue }
func (m *jiraMessage) Loggable() bool   { return m.issue.Summary != "" }
