package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	_ "database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"restapi/internal/models"
	"restapi/internal/repository/sqlconnect"
	"restapi/pkg/utils"
	"strconv"
	"time"
	"github.com/go-mail/mail/v2"
)

func GetExecsHandler(w http.ResponseWriter, r *http.Request) {
	var execs []models.Exec
	execs, err := sqlconnect.GetExecsDbHandler(execs, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := struct {
		Status string        `json:"status"`
		Count  int           `json:"count"`
		Data   []models.Exec `json:"data"`
	}{
		Status: "success",
		Count:  len(execs),
		Data:   execs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetOneExecHandler(w http.ResponseWriter, r *http.Request) {

	idstr := r.PathValue("id")

	id, err := strconv.Atoi(idstr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	Exec, err := sqlconnect.GetTeacherById(w, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Exec)

}

func AddExecsHandler(w http.ResponseWriter, r *http.Request) {

	var newExecs []models.Exec
	var rawExecs []map[string]interface{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "error reading body data", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	err = json.Unmarshal(body, &rawExecs)
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &newExecs)
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fields := utils.GetFieldNames(models.Exec{})

	allowedFields := make(map[string]struct{})
	for _, field := range fields {
		allowedFields[field] = struct{}{}
	}

	for _, Exec := range newExecs {
		val := reflect.ValueOf(Exec)
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			if field.Kind() == reflect.String && field.String() == "" {
				http.Error(w, "All fields are required", http.StatusBadRequest)
				return
			}
		}
	}

	for _, Exec := range rawExecs {
		for key := range Exec {
			_, ok := allowedFields[key]
			if !ok {
				http.Error(w, "Unacceptable field found in request, Only allwed fields are allowed", http.StatusBadRequest)
				return
			}
		}
	}

	addedExecs, err := sqlconnect.AddExecDbHandler(newExecs)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	Response := struct {
		Status string        `json:"status"`
		Count  int           `json:"count"`
		Data   []models.Exec `json:"data"`
	}{
		Status: "success",
		Count:  len(addedExecs),
		Data:   addedExecs,
	}
	json.NewEncoder(w).Encode(Response)
}

// PATCH /teachers/{id}
func PatchExecsHandler(w http.ResponseWriter, r *http.Request) {

	var updates []map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&updates)
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid Request Payload", http.StatusBadRequest)
		return
	}

	err = sqlconnect.PatchExecs(updates)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func PatchOneExecHandler(w http.ResponseWriter, r *http.Request) {
	idstr := r.PathValue("id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid Teacher ID", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	err = json.NewDecoder(r.Body).Decode(&updates)
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid Request Payload", http.StatusBadRequest)
		return
	}

	updatedExec, err := sqlconnect.PatchOneExec(id, updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedExec)

}

func DeleteOneExecHandler(w http.ResponseWriter, r *http.Request) {
	idstr := r.PathValue("id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid Teacher ID", http.StatusBadRequest)
		return
	}

	err = sqlconnect.DeleteOneExec(id)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status string `json:"status"`
		ID     int    `json:"id"`
	}{
		Status: "Exec Successfully Deleted",
		ID:     id,
	}
	json.NewEncoder(w).Encode(response)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req models.Exec
	//Data Validation
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	//search if user actually exist
	user, err := sqlconnect.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusInternalServerError)
		return
	}
	//is user active
	if user.InactiveStatus {
		http.Error(w, "Account is inactive", http.StatusForbidden)
		return
	}

	//verify the password
	err = utils.VerifyPassword(req.Password, user.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	//generate token
	tokenString, err := utils.SignToken(strconv.Itoa(user.Id), user.Username, user.Role)
	if err != nil {
		http.Error(w, "Coud not create login token", http.StatusInternalServerError)
		return
	}
	//send token as a response or as a cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().Add(24 * time.Hour),
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "testabc",
		Value:    "testing1231",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().Add(24 * time.Hour),
		SameSite: http.SameSiteStrictMode,
	})

	//Response body

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Token string `json:"token"`
	}{
		Token: tokenString,
	}
	json.NewEncoder(w).Encode(response)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Unix(0, 0),
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"Logged out succesfully"}`))
}

func UpdatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	idstr := r.PathValue("id")
	userId, err := strconv.Atoi(idstr)
	if err != nil {
		http.Error(w, "Invalid execs Id", http.StatusBadGateway)
	}

	var req models.UpdatePasswordRequest
	json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
	}
	r.Body.Close()

	if req.CurrentPassword == "" || req.NewPassword == "" {
		http.Error(w, "Please enter password", http.StatusBadRequest)
		return
	}

	_, err = sqlconnect.UpdatePasswordInDb(userId, req.CurrentPassword, req.NewPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// //send token as a response or as a cookie
	// http.SetCookie(w, &http.Cookie{
	// 	Name:     "Bearer",
	// 	Value:    token,
	// 	Path:     "/",
	// 	HttpOnly: true,
	// 	Secure:   true,
	// 	Expires:  time.Now().Add(24 * time.Hour),
	// 	SameSite: http.SameSiteStrictMode,
	// })

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Message string `json:"message"`
	}{
		Message: "Password updated successfully",
	}
	json.NewEncoder(w).Encode(response)

}

func ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err!=nil{
		http.Error(w,"Invalid request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	db,err:= sqlconnect.ConnectDb()
	if err!=nil{
		utils.ErrorHandler(err, "Internal error")
		return
	}
	defer db.Close()

	var exec models.Exec
	err=db.QueryRow("SELECT id FROM execs WHERE email=?", req.Email).Scan(&exec.Id)
	if err!=nil{
		utils.ErrorHandler(err, "User not found")
		return
	}

	duration, err:=strconv.Atoi(os.Getenv("RESET_TOKEN_EXP_DURATION"))
	if err!=nil{
		utils.ErrorHandler(err, "Failed to send reset email")
		return
	}

	mins := time.Duration(duration)

	expiry:= time.Now().Add(mins*time.Minute).Format(time.RFC3339)

	tokenBytes := make([]byte,32)
	_,err=rand.Read(tokenBytes)
	if err!=nil{
		utils.ErrorHandler(err, "Failed to send password reset email")
		return
	}

	log.Println("tokenBytes:", tokenBytes)
	token := hex.EncodeToString(tokenBytes)
	log.Println("token: ", token)

	hashedToken := sha256.Sum256(tokenBytes)
	log.Println("hashedToken:",hashedToken)

	hashedTokenString:= hex.EncodeToString(hashedToken[:])

	_,err=db.Exec("UPDATE execs SET password_reset_token =?, password_token_expires=? WHERE id=?", hashedTokenString, expiry, exec.Id)
	if err!=nil{
		utils.ErrorHandler(err, "Failed to send password reset email")
		return
	}

	//Send to reset email
	resetURL := fmt.Sprintf("https://localhost:3000/execs/resetpassword/reset/%s",token)
	message := fmt.Sprintf("Forgot your password? Reset your password using the following link: \n%s\nif you didn't request a password reset, please ignore this email. This link is only valid for %d minutes.", resetURL, int(mins))

	m:=mail.NewMessage()
	m.SetHeader("From","schooladmin@school.com")
	m.SetHeader("To", req.Email)
	m.SetHeader("Subject","Your password reset link")
	m.SetBody("text/plain", message)

	d:=mail.NewDialer("localhost", 1025, "","")
	err=d.DialAndSend(m)
	if err!=nil{
		utils.ErrorHandler(err, "Failed to send password reset email")
		return
	}

	// respond with success message

	fmt.Fprintf(w,"PAssword reset link sent to %s",req.Email)



}
