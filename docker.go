package dockertest

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ErrTimeout represents timeout error.
var ErrTimeout = fmt.Errorf("dockertest: timeout")

const defaultTimeout = 3 * time.Minute

// Container represents Docker container.
type Container struct {
	ID          string
	Mapped      map[string]string
	WaitTimeout time.Duration // it is set to 3 minutes as the default by New.
}

type Config struct {
	Image       string            // "image[:version]" such as "postgres:latest".
	Env         map[string]string // Env["ENV_NAME"] = "VALUE"
	PortMapping map[string]string // PortMapping["port/proto"] = "host:port"
}

// New runs new Docker container.
func New(conf Config) (*Container, error) {
	args := []string{"run", "-d"}
	for p, h := range conf.PortMapping {
		args = append(args, "-p", h+":"+p)
	}
	for key, val := range conf.Env {
		args = append(args, "-e", key+"="+val)
	}
	args = append(args, conf.Image)

	o, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("dockertest: docker run failed: %v: %s", err, o)
	}
	c := &Container{
		ID:          strings.TrimSpace(string(o)),
		WaitTimeout: defaultTimeout,
	}
	if c.Mapped, err = getMapping(c.ID, conf.PortMapping); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func getMapping(ID string, port map[string]string) (map[string]string, error) {
	o, err := exec.Command("docker", "inspect", ID).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("dockertest: docker inspect failed: %v: %s", err, o)
	}

	var r []struct {
		NetworkSettings struct {
			Ports map[string][]struct {
				HostIp   string
				HostPort string
			}
		}
	}
	if err = json.Unmarshal(o, &r); err != nil {
		return nil, fmt.Errorf("dockertest: could not parse docker inspect output: %v: %s", err, o)
	}
	if len(r) != 1 {
		return nil, fmt.Errorf("dockertest: more than one result: %d", len(r))
	}
	m := map[string]string{}
	for p := range port {
		ps := r[0].NetworkSettings.Ports[p]
		if len(ps) != 1 {
			return nil, fmt.Errorf("dockertest: more than one port mapping data: %d", len(ps))
		}
		m[p] = ps[0].HostIp + ":" + ps[0].HostPort
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

// Close kills and removes the container.
func (c *Container) Close() error {
	o, err := exec.Command("docker", "rm", "-f", c.ID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("dockertest: docker rm failed: %v: %s", err, o)
	}
	return nil
}