package dockertest_test

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/oov/dockertest"
)

func Example_mySQL() {
	const (
		User     = "dockertest"
		Password = "mypassword"
		DBName   = "dockertestdb"
	)
	c, err := dockertest.New(dockertest.Config{
		Image: "mysql", // or "mysql:latest"
		Name:  "dockertest-mysql",
		PortMapping: map[string]string{
			"3306/tcp": "auto",
		},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": Password,
			"MYSQL_DATABASE":      DBName,
			"MYSQL_USER":          User,
			"MYSQL_PASSWORD":      Password,
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

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4",
		User, Password, c.Mapped["3306/tcp"], DBName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var r int
	if err = db.QueryRow("SELECT LENGTH('the answer to life the universe&everything')").Scan(&r); err != nil {
		panic(err)
	}
	fmt.Println(r)
	// Output:
	// 42
}
