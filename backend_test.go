package main

import (
	"context"
	"fmt"
	"testing"

	"encoding/json"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/openai/openai-go/v3"
)

func TestBackend(t *testing.T) {
	var db *sqlx.DB
	db, err := sqlx.Connect("sqlite3", "data.db")

	err = db.Ping()
	if err != nil {
		t.Error(err.Error())
	}

}

func TestInsertVocab(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Connect("sqlite3", "data.db")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	curVoc, err := GetCurVocab(db)
	if err != nil {
		t.Error(err.Error())
	}

	client := openai.NewClient()
	question := "Buatkan data vocab"

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "english_vocabulary",
		Description: openai.String("10 English Vocabulary dengan tema buku The Ragged Trousered Philanthropists kecuali vocab berikut: " + curVoc),
		Schema:      VocabularyResponseSchema,
		Strict:      openai.Bool(true),
	}

	chat, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(question),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
		Model: openai.ChatModelGPT4o2024_08_06,
	})

	if err != nil {
		panic(err.Error())
	}
	var vocabRes Vocabularies
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &vocabRes)

	if err != nil {
		fmt.Println(err.Error())
	}

	//println(chat.Choices[0].Message.Content)

	for _, vRes := range vocabRes.Vocab {
		query := "INSERT INTO vocab (word, pos, core_meaning, common_collocations, example_sentence, register, notes ) values (?, ?, ?, ?, ?, ?, ?)"
		valCC, err := json.Marshal(vRes.CommonCollocations)
		if err != nil {
			t.Error(err.Error())

		}

		_, err = db.Exec(query, vRes.Word, vRes.POS, vRes.CoreMeaning, string(valCC), vRes.ExampleSentence, vRes.Register, vRes.Notes)
		if err != nil {
			t.Error(err.Error())
		}
	}

}

func TestGetVocab(t *testing.T) {
	db, err := ConnectDB()
	if err != nil {
		t.Error(err.Error())
		return
	}

	res, err := GetCurVocab(db)

	if err != nil {
		t.Error(err.Error())
	}

	fmt.Println(res)
}

func TestAnswer(t *testing.T) {
	ctx := context.Background()

	client := openai.NewClient()
	question := "Verifikasi data jawaban vocab dan aswernya berikut benar atau tidak isikan respon is_correct dan tidak karena kamu adalah linguistics validator"

	var answereds []Answer
	answer1 := Answer{
		Id:        1,
		Word:      "peculiar",
		IsCorrect: false,
		Answer:    "Aneh/orang yang aneh",
	}
	answer2 := Answer{
		Id:        2,
		Word:      "bespoke",
		IsCorrect: false,
		Answer:    "pasaran/ada banyak",
	}

	answereds = append(answereds, answer1)
	answereds = append(answereds, answer2)

	answersJSON, err := json.Marshal(answereds)
	if err != nil {
		t.Error(err.Error())
	}
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "english_vocabulary",
		Description: openai.String("verifikasi jawaban berikut"),
		Schema:      AnswerResponseSchema,
		Strict:      openai.Bool(true),
	}

	chat, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(question),
			openai.UserMessage(fmt.Sprintf("User answers (JSON):%s", answersJSON)),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
		Model: openai.ChatModelGPT4o2024_08_06,
	})

	if err != nil {
		panic(err.Error())
	}
	var vocabRes Answers
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &vocabRes)

	if err != nil {
		fmt.Println(err.Error())
	}

	//println(chat.Choices[0].Message.Content)
	db, err := ConnectDB()
	if err != nil {
		t.Error(err.Error())
	}
	for _, vRes := range vocabRes.Answers {
		err = InsertVocabAnswer(db, vRes)
		if err != nil {
			t.Error(err.Error())
		}
	}

}

func TestSelectRandom(t *testing.T) {
	db, err := ConnectDB()
	if err != nil {
		t.Error(err.Error())
	}

	res, err := SelectRandomVocab(db)
	if err != nil {
		t.Error(err.Error())
	}

	fmt.Println(res)

}

func TestVocabularyList(t *testing.T) {
	db, err := ConnectDB()
	if err != nil {
		t.Error(err.Error())
	}
	var res []VocabularyList
	err = db.Select(&res, "select id, word, core_meaning from vocab;")
	if err != nil {
		t.Error(err.Error())
	}
}
