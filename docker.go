package dockertest

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ErrTimeout represents timeout error.
var ErrTimeout = fmt.Errorf("dockertest: timeout")

const defaultTimeout = 3 * time.Minute

type Mapped struct {
	Host       string
	DockerHost string
	Port       string
}

func (m Mapped) HostAndPort() string {
	return m.Host + ":" + m.Port
}

func (m Mapped) DockerHostAndPort() string {
	return m.DockerHost + ":" + m.Port
}

func (m Mapped) String() string {
	return m.DockerHostAndPort()
}

// Container represents Docker container.
type Container struct {
	ID          string
	Mapped      map[string]Mapped
	WaitTimeout time.Duration // it is set to 3 minutes as the default by New.
	runArgs     []string
	created     bool
}

type Config struct {
	Image       string            // "image[:version]" such as "postgres:latest".
	Name        string            // used for to reuse existing container.
	Args        []string          // Additional parameters for the container.
	DockerArgs  []string          // Additional parameters for the docker.
	Env         map[string]string // Env["ENV_NAME"] = "VALUE"
	PortMapping map[string]string // PortMapping["port/proto"] = "host:port"
}

func dockerHost() string {
	return evalDockerHost(os.Getenv("DOCKER_HOST"))
}

func evalDockerHost(envvar string) string {
	dhURL, err := url.Parse(envvar)
	if envvar == "" || err != nil {
		return "127.0.0.1"
	}
	h, _, err := net.SplitHostPort(dhURL.Host)
	if err != nil {
		return dhURL.Host
	}
	return h
}

func suggestMappingHost(envvar string) string {
	if evalDockerHost(envvar) == "127.0.0.1" {
		return "127.0.0.1"
	}
	return "0.0.0.0"
}

func SuggestMappingHost() string {
	return suggestMappingHost(os.Getenv("DOCKER_HOST"))
}

func reuse(conf *Config) (*Container, error) {
	o, err := exec.Command("docker", "inspect", conf.Name).CombinedOutput()
	if err != nil {
		if strings.Contains(string(o), "No such image or container") {
			return nil, nil
		}
		return nil, fmt.Errorf("dockertest: docker inspect failed: %v: %s", err, o)
	}

	var r []struct {
		Id string
	}
	if err = json.Unmarshal(o, &r); err != nil {
		return nil, fmt.Errorf("dockertest: could not parse docker inspect output: %v: %s", err, o)
	}
	switch len(r) {
	case 0:
		return nil, fmt.Errorf("dockertest: could not find network setting")
	case 1:
		break
	default:
		return nil, fmt.Errorf("dockertest: more than one result: %d", len(r))
	}

	c := &Container{
		ID:          r[0].Id,
		WaitTimeout: defaultTimeout,
	}
	if c.Mapped, err = parsePortMapping(o, conf.PortMapping); err != nil {
		return nil, err
	}
	return c, nil
}

// New runs new Docker container.
func New(conf Config) (*Container, error) {
	if conf.Name != "" {
		c, err := reuse(&conf)
		if err != nil {
			return nil, err
		}
		if c != nil {
			return c, nil
		}
	}

	args := []string{"run", "-d"}
	for p, h := range conf.PortMapping {
		if h == "auto" {
			args = append(args, "-p", SuggestMappingHost()+":0:"+p)
		} else {
			args = append(args, "-p", h+":"+p)
		}
	}
	for key, val := range conf.Env {
		args = append(args, "-e", key+"="+val)
	}
	if conf.DockerArgs != nil {
		args = append(args, conf.DockerArgs...)
	}
	args = append(args, conf.Image)
	if conf.Args != nil {
		args = append(args, conf.Args...)
	}

	o, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("dockertest: docker run failed: %v: %s", err, o)
	}
	c := &Container{
		ID:          strings.TrimSpace(string(o)),
		WaitTimeout: defaultTimeout,
		runArgs:     args,
		created:     true,
	}
	if c.Mapped, err = getMapping(c.ID, conf.PortMapping); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// RunArgs returns parameters that was used for "docker run".
func (c *Container) RunArgs() []string {
	return c.runArgs
}

func getMapping(ID string, port map[string]string) (map[string]Mapped, error) {
	o, err := exec.Command("docker", "inspect", ID).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("dockertest: docker inspect failed: %v: %s", err, o)
	}

	return parsePortMapping(o, port)
}

func parsePortMapping(b []byte, port map[string]string) (map[string]Mapped, error) {
	var r []struct {
		NetworkSettings struct {
			Ports map[string][]struct {
				HostIp   string
				HostPort string
			}
		}
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("dockertest: could not parse docker inspect output: %v: %s", err, b)
	}
	switch len(r) {
	case 0:
		return nil, fmt.Errorf("dockertest: could not find network setting")
	case 1:
		break
	default:
		return nil, fmt.Errorf("dockertest: more than one result: %d", len(r))
	}

	m := map[string]Mapped{}
	for p := range port {
		ps := r[0].NetworkSettings.Ports[p]
		switch len(ps) {
		case 0:
			return nil, fmt.Errorf("dockertest: could not find port mapping data")
		case 1:
			break
		default:
			return nil, fmt.Errorf("dockertest: more than one port mapping data: %d", len(ps))
		}
		m[p] = Mapped{
			Host:       ps[0].HostIp,
			DockerHost: dockerHost(),
			Port:       ps[0].HostPort,
		}
	}
	return m, nil
}

func getProcNetEntries(ID string, target string) (entries, error) {
	o, err := exec.Command("docker", "exec", ID, "cat", "/proc/net/"+target).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("dockertest: docker exec failed: %v: %s", err, o)
	}
	return parseProcNet(string(o))
}

func ready(ID string, proto string, port []int) (bool, error) {
	if port == nil || len(port) == 0 {
		return true, nil
	}

	i := -1
	es, err := getProcNetEntries(ID, proto)
	if err != nil {
		return false, err
	}
	for j, p := range port {
		if es.FindByLocalPort(p) == -1 {
			break
		}
		i = j
	}
	if i == len(port)-1 {
		return true, nil
	}

	// If could not find all entry, merge IPv6 entries and retry.
	es6, err := getProcNetEntries(ID, proto+"6")
	if err != nil {
		return false, err
	}
	es = append(es, es6...)
	for j := i + 1; j < len(port); j++ {
		p := port[j]
		if es.FindByLocalPort(p) == -1 {
			break
		}
		i = j
	}
	return i == len(port)-1, nil
}

func appendPort(tcp []int, udp []int, port string) ([]int, []int) {
	if len(port) < len("1/tcp") {
		return tcp, udp
	}
	var i int
	if _, err := fmt.Sscan(port[:len(port)-4], &i); err != nil {
		return tcp, udp
	}
	switch port[len(port)-3:] {
	case "tcp":
		return append(tcp, i), udp
	case "udp":
		return tcp, append(udp, i)
	}
	return tcp, udp
}

// Wait waits until the container has started listening.
// If wait for all ports that passed at New, can pass nil to the port.
// When Wait has timed out, Wait returns ErrTimeout.
//
// Internally, Wait reads "/proc/net/(tcp|udp)6?" that exists inside the container.
func (c *Container) Wait(port []string) error {
	var tcp, udp []int
	if port == nil {
		for p := range c.Mapped {
			tcp, udp = appendPort(tcp, udp, p)
		}
	} else {
		for _, p := range port {
			tcp, udp = appendPort(tcp, udp, p)
		}
	}

	var ok bool
	var err error
	retryDuration := time.Duration(0)
	try := time.NewTimer(retryDuration)
	timeOut := time.After(c.WaitTimeout)
	for {
		select {
		case <-try.C:
			ok, err = ready(c.ID, "tcp", tcp)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			ok, err = ready(c.ID, "udp", udp)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			return nil
		case <-timeOut:
			try.Stop()
			return ErrTimeout
		}
		if retryDuration < time.Second {
			retryDuration += 100 * time.Millisecond
		}
		try.Reset(retryDuration)
	}
	return nil
}

// Reused reports whenever the container was existed one.
func (c *Container) Reused() bool {
	return !c.created
}

// Close kills and removes the container.
// However if you are reusing existing container, it is not removed.
func (c *Container) Close() error {
	if !c.created {
		return nil
	}

	o, err := exec.Command("docker", "rm", "-f", c.ID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("dockertest: docker rm failed: %v: %s", err, o)
	}
	return nil
}
