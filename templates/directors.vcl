{{range .}}
    {{range .Backends}}
        backend {{.Name}} { .host = "{{.Address}}"; .port = "{{.Port}}"; }
    {{end}}
{{end}}

{{range .}}
    director {{.Name}} round-robin {
        {{range .Backends}}
            { .backend = {{.Name}}; }
        {{end}}
    }
{{end}}
