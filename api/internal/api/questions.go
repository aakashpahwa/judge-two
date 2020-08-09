package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AddQuestionResponse : ID of the added question
type AddQuestionResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

var validInputFile = regexp.MustCompile(`^input/input([0-9]+)\.([a-zA-Z]+)$`)
var validOutputFile = regexp.MustCompile(`^output/output([0-9]+)\.([a-zA-Z]+)$`)

func (api *API) addQuestionHandler(w http.ResponseWriter, r *http.Request) {
	// Time limit for the question in seconds
	timeStr := r.FormValue("time")
	if len(timeStr) <= 0 {
		api.Log.Info("Time field missing")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	time, err := strconv.Atoi(timeStr)
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	// Name for question
	name := r.FormValue("name")
	if len(name) <= 0 {
		api.Log.Info("Name field missing")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	// ObjectID for the new question
	ID := primitive.NewObjectID()

	// Zip file for the testcases of the new question
	file, handler, err := r.FormFile("testcases")
	defer file.Close()
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	if !strings.HasSuffix(handler.Filename, ".zip") {
		api.Log.Info("Testcases should be a zip file, please refer <link> for more info")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	// Target destination paths for the testcases
	folderPath := fmt.Sprintf("testcases/%s/", ID.Hex())
	filePath := folderPath + handler.Filename
	if err = os.MkdirAll(folderPath, os.ModePerm); err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	if err = os.MkdirAll(folderPath+"input/", os.ModePerm); err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	if err = os.MkdirAll(folderPath+"output/", os.ModePerm); err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	// Target path for saving the zip
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	defer f.Close()
	if err != nil {
		os.RemoveAll(folderPath)
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	_, err = io.Copy(f, file)
	if err != nil {
		os.RemoveAll(folderPath)
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	defer os.Remove(filePath)

	// Zip reader object
	zipr, err := zip.OpenReader(filePath)
	if err != nil {
		os.RemoveAll(folderPath)
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	defer zipr.Close()

	validZip := true
	inputFiles := make(map[int]bool)
	outputFiles := make(map[int]bool)

	for _, item := range zipr.File {
		if item.Name == "input/" || item.Name == "output/" {
			continue
		}
		if validInputFile.Match([]byte(item.Name)) {
			testcaseFile := strings.TrimPrefix(item.Name, "input/input")
			dotIndex := strings.Index(testcaseFile, ".")
			if dotIndex != -1 {
				testcaseFile = testcaseFile[0:dotIndex]
			}
			if len(testcaseFile) <= 0 {
				validZip = false
				break
			}
			fileNumber, err := strconv.Atoi(testcaseFile)
			if err != nil {
				validZip = false
				break
			}
			if _, ok := inputFiles[fileNumber]; ok && inputFiles[fileNumber] {
				validZip = false
				break
			} else {
				inputFiles[fileNumber] = true
			}
		} else if validOutputFile.Match([]byte(item.Name)) {
			testcaseFile := strings.TrimPrefix(item.Name, "output/output")
			dotIndex := strings.Index(testcaseFile, ".")
			if dotIndex != -1 {
				testcaseFile = testcaseFile[0:dotIndex]
			}
			if len(testcaseFile) <= 0 {
				validZip = false
				break
			}
			fileNumber, err := strconv.Atoi(testcaseFile)
			if err != nil {
				validZip = false
				break
			}
			if _, ok := outputFiles[fileNumber]; ok && outputFiles[fileNumber] {
				validZip = false
				break
			} else {
				outputFiles[fileNumber] = true
			}
		} else {
			validZip = false
			break
		}
	}

	if !validZip {
		os.RemoveAll(folderPath)
		api.Log.Info("Not a valid zip")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	json.NewEncoder(w).Encode(AddQuestionResponse{
		Success: true,
		ID:      ID.Hex(),
	})

	api.Log.Info(fmt.Sprintf("Copying files for question %s...", ID.Hex()))

	finalTestcases := make(map[int]int)
	currentTestcase := 0

	for _, item := range zipr.File {
		if item.Name == "input/" || item.Name == "output/" {
			continue
		}
		if validInputFile.Match([]byte(item.Name)) {
			testcaseFile := strings.TrimPrefix(item.Name, "input/input")
			dotIndex := strings.Index(testcaseFile, ".")
			if dotIndex != -1 {
				testcaseFile = testcaseFile[0:dotIndex]
			}
			if len(testcaseFile) <= 0 {
				validZip = false
				break
			}
			fileNumber, err := strconv.Atoi(testcaseFile)
			if err != nil {
				validZip = false
				break
			}
			vali, oki := inputFiles[fileNumber]
			valo, oko := outputFiles[fileNumber]
			if oki && vali && oko && valo {
				currentTestcase++
				finalTestcases[fileNumber] = currentTestcase
			}
		}
	}

	for _, item := range zipr.File {
		if item.Name == "input/" || item.Name == "output/" {
			continue
		}
		if validInputFile.Match([]byte(item.Name)) {
			testcaseFile := strings.TrimPrefix(item.Name, "input/input")
			dotIndex := strings.Index(testcaseFile, ".")
			if dotIndex != -1 {
				testcaseFile = testcaseFile[0:dotIndex]
			}
			if len(testcaseFile) <= 0 {
				validZip = false
				break
			}
			fileNumber, err := strconv.Atoi(testcaseFile)
			if err != nil {
				validZip = false
				break
			}
			if val, ok := finalTestcases[fileNumber]; ok {
				targetFile, err := os.OpenFile(fmt.Sprintf("%sinput/input%d.txt", folderPath, val), os.O_RDWR|os.O_CREATE, 0666)
				if err != nil {
					validZip = false
					break
				}
				srcZipFile, err := item.Open()
				_, err = io.Copy(targetFile, srcZipFile)
				if err != nil {
					validZip = false
					break
				}
				targetFile.Close()
				srcZipFile.Close()
			}
		} else if validOutputFile.Match([]byte(item.Name)) {
			testcaseFile := strings.TrimPrefix(item.Name, "output/output")
			dotIndex := strings.Index(testcaseFile, ".")
			if dotIndex != -1 {
				testcaseFile = testcaseFile[0:dotIndex]
			}
			if len(testcaseFile) <= 0 {
				validZip = false
				break
			}
			fileNumber, err := strconv.Atoi(testcaseFile)
			if err != nil {
				validZip = false
				break
			}
			if val, ok := finalTestcases[fileNumber]; ok {
				targetFile, err := os.OpenFile(fmt.Sprintf("%soutput/output%d.txt", folderPath, val), os.O_RDWR|os.O_CREATE, 0666)
				if err != nil {
					validZip = false
					break
				}
				srcZipFile, err := item.Open()
				_, err = io.Copy(targetFile, srcZipFile)
				if err != nil {
					validZip = false
					break
				}
				targetFile.Close()
				srcZipFile.Close()
			}
		} else {
			validZip = false
			break
		}
	}

	if !validZip {
		os.RemoveAll(folderPath)
		api.Log.Info("Not a valid zip")
		return
	}

	question := Question{
		ID:           ID,
		Time:         time,
		Name:         name,
		NumTestcases: currentTestcase,
	}

	_, err = api.Db.Collection("questions").InsertOne(r.Context(), question)
	if err != nil {
		os.RemoveAll(folderPath)
		api.Log.Info(err.Error())
		return
	}

	api.Log.Info(fmt.Sprintf("Extraction done for question %s...", ID.Hex()))
}

func (api *API) editTestcasesHandler(w http.ResponseWriter, r *http.Request) {
	// ID of the question whose testcases need to be edited
	id := r.FormValue("id")
	if len(id) <= 0 {
		api.Log.Info("Missing id field")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	ID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	// Checking if the question with this ID exists
	singleResult := api.Db.Collection("questions").FindOne(r.Context(), bson.M{"_id": bson.M{"$eq": ID}})
	if singleResult.Err() != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	// Zip file for the testcases of the new question
	file, handler, err := r.FormFile("testcases")
	defer file.Close()
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	if !strings.HasSuffix(handler.Filename, ".zip") {
		api.Log.Info("Testcases should be a zip file, please refer <link> for more info")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	// Target destination paths for the testcases
	folderPath := fmt.Sprintf("testcases/%s/", ID.Hex())
	filePath := folderPath + handler.Filename
	if err = os.MkdirAll(folderPath, os.ModePerm); err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	// Target path for saving the zip
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	defer f.Close()
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	_, err = io.Copy(f, file)
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	defer os.Remove(filePath)

	// Zip reader object
	zipr, err := zip.OpenReader(filePath)
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	defer zipr.Close()

	validZip := true
	inputFiles := make(map[int]bool)
	outputFiles := make(map[int]bool)

	for _, item := range zipr.File {
		if item.Name == "input/" || item.Name == "output/" {
			continue
		}
		if validInputFile.Match([]byte(item.Name)) {
			testcaseFile := strings.TrimPrefix(item.Name, "input/input")
			dotIndex := strings.Index(testcaseFile, ".")
			if dotIndex != -1 {
				testcaseFile = testcaseFile[0:dotIndex]
			}
			if len(testcaseFile) <= 0 {
				validZip = false
				break
			}
			fileNumber, err := strconv.Atoi(testcaseFile)
			if err != nil {
				validZip = false
				break
			}
			if _, ok := inputFiles[fileNumber]; ok && inputFiles[fileNumber] {
				validZip = false
				break
			} else {
				inputFiles[fileNumber] = true
			}
		} else if validOutputFile.Match([]byte(item.Name)) {
			testcaseFile := strings.TrimPrefix(item.Name, "output/output")
			dotIndex := strings.Index(testcaseFile, ".")
			if dotIndex != -1 {
				testcaseFile = testcaseFile[0:dotIndex]
			}
			if len(testcaseFile) <= 0 {
				validZip = false
				break
			}
			fileNumber, err := strconv.Atoi(testcaseFile)
			if err != nil {
				validZip = false
				break
			}
			if _, ok := outputFiles[fileNumber]; ok && outputFiles[fileNumber] {
				validZip = false
				break
			} else {
				outputFiles[fileNumber] = true
			}
		} else {
			validZip = false
			break
		}
	}

	if !validZip {
		api.Log.Info("Not a valid zip")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	json.NewEncoder(w).Encode(TemplateResponse{
		Success: true,
	})

	api.Log.Info(fmt.Sprintf("Copying files for question %s...", ID.Hex()))

	// Cleaning out old testcases
	if err = os.RemoveAll(folderPath + "input/"); err != nil {
		os.RemoveAll(folderPath)
		api.Log.Info(err.Error())
		return
	}
	if err = os.RemoveAll(folderPath + "output/"); err != nil {
		os.RemoveAll(folderPath)
		api.Log.Info(err.Error())
		return
	}
	if err = os.MkdirAll(folderPath+"input/", os.ModePerm); err != nil {
		os.RemoveAll(folderPath)
		api.Log.Info(err.Error())
		return
	}
	if err = os.MkdirAll(folderPath+"output/", os.ModePerm); err != nil {
		os.RemoveAll(folderPath)
		api.Log.Info(err.Error())
		return
	}

	finalTestcases := make(map[int]int)
	currentTestcase := 0

	for _, item := range zipr.File {
		if item.Name == "input/" || item.Name == "output/" {
			continue
		}
		if validInputFile.Match([]byte(item.Name)) {
			testcaseFile := strings.TrimPrefix(item.Name, "input/input")
			dotIndex := strings.Index(testcaseFile, ".")
			if dotIndex != -1 {
				testcaseFile = testcaseFile[0:dotIndex]
			}
			if len(testcaseFile) <= 0 {
				validZip = false
				break
			}
			fileNumber, err := strconv.Atoi(testcaseFile)
			if err != nil {
				validZip = false
				break
			}
			vali, oki := inputFiles[fileNumber]
			valo, oko := outputFiles[fileNumber]
			if oki && vali && oko && valo {
				currentTestcase++
				finalTestcases[fileNumber] = currentTestcase
			}
		}
	}

	for _, item := range zipr.File {
		if item.Name == "input/" || item.Name == "output/" {
			continue
		}
		if validInputFile.Match([]byte(item.Name)) {
			testcaseFile := strings.TrimPrefix(item.Name, "input/input")
			dotIndex := strings.Index(testcaseFile, ".")
			if dotIndex != -1 {
				testcaseFile = testcaseFile[0:dotIndex]
			}
			if len(testcaseFile) <= 0 {
				validZip = false
				break
			}
			fileNumber, err := strconv.Atoi(testcaseFile)
			if err != nil {
				validZip = false
				break
			}
			if val, ok := finalTestcases[fileNumber]; ok {
				targetFile, err := os.OpenFile(fmt.Sprintf("%sinput/input%d.txt", folderPath, val), os.O_RDWR|os.O_CREATE, 0666)
				if err != nil {
					validZip = false
					break
				}
				srcZipFile, err := item.Open()
				_, err = io.Copy(targetFile, srcZipFile)
				if err != nil {
					validZip = false
					break
				}
				targetFile.Close()
				srcZipFile.Close()
			}
		} else if validOutputFile.Match([]byte(item.Name)) {
			testcaseFile := strings.TrimPrefix(item.Name, "output/output")
			dotIndex := strings.Index(testcaseFile, ".")
			if dotIndex != -1 {
				testcaseFile = testcaseFile[0:dotIndex]
			}
			if len(testcaseFile) <= 0 {
				validZip = false
				break
			}
			fileNumber, err := strconv.Atoi(testcaseFile)
			if err != nil {
				validZip = false
				break
			}
			if val, ok := finalTestcases[fileNumber]; ok {
				targetFile, err := os.OpenFile(fmt.Sprintf("%soutput/output%d.txt", folderPath, val), os.O_RDWR|os.O_CREATE, 0666)
				if err != nil {
					validZip = false
					break
				}
				srcZipFile, err := item.Open()
				_, err = io.Copy(targetFile, srcZipFile)
				if err != nil {
					validZip = false
					break
				}
				targetFile.Close()
				srcZipFile.Close()
			}
		} else {
			validZip = false
			break
		}
	}

	if !validZip {
		os.RemoveAll(folderPath)
		api.Log.Info("Not a valid zip")
		return
	}

	_, err = api.Db.Collection("questions").UpdateOne(r.Context(), bson.M{"_id": bson.M{"$eq": ID}}, bson.M{"$set": bson.M{"num_testcases": currentTestcase}})
	if err != nil {
		os.RemoveAll(folderPath)
		api.Log.Info(err.Error())
		return
	}

	api.Log.Info(fmt.Sprintf("Extraction done for question %s...", ID.Hex()))
}

func (api *API) editQuestionHandler(w http.ResponseWriter, r *http.Request) {
	// ID of the question whose testcases need to be edited
	id := r.FormValue("id")
	if len(id) <= 0 {
		api.Log.Info("Missing id field")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	ID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	// Time limit for the question in seconds
	timeStr := r.FormValue("time")
	if len(timeStr) <= 0 {
		api.Log.Info("Time field missing")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	time, err := strconv.Atoi(timeStr)
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	// Name for question
	name := r.FormValue("name")
	if len(name) <= 0 {
		api.Log.Info("Name field missing")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	updateResult, err := api.Db.Collection("questions").UpdateOne(r.Context(), bson.M{"_id": bson.M{"$eq": ID}}, bson.M{"$set": bson.M{"time": time, "name": name}})
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	if updateResult.MatchedCount <= 0 {
		api.Log.Info("No such question with this ID")
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

func (api *API) deleteQuestionHandler(w http.ResponseWriter, r *http.Request) {
	// ID of the question whose testcases need to be edited
	id := r.FormValue("id")
	if len(id) <= 0 {
		api.Log.Info("Missing id field")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	ID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	deleteResult, err := api.Db.Collection("questions").DeleteOne(r.Context(), bson.M{"_id": bson.M{"$eq": ID}})
	if err != nil {
		api.Log.Info(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}
	if deleteResult.DeletedCount <= 0 {
		api.Log.Info("No such question with this ID")
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
