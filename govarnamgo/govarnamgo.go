package govarnamgo

/* Golang bindings for govarnam library */

// #cgo pkg-config: govarnam
// #include "libgovarnam.h"
// #include "stdlib.h"
import "C"

import (
	"context"
	"fmt"
	"unsafe"
)

// Config  config values
type Config struct {
	IndicDigits                bool
	DictionarySuggestionsLimit int
	TokenizerSuggestionsLimit  int
	TokenizerSuggestionsAlways bool
}

// VarnamHandle for making things easier
type VarnamHandle struct {
	connectionID C.int
}

// Suggestion suggestion
type Suggestion struct {
	Word      string
	Weight    int
	LearnedOn int
}

// TransliterationResult result
type TransliterationResult struct {
	ExactMatches                 []Suggestion
	DictionarySuggestions        []Suggestion
	PatternDictionarySuggestions []Suggestion
	TokenizerSuggestions         []Suggestion
	GreedyTokenized              []Suggestion
}

// SchemeDetails of VST
type SchemeDetails struct {
	Identifier   string
	LangCode     string
	DisplayName  string
	Author       string
	CompiledDate string
	IsStable     bool
}

// LearnStatus output of bulk learn
type LearnStatus struct {
	TotalWords  int
	FailedWords int
}

// Symbol result from VST
type Symbol struct {
	Identifier      int
	Type            int
	MatchType       int
	Pattern         string
	Value1          string
	Value2          string
	Value3          string
	Tag             string
	Weight          int
	Priority        int
	AcceptCondition int
	Flags           int
}

// Convert a C Suggestion to Go
func makeSuggestion(cSug *C.struct_Suggestion_t) Suggestion {
	var sug Suggestion
	sug.Word = C.GoString(cSug.Word)
	sug.Weight = int(cSug.Weight)
	sug.LearnedOn = int(cSug.LearnedOn)

	return sug
}

func makeGoTransliterationResult(ctx context.Context, cResult *C.struct_TransliterationResult_t) TransliterationResult {
	var result TransliterationResult

	select {
	case <-ctx.Done():
		return result
	default:
		var i int

		var exactMatches []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.ExactMatches)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.ExactMatches, C.int(i)))
			sug := makeSuggestion(cSug)
			exactMatches = append(exactMatches, sug)
			i++
		}
		result.ExactMatches = exactMatches

		var dictionarySuggestions []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.DictionarySuggestions)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.DictionarySuggestions, C.int(i)))
			sug := makeSuggestion(cSug)
			dictionarySuggestions = append(dictionarySuggestions, sug)
			i++
		}
		result.DictionarySuggestions = dictionarySuggestions

		var patternDictionarySuggestions []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.PatternDictionarySuggestions)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.PatternDictionarySuggestions, C.int(i)))
			sug := makeSuggestion(cSug)
			patternDictionarySuggestions = append(patternDictionarySuggestions, sug)
			i++
		}
		result.PatternDictionarySuggestions = patternDictionarySuggestions

		var tokenizerSuggestions []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.TokenizerSuggestions)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.TokenizerSuggestions, C.int(i)))
			sug := makeSuggestion(cSug)
			tokenizerSuggestions = append(tokenizerSuggestions, sug)
			i++
		}
		result.TokenizerSuggestions = tokenizerSuggestions

		var greedyTokenized []Suggestion
		i = 0
		for i < int(C.varray_length(cResult.GreedyTokenized)) {
			cSug := (*C.Suggestion)(C.varray_get(cResult.GreedyTokenized, C.int(i)))
			sug := makeSuggestion(cSug)
			greedyTokenized = append(greedyTokenized, sug)
			i++
		}
		result.GreedyTokenized = greedyTokenized

		go C.destroyTransliterationResult(cResult)

		return result
	}
}

//VarnamError Custom error for varnam
type VarnamError struct {
	errorCode int
	message   string
}

// Error mimicking error package's function
func (err *VarnamError) Error() string {
	return err.message
}

func (handle *VarnamHandle) checkError(code C.int) *VarnamError {
	if code == C.VARNAM_SUCCESS {
		return nil
	}
	return &VarnamError{int(code), handle.GetLastError()}
}

// InitFromID Initialize
func InitFromID(id string) (*VarnamHandle, error) {
	handleID := C.int(0)
	cID := C.CString(id)
	err := C.varnam_init_from_id(cID, unsafe.Pointer(&handleID))
	C.free(unsafe.Pointer(cID))

	if err != C.VARNAM_SUCCESS {
		return nil, fmt.Errorf(C.GoString(C.varnam_get_last_error(handleID)))
	}
	return &VarnamHandle{handleID}, nil
}

// GetLastError get last error
func (handle *VarnamHandle) GetLastError() string {
	cStr := C.varnam_get_last_error(handle.connectionID)
	goStr := C.GoString(cStr)
	C.free(unsafe.Pointer(cStr))
	return goStr
}

// Close db connections and end varnam
func (handle *VarnamHandle) Close() *VarnamError {
	err := C.varnam_close(handle.connectionID)
	return handle.checkError(err)
}

// Debug turn debug on/off
func (handle *VarnamHandle) Debug(val bool) {
	if val {
		C.varnam_debug(handle.connectionID, C.int(1))
	} else {
		C.varnam_debug(handle.connectionID, C.int(0))
	}
}

// SetConfig set config
func (handle *VarnamHandle) SetConfig(config Config) {
	C.varnam_set_dictionary_suggestions_limit(handle.connectionID, C.int(config.DictionarySuggestionsLimit))

	C.varnam_set_tokenizer_suggestions_limit(handle.connectionID, C.int(config.TokenizerSuggestionsLimit))

	if config.IndicDigits {
		C.varnam_set_indic_digits(handle.connectionID, C.int(1))
	} else {
		C.varnam_set_indic_digits(handle.connectionID, C.int(0))
	}
}

func (handle *VarnamHandle) cgoGetTransliterationResult(operationID C.int, resultChannel chan<- *C.struct_TransliterationResult_t, word string) {
	cWord := C.CString(word)
	defer C.free(unsafe.Pointer(cWord))
	cResult := C.varnam_transliterate_with_id(handle.connectionID, operationID, cWord)

	resultChannel <- cResult
	close(resultChannel)
}

var transliterationOperationCount = C.int(0)

// Transliterate transilterate
func (handle *VarnamHandle) Transliterate(ctx context.Context, word string) TransliterationResult {
	operationID := transliterationOperationCount
	transliterationOperationCount++

	channel := make(chan *C.struct_TransliterationResult_t)

	go handle.cgoGetTransliterationResult(operationID, channel, word)

	select {
	case <-ctx.Done():
		C.varnam_cancel(operationID)
		var result TransliterationResult
		return result
	case cResult := <-channel:
		return makeGoTransliterationResult(ctx, cResult)
	}
}

// ReverseTransliterate reverse transilterate
func (handle *VarnamHandle) ReverseTransliterate(word string) ([]Suggestion, error) {
	var sugs []Suggestion
	cWord := C.CString(word)

	result := C.varnam_reverse_transliterate(handle.connectionID, cWord)
	if result == nil {
		return sugs, fmt.Errorf(handle.GetLastError())
	}

	i := 0
	for i < int(C.varray_length(result)) {
		cSug := (*C.Suggestion)(C.varray_get(result, C.int(i)))
		sug := makeSuggestion(cSug)
		sugs = append(sugs, sug)
		i++
	}
	return sugs, nil
}

// Train train a pattern => word
func (handle *VarnamHandle) Train(pattern string, word string) *VarnamError {
	cPattern := C.CString(pattern)
	cWord := C.CString(word)

	err := C.varnam_train(handle.connectionID, cPattern, cWord)

	C.free(unsafe.Pointer(cPattern))
	C.free(unsafe.Pointer(cWord))

	return handle.checkError(err)
}

// Learn a word
func (handle *VarnamHandle) Learn(word string, weight int) *VarnamError {
	cWord := C.CString(word)

	err := C.varnam_learn(handle.connectionID, cWord, C.int(weight))

	C.free(unsafe.Pointer(cWord))

	return handle.checkError(err)
}

// Unlearn a word
func (handle *VarnamHandle) Unlearn(word string) *VarnamError {
	cWord := C.CString(word)

	err := C.varnam_unlearn(handle.connectionID, cWord)

	C.free(unsafe.Pointer(cWord))

	return handle.checkError(err)
}

// LearnFromFile learn words from a file
func (handle *VarnamHandle) LearnFromFile(filePath string) (LearnStatus, *VarnamError) {
	var learnStatus LearnStatus

	cFilePath := C.CString(filePath)

	cLearnStatus := C.varnam_learn_from_file(handle.connectionID, cFilePath)
	if cLearnStatus == nil {
		return learnStatus, &VarnamError{int(C.VARNAM_ERROR), handle.GetLastError()}
	}

	learnStatus = LearnStatus{int(cLearnStatus.TotalWords), int(cLearnStatus.FailedWords)}

	return learnStatus, nil
}

// TrainFromFile train pattern => word from a file
func (handle *VarnamHandle) TrainFromFile(filePath string) *VarnamError {
	cFilePath := C.CString(filePath)
	err := C.varnam_train_from_file(handle.connectionID, cFilePath)
	return handle.checkError(err)
}

// Export learnigns to a file
func (handle *VarnamHandle) Export(filePath string) *VarnamError {
	cFilePath := C.CString(filePath)
	err := C.varnam_export(handle.connectionID, cFilePath)
	return handle.checkError(err)
}

// Import learnigns to a file
func (handle *VarnamHandle) Import(filePath string) *VarnamError {
	cFilePath := C.CString(filePath)
	err := C.varnam_import(handle.connectionID, cFilePath)
	return handle.checkError(err)
}

// GetVSTPath Get path to VST of current handle
func (handle *VarnamHandle) GetVSTPath() string {
	cStr := C.varnam_get_vst_path(handle.connectionID)
	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr)
}

var searchSymbolTableCount = C.int(0)

// SearchSymbolTable search VST
func (handle *VarnamHandle) SearchSymbolTable(searchCriteria Symbol) []Symbol {
	Identifier := C.int(searchCriteria.Identifier)
	Type := C.int(searchCriteria.Type)
	MatchType := C.int(searchCriteria.MatchType)
	Pattern := C.CString(searchCriteria.Pattern)
	Value1 := C.CString(searchCriteria.Value1)
	Value2 := C.CString(searchCriteria.Value2)
	Value3 := C.CString(searchCriteria.Value3)
	Tag := C.CString(searchCriteria.Tag)
	Weight := C.int(searchCriteria.Weight)
	Priority := C.int(searchCriteria.Priority)
	AcceptCondition := C.int(searchCriteria.AcceptCondition)
	Flags := C.int(searchCriteria.Flags)

	symbol := C.makeSymbol(Identifier, Type, MatchType, Pattern, Value1, Value2, Value3, Tag, Weight, Priority, AcceptCondition, Flags)

	cResult := C.varnam_search_symbol_table(handle.connectionID, searchSymbolTableCount, *symbol)

	var goResults []Symbol

	if cResult == nil {
		return goResults
	}

	i := 0
	for i < int(C.varray_length(cResult)) {
		result := (*C.Symbol)(C.varray_get(cResult, C.int(i)))

		var goResult Symbol
		goResult.Identifier = int(result.Identifier)
		goResult.Type = int(result.Type)
		goResult.MatchType = int(result.MatchType)
		goResult.Pattern = C.GoString(result.Pattern)
		goResult.Value1 = C.GoString(result.Value1)
		goResult.Value2 = C.GoString(result.Value2)
		goResult.Value3 = C.GoString(result.Value3)
		goResult.Tag = C.GoString(result.Tag)
		goResult.Weight = int(result.Weight)
		goResult.Priority = int(result.Priority)
		goResult.AcceptCondition = int(result.AcceptCondition)
		goResult.Flags = int(result.Flags)

		goResults = append(goResults, goResult)
		i++
	}

	return goResults
}

// GetVSTDir Get path to directory containging the VSTs
func GetVSTDir() string {
	cStr := C.varnam_get_vst_dir()
	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr)
}

// GetAllSchemeDetails get all available scheme details. The bool is for error
func GetAllSchemeDetails() ([]SchemeDetails, bool) {
	cSchemeDetails := C.varnam_get_all_scheme_details()

	if cSchemeDetails == nil {
		return nil, true
	}

	var schemeDetails []SchemeDetails
	i := 0
	for i < int(C.varray_length(cSchemeDetails)) {
		cSD := (*C.SchemeDetails)(C.varray_get(cSchemeDetails, C.int(i)))

		isStable := true
		if cSD.IsStable == 0 {
			isStable = false
		}

		sd := SchemeDetails{
			C.GoString(cSD.Identifier),
			C.GoString(cSD.LangCode),
			C.GoString(cSD.DisplayName),
			C.GoString(cSD.Author),
			C.GoString(cSD.CompiledDate),
			isStable,
		}

		schemeDetails = append(schemeDetails, sd)
		i++
	}

	go C.destroySchemeDetailsArray(unsafe.Pointer(cSchemeDetails))

	return schemeDetails, false
}
