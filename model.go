package main

type VocabularyRes struct {
	Id                 int    `json:"id"`
	Word               string `json:"word" db:"word" jsonschema:"description=The vocabulary word"`
	POS                string `json:"pos" db:"pos" jsonschema:"enum=verb,enum=noun,enum=adjective,enum=adverb,enum=pronoun,enum=preposition,enum=conjunction,enum=interjection,description=Part of speech"`
	CoreMeaning        string `json:"core_meaning" db:"core_meaning" jsonschema:"description=Arti dalam bahasa Inggris dan Indonesia, concise and conceptual"`
	CommonCollocations string `json:"common_collocations" db:"common_collocations" jsonschema:"description=Common word pairings or phrases"`
	ExampleSentence    string `json:"example_sentence" db:"example_sentence" jsonschema:"description=A natural example sentence using the word"`
	Register           string `json:"register" db:"register" jsonschema:"enum=formal,enum=neutral,enum=informal,enum=technical,enum=academic,description=Usage register"`
	Notes              string `json:"notes" db:"notes" jsonschema:"description=Differences from similar words, synonyms, or personal/contextual notes"`
}

type Vocabulary struct {
	Word               string `json:"word" jsonschema:"description=The vocabulary word"`
	POS                string `json:"pos" jsonschema:"enum=verb,enum=noun,enum=adjective,enum=adverb,enum=pronoun,enum=preposition,enum=conjunction,enum=interjection,description=Part of speech"`
	CoreMeaning        string `json:"core_meaning" jsonschema:"description=Arti dalam bahasa Inggris dan Indonesia, concise and conceptual"`
	CommonCollocations string `json:"common_collocations" jsonschema:"description=Common word pairings or phrases"`
	ExampleSentence    string `json:"example_sentence" jsonschema:"description=A natural example sentence using the word"`
	Register           string `json:"register" jsonschema:"enum=formal,enum=neutral,enum=informal,enum=technical,enum=academic,description=Usage register"`
	Notes              string `json:"notes" jsonschema:"description=Differences from similar words, synonyms, or personal/contextual notes"`
}

type Vocabularies struct {
	Vocab []Vocabulary `json:"vocab"`
}

type Answer struct {
	Id         int    `json:"id" jsonschema:"description=Id of the answer"`
	Word       string `json:"word" jsonschema:"description=The vocabulary word"`
	Answer     string `json:"answer" jsonschema:"description=Answer probably in Indonesia that needed to verify"`
	IsCorrect  bool   `json:"is_correct" jsonschema:"description=Boolean base on the word and answer translation"`
	RealAnswer string `json:"real_answer" jsonschema:"description=String base on The real meaning of word in English or Bahasa Indonesia if the answer is incorrect"`
}

type VocabLearn struct {
	Id    int    `json:"id"`
	Word  string `json:"word"`
	Count int    `json:"count"`
}

type Answers struct {
	Answers []Answer `json:"answers"`
}

type VocabularyList struct {
	Id          int    `json:"id"`
	Word        string `json:"word"`
	CoreMeaning string `json:"core_meaning" db:"core_meaning"`
}
