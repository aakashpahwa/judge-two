package api

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Question : Structure for the question documents
type Question struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	Time         int                `bson:"time" json:"time"`
	Name         string             `bson:"name" json:"name"`
	NumTestcases int                `bson:"num_testcases" json:"num_testcases"`
}

// Language : Structure for the language documents
type Language struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	Name     string             `bson:"name" json:"name"`
	Time     int                `bson:"time" json:"time"`
	Filename string             `bson:"filename" json:"filename"`
	Compile  string             `bson:"compile" json:"compile"`
	Execute  string             `bson:"execute" json:"execute"`
}

// Submission : Structure for the submission documents
type Submission struct {
	ID         primitive.ObjectID `bson:"_id" json:"id"`
	LanguageID primitive.ObjectID `bson:"lang_id" json:"lang_id"`
	QuestionID primitive.ObjectID `bson:"ques_id" json:"ques_id"`
	Testcases  map[int]string     `bson:"testcases" json:"testcases"`
}

// TemplateResponse : Fields for normal response
type TemplateResponse struct {
	Success bool `json:"success"`
}
