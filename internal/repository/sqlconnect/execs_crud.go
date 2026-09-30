package sqlconnect

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"restapi/internal/models"
	"restapi/pkg/utils"
	"strconv"
	"time"
)

func GetExecsDbHandler(execs []models.Exec, r *http.Request) ([]models.Exec, error) {
	db, err := ConnectDb()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer db.Close()

	query := "SELECT id, first_name, last_name, email, username, user_created_at, inactive_status, role FROM execs WHERE 1=1"
	var args []interface{}

	query, args = utils.AddFilters(r, query, args)
	query = utils.AddSorting(r, query)

	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Println(err)
		return nil, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer rows.Close()

	for rows.Next() {
		var exec models.Exec
		err := rows.Scan(&exec.Id, &exec.FirstName, &exec.LastName, &exec.Email, &exec.Username, &exec.UserCreatedAt, &exec.InactiveStatus, &exec.Role)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error retrieving data")
		}
		execs = append(execs, exec)
	}
	return execs, nil
}

func GetExecById(w http.ResponseWriter, id int) (models.Exec, error) {
	db, err := ConnectDb()
	if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "Error adding data")
	}
	defer db.Close()

	var exec models.Exec

	err = db.QueryRow("SELECT id, first_name, last_name, email, username, inactive_status, role FROM execs WHERE id=?", id).Scan(&exec.Id, &exec.FirstName, &exec.LastName, &exec.Email, &exec.Username, &exec.InactiveStatus, &exec.Role)
	if err == sql.ErrNoRows {
		return models.Exec{}, utils.ErrorHandler(err, "Error adding data")
	} else if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "Error adding data")
	}
	return exec, nil
}

func AddExecDbHandler(newExecs []models.Exec) ([]models.Exec, error) {
	db, err := ConnectDb()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error adding data")
	}
	defer db.Close()

	// stmt, err := db.Prepare("INSERT INTO execs (first_name, last_name, email, class, subject) VALUES(?,?,?,?,?)")
	stmt, err := db.Prepare(utils.GenerateInsertQuery("execs", models.Exec{}))
	if err != nil {
		fmt.Println("Prepare error:", err)
		return nil, utils.ErrorHandler(err, "Error adding data")
	}

	addedExecs := make([]models.Exec, len(newExecs))
	for i, newExec := range newExecs {
		newExec.Password, err = utils.HashPassword(newExec.Password)
		if err != nil {
			return nil, utils.ErrorHandler(err, "error adding exec into database")
		}

		values := utils.GetStructValues(newExec)
		result, err := stmt.Exec(values...)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		lastID, err := result.LastInsertId()
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		newExec.Id = int(lastID)

		addedExecs[i] = newExec

	}
	stmt.Close()
	return addedExecs, nil
}

func PatchExecs(updates []map[string]interface{}) error {
	db, err := ConnectDb()
	if err != nil {
		return utils.ErrorHandler(err, "Error updating data")
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return utils.ErrorHandler(err, "Error updating data")
	}

	for _, update := range updates {
		idStr, ok := update["id"].(string)
		if !ok {
			tx.Rollback()
			fmt.Println("1")
			return utils.ErrorHandler(err, "Invalid ID")
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			tx.Rollback()
			return utils.ErrorHandler(err, "Invalid ID")
		}
		var execFromDb models.Exec

		err = db.QueryRow("SELECT id, first_name, last_name, email, username FROM execs WHERE id = ?", id).Scan(&execFromDb.Id, &execFromDb.FirstName, &execFromDb.LastName, &execFromDb.Email, &execFromDb.Username)
		if err != nil {
			if err == sql.ErrNoRows {
				tx.Rollback()
				return utils.ErrorHandler(err, "exec not found")
			}
			tx.Rollback()
			return utils.ErrorHandler(err, "Error updating data")
		}

		execVal := reflect.ValueOf(&execFromDb).Elem()
		execType := execVal.Type()

		for k, v := range update {
			if k == "id" {
				continue
			}
			for i := 0; i < execVal.NumField(); i++ {
				field := execType.Field(i)
				if field.Tag.Get("json") == k+",omitempty" {
					fieldVal := execVal.Field(i)
					if fieldVal.CanSet() {
						val := reflect.ValueOf(v)
						if val.Type().ConvertibleTo(fieldVal.Type()) {
							fieldVal.Set(val.Convert(fieldVal.Type()))
						} else {
							tx.Rollback()
							log.Printf("Cannot convert %v to %v", val.Type(), fieldVal.Type())
							return err
						}
					}
					break
				}
			}
		}
		_, err = tx.Exec("UPDATE execs SET first_name =?, last_name =?, email=?, username=? WHERE id=?", execFromDb.FirstName, execFromDb.LastName, execFromDb.Email, execFromDb.Username, execFromDb.Id)
		if err != nil {
			return utils.ErrorHandler(err, "Error updating data")
		}
	}
	err = tx.Commit()
	if err != nil {
		return utils.ErrorHandler(err, "Error updating data")
	}
	return nil
}

func PatchOneExec(id int, updates map[string]interface{}) (models.Exec, error) {
	db, err := ConnectDb()
	if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "Error updating data")
	}
	defer db.Close()

	var existingExec models.Exec
	err = db.QueryRow("SELECT id, first_name, last_name, email, username FROM execs WHERE id =?", id).Scan(&existingExec.Id, &existingExec.FirstName, &existingExec.LastName, &existingExec.Email, &existingExec.Username)
	if err == sql.ErrNoRows {
		return models.Exec{}, utils.ErrorHandler(err, "Error updating data")
	}

	if err == sql.ErrNoRows {
		return models.Exec{}, utils.ErrorHandler(err, "exec not found")
	}

	if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "Error updating data")
	}

	execVal := reflect.ValueOf(&existingExec).Elem()
	execType := execVal.Type()

	for k, v := range updates {
		for i := 0; i < execVal.NumField(); i++ {
			field := execType.Field(i)
			if field.Tag.Get("json") == k+",omitempty" {
				if execVal.Field(i).CanSet() {
					execVal.Field(i).Set(reflect.ValueOf(v).Convert(execVal.Field(i).Type()))
				}
			}
		}
	}

	_, err = db.Exec("UPDATE execs SET first_name =?, last_name =?, email=?, username=? WHERE id=?", existingExec.FirstName, existingExec.LastName, existingExec.Email, existingExec.Username, existingExec.Id)
	if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "Error updating data")
	}
	return existingExec, nil
}

func DeleteOneExec(id int) error {
	db, err := ConnectDb()
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}
	defer db.Close()

	result, err := db.Exec("DELETE FROM execs WHERE id=?", id)
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	if rowsAffected == 0 {
		return utils.ErrorHandler(err, "exec not found")
	}
	return nil
}

func GetUserByUsername(username string) (*models.Exec, error) {
	db, err := ConnectDb()
	if err != nil {
		return nil, utils.ErrorHandler(err, "error updting data")
	}
	defer db.Close()

	user := &models.Exec{}
	err = db.QueryRow("SELECT id, first_name, last_name, email, username, password, inactive_status, role FROM execs WHERE username=?", username).Scan(&user.Id, &user.FirstName, &user.LastName, &user.Email, &user.Username, &user.Password, &user.InactiveStatus, &user.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, utils.ErrorHandler(err, "user not found")
		}
		return nil, utils.ErrorHandler(err, "Database error")
	}
	return user, nil
}

func UpdatePasswordInDb(userId int, currentPassword, NewPassword string) (bool, error) {
	db, err := ConnectDb()
	if err != nil {
		return false, utils.ErrorHandler(err, "database connection error")

	}
	defer db.Close()

	var username string
	var userPassword string
	var userRole string

	err = db.QueryRow("SELECT username, password, role FROM execs WHERE id=?", userId).Scan(&username, &userPassword, &userRole)
	if err != nil {

		return false, utils.ErrorHandler(err, "User not found")
	}

	err = utils.VerifyPassword(currentPassword, userPassword)
	if err != nil {

		return false, utils.ErrorHandler(err, "The password you entered does not match the current password")
	}

	hashedPassword, err := utils.HashPassword(NewPassword)
	if err != nil {
		utils.ErrorHandler(err, "internal Error")
	}

	currentTime := time.Now().Format(time.RFC3339)

	_, err = db.Exec("UPDATE execs SET password =?, password_changed_at=? WHERE id=?", hashedPassword, currentTime, userId)
	if err != nil {

		return false, utils.ErrorHandler(err, "Failed to update the password")
	}

	// token, err:=utils.SignToken(strconv.Itoa(userId),username,userRole)
	// if err!=nil{
	// 	utils.ErrorHandler(err, "Password update, Could not create token")
	// 	return
	// }

	return true, nil
}
