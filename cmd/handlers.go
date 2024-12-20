package main

import (
	"1008001/splitwiser/internal/funcs"
	"1008001/splitwiser/internal/models"
	"1008001/splitwiser/internal/store"
	"1008001/splitwiser/web"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/schema"
	"github.com/urfave/negroni"
)

var router *http.ServeMux
var templates *template.Template
var decoder = schema.NewDecoder()

func renderTemplate(t *template.Template, w http.ResponseWriter, tmpl string, data any) {
	err := t.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var timeConverter = func(value string) reflect.Value {
	if v, err := time.Parse("2006-01-02", value); err == nil {
		return reflect.ValueOf(v)
	}
	return reflect.Value{} // this is the same as the private const invalidType
}

func init() {
	router = http.NewServeMux()
	decoder.RegisterConverter(time.Time{}, timeConverter)
	// var templates = template.Must(template.ParseGlob("web/templates/*.tmpl")).Funcs(funcs.TemplateFuncs)
	var err error
	templates, err = template.New("").Funcs(funcs.TemplateFuncs).ParseFS(web.EmbeddedFiles, "templates/*.tmpl")
	if err != nil {
		slog.Error(err.Error())
	}
}

func routes() http.Handler {
	fileServer := http.FileServer(http.FS(web.EmbeddedFiles))
	router.Handle("GET /static/", fileServer)

	router.HandleFunc("GET /{$}", Index)
	router.HandleFunc("POST /t/new", NewTrip)
	router.HandleFunc("GET /t/{tripId}", GetTrip)
	router.HandleFunc("POST /t/{tripId}", UpdateTrip)
	router.HandleFunc("POST /t/{tripId}/u", AddUser)
	router.HandleFunc("POST /t/{tripId}/u/{userId}", DeleteUser)
	router.HandleFunc("POST /t/{tripId}/s", UpdateSchedule)
	router.HandleFunc("POST /t/{tripId}/e", NewExpense)

	n := negroni.Classic() // default middleware: panic recovery, logger, static serving
	n.UseHandler(router)

	return n
}

func Index(w http.ResponseWriter, r *http.Request) {
	renderTemplate(templates, w, "index", nil)
}

func NewTrip(w http.ResponseWriter, r *http.Request) {
	trip := models.NewTrip()
	store.SaveTrip(trip)
	http.Redirect(w, r, fmt.Sprintf("/t/%s", trip.Id), http.StatusSeeOther)
}

func GetTrip(w http.ResponseWriter, r *http.Request) {
	tripId := r.PathValue("tripId")
	tripData := store.GetTripData(tripId)
	if tripData == nil {
		http.Error(w, "trip not found", http.StatusNotFound)
	} else {
		renderTemplate(templates, w, "trip", tripData)
	}
}

func UpdateTrip(w http.ResponseWriter, r *http.Request) {
	tripId := r.PathValue("tripId")
	err := r.ParseForm()
	if err != nil {
		slog.Error(err.Error())
	}

	trip := new(models.Trip)
	trip.Id = tripId
	err = decoder.Decode(trip, r.PostForm)
	if err != nil {
		slog.Error(err.Error(), "postform", r.PostForm)
	}
	store.SaveTrip(trip)
	tripData := store.GetTripData(trip.Id)
	renderTemplate(templates, w, "trip_detail", tripData)
}

func AddUser(w http.ResponseWriter, r *http.Request) {
	tripId := r.PathValue("tripId")

	name := r.PostFormValue("name")
	user := models.NewUser()
	user.Name = name
	store.AddUser(tripId, user)

	tripData := store.GetTripData(tripId)
	renderTemplate(templates, w, "trip_detail", tripData)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	tripId := r.PathValue("tripId")
	trip := store.GetTripData(tripId)

	// TODO can only delete if not involved in expenses - check!

	// userId := r.PostFormValue("userId")
	renderTemplate(templates, w, "trip_detail", trip)
}

func UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	tripId := r.PathValue("tripId")
	err := r.ParseForm()
	if err != nil {
		slog.Error(err.Error())
	}
	schedule := make([]*models.ScheduleEntry, 0)
	users := store.GetUsers(tripId)
	for k := range r.PostForm {
		se, err := models.NewScheduleEntry(users, k)
		if err != nil {
			slog.Error(err.Error())
		}
		schedule = append(schedule, se)
	}
	store.SaveSchedule(tripId, schedule)
	tripData := store.GetTripData(tripId)
	renderTemplate(templates, w, "trip_detail", tripData)
}

func NewExpense(w http.ResponseWriter, r *http.Request) {
	tripId := r.PathValue("tripId")
	err := r.ParseForm()
	if err != nil {
		slog.Error(err.Error())
	}
	expense := new(models.Expense)
	err = decoder.Decode(expense, r.PostForm)
	if err != nil {
		slog.Error(err.Error(), "postform", r.PostForm)
	}
	store.SaveExpense(tripId, expense)
	tripData := store.GetTripData(tripId)
	renderTemplate(templates, w, "trip_detail", tripData)
}
