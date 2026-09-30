package sqlconnect

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_"github.com/joho/godotenv"
)

func ConnectDb() (*sql.DB,error){

	// err:=godotenv.Load()
	// if err!=nil{
	// 	fmt.Println("Error:",err)
	// 	return nil,err
	// }

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName:= os.Getenv("DB_NAME")
	host:=os.Getenv("HOST")
	dbPort:=os.Getenv("DB_PORT")
	// connectionString := "root:man123456@tcp(127.0.0.1:3306)/" + dbName
	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",user,password,host,dbPort,dbName)
	sqlDB,err:=sql.Open("mysql",connectionString)
	if err!=nil{
		// panic(err)
		return nil, err
	}
	return  sqlDB,nil
}