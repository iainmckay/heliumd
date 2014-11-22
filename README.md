# Heliumd

[Etcd](https://github.com/coreos/etcd) aware daemon for configuring [Varnish](https://www.varnish-cache.org/)

Heliumd stores Varnish configuration inside Etcd, listens for changes and reconfigures Varnish. It is intended to be used in a Dockerized environment and can be used alongside a service such as [registrator](https://github.com/progrium/registrator/).

## Starting Heliumd

Heliumd starts with sane defaults. It expects Varnish to be running on the same host with the admin interface running on port 6082. It will write two files by default, /etc/varnish/directors.vcl and /etc/varnish/default.vcl. All defaults can be overriden, you can see a list by running heliumd with --help.

To start Heliumd you must provide the address to a working Etcd instance.

    $ ./heliumd http://127.0.0.1:4001

## Running the Docker container

This repository contains a working Dockerfile, it exposes the web interface on port 80 and the admin interface on port 6082. To get things up and running you simply run:

    $ docker run -d -P -e ETCD_PEER=http://127.0.0.1:4001 iainmckay/heliumd

The container can be customized with these environment variables:

Name | Default Value | Description
--- | --- | ---
ETCD_KEY | /varnish | Which Etcd path to watch for configuration
ETCD_PEER | http://127.0.0.1:4001 | The Etcd peer to connect to
OUT_DIRECTORS | /etc/varnish/directors.vcl | Where to output the directors
OUT_VCL | /etc/varnish/default.vcl | Where to output the main VCL
VARNISH_ALLOCATION | 100m | How much memory to allocate to Varnish
VARNISH_SECRET | /etc/varnish/secret | Where to store the varnish secret

## Adding hosts and upstreams

By default Heliumd will watch `/varnish` in Etcd for host and upstream changes. The expected format for host and upstreams is the same used in [Vulcand](https://github.com/mailgun/vulcand) so it is possible to run this in conjunction with Vulcand.

### Adding hosts

Hosts consist of a domain, a path under that domain, a VCL and an upstream to route traffic to.

```
/varnish/hosts/api.example.org/locations/super-service/path /some-super-service
/varnish/hosts/api.example.org/locations/super-service/upstream super-service
/varnish/hosts/api.example.org/locations/super-service/vcl passthrough
```

In this example, we are going to have Varnish route traffic for `api.example.org/some-super-service` through an upstream called `super-service` and use a VCL called `passthrough`.

Heliumd is by default, opinionated, on how to setup Varnish. It supports being able to use a separate VCL per location. You can see a working example for `passthrough` in the VCL directory [here](https://github.com/iainmckay/heliumd/tree/master/vcl/). You could write the provided templates to not use this VCL setup.

### Adding upstreams

Upstreams are a collection of backends to send traffic to.

```
/varnish/upstreams/super-service/endpoints/e1 http://127.0.0.1:8080
/varnish/upstreams/super-service/endpoints/e2 http://127.0.0.1:8081
/varnish/upstreams/super-service/endpoints/e3 http://127.0.0.1:8082
```

Continuing the `some-super-service` example in the previous section. Here we register 3 backends on 3 different ports. Only HTTP is supported.

You can use a helper container such as [registrator](https://github.com/progrium/registrator/) to manage these automatically. [PR #53](https://github.com/progrium/registrator/pull/53) adds support for the Vulcand format that Heliumd uses, it will hopefully be merged in to master soon.

## SSL

It is recommended you run Nginx or another reverse-proxy in front of Varnish to provide SSL termination.
