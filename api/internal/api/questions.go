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

	name := r.FormValue("name")
	if len(name) <= 0 {
		api.Log.Info("Name field missing")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	ID := primitive.NewObjectID()

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
	var inputCasesCount, outputCasesCount int = 0, 0
	var inputItemsCount, outputItemsCount int = 0, 0
	for _, item := range zipr.File {
		fmt.Println(item.Name)
		if item.Name == "input/" || item.Name == "output/" {
			continue
		}
		if validInputFile.Match([]byte(item.Name)) {
			inputItemsCount++
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
			if fileNumber > inputCasesCount {
				inputCasesCount = fileNumber
			}
			targetFile, err := os.OpenFile(folderPath+fmt.Sprintf("input/input%d.txt", fileNumber), os.O_RDWR|os.O_CREATE, 0666)
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
		} else if validOutputFile.Match([]byte(item.Name)) {
			outputItemsCount++
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
			if fileNumber > outputCasesCount {
				outputCasesCount = fileNumber
			}
			targetFile, err := os.OpenFile(folderPath+fmt.Sprintf("output/output%d.txt", fileNumber), os.O_RDWR|os.O_CREATE, 0666)
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

	if inputCasesCount == 0 || inputItemsCount == 0 || outputCasesCount == 0 || outputItemsCount == 0 || inputCasesCount != inputItemsCount || outputCasesCount != outputItemsCount || inputCasesCount != outputCasesCount || inputItemsCount != outputItemsCount {
		os.RemoveAll(folderPath)
		api.Log.Info("Not a valid zip")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TemplateResponse{
			Success: false,
		})
		return
	}

	question := Question{
		ID:           ID,
		Time:         time,
		Name:         name,
		NumTestcases: inputCasesCount,
	}

	_, err = api.Db.Collection("questions").InsertOne(r.Context(), question)
	if err != nil {
		os.RemoveAll(folderPath)
		api.Log.Info(err.Error())
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
}
