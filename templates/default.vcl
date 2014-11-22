include "directors.vcl";

backend default {
    .host = "127.0.0.1";
    .port = "65535";
}

sub vcl_recv {
    set req.backend = default;

    {{range .}}
        if (req.http.host == "{{.Name}}") {
            {{range .Paths}}
                if (req.url ~ "^{{.Path}}") {
                    set req.backend = {{.Director}};
                    include "/etc/varnish/vcl/{{.VCL}}/recv.vcl";
                }
            {{end}}
        }
    {{end}}
}

sub vcl_fetch {
    {{range .}}
        if (req.http.host == "{{.Name}}") {
            {{range .Paths}}
                if (req.url ~ "^{{.Path}}") {
                    include "/etc/varnish/vcl/{{.VCL}}/fetch.vcl";
                }
            {{end}}
        }
    {{end}}
}

sub vcl_deliver {
    {{range .}}
        if (req.http.host == "{{.Name}}") {
            {{range .Paths}}
                if (req.url ~ "^{{.Path}}") {
                    include "/etc/varnish/vcl/{{.VCL}}/deliver.vcl";
                }
            {{end}}
        }
    {{end}}
}

sub vcl_hit {
    {{range .}}
        if (req.http.host == "^{{.Name}}") {
            {{range .Paths}}
                if (req.url ~ "^{{.Path}}") {
                    include "/etc/varnish/vcl/{{.VCL}}/hit.vcl";
                }
            {{end}}
        }
    {{end}}
}

sub vcl_miss {
    {{range .}}
        if (req.http.host == "^{{.Name}}") {
            {{range .Paths}}
                if (req.url ~ "^{{.Path}}") {
                    include "/etc/varnish/vcl/{{.VCL}}/miss.vcl";
                }
            {{end}}
        }
    {{end}}
}
