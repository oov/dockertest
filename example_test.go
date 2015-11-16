package dockertest_test

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/oov/dockertest"
)

func Example() {
	const (
		User     = "dockertestdb"
		Password = "mypassword"
		DBName   = User
	)
	c, err := dockertest.New(dockertest.Config{
		Image: "postgres", // or "postgres:latest"
		PortMapping: map[string]string{
			"5432/tcp": "127.0.0.1:0",
		},
		Env: map[string]string{
			"POSTGRES_USER":     User,
			"POSTGRES_PASSWORD": Password,
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

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		User, Password, c.Mapped["5432/tcp"], DBName)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

ping:
	if err = db.Ping(); err != nil {
		// Sometimes fails with this error, so we need ignore and retry later.
		if err.Error() == "pq: the database system is starting up" {
			time.Sleep(50 * time.Millisecond)
			goto ping
		}
		panic(err)
	}

	var r int
	if err = db.QueryRow("SELECT LENGTH('the answer to life the universe&everything')").Scan(&r); err != nil {
		panic(err)
	}
	fmt.Println(r)
	// Output:
	// 42
}
