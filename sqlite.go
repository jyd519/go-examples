package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

func isTableExists(db *sql.DB, table string) (bool, error) {
	var cnt int
	err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&cnt)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		checkErr(err)
	default:
	}
	return cnt > 0, nil
}

func createdb(db *sql.DB) {

	var err error
	var exists bool

	exists, err = isTableExists(db, "userinfo")
	checkErr(err)
	if !exists {
		db.Exec(`CREATE TABLE userinfo (
			uid INTEGER PRIMARY KEY AUTOINCREMENT,
			username VARCHAR(64) NULL,
			departname VARCHAR(64) NULL,
			created DATE NULL);`)
	}

	exists, err = isTableExists(db, "userdeatail")
	checkErr(err)
	if !exists {
		db.Exec(` CREATE TABLE userdetail (
			uid INT(10) NULL,
			intro TEXT NULL,
			profile TEXT NULL,
			PRIMARY KEY (uid)); `)
	}
}

func main() {
	db, err := sql.Open("sqlite3", "./foo.db")
	checkErr(err)

	defer db.Close()

	createdb(db)

	//插入数据
	stmt, err := db.Prepare("INSERT INTO userinfo(username, departname, created) values(?,?,?)")
	checkErr(err)

	res, err := stmt.Exec("astaxie", "研发部门", "2012-12-09")
	checkErr(err)

	id, err := res.LastInsertId()
	checkErr(err)

	fmt.Println(id)
	//更新数据
	stmt, err = db.Prepare("update userinfo set username=? where uid=?")
	checkErr(err)

	res, err = stmt.Exec("astaxieupdate", id)
	checkErr(err)

	affect, err := res.RowsAffected() // 1
	checkErr(err)

	fmt.Println(affect)

	//查询数据
	rows, err := db.Query("SELECT * FROM userinfo")
	checkErr(err)

	for rows.Next() {
		var uid int
		var username string
		var department string
		var created time.Time
		err = rows.Scan(&uid, &username, &department, &created)
		checkErr(err)
		fmt.Println(uid, username, department, created)
	}

	//删除数据
	stmt, err = db.Prepare("delete from userinfo where uid=?")
	checkErr(err)

	res, err = stmt.Exec(id)
	checkErr(err)

	affect, err = res.RowsAffected()
	checkErr(err)

	fmt.Println(affect)

}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
