package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

// API : Structure for the main app object
type API struct {
	Log    *zap.Logger
	Router *mux.Router
	Db     *mongo.Database
}

func jsonResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// StartAPI : Returns an API object
func StartAPI() API {
	var api API

	api.mountLogger()
	api.mountRouter()
	api.mountDatabase()

	return api
}

// Run : Start the server
func (api *API) Run() {
	server := &http.Server{
		Addr:         "0.0.0.0:8080",
		WriteTimeout: time.Second * 5 * 60,
		ReadTimeout:  time.Second * 5 * 60,
		IdleTimeout:  time.Second * 5 * 60,
		Handler:      api.Router,
	}

	go func() {
		api.Log.Info("Listening on port 8080")
		if err := server.ListenAndServe(); err != nil {
			api.Log.Info(err.Error())
			panic(errors.New("Server launch error"))
		}
	}()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	<-signalChannel

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	server.Shutdown(ctx)
	api.Log.Info("bye")
}

func (api *API) mountLogger() {
	var cfg zap.Config
	cfg = zap.NewDevelopmentConfig()

	cfg.OutputPaths = []string{
		"stdout",
	}
	cfg.ErrorOutputPaths = []string{
		"stderr",
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(errors.New("Could not initialize logger"))
	}
	defer logger.Sync()

	api.Log = logger
}

func (api *API) mountRouter() {
	api.Router = mux.NewRouter()
	api.Router.Use(jsonResponse)

	// Languages
	api.Router.HandleFunc("/addLanguage", api.addLanguageHandler).Methods("POST")
	api.Router.HandleFunc("/editLanguage", api.editLanguageHandler).Methods("POST")
	api.Router.HandleFunc("/deleteLanguage", api.deleteLanguageHandler).Methods("POST")

	// Questions
	api.Router.HandleFunc("/addQuestion", api.addQuestionHandler).Methods("POST")
	api.Router.HandleFunc("/editTestcases", api.editTestcasesHandler).Methods("POST")
	api.Router.HandleFunc("/editQuestion", api.editQuestionHandler).Methods("POST")
	api.Router.HandleFunc("/deleteQuestion", api.deleteQuestionHandler).Methods("POST")
}

func (api *API) mountDatabase() {
	//clientOpts := options.Client().ApplyURI("mongodb://mongo-0.mongo,mongo-1.mongo,mongo-2.mongo:27017/?replicaSet=rs0&readPreference=secondaryPreferred")
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOpts)
	if err != nil {
		api.Log.Info("Database connection failed, URI")
		panic(err)
	}

	err = client.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		api.Log.Info("Database connection failed, Ping")
		panic(err)
	} else {
		api.Log.Info("Connected to Database")
	}

	api.Db = client.Database("judge")

	// Initialize database with languages table if not available
	cols, err := api.Db.ListCollectionNames(context.TODO(), bson.D{})
	foundLanguagesTable := false
	for _, result := range cols {
		if result == "languages" {
			foundLanguagesTable = true
			break
		}
	}
	if !foundLanguagesTable {
		api.Db.Collection("languages").InsertMany(context.TODO(), []interface{}{
			Language{ID: primitive.NewObjectID(), Name: "C++", Time: 2, Filename: "main.cpp", Compile: "g++ -O2 --std=c++17 /tmp/runner/main.cpp", Execute: "/tmp/runner/a.out"},
			Language{ID: primitive.NewObjectID(), Name: "Java 8", Time: 4, Filename: "Main.java", Compile: "javac /tmp/runner/Main.java", Execute: "java -cp /tmp/runner Main"},
			Language{ID: primitive.NewObjectID(), Name: "Python 3", Time: 6, Filename: "main.py", Compile: "compilation-not-needed", Execute: "python3 /tmp/runner/main.py"},
		}, options.InsertMany().SetOrdered(false))
	}
}
