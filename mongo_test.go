package dockertest_test

import (
	"fmt"

	"github.com/oov/dockertest"
	"gopkg.in/mgo.v2"
)

func Example_mongoDB() {
	c, err := dockertest.New(dockertest.Config{
		Image: "mongo", // or "redis:latest"
		PortMapping: map[string]string{
			"27017/tcp": "auto",
		},
		Args: []string{"--storageEngine", "wiredTiger"},
	})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	// wait until the container has started listening
	if err = c.Wait(nil); err != nil {
		panic(err)
	}

	session, err := mgo.Dial(c.Mapped["27017/tcp"].String())
	if err != nil {
		panic(err)
	}
	defer session.Close()

	type Data struct {
		S string
	}
	col := session.DB("test").C("data")
	if err = col.Insert(&Data{S: "the answer to life the universe&everything"}); err != nil {
		panic(err)
	}
	var r Data
	if err = col.Find(nil).One(&r); err != nil {
		panic(err)
	}
	fmt.Println(len(r.S))
	// Output:
	// 42
}
