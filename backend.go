package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/openai/openai-go/v3"
)

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

var VocabularyResponseSchema = GenerateSchema[Vocabularies]()
var AnswerResponseSchema = GenerateSchema[Answers]()

func SelectRandomVocab(db *sqlx.DB) ([]VocabLearn, error) {
	query := `	
	select id, word, count
	from vocab
	where count = (select min(count) from vocab)
	order by random()
	limit 5;
`
	var res []VocabLearn
	err := db.Select(&res, query)
	if err != nil {
		return res, err
	}
	return res, nil
}

func JawabRandomVocab(answereds []Answer) ([]Answer, error) {
	ctx := context.Background()
	client := openai.NewClient()
	var vocabRes Answers
	question := "Verifikasi data jawaban vocab dan aswernya berikut benar atau tidak isikan respon is_correct dan tidak karena kamu adalah linguistics validator"

	answersJSON, err := json.Marshal(answereds)
	if err != nil {
		return vocabRes.Answers, err
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
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &vocabRes)

	if err != nil {
		fmt.Println(err.Error())
	}

	//println(chat.Choices[0].Message.Content)
	db, err := ConnectDB()
	if err != nil {
		return vocabRes.Answers, err
	}
	//ga insert dulu
	for _, vRes := range vocabRes.Answers {
		err = InsertVocabAnswer(db, vRes)
		if err != nil {
			return vocabRes.Answers, err
		}

		if vRes.IsCorrect {
			updateVocabCount(db, vRes.Id)
		}

	}

	return vocabRes.Answers, nil
}

func InsertVocabAnswer(db *sqlx.DB, val Answer) error {
	query := "insert into vocab_answer(id_vocab, jawaban, is_correct, created_at) values (?, ?, ?, ?)"

	datex := time.Now().String()
	_, err := db.Exec(query, val.Id, val.Answer, val.IsCorrect, datex)
	if err != nil {
		return err
	}
	return nil
}

func updateVocabCount(db *sqlx.DB, id int) error {
	query := "update vocab set count = count + 1 where id = ?"
	_, err := db.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

func InsertVocabulary(db *sqlx.DB, theme string, num int) error {
	ctx := context.Background()

	db, err := sqlx.Connect("sqlite3", "data.db")
	if err != nil {
		return err
	}
	curVoc, err := GetCurVocab(db)
	if err != nil {
		return err
	}

	numx := strconv.Itoa(num)
	client := openai.NewClient()
	question := "Buatkan data vocab"

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "english_vocabulary",
		Description: openai.String((numx) + " English Vocabulary dengan tema buku " + theme + " kecuali vocab berikut: " + curVoc),
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

	for _, vRes := range vocabRes.Vocab {
		query := "INSERT INTO vocab (word, pos, core_meaning, common_collocations, example_sentence, register, notes ) values (?, ?, ?, ?, ?, ?, ?)"
		valCC, err := json.Marshal(vRes.CommonCollocations)
		if err != nil {
			return err

		}

		_, err = db.Exec(query, vRes.Word, vRes.POS, vRes.CoreMeaning, string(valCC), vRes.ExampleSentence, vRes.Register, vRes.Notes)
		if err != nil {
			return err
		}
	}

	return nil

}

func GetCurVocab(db *sqlx.DB) (string, error) {
	var res []string

	err := db.Select(&res, "select word from vocab")
	if err != nil {
		return "", err
	}

	wordSJ := strings.Join(res, ", ")
	return wordSJ, nil
}

func AllVocabulary(db *sqlx.DB) ([]VocabularyList, error) {
	var res []VocabularyList
	err := db.Select(&res, "select id, word, core_meaning from vocab;")
	if err != nil {
		return res, err
	}

	return res, nil

}
