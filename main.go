package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ClassOffer struct {
	ID         int    `gorm:"primaryKey"`
	StudentID   string //studentId
	Offer       string
	Want        string
	IsCompleted string //studentId
}

var db *gorm.DB

func main() {
	initDb()
	migrateDb()
	initRouter()
}

func initDb() {
	dsn := "user:password@tcp(127.0.0.1:3306)/Assignment2?charset=utf8mb4&parseTime=True&loc=Local"

	var err error

	//set global var "db"
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database with DSN: " + dsn)
	}
}

func migrateDb() {
	// GORM Automigrate https://gorm.io/docs/migration.html
	// Creates/Updates DB table acording to struct
	err := db.AutoMigrate(&ClassOffer{})
	if err != nil {
		panic("DB Migration failed with error: " + err.Error())
	}
}

func initRouter() {
	router := mux.NewRouter()

	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"})

	router.HandleFunc("/classOffers", getClassOffers).Methods("GET") //has url query params
	router.HandleFunc("/classOffers", createClassOffer).Methods("POST")
	router.HandleFunc("/classOffers/{id}", updateClassOffer).Methods("PUT")
	router.HandleFunc("/classOffers/{id}", deleteClassOffer).Methods("DELETE")

	portNo := 9021

	fmt.Printf("ClassOffer Microservice running on port %s...\n", portNo)
	err := http.ListenAndServe(":"+portNo, handlers.CORS(originsOk, headersOk, methodsOk)(router))
	if err != nil {
		panic("InitRouter failed with error: " + err.Error())
	}
}
/////////////////////////
//                     //
//    HTTP Functions   //
//                     //
/////////////////////////

func getClassOffers(w http.ResponseWriter, r *http.Request) {
	var classOffers []ClassOffer

	urlParams := r.URL.Query()
	queryWant, haveWant := urlParams["want"]
	queryOffer, haveOffer := urlParams["offer"]

	if haveWant && !haveOffer {
		want := queryWant[0]
		db.Where("want = ?", want).Find(&classOffers)
	} else if !haveWant && haveOffer {
		offer := queryOffer[0]
		db.Where("offer = ?", offer).Find(&classOffers)
	} else if haveWant && haveOffer {
		want := queryWant[0]
		offer := queryOffer[0]
		db.Where("want = ? AND offer = ?", want, offer).Find(&classOffers)
	} else {
		db.Find(&classOffers)
	}

	httpRespondWith(w, http.StatusOK, classOffers)
}

func createClassOffer(w http.ResponseWriter, r *http.Request) {
	var classOffer ClassOffer

	decodeErr := json.NewDecoder(r.Body).Decode(&classOffer)
	if decodeErr != nil {
		httpRespondWith(w, http.StatusBadRequest, "Invalid JSON: "+decodeErr.Error())
		return
	}

	//validate empty fields
	if isFieldMissing(w, classOffer.CreatedBy, "createdBy") ||
		isFieldMissing(w, classOffer.Want, "want") ||
		isFieldMissing(w, classOffer.Offer, "offer") {
		return
	}

	//Disallow manual setting of Id
	//GORM will automatically assign Id if it is zero value
	classOffer.Id = 0

	//Offers should be incomplete on creation
	classOffer.CompletedBy = ""

	dbErr := db.Create(&classOffer).Error
	if dbErr != nil {
		httpRespondWith(w, http.StatusBadRequest, "Invalid Data")
		return
	}

	httpRespondWith(w, http.StatusOK, classOffer)
}

func updateClassOffer(w http.ResponseWriter, r *http.Request) {
	var classOffer ClassOffer

	decodeErr := json.NewDecoder(r.Body).Decode(&classOffer)
	if decodeErr != nil {
		httpRespondWith(w, http.StatusBadRequest, "Invalid JSON: "+decodeErr.Error())
		return
	}

	params := mux.Vars(r)
	id := params["id"]

	dbErr := db.Model(&ClassOffer{}).Where("id = ?", id).Updates(classOffer).Error

	if dbErr != nil {
		httpRespondWith(w, http.StatusBadRequest, "Invalid Data")
		return
	}

	var newClassOffer ClassOffer
	db.Where("id = ?", id).First(&newClassOffer)

	httpRespondWith(w, http.StatusOK, newClassOffer)
}

func deleteClassOffer(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	//check user exist
	idExist := existInDb("id", id)
	if !idExist {
		httpRespondWith(w, http.StatusNotFound, "Offer doesn't exist")
		return
	}

	//delete from driver where id = id
	db.Delete(&ClassOffer{}, id)

	httpRespondWith(w, http.StatusOK, fmt.Sprintf("Class Offer of ID %s successfully deleted", id))
}

func httpRespondWith(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

/*
This function checks whether a data has a zero value.
If it does, it will return true and write a http response
*/
func isFieldMissing(w http.ResponseWriter, data interface{}, fieldName string) bool {
	if isZero(data) {
		errorMsg := fmt.Sprintf("%s field is missing/incorrect.", fieldName)
		httpRespondWith(w, http.StatusBadRequest, errorMsg)
		return true
	}
	return false
}

/*
Check whether "data" has zero value
e.g. empty object, "false", 0, etc.
*/
func isZero(data interface{}) bool {
	value := reflect.ValueOf(data)
	return value.IsZero()
}

/*
This function returns whether a row with the given value for the given fieldName exists
*/
func existInDb(fieldName string, value interface{}) bool {
	var dbClassOffer ClassOffer

	//if email doens't exist, db.First returns a ErrRecordNotFound
	err := db.Where(fieldName+" = ?", value).First(&dbClassOffer).Error
	return err != gorm.ErrRecordNotFound
}
