{{- if .Entries }}
| ID | State | Title | Labels | Created |
| --- | --- | --- | --- | --- |
{{- range .Entries }}
| {{ mdCell (default " " .Match.Short) }} | {{ mdCell (default " " .Match.State) }} | {{ mdCell (default " " .Issue.Title) }} | {{ mdCell (default " " (join .Issue.Labels ", ")) }} | {{ mdCell (default " " (.Issue.Created.Format "2006-01-02")) }} |
{{- end }}
{{- end }}
