package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("sqlite3", "./info.db")
	if err != nil {
		fmt.Println(err)
	}
	//	defer db.Close()

	sqlStmt := `
	CREATE TABLE friends (
  id varchar(255) NOT NULL,
  name varchar(255) DEFAULT NULL,
  PRIMARY KEY (id)
);
	`

	//	sqlStmt := `
	//	create table friends (id integer not null primary key, name text);
	//	delete from friends;
	//	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		//		return
	}

	sqlStmt = `
	create table user (id integer not null primary key, name text);
	delete from user;
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		//		return
	}

	//	Friends_add("123456")

	fs, err := Friends_getall()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(fs)

	//将所有用户id载入内存，方便查询
	if loadFriends() != nil {
		panic("载入用户到内存失败 查询数据库错误")
	}
}
