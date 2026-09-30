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
)




func GetTeachersDbHandler(teachers []models.Teacher, r *http.Request) ([]models.Teacher, error) {
	db, err := ConnectDb()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer db.Close()

	query := "SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE 1=1"
	var args []interface{}

	query, args = utils.AddFilters(r, query, args)
	query = utils.AddSorting(r, query)

	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Println(err)
		return nil, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer rows.Close()

	// teacher := make([]models.Teacher, 0)
	for rows.Next() {
		var teacher models.Teacher
		err := rows.Scan(&teacher.Id, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error retrieving data")
		}
		teachers = append(teachers, teacher)
	}
	return teachers, nil
}

func GetTeacherById(w http.ResponseWriter, id int) (models.Teacher, error) {
	db, err := ConnectDb()
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error adding data")
	}
	defer db.Close()

	var teacher models.Teacher

	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id=?", id).Scan(&teacher.Id, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
	if err == sql.ErrNoRows {
		return models.Teacher{}, utils.ErrorHandler(err, "Error adding data")
	} else if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error adding data")
	}
	return teacher, nil
}

func AddTeacherDbHandler(newTeachers []models.Teacher) ([]models.Teacher, error) {
	db, err := ConnectDb()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error adding data")
	}
	defer db.Close()

	// stmt, err := db.Prepare("INSERT INTO teachers (first_name, last_name, email, class, subject) VALUES(?,?,?,?,?)")
	stmt, err := db.Prepare(utils.GenerateInsertQuery("teachers", models.Teacher{}))
	if err != nil {
		fmt.Println("Prepare error:", err)
		return nil, utils.ErrorHandler(err, "Error adding data")
	}

	addedTeachers := make([]models.Teacher, len(newTeachers))
	for i, newTeacher := range newTeachers {
		// result, err := stmt.Exec(newTeacher.FirstName, newTeacher.LastName, newTeacher.Email, newTeacher.Class, newTeacher.Subject)
		values := utils.GetStructValues(newTeacher)
		result, err := stmt.Exec(values...)
		if err != nil {
			fmt.Println("Exec error:", err)
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		lastID, err := result.LastInsertId()
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		newTeacher.Id = int(lastID)

		addedTeachers[i] = newTeacher

	}
	stmt.Close()
	return addedTeachers, nil
}

func UpdateTeacher(id int, updatedTeacher models.Teacher) (models.Teacher, error) {
	db, err := ConnectDb()
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}
	defer db.Close()

	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id =?", id).Scan(&existingTeacher.Id, &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)
	if err == sql.ErrNoRows {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}

	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}

	updatedTeacher.Id = existingTeacher.Id
	_, err = db.Exec("UPDATE teachers SET first_name =?, last_name =?, email=?, class=?, subject=? WHERE id=?", updatedTeacher.FirstName, updatedTeacher.LastName, updatedTeacher.Email, updatedTeacher.Class, updatedTeacher.Subject, updatedTeacher.Id)
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}
	return updatedTeacher, nil
}

func PatchTeachers(updates []map[string]interface{}) error {
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
			return utils.ErrorHandler(err, "Invalid ID  ")
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			tx.Rollback()
			return utils.ErrorHandler(err, "Invalid ID")
		}
		var teacherFromDb models.Teacher

		err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&teacherFromDb.Id, &teacherFromDb.FirstName, &teacherFromDb.LastName, &teacherFromDb.Email, &teacherFromDb.Class, &teacherFromDb.Subject)
		if err != nil {
			if err == sql.ErrNoRows {
				tx.Rollback()
				return utils.ErrorHandler(err, "Teacher not found")
			}
			tx.Rollback()
			return utils.ErrorHandler(err, "Error updating data")
		}

		teacherVal := reflect.ValueOf(&teacherFromDb).Elem()
		teacherType := teacherVal.Type()

		for k, v := range update {
			if k == "id" {
				continue
			}
			for i := 0; i < teacherVal.NumField(); i++ {
				field := teacherType.Field(i)
				if field.Tag.Get("json") == k+",omitempty" {
					fieldVal := teacherVal.Field(i)
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
		_, err = tx.Exec("UPDATE teachers SET first_name =?, last_name =?, email=?, class=?, subject=? WHERE id=?", teacherFromDb.FirstName, teacherFromDb.LastName, teacherFromDb.Email, teacherFromDb.Class, teacherFromDb.Subject, teacherFromDb.Id)
		if err != nil {
			tx.Rollback()
			return utils.ErrorHandler(err, "Error updating data")
		}
	}
	err = tx.Commit()
	if err != nil {
		return utils.ErrorHandler(err, "Error updating data")
	}
	return nil
}

func PatchOneTeacher(id int, updates map[string]interface{}) (models.Teacher, error) {
	db, err := ConnectDb()
	if err != nil {
		log.Println(err)
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}
	defer db.Close()

	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id =?", id).Scan(&existingTeacher.Id, &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)
	if err == sql.ErrNoRows {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}

	if err == sql.ErrNoRows {
		return models.Teacher{}, utils.ErrorHandler(err, "Teacher not found")
	}

	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}

	teacherVal := reflect.ValueOf(&existingTeacher).Elem()
	teacherType := teacherVal.Type()

	for k, v := range updates {
		for i := 0; i < teacherVal.NumField(); i++ {
			field := teacherType.Field(i)
			if field.Tag.Get("json") == k+",omitempty" {
				if teacherVal.Field(i).CanSet() {
					teacherVal.Field(i).Set(reflect.ValueOf(v).Convert(teacherVal.Field(i).Type()))
				}
			}
		}
	}

	_, err = db.Exec("UPDATE teachers SET first_name =?, last_name =?, email=?, class=?, subject=? WHERE id=?", existingTeacher.FirstName, existingTeacher.LastName, existingTeacher.Email, existingTeacher.Class, existingTeacher.Subject, existingTeacher.Id)
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}
	return existingTeacher, nil
}

func DeleteOneTeacher(id int) error {
	db, err := ConnectDb()
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}
	defer db.Close()

	result, err := db.Exec("DELETE FROM teachers WHERE id=?", id)
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	if rowsAffected == 0 {
		return utils.ErrorHandler(err, "Teacher not found")
	}

	//w.WriteHeader(http.StatusNoContent)

	return nil
}

func DeleteTeachers(ids []int) ([]int, error) {
	db, err := ConnectDb()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error deleting data")
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error deleting data")
	}

	stmt, err := tx.Prepare("DELETE FROM teachers WHERE id=?")
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

func GetStudentByTeacherIdDb(teacherId string, students []models.Student) ([]models.Student, error) {
	db, err := ConnectDb()
	if err != nil {
		log.Println(err)
		return nil, utils.ErrorHandler(err, "error updating data")
	}
	defer db.Close()

	query := "SELECT id, first_name, last_name, email, class FROM students WHERE class=(SELECT class from teachers WHERE id=?)"
	rows, err := db.Query(query, teacherId)
	if err != nil {
		return nil, utils.ErrorHandler(err, "error retrieving data")
	}
	defer rows.Close()

	for rows.Next() {
		var student models.Student
		err = rows.Scan(&student.Id, &student.FirstName, &student.LastName, &student.Email, &student.Class)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Invalid class")
		}
		students = append(students, student)
	}
	err = rows.Err()
	if err != nil {
		return nil, utils.ErrorHandler(err, "error retrieving data")
	}
	return students, nil
}

func GetStudentsCountByTeacherIdDb(teacherId string) (int,error) {
	db, err := ConnectDb()
	if err != nil {
		log.Println(err)
		return 0,utils.ErrorHandler(err,"error retrieving data")
	}
	defer db.Close()

	query := "SELECT COUNT(*) FROM students WHERE class=(SELECT class FROM teachers WHERE id=?)"

	var studentsCount int
	err = db.QueryRow(query, teacherId).Scan(&studentsCount)
	if err != nil {
		return 0,utils.ErrorHandler(err,"error retrieving data")
	}
	return studentsCount,nil
}

