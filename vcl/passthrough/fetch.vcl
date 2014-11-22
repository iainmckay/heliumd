if (beresp.http.Surrogate-Control ~ "ESI/1.0") {
    unset beresp.http.Surrogate-Control;
    set beresp.do_esi = true;
}

if (beresp.status == 301 || beresp.status == 302) {
    return (hit_for_pass);
}

if (beresp.status == 404) {
    set beresp.ttl = 30s;
    return (hit_for_pass);
}

# Serve pages from the cache should we get a sudden error and re-check in one minute
if (beresp.status >= 500) {
    set beresp.grace = 1s;
    set beresp.ttl = 1s;
    return (hit_for_pass);
}

unset beresp.http.set-cookie;
