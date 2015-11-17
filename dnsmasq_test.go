package dockertest_test

import (
	"fmt"

	"github.com/miekg/dns"
	"github.com/oov/dockertest"
)

func Example_dnsmasq() {
	c, err := dockertest.New(dockertest.Config{
		Image: "andyshinn/dnsmasq", // or "andyshinn/dnsmasq:latest"
		Args: []string{
			"--user=root",
			"--address=/the-answer-to-life-the-universe-and-everything/42.42.42.42",
		},
		PortMapping: map[string]string{
			"53/udp": "auto",
			"53/tcp": "auto",
		},
	})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	// wait until the container has started listening
	if err = c.Wait(nil); err != nil {
		panic(err)
	}

	var m dns.Msg
	m.SetQuestion("the-answer-to-life-the-universe-and-everything.", dns.TypeA)

	in, _, err := (&dns.Client{}).Exchange(&m, c.Mapped["53/udp"].String())
	if err != nil {
		panic(err)
	}
	fmt.Println("UDP", in.Answer[0].(*dns.A).A)

	in, _, err = (&dns.Client{Net: "tcp"}).Exchange(&m, c.Mapped["53/tcp"].String())
	if err != nil {
		panic(err)
	}
	fmt.Println("TCP", in.Answer[0].(*dns.A).A)

	// Output:
	// UDP 42.42.42.42
	// TCP 42.42.42.42
}
