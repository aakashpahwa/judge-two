package api

import (
	"encoding/json"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddLanguageRequest : Add language support
type AddLanguageRequest struct {
	Name     string `json:"name"`
	Time     int    `json:"time"`
	Filename string `json:"filename"`
	Compile  string `json:"compile"`
	Execute  string `json:"execute"`
}

func (r AddLanguageRequest) validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required),
		validation.Field(&r.Time, validation.Required),
		validation.Field(&r.Filename, validation.Required),
		validation.Field(&r.Compile, validation.Required),
		validation.Field(&r.Execute, validation.Required),
	)
}

// AddLanguageResponse : ID of the added language
type AddLanguageResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

// EditLanguageRequest : Edit language support
type EditLanguageRequest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Time     int    `json:"time"`
	Filename string `json:"filename"`
	Compile  string `json:"compile"`
	Execute  string `json:"execute"`
}

func (r EditLanguageRequest) validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.ID, validation.Required),
		validation.Field(&r.Name, validation.Required),
		validation.Field(&r.Time, validation.Required),
		validation.Field(&r.Filename, validation.Required),
		validation.Field(&r.Compile, validation.Required),
		validation.Field(&r.Execute, validation.Required),
	)
}

// DeleteLanguageRequest : Delete language support
type DeleteLanguageRequest struct {
	ID string `json:"id"`
}

func (r DeleteLanguageRequest) validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.ID, validation.Required),
	)
}

func (api *API) addLanguageHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody AddLanguageRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	if err := reqBody.validate(); err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	language := Language{
		ID:       primitive.NewObjectID(),
		Name:     reqBody.Name,
		Time:     reqBody.Time,
		Filename: reqBody.Filename,
		Compile:  reqBody.Compile,
		Execute:  reqBody.Execute,
	}

	_, err := api.Db.Collection("languages").InsertOne(r.Context(), language)
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	json.NewEncoder(w).Encode(AddLanguageResponse{
		Success: true,
		ID:      language.ID.Hex(),
	})
}

func (api *API) editLanguageHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody EditLanguageRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	if err := reqBody.validate(); err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	objID, err := primitive.ObjectIDFromHex(reqBody.ID)
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	_, err = api.Db.Collection("languages").UpdateOne(r.Context(), bson.M{"_id": bson.M{"$eq": objID}}, bson.M{"$set": bson.M{"name": reqBody.Name, "time": reqBody.Time, "filename": reqBody.Filename, "compile": reqBody.Compile, "execute": reqBody.Execute}})
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	json.NewEncoder(w).Encode(TemplateResponse{
		Success: true,
	})
}

func (api *API) deleteLanguageHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody DeleteLanguageRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	if err := reqBody.validate(); err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	objID, err := primitive.ObjectIDFromHex(reqBody.ID)
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	_, err = api.Db.Collection("languages").DeleteOne(r.Context(), bson.M{"_id": bson.M{"$eq": objID}}, &options.DeleteOptions{})
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	json.NewEncoder(w).Encode(TemplateResponse{
		Success: true,
	})
}
