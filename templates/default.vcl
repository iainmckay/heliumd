include "directors.vcl";

backend default {
    .host = "127.0.0.1";
    .port = "65535";
}

acl purge {
    "localhost";
    "10.0.0.0"/24;
}

sub vcl_recv {
    set req.backend = default;

    {{range $index, $host := .}}
        {{if $index}}else{{end}} if (req.http.host == "{{$host.Name}}") {
            {{range $index, $value := $host.Paths}}
                {{if $index}}else{{end}} if (req.url ~ "^{{$value.Path}}") {
                    set req.backend = {{$value.Director}};
                    include "/etc/varnish/vcl/{{$value.VCL}}/recv.vcl";
                }
            {{end}}
        }
    {{end}}
}

sub vcl_fetch {
    {{range $index, $host := .}}
        {{if $index}}else{{end}} if (req.http.host == "{{$host.Name}}") {
            {{range $index, $value := $host.Paths}}
                {{if $index}}else{{end}} if (req.url ~ "^{{$value.Path}}") {
                    include "/etc/varnish/vcl/{{$value.VCL}}/fetch.vcl";
                }
            {{end}}
        }
    {{end}}
}

sub vcl_deliver {
    {{range $index, $host := .}}
        {{if $index}}else{{end}} if (req.http.host == "{{$host.Name}}") {
            {{range $index, $value := $host.Paths}}
                {{if $index}}else{{end}} if (req.url ~ "^{{$value.Path}}") {
                    include "/etc/varnish/vcl/{{$value.VCL}}/deliver.vcl";
                }
            {{end}}
        }
    {{end}}
}

sub vcl_hit {
    {{range $index, $host := .}}
        {{if $index}}else{{end}} if (req.http.host == "{{$host.Name}}") {
            {{range $index, $value := $host.Paths}}
                {{if $index}}else{{end}} if (req.url ~ "^{{$value.Path}}") {
                    include "/etc/varnish/vcl/{{$value.VCL}}/hit.vcl";
                }
            {{end}}
        }
    {{end}}
}

sub vcl_miss {
    {{range $index, $host := .}}
        {{if $index}}else{{end}} if (req.http.host == "{{$host.Name}}") {
            {{range $index, $value := $host.Paths}}
                {{if $index}}else{{end}} if (req.url ~ "^{{$value.Path}}") {
                    include "/etc/varnish/vcl/{{$value.VCL}}/miss.vcl";
                }
            {{end}}
        }
    {{end}}
}
