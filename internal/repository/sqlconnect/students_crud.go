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
	"strings"
)

// func isvalidSortedOrder(order string) bool {
// 	return order == "asc" || order == "desc"
// }

// func isValidSortField(field string) bool {
// 	validFields := map[string]bool{
// 		"first_name": true,
// 		"last_name":  true,
// 		"email":      true,
// 		"class":      true,
// 		"subject":    true,
// 	}
// 	return validFields[field]
// }

func GetStudentsDbHandler(students []models.Student, r *http.Request) ([]models.Student, error) {
	db, err := ConnectDb()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer db.Close()

	query := "SELECT id, first_name, last_name, email, class FROM students WHERE 1=1"
	var args []interface{}

	query, args = utils.AddFilters(r, query, args)
	query = utils.AddSorting(r, query)

	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Println(err)
		return nil, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer rows.Close()

	// student := make([]models.Student, 0)
	for rows.Next() {
		var student models.Student
		err := rows.Scan(&student.Id, &student.FirstName, &student.LastName, &student.Email, &student.Class)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error retrieving data")
		}
		students = append(students, student)
	}
	return students, nil
}

func GetStudentById(w http.ResponseWriter, id int) (models.Student, error) {
	db, err := ConnectDb()
	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error adding data")
	}
	defer db.Close()

	var student models.Student

	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id=?", id).Scan(&student.Id, &student.FirstName, &student.LastName, &student.Email, &student.Class)
	if err == sql.ErrNoRows {
		return models.Student{}, utils.ErrorHandler(err, "Error adding data")
	} else if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error adding data")
	}
	return student, nil
}

func AddStudentDbHandler(newStudents []models.Student) ([]models.Student, error) {
	db, err := ConnectDb()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error adding data")
	}
	defer db.Close()

	// stmt, err := db.Prepare("INSERT INTO students (first_name, last_name, email, class, subject) VALUES(?,?,?,?,?)")
	stmt, err := db.Prepare(utils.GenerateInsertQuery("students", models.Student{}))
	if err != nil {
		fmt.Println("Prepare error:", err)
		return nil, utils.ErrorHandler(err, "Error adding data")
	}

	addedStudents := make([]models.Student, len(newStudents))
	for i, newStudent := range newStudents {
		// result, err := stmt.Exec(newstudent.FirstName, newstudent.LastName, newstudent.Email, newstudent.Class, newstudent.Subject)
		values := utils.GetStructValues(newStudent)
		result, err := stmt.Exec(values...)
		if err != nil {
			if strings.Contains(err.Error(),"Cannot add or update a child row: a foreign key constraint fails (`school`.`students`, CONSTRAINT `students_ibfk_1` FOREIGN KEY (`class`) REFERENCES `teachers` (`class`))"){
				return nil,utils.ErrorHandler(err,"class/class teacher does not exist")
			}
			fmt.Println("Exec error:", err)
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		lastID, err := result.LastInsertId()
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		newStudent.Id = int(lastID)

		addedStudents[i] = newStudent

	}
	stmt.Close()
	return addedStudents, nil
}

func UpdateStudent(id int, updatedStudent models.Student) (models.Student, error) {
	db, err := ConnectDb()
	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}
	defer db.Close()

	var existingStudent models.Student
	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id =?", id).Scan(&existingStudent.Id, &existingStudent.FirstName, &existingStudent.LastName, &existingStudent.Email, &existingStudent.Class)
	if err == sql.ErrNoRows {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}

	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}

	updatedStudent.Id = existingStudent.Id
	_, err = db.Exec("UPDATE students SET first_name =?, last_name =?, email=?, class=? WHERE id=?", updatedStudent.FirstName, updatedStudent.LastName, updatedStudent.Email, updatedStudent.Class, updatedStudent.Id)
	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}
	return updatedStudent, nil
}

func PatchStudents(updates []map[string]interface{}) error {
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
		var studentFromDb models.Student

		err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&studentFromDb.Id, &studentFromDb.FirstName, &studentFromDb.LastName, &studentFromDb.Email, &studentFromDb.Class)
		if err != nil {
			if err == sql.ErrNoRows {
				tx.Rollback()
				return utils.ErrorHandler(err, "student not found")
			}
			tx.Rollback()
			return utils.ErrorHandler(err, "Error updating data")
		}

		studentVal := reflect.ValueOf(&studentFromDb).Elem()
		studentType := studentVal.Type()

		for k, v := range update {
			if k == "id" {
				continue
			}
			for i := 0; i < studentVal.NumField(); i++ {
				field := studentType.Field(i)
				if field.Tag.Get("json") == k+",omitempty" {
					fieldVal := studentVal.Field(i)
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
		_, err = tx.Exec("UPDATE students SET first_name =?, last_name =?, email=?, class=? WHERE id=?", studentFromDb.FirstName, studentFromDb.LastName, studentFromDb.Email, studentFromDb.Class, studentFromDb.Id)
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

func PatchOneStudent(id int, updates map[string]interface{}) (models.Student, error) {
	db, err := ConnectDb()
	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}
	defer db.Close()

	var existingStudent models.Student
	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id =?", id).Scan(&existingStudent.Id, &existingStudent.FirstName, &existingStudent.LastName, &existingStudent.Email, &existingStudent.Class)
	if err == sql.ErrNoRows {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}

	if err == sql.ErrNoRows {
		return models.Student{}, utils.ErrorHandler(err, "student not found")
	}

	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}

	studentVal := reflect.ValueOf(&existingStudent).Elem()
	studentType := studentVal.Type()

	for k, v := range updates {
		for i := 0; i < studentVal.NumField(); i++ {
			field := studentType.Field(i)
			if field.Tag.Get("json") == k+",omitempty" {
				if studentVal.Field(i).CanSet() {
					studentVal.Field(i).Set(reflect.ValueOf(v).Convert(studentVal.Field(i).Type()))
				}
			}
		}
	}

	_, err = db.Exec("UPDATE students SET first_name =?, last_name =?, email=?, class=? WHERE id=?", existingStudent.FirstName, existingStudent.LastName, existingStudent.Email, existingStudent.Class, existingStudent.Id)
	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}
	return existingStudent, nil
}

func DeleteOneStudent(id int) error {
	db, err := ConnectDb()
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}
	defer db.Close()

	result, err := db.Exec("DELETE FROM students WHERE id=?", id)
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	if rowsAffected == 0 {
		return utils.ErrorHandler(err, "student not found")
	}

	//w.WriteHeader(http.StatusNoContent)

	return nil
}

func DeleteStudents(ids []int) ([]int, error) {
	db, err := ConnectDb()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error deleting data")
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error deleting data")
	}

	stmt, err := tx.Prepare("DELETE FROM students WHERE id=?")
	if err != nil {
		tx.Rollback()
		return nil, utils.ErrorHandler(err, "Error deleting data")
	}
	defer stmt.Close()

	deletedIds := []int{}
	for _, id := range ids {
		result, err := stmt.Exec(id)
		if err != nil {
			tx.Rollback()
			return nil, utils.ErrorHandler(err, "Error deleting data")
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error deleting data")
		}

		if rowsAffected > 0 {
			deletedIds = append(deletedIds, id)
		}

		if rowsAffected < 1 {
			tx.Rollback()
			return nil, utils.ErrorHandler(err, fmt.Sprintf("ID %d not found", id))
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error deleting data")
	}

	if len(deletedIds) < 1 {
		return nil, utils.ErrorHandler(err, "Error deleting data")
	}
	return deletedIds, nil
}
