package main

import (
	"bytes"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/coreos/go-etcd/etcd"
)

var prefix = flag.String("key", "/varnish", "etcd key to watch. Include the leading slash")
var templates = flag.String("templates", "./templates", "Path to the varnish templates")
var directorsOut = flag.String("directors", "/etc/varnish/directors.vcl", "The file to write the directors to")
var varnishSecret = flag.String("secret", "/etc/varnish/secret", "The secret to use when connecting to varnish")
var varnishServer = flag.String("varnish", "127.0.0.1:6082", "Varnish server to connect to, with admin port")
var varnishAdmin = flag.String("varnishadm", "/usr/bin/varnishadm", "The varnishadm program")
var vclOut = flag.String("vcl", "/etc/varnish/default.vcl", "The file to write the main VCL to")

var Error *log.Logger
var Info *log.Logger
var Warning *log.Logger

var nameRegexp *regexp.Regexp

type Def struct {
	Hosts     []*Host
	Directors map[string]*Director
}

type Host struct {
	Name  string
	Paths []*HostPath
}

type HostPath struct {
	Director string
	Path     string
	VCL      string
}

type Director struct {
	Backends []*Backend
	Name     string
}

type Backend struct {
	Address string
	Name    string
	Port    int
}

func assert(err error) {
	if err != nil {
		Error.Fatal(err)
	}
}

func parse(hosts *etcd.Response, upstreams *etcd.Response) Def {
	def := Def{
		Hosts:     make([]*Host, 0),
		Directors: make(map[string]*Director),
	}

	// read in all the configured hosts
	for i := 0; i < len(hosts.Node.Nodes); i++ {
		parsedHost := parseHost(hosts.Node.Nodes[i])

		if parsedHost != nil && len(parsedHost.Paths) > 0 {
			def.Hosts = append(def.Hosts, parsedHost)

			// create an entry for each upstream the host uses
			for j := 0; j < len(parsedHost.Paths); j++ {
				directorName := parsedHost.Paths[j].Director

				if _, state := def.Directors[directorName]; state == false {
					def.Directors[directorName] = &Director{
						Backends: make([]*Backend, 0),
						Name:     directorName,
					}
				}
			}
		}
	}

	// read in all the registered backends
	for i := 0; i < len(upstreams.Node.Nodes); i++ {
		node := upstreams.Node.Nodes[i]
		directorName := formatNameForVarnish(path.Base(node.Key))

		if director, state := def.Directors[directorName]; state == true {
			for j := 0; j < len(node.Nodes); j++ {
				if path.Base(node.Nodes[j].Key) == "endpoints" {
					for k := 0; k < len(node.Nodes[j].Nodes); k++ {
						endpointNode := node.Nodes[j].Nodes[k]
						address, port, err := parseEndpoint(endpointNode.Value)

						if err != nil {
							Error.Printf("Invalid endpoint address, %s, expected http://address:port", endpointNode.Value)
						} else {
							backend := &Backend{
								Name:    director.Name + "_" + strconv.Itoa(len(director.Backends)),
								Address: address,
								Port:    port,
							}

							director.Backends = append(director.Backends, backend)
						}
					}
				}
			}
		}
	}

	return def
}

func parseHost(host *etcd.Node) *Host {
	templateHost := &Host{
		Name:  path.Base(host.Key),
		Paths: make([]*HostPath, 0),
	}

	for i := 0; i < len(host.Nodes); i++ {
		if path.Base(host.Nodes[i].Key) == "locations" {
			for j := 0; j < len(host.Nodes[i].Nodes); j++ {
				hostPath := parseHostPath(host.Nodes[i].Nodes[j], templateHost)

				if hostPath != nil {
					templateHost.Paths = append(templateHost.Paths, hostPath)
				}
			}
		}
	}

	return templateHost
}

func parseHostPath(pathNode *etcd.Node, host *Host) *HostPath {
	hostPath := &HostPath{}
	pathKey := path.Base(pathNode.Key)
	missingComponents := make([]string, 0)

	for i := 0; i < len(pathNode.Nodes); i++ {
		node := pathNode.Nodes[i]
		key := path.Base(node.Key)

		switch key {
		case "path":
			hostPath.Path = node.Value

		case "upstream":
			hostPath.Director = node.Value

		case "vcl":
			hostPath.VCL = node.Value
		}
	}

	if len(hostPath.Path) == 0 {
		missingComponents = append(missingComponents, "path")
	}

	if len(hostPath.Director) == 0 {
		missingComponents = append(missingComponents, "upstream")
	}

	if len(hostPath.VCL) == 0 {
		missingComponents = append(missingComponents, "vcl")
	}

	if len(missingComponents) > 0 {
		Warning.Printf("Found %s/%s but missing components [%s], ignoring", host.Name, pathKey, strings.Join(missingComponents, ","))
	} else {
		hostPath.Director = formatNameForVarnish(hostPath.Director)
		return hostPath
	}

	return nil
}

func writeHosts(t *template.Template, hosts []*Host) error {
	var buf bytes.Buffer
	err := t.Execute(&buf, hosts)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(*vclOut, buf.Bytes(), 0700)
}

func writeDirectors(t *template.Template, directors map[string]*Director) error {
	var buf bytes.Buffer
	err := t.Execute(&buf, directors)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(*directorsOut, buf.Bytes(), 0700)
}

func reloadVarnish() error {
	vclName := "vcl" + strconv.FormatInt(time.Now().Unix(), 10)

	load := exec.Command(*varnishAdmin, "-T"+*varnishServer, "-S"+*varnishSecret, "vcl.load "+vclName+" "+*vclOut)
	use := exec.Command(*varnishAdmin, "-T"+*varnishServer, "-S"+*varnishSecret, "vcl.use "+vclName)

	var err error
	var out []byte

	if out, err = load.CombinedOutput(); err != nil {
		Error.Println("Problem compiling new VCL,", err.Error(), string(out))
	} else if err = use.Run(); err != nil {
		Error.Println("Problem switching to new VCL, ", err.Error())
	} else {
		Info.Println("Configuration updated")
	}

	return nil
}

func formatNameForVarnish(name string) string {
	return nameRegexp.ReplaceAllString(name, "")
}

func parseEndpoint(endpoint string) (address string, port int, err error) {
	uri, err := url.Parse(endpoint)

	if err != nil {
		return "", 0, errors.New("Invalid address")
	} else if uri.Host == "" {
		return "", 0, errors.New("Must provide a host")
	}

	parts := strings.Split(uri.Host, ":")
	address = parts[0]

	if len(parts) == 1 {
		port = 80
	} else {
		port, _ = strconv.Atoi(parts[1])
	}

	return
}

func process(client *etcd.Client, vclTemplate *template.Template, upstreamTemplate *template.Template) {
	if hosts, err := client.Get(*prefix+"/hosts", true, true); err != nil {
		Error.Println("Reading hosts from etcd,", err.Error())
	} else if upstreams, err := client.Get(*prefix+"/upstreams", true, true); err != nil {
		Error.Println("Reading upstreams from etcd,", err.Error())
	} else {
		def := parse(hosts, upstreams)

		if err = writeHosts(vclTemplate, def.Hosts); err != nil {
			Error.Println("Writing hosts,", err.Error())
		} else if err = writeDirectors(upstreamTemplate, def.Directors); err != nil {
			Error.Println("Writing directors,", err.Error())
		} else if err = reloadVarnish(); err != nil {
			Error.Println("Trying to reload varnish,", err.Error())
		}
	}
}

func main() {
	Error = log.New(os.Stderr, "ERROR: ", log.Ltime|log.Lshortfile)
	Info = log.New(os.Stdout, "INFO: ", log.Ltime|log.Lshortfile)
	Warning = log.New(os.Stdout, "WARN: ", log.Ltime|log.Lshortfile)

	nameRegexp = regexp.MustCompile(`[\W]`)

	flag.Parse()

	// arg 0 is the etcd server address
	uri, err := url.Parse(flag.Arg(0))
	assert(err)

	if uri.Host == "" {
		Error.Fatal("Invalid etcd host, expected format http://127.0.0.1:4001")
	}

	// compile the templates, exit early if there's a problem
	vclTemplate := template.New("default.vcl")
	vclTemplate, err = vclTemplate.ParseFiles(*templates + "/default.vcl")
	assert(err)

	directorTemplate := template.New("directors.vcl")
	directorTemplate, err = directorTemplate.ParseFiles(*templates + "/directors.vcl")
	assert(err)

	// connect to etcd and start watching
	urls := []string{uri.String()}
	watchChan := make(chan *etcd.Response)
	client := etcd.NewClient(urls)

	process(client, vclTemplate, directorTemplate)

	go client.Watch(*prefix, 0, true, watchChan, nil)

	Info.Printf("Listening for etcd events from %s%s...", uri.String(), *prefix)

	for _ = range watchChan {
		Info.Println("Change detected...")
		process(client, vclTemplate, directorTemplate)
	}

	Error.Fatal("etcd watch loop closed")
}
