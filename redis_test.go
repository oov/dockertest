package dockertest_test

import (
	"fmt"

	"github.com/oov/dockertest"
	"gopkg.in/redis.v3"
)

func Example_redis() {
	c, err := dockertest.New(dockertest.Config{
		Image: "redis", // or "redis:latest"
		Name:  "dockertest-redis",
		PortMapping: map[string]string{
			"6379/tcp": "auto",
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

	r := redis.NewClient(&redis.Options{
		Addr: c.Mapped["6379/tcp"].String(),
	})
	pong, err := r.Ping().Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(pong)
	// Output:
	// PONG
}
